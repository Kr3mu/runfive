package serverfs

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Kr3mu/runfive/internal/models"
)

const configFilename = "server.toml"

type artifactLookup interface {
	IsInstalled(version string) bool
}

// Registry manages server configs stored on disk.
type Registry struct {
	rootDir   string
	artifacts artifactLookup
}

// Config is the minimal persisted server config used by v1.
type Config struct {
	Name            string
	ArtifactVersion string
}

// NewRegistry creates a filesystem-backed server registry.
func NewRegistry(rootDir string, artifacts artifactLookup) *Registry {
	return &Registry{rootDir: rootDir, artifacts: artifacts}
}

// List reads all server.toml files under the servers root.
func (r *Registry) List() ([]models.ManagedServer, error) {
	entries, err := os.ReadDir(r.rootDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []models.ManagedServer{}, nil
		}
		return nil, fmt.Errorf("list servers: %w", err)
	}

	servers := make([]models.ManagedServer, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		config, err := readConfig(filepath.Join(r.rootDir, entry.Name(), configFilename))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}

		servers = append(servers, models.ManagedServer{
			ID:              entry.Name(),
			Name:            config.Name,
			Status:          models.ServerStatusStopped,
			Address:         "",
			PlayerCount:     0,
			MaxPlayers:      0,
			CPU:             0,
			RamMB:           0,
			TickMs:          0,
			ArtifactVersion: config.ArtifactVersion,
		})
	}

	sort.Slice(servers, func(i, j int) bool {
		return strings.ToLower(servers[i].Name) < strings.ToLower(servers[j].Name)
	})

	return servers, nil
}

// Create persists a new server folder and its server.toml config.
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

	if err := writeConfig(filepath.Join(serverDir, configFilename), Config{
		Name:            name,
		ArtifactVersion: artifactVersion,
	}); err != nil {
		return models.ManagedServer{}, err
	}

	return models.ManagedServer{
		ID:              dirID,
		Name:            name,
		Status:          models.ServerStatusStopped,
		Address:         "",
		PlayerCount:     0,
		MaxPlayers:      0,
		CPU:             0,
		RamMB:           0,
		TickMs:          0,
		ArtifactVersion: artifactVersion,
	}, nil
}

// ArtifactReferences returns all server IDs pointing at the given artifact version.
func (r *Registry) ArtifactReferences(version string) ([]string, error) {
	servers, err := r.List()
	if err != nil {
		return nil, err
	}

	refs := make([]string, 0)
	for _, server := range servers {
		if server.ArtifactVersion == version {
			refs = append(refs, server.ID)
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

func readConfig(path string) (Config, error) {
	//nolint:gosec // path is constructed from the registry root plus discovered server directories.
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close server config: %w", closeErr)
		}
	}()

	var cfg Config
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			continue
		}

		switch key {
		case "name":
			cfg.Name = unquoted
		case "artifact_version":
			cfg.ArtifactVersion = unquoted
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("read server config: %w", err)
	}
	if cfg.Name == "" {
		return Config{}, fmt.Errorf("read server config %s: missing name", path)
	}
	if cfg.ArtifactVersion == "" {
		return Config{}, fmt.Errorf("read server config %s: missing artifact_version", path)
	}
	return cfg, nil
}

func writeConfig(path string, cfg Config) error {
	tempPath := path + ".tmp"
	content := fmt.Sprintf("name = %s\nartifact_version = %s\n", strconv.Quote(cfg.Name), strconv.Quote(cfg.ArtifactVersion))
	if err := os.WriteFile(tempPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write server config: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("finalize server config: %w", err)
	}
	return nil
}
