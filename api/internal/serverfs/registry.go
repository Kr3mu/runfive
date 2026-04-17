// Package serverfs manages the on-disk representation of panel-owned servers.
package serverfs

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pelletier/go-toml/v2"

	"github.com/runfivedev/runfive/internal/models"
)

const (
	configFilename = "server.toml"
	debounceDelay  = 500 * time.Millisecond
)

// ServerConfig is the on-disk schema for a single server's configuration.
type ServerConfig struct {
	Name            string          `toml:"name"`
	ArtifactVersion string          `toml:"artifact_version"`
	Network         NetworkConfig   `toml:"network"`
	License         LicenseConfig   `toml:"license"`
	Gameplay        GameplayConfig  `toml:"gameplay"`
	Resources       ResourcesConfig `toml:"resources"`
}

// NetworkConfig maps to the [network] table.
type NetworkConfig struct {
	Port             int    `toml:"port"`
	MaxClients       int    `toml:"max_clients"`
	EnforceGameBuild string `toml:"enforce_game_build,omitempty"`
}

// LicenseConfig holds the panel-encrypted fxserver license key. The plaintext
// never touches this struct — KeyEncrypted stores the base64 ciphertext
// produced by auth.FieldEncryptor and is treated as opaque by the registry.
type LicenseConfig struct {
	KeyEncrypted string `toml:"key_encrypted"`
}

// GameplayConfig holds values the launcher forwards to fxserver as CLI args.
// OneSync in particular must live on the command line — setting it via
// server.cfg is deprecated in fxserver as of 2026.
type GameplayConfig struct {
	OneSync string `toml:"onesync"`
}

// ResourcesConfig lists the resources the launcher should ensure on start.
type ResourcesConfig struct {
	Ensure []string `toml:"ensure"`
}

// LaunchSpec contains the resolved filesystem/runtime inputs required to start
// one managed server process.
type LaunchSpec struct {
	ID              string
	Name            string
	ServerDir       string
	ConfigPath      string
	ArtifactVersion string
	OneSync         string
}

type artifactLookup interface {
	IsInstalled(version string) bool
}

// entry is one row of the in-memory cache. Invalid entries are kept so the
// admin UI can surface the failure reason instead of silently hiding the dir.
type entry struct {
	id      string
	config  ServerConfig
	invalid bool
	reason  string
}

// WarningHandler receives non-fatal messages produced while rendering the
// generated server.cfg (e.g. a TOML value was rejected by the sanitizer).
// The handler runs synchronously from the reload goroutine and MUST NOT
// block or call back into the Registry, or it will deadlock the reload.
type WarningHandler func(serverID, message string)

// Registry manages the file-backed set of server configurations and keeps an
// in-memory cache in sync via an fsnotify watcher.
type Registry struct {
	rootDir   string
	artifacts artifactLookup
	cipher    FieldCipher

	mu       sync.RWMutex
	entries  map[string]*entry
	onWarn   WarningHandler
	onWarnMu sync.RWMutex
}

// NewRegistry creates a registry rooted at rootDir and performs an initial
// load. Call StartWatcher separately once the caller is ready to consume
// filesystem events so tests can opt out. The cipher is used both to render
// generated server.cfg files (decrypt) and to accept new secrets on create
// (encrypt); pass nil if no encrypted fields are in use.
func NewRegistry(rootDir string, artifacts artifactLookup, cipher FieldCipher) (*Registry, error) {
	r := &Registry{
		rootDir:   rootDir,
		artifacts: artifacts,
		cipher:    cipher,
		entries:   make(map[string]*entry),
	}
	if err := r.Reload(); err != nil {
		return nil, err
	}
	return r, nil
}

// Reload rebuilds the in-memory cache from disk. Individual parse errors and
// port collisions mark the affected entries as invalid but never abort the
// whole reload — one bad config does not take down the registry.
func (r *Registry) Reload() error {
	dirEntries, err := os.ReadDir(r.rootDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			r.mu.Lock()
			r.entries = make(map[string]*entry)
			r.mu.Unlock()
			return nil
		}
		return fmt.Errorf("read servers dir: %w", err)
	}

	next := make(map[string]*entry, len(dirEntries))
	portOwners := make(map[int]string)

	for _, de := range dirEntries {
		if !de.IsDir() {
			continue
		}
		id := de.Name()
		path := filepath.Join(r.rootDir, id, configFilename)

		cfg, err := readServerConfig(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			log.Printf("[serverfs] %s: %v", id, err)
			next[id] = &entry{id: id, invalid: true, reason: err.Error()}
			continue
		}

		// Only the later-scanned entry is marked invalid on a port collision.
		// os.ReadDir returns entries in lexical order, so the first-alphabetical
		// server keeps its slot — operators can then rename or delete the
		// duplicate without their working server getting demoted along with it.
		if cfg.Network.Port > 0 {
			if other, clash := portOwners[cfg.Network.Port]; clash {
				reason := fmt.Sprintf("port %d already claimed by %q", cfg.Network.Port, other)
				log.Printf("[serverfs] %s: %s", id, reason)
				next[id] = &entry{id: id, config: cfg, invalid: true, reason: reason}
				continue
			}
			portOwners[cfg.Network.Port] = id
		}

		next[id] = &entry{id: id, config: cfg}
	}

	r.mu.Lock()
	r.entries = next
	r.mu.Unlock()

	for id, e := range next {
		if e.invalid {
			continue
		}
		serverDir := filepath.Join(r.rootDir, id)
		warnings, err := writeGeneratedServerCfg(serverDir, &e.config, r.cipher)
		if err != nil {
			log.Printf("[serverfs] %s: generate server.cfg: %v", id, err)
		}
		for _, w := range warnings {
			log.Printf("[serverfs] %s: %s", id, w)
			r.emitWarning(id, w)
		}
	}

	return nil
}

// SetWarningHandler installs a callback invoked once per non-fatal
// server.cfg rendering warning. Pass nil to clear. Safe to call at any time.
func (r *Registry) SetWarningHandler(fn WarningHandler) {
	r.onWarnMu.Lock()
	r.onWarn = fn
	r.onWarnMu.Unlock()
}

func (r *Registry) emitWarning(serverID, message string) {
	r.onWarnMu.RLock()
	fn := r.onWarn
	r.onWarnMu.RUnlock()
	if fn != nil {
		fn(serverID, message)
	}
}

// StartWatcher begins watching the servers directory in a goroutine. Events
// are debounced into full reloads — at our scale a full rescan is cheaper
// than tracking per-file deltas. Failures are soft: logged and ignored so
// the panel still boots with the cached registry.
func (r *Registry) StartWatcher(ctx context.Context) {
	if err := os.MkdirAll(r.rootDir, 0o750); err != nil {
		log.Printf("[serverfs] watcher: ensure root dir: %v", err)
		return
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("[serverfs] watcher: create: %v", err)
		return
	}

	if err := w.Add(r.rootDir); err != nil {
		log.Printf("[serverfs] watcher: watch %s: %v", r.rootDir, err)
		_ = w.Close()
		return
	}

	r.addSubdirWatches(w)
	go r.watchLoop(ctx, w)
}

func (r *Registry) addSubdirWatches(w *fsnotify.Watcher) {
	subs, err := os.ReadDir(r.rootDir)
	if err != nil {
		return
	}
	for _, sub := range subs {
		if !sub.IsDir() {
			continue
		}
		if err := w.Add(filepath.Join(r.rootDir, sub.Name())); err != nil {
			log.Printf("[serverfs] watcher: add %s: %v", sub.Name(), err)
		}
	}
}

func (r *Registry) watchLoop(ctx context.Context, w *fsnotify.Watcher) {
	defer func() { _ = w.Close() }()

	rootClean := filepath.Clean(r.rootDir)

	var pending *time.Timer
	trigger := func() {
		if err := r.Reload(); err != nil {
			log.Printf("[serverfs] reload: %v", err)
			return
		}
		r.addSubdirWatches(w)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-w.Events:
			if !ok {
				return
			}
			if !isRelevantEvent(ev, rootClean) {
				continue
			}
			if pending != nil {
				pending.Stop()
			}
			pending = time.AfterFunc(debounceDelay, trigger)
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			log.Printf("[serverfs] watcher error: %v", err)
		}
	}
}

// isRelevantEvent drops fsnotify noise produced by fxserver at runtime —
// cache files, logs, lockfiles inside a server directory. We only want to
// reload when a server.toml actually changes or when a server subdirectory
// is added/removed at the root.
func isRelevantEvent(ev fsnotify.Event, rootClean string) bool {
	if ev.Op == fsnotify.Chmod {
		return false
	}
	if strings.HasSuffix(ev.Name, ".toml") {
		return true
	}
	// Top-level Create/Remove/Rename = a server directory appeared or
	// disappeared. We still need to reload so the watch list stays in sync.
	if filepath.Dir(ev.Name) == rootClean &&
		(ev.Has(fsnotify.Create) || ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename)) {
		return true
	}
	return false
}

// List returns a sorted snapshot of valid entries in API-facing form.
func (r *Registry) List() ([]models.ManagedServer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	servers := make([]models.ManagedServer, 0, len(r.entries))
	for _, e := range r.entries {
		if e.invalid {
			continue
		}
		servers = append(servers, toManagedServer(e))
	}
	sort.Slice(servers, func(i, j int) bool {
		return strings.ToLower(servers[i].Name) < strings.ToLower(servers[j].Name)
	})
	return servers, nil
}

// Get returns a single valid entry by directory id.
func (r *Registry) Get(id string) (models.ManagedServer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.entries[id]
	if !ok || e.invalid {
		return models.ManagedServer{}, false
	}
	return toManagedServer(e), true
}

// LaunchSpec returns the resolved launch inputs for a valid server entry.
func (r *Registry) LaunchSpec(id string) (LaunchSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.entries[id]
	if !ok || e.invalid {
		return LaunchSpec{}, false
	}

	serverDir := filepath.Join(r.rootDir, id)
	return LaunchSpec{
		ID:              id,
		Name:            e.config.Name,
		ServerDir:       serverDir,
		ConfigPath:      filepath.Join(serverDir, configurationsDir, generatedCfgFile),
		ArtifactVersion: e.config.ArtifactVersion,
		OneSync:         strings.TrimSpace(e.config.Gameplay.OneSync),
	}, true
}

// HasServer reports whether id currently exists as a valid entry. The
// permission middleware uses this to fail-closed on orphaned RBAC references.
func (r *Registry) HasServer(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.entries[id]
	return ok && !e.invalid
}

// Create allocates a new server directory, writes the default server.toml,
// and refreshes the cache. licenseKey is the optional plaintext Cfx.re key
// (cfxk_...) which is encrypted with the registry cipher before being
// persisted. port is the TCP/UDP endpoint port — pass 0 to let the registry
// pick the next free slot. maxPlayers is the sv_maxclients value — pass 0
// to use the panel default. Pass an empty licenseKey to create a server
// without a license key.
func (r *Registry) Create(name, artifactVersion, licenseKey string, port, maxPlayers int) (models.ManagedServer, error) {
	name = strings.TrimSpace(name)
	artifactVersion = strings.TrimSpace(artifactVersion)
	licenseKey = strings.TrimSpace(licenseKey)

	if name == "" {
		return models.ManagedServer{}, fmt.Errorf("server name is required")
	}
	if artifactVersion == "" {
		return models.ManagedServer{}, fmt.Errorf("artifact version is required")
	}
	if r.artifacts != nil && !r.artifacts.IsInstalled(artifactVersion) {
		return models.ManagedServer{}, fmt.Errorf("artifact version %s is not installed", artifactVersion)
	}

	resolvedPort, err := r.resolvePort(port)
	if err != nil {
		return models.ManagedServer{}, err
	}

	resolvedMax, err := resolveMaxPlayers(maxPlayers)
	if err != nil {
		return models.ManagedServer{}, err
	}

	encryptedLicense, err := r.encryptLicense(licenseKey)
	if err != nil {
		return models.ManagedServer{}, err
	}

	if err := os.MkdirAll(r.rootDir, 0o750); err != nil {
		return models.ManagedServer{}, fmt.Errorf("create servers root: %w", err)
	}

	dirID, err := r.allocateDirID(name)
	if err != nil {
		return models.ManagedServer{}, err
	}

	serverDir := filepath.Join(r.rootDir, dirID)
	if err := os.MkdirAll(serverDir, 0o750); err != nil {
		return models.ManagedServer{}, fmt.Errorf("create server dir: %w", err)
	}

	cfg := defaultServerConfig(name, artifactVersion)
	cfg.License.KeyEncrypted = encryptedLicense
	cfg.Network.Port = resolvedPort
	cfg.Network.MaxClients = resolvedMax
	if err := writeServerConfig(filepath.Join(serverDir, configFilename), &cfg); err != nil {
		return models.ManagedServer{}, err
	}

	if err := r.Reload(); err != nil {
		return models.ManagedServer{}, fmt.Errorf("reload after create: %w", err)
	}

	// Surface the new entry even if the reload immediately flagged it as
	// invalid (e.g. a deliberate port-collision override). The HTTP caller
	// still wants a success response; the invalid reason will show up in
	// the next List() call so the operator can act on it.
	r.mu.RLock()
	e, ok := r.entries[dirID]
	r.mu.RUnlock()
	if !ok {
		return models.ManagedServer{}, fmt.Errorf("server %s missing from cache after create", dirID)
	}
	return toManagedServer(e), nil
}

// encryptLicense converts a plaintext Cfx.re key to the base64 AES-GCM blob
// stored in server.toml. An empty key returns "" (no key persisted). A key
// without the `cfxk_` prefix is rejected so typos don't silently produce an
// unbootable server.
func (r *Registry) encryptLicense(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	if !strings.HasPrefix(plain, "cfxk_") {
		return "", fmt.Errorf("license key must start with cfxk_")
	}
	if r.cipher == nil {
		return "", fmt.Errorf("license key provided but no field cipher is configured")
	}
	ciphertext, err := r.cipher.Encrypt([]byte(plain))
	if err != nil {
		return "", fmt.Errorf("encrypt license key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// ArtifactReferences returns server IDs pointing at the given artifact version.
func (r *Registry) ArtifactReferences(version string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	refs := make([]string, 0)
	for id, e := range r.entries {
		if e.config.ArtifactVersion == version {
			refs = append(refs, id)
		}
	}
	sort.Strings(refs)
	return refs, nil
}

func (r *Registry) allocateDirID(name string) (string, error) {
	base := sanitizeDirName(name)
	candidate := base

	for i := 2; ; i++ {
		_, err := os.Stat(filepath.Join(r.rootDir, candidate))
		if errors.Is(err, fs.ErrNotExist) {
			return candidate, nil
		}
		if err != nil {
			return "", fmt.Errorf("check server dir collision: %w", err)
		}
		candidate = base + "_" + strconv.Itoa(i)
	}
}

var (
	whitespaceRe  = regexp.MustCompile(`\s+`)
	unsupportedRe = regexp.MustCompile(`[^A-Za-z0-9_-]+`)
	multiUndersRe = regexp.MustCompile(`_+`)
)

// defaultBasePort is where the auto-allocator starts looking for a free slot.
// 30120 is the fxserver convention and matches what human-edited configs use.
const defaultBasePort = 30120

// Port range excludes privileged ports (<1024) so operators don't accidentally
// ask fxserver to bind 22 or 80. The upper bound is the TCP/UDP maximum.
const (
	minUserPort = 1024
	maxUserPort = 65535
)

// Max-players bounds: fxserver supports up to 2048 slots, anything higher
// silently gets clamped. Zero is reserved for "use the default".
const (
	defaultMaxPlayers = 32
	minMaxPlayers     = 1
	maxMaxPlayers     = 2048
)

// resolveMaxPlayers validates a caller-supplied slot count or falls back to
// the panel default when the value is zero.
func resolveMaxPlayers(value int) (int, error) {
	if value == 0 {
		return defaultMaxPlayers, nil
	}
	if value < minMaxPlayers || value > maxMaxPlayers {
		return 0, fmt.Errorf("max players must be between %d and %d", minMaxPlayers, maxMaxPlayers)
	}
	return value, nil
}

// allocatePort picks the next unused port at or above defaultBasePort. Only
// ports currently held by another entry are skipped — a server that has been
// deleted frees its port immediately on the next reload.
func (r *Registry) allocatePort() int {
	taken := r.takenPorts()
	for port := defaultBasePort; port <= maxUserPort; port++ {
		if _, clash := taken[port]; !clash {
			return port
		}
	}
	return defaultBasePort
}

// resolvePort validates a caller-supplied port (range only) or falls back to
// allocatePort when port is zero. Port collisions are NOT rejected here —
// the UI warns the operator and requires an explicit second click to
// acknowledge the conflict; the reload pass downgrades the later entry to
// invalid so the duplicate is still visibly flagged in the dashboard.
func (r *Registry) resolvePort(port int) (int, error) {
	if port == 0 {
		return r.allocatePort(), nil
	}
	if port < minUserPort || port > maxUserPort {
		return 0, fmt.Errorf("port %d is outside the allowed range %d-%d", port, minUserPort, maxUserPort)
	}
	return port, nil
}

// takenPorts returns a map of currently claimed ports → owning server name,
// for both conflict detection and auto-allocation. Invalid entries are
// included so we don't paper over a pre-existing collision.
func (r *Registry) takenPorts() map[int]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	taken := make(map[int]string, len(r.entries))
	for _, e := range r.entries {
		if e.config.Network.Port > 0 {
			taken[e.config.Network.Port] = e.config.Name
		}
	}
	return taken
}

func sanitizeDirName(name string) string {
	sanitized := strings.TrimSpace(name)
	sanitized = whitespaceRe.ReplaceAllString(sanitized, "_")
	sanitized = unsupportedRe.ReplaceAllString(sanitized, "_")
	sanitized = multiUndersRe.ReplaceAllString(sanitized, "_")
	sanitized = strings.Trim(sanitized, "_-.")
	if sanitized == "" {
		return "server"
	}
	return sanitized
}

func defaultServerConfig(name, artifactVersion string) ServerConfig {
	return ServerConfig{
		Name:            name,
		ArtifactVersion: artifactVersion,
		Network: NetworkConfig{
			Port:       30120,
			MaxClients: 32,
		},
		Gameplay: GameplayConfig{
			OneSync: "on",
		},
		Resources: ResourcesConfig{
			Ensure: []string{"mapmanager", "chat", "spawnmanager", "sessionmanager", "hardcap"},
		},
	}
}

const tomlHeader = "# runfive server config — managed by the panel\n" +
	"# hand-edits may be rewritten on every panel write\n\n"

func readServerConfig(path string) (ServerConfig, error) {
	//nolint:gosec // path is constructed from the registry root plus discovered server directories.
	data, err := os.ReadFile(path)
	if err != nil {
		return ServerConfig{}, err
	}
	var cfg ServerConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return ServerConfig{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.Name == "" {
		return ServerConfig{}, fmt.Errorf("%s: missing name", path)
	}
	if cfg.ArtifactVersion == "" {
		return ServerConfig{}, fmt.Errorf("%s: missing artifact_version", path)
	}
	return cfg, nil
}

func writeServerConfig(path string, cfg *ServerConfig) error {
	body, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal server config: %w", err)
	}

	out := append([]byte(tomlHeader), body...)

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return fmt.Errorf("write server config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("finalize server config: %w", err)
	}
	return nil
}

func toManagedServer(e *entry) models.ManagedServer {
	return models.ManagedServer{
		ID:              e.id,
		Name:            e.config.Name,
		Status:          models.ServerStatusStopped,
		Address:         "",
		Port:            e.config.Network.Port,
		PlayerCount:     0,
		MaxPlayers:      e.config.Network.MaxClients,
		CPU:             0,
		RamMB:           0,
		TickMs:          0,
		ArtifactVersion: e.config.ArtifactVersion,
	}
}
