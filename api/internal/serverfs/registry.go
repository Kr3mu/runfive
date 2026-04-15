// Package serverfs manages the on-disk representation of panel-owned servers.
package serverfs

import (
	"context"
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

// Registry manages the file-backed set of server configurations and keeps an
// in-memory cache in sync via an fsnotify watcher.
type Registry struct {
	rootDir   string
	artifacts artifactLookup
	dec       FieldDecryptor

	mu      sync.RWMutex
	entries map[string]*entry
}

// NewRegistry creates a registry rooted at rootDir and performs an initial
// load. Call StartWatcher separately once the caller is ready to consume
// filesystem events so tests can opt out. The decryptor is used to render
// generated server.cfg files containing sensitive values such as the license
// key; pass nil if no encrypted fields are in use.
func NewRegistry(rootDir string, artifacts artifactLookup, dec FieldDecryptor) (*Registry, error) {
	r := &Registry{
		rootDir:   rootDir,
		artifacts: artifacts,
		dec:       dec,
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

		if cfg.Network.Port > 0 {
			if other, clash := portOwners[cfg.Network.Port]; clash {
				reason := fmt.Sprintf("port %d already claimed by %q", cfg.Network.Port, other)
				log.Printf("[serverfs] %s: %s", id, reason)
				next[id] = &entry{id: id, config: cfg, invalid: true, reason: reason}
				if existing, ok := next[other]; ok {
					existing.invalid = true
					existing.reason = fmt.Sprintf("port %d also claimed by %q", cfg.Network.Port, id)
				}
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
		if err := writeGeneratedServerCfg(serverDir, &e.config, r.dec); err != nil {
			log.Printf("[serverfs] %s: generate server.cfg: %v", id, err)
		}
	}

	return nil
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
			if ev.Op == fsnotify.Chmod {
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
// and refreshes the cache.
func (r *Registry) Create(name, artifactVersion string) (models.ManagedServer, error) {
	name = strings.TrimSpace(name)
	artifactVersion = strings.TrimSpace(artifactVersion)

	if name == "" {
		return models.ManagedServer{}, fmt.Errorf("server name is required")
	}
	if artifactVersion == "" {
		return models.ManagedServer{}, fmt.Errorf("artifact version is required")
	}
	if r.artifacts != nil && !r.artifacts.IsInstalled(artifactVersion) {
		return models.ManagedServer{}, fmt.Errorf("artifact version %s is not installed", artifactVersion)
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
	if err := writeServerConfig(filepath.Join(serverDir, configFilename), &cfg); err != nil {
		return models.ManagedServer{}, err
	}

	if err := r.Reload(); err != nil {
		return models.ManagedServer{}, fmt.Errorf("reload after create: %w", err)
	}

	srv, ok := r.Get(dirID)
	if !ok {
		return models.ManagedServer{}, fmt.Errorf("server %s missing from cache after create", dirID)
	}
	return srv, nil
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
		PlayerCount:     0,
		MaxPlayers:      e.config.Network.MaxClients,
		CPU:             0,
		RamMB:           0,
		TickMs:          0,
		ArtifactVersion: e.config.ArtifactVersion,
	}
}
