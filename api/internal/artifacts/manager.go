package artifacts

import (
	"archive/tar"
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Kr3mu/runfive/internal/fxserver"
	"github.com/Kr3mu/runfive/internal/models"
)

type upstreamClient interface {
	HostOS() string
	ArchiveExtension() string
	DownloadURL(tag string) string
	ListVersions(context.Context) ([]string, error)
	ResolveTag(context.Context, string) (string, error)
}

// Manager coordinates shared artifact installs on disk.
type Manager struct {
	rootDir        string
	upstream       upstreamClient
	downloadArchive func(context.Context, string, string) error
	extractArchive  func(string, string) error

	lockMu sync.Mutex
	locks  map[string]*sync.Mutex
}

// NewManager creates an artifact manager for the configured root directory.
func NewManager(rootDir string) (*Manager, error) {
	client, err := fxserver.NewClient()
	if err != nil {
		return nil, err
	}

	return &Manager{
		rootDir:         rootDir,
		upstream:        client,
		downloadArchive: downloadToFile,
		extractArchive: func(archivePath, dest string) error {
			return extractArchive(client.HostOS(), archivePath, dest)
		},
		locks: make(map[string]*sync.Mutex),
	}, nil
}

// HostOS returns the artifact tree managed on this machine.
func (m *Manager) HostOS() string {
	return m.upstream.HostOS()
}

// ListInstalled returns extracted versions already present on disk.
func (m *Manager) ListInstalled() ([]models.InstalledArtifact, error) {
	root := m.osRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []models.InstalledArtifact{}, nil
		}
		return nil, fmt.Errorf("list installed artifacts: %w", err)
	}

	installed := make([]models.InstalledArtifact, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		version := entry.Name()
		installed = append(installed, models.InstalledArtifact{
			OS:      m.HostOS(),
			Version: version,
			Path:    filepath.Join(root, version),
		})
	}

	sortInstalled(installed)
	return installed, nil
}

// ListAvailable returns all upstream versions, annotated with local install state.
func (m *Manager) ListAvailable(ctx context.Context) ([]models.AvailableArtifactVersion, error) {
	versions, err := m.upstream.ListVersions(ctx)
	if err != nil {
		return nil, err
	}

	installed, err := m.ListInstalled()
	if err != nil {
		return nil, err
	}

	installedSet := make(map[string]bool, len(installed))
	for _, item := range installed {
		installedSet[item.Version] = true
	}

	available := make([]models.AvailableArtifactVersion, 0, len(versions))
	for _, version := range versions {
		available = append(available, models.AvailableArtifactVersion{
			Version:   version,
			Installed: installedSet[version],
		})
	}

	return available, nil
}

// IsInstalled returns true when the version directory already exists.
func (m *Manager) IsInstalled(version string) bool {
	info, err := os.Stat(m.installedDir(version))
	return err == nil && info.IsDir()
}

// Install downloads and extracts a version if it is not already installed.
func (m *Manager) Install(ctx context.Context, version string) (models.InstalledArtifact, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return models.InstalledArtifact{}, fmt.Errorf("artifact version is required")
	}

	lock := m.lockFor(version)
	lock.Lock()
	defer lock.Unlock()

	if m.IsInstalled(version) {
		return m.installedRecord(version), nil
	}

	tag, err := m.upstream.ResolveTag(ctx, version)
	if err != nil {
		return models.InstalledArtifact{}, err
	}

	stagingRoot := filepath.Join(m.rootDir, ".tmp", m.HostOS(), fmt.Sprintf("%s-%d", version, time.Now().UnixNano()))
	extractRoot := filepath.Join(stagingRoot, "files")
	archivePath := filepath.Join(stagingRoot, "artifact"+m.upstream.ArchiveExtension())
	finalDir := m.installedDir(version)

	if err := os.MkdirAll(extractRoot, 0o755); err != nil {
		return models.InstalledArtifact{}, fmt.Errorf("create staging dir: %w", err)
	}
	defer os.RemoveAll(stagingRoot)

	if err := m.downloadArchive(ctx, m.upstream.DownloadURL(tag), archivePath); err != nil {
		return models.InstalledArtifact{}, err
	}

	if err := m.extractArchive(archivePath, extractRoot); err != nil {
		return models.InstalledArtifact{}, err
	}

	if err := os.MkdirAll(filepath.Dir(finalDir), 0o755); err != nil {
		return models.InstalledArtifact{}, fmt.Errorf("create artifact OS dir: %w", err)
	}

	if err := os.Rename(extractRoot, finalDir); err != nil {
		if m.IsInstalled(version) {
			return m.installedRecord(version), nil
		}
		return models.InstalledArtifact{}, fmt.Errorf("finalize artifact install: %w", err)
	}

	return m.installedRecord(version), nil
}

// Delete removes an installed artifact directory.
func (m *Manager) Delete(version string) error {
	lock := m.lockFor(version)
	lock.Lock()
	defer lock.Unlock()

	if !m.IsInstalled(version) {
		return fs.ErrNotExist
	}

	if err := os.RemoveAll(m.installedDir(version)); err != nil {
		return fmt.Errorf("delete artifact: %w", err)
	}
	return nil
}

func (m *Manager) osRoot() string {
	return filepath.Join(m.rootDir, m.HostOS())
}

func (m *Manager) installedDir(version string) string {
	return filepath.Join(m.osRoot(), version)
}

func (m *Manager) installedRecord(version string) models.InstalledArtifact {
	return models.InstalledArtifact{
		OS:      m.HostOS(),
		Version: version,
		Path:    m.installedDir(version),
	}
}

func (m *Manager) lockFor(version string) *sync.Mutex {
	m.lockMu.Lock()
	defer m.lockMu.Unlock()

	lock, ok := m.locks[version]
	if !ok {
		lock = &sync.Mutex{}
		m.locks[version] = lock
	}
	return lock
}

func sortInstalled(items []models.InstalledArtifact) {
	sort.Slice(items, func(i, j int) bool {
		left, leftErr := strconv.Atoi(items[i].Version)
		right, rightErr := strconv.Atoi(items[j].Version)
		if leftErr == nil && rightErr == nil {
			return left > right
		}
		return items[i].Version > items[j].Version
	})
}

func downloadToFile(ctx context.Context, url string, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("download artifact: build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download artifact: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download artifact: unexpected status %d", resp.StatusCode)
	}

	file, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("download artifact: create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("download artifact: write file: %w", err)
	}

	return nil
}

func extractArchive(hostOS, archivePath, dest string) error {
	switch hostOS {
	case "windows":
		return extractZip(archivePath, dest)
	case "linux":
		return extractTarXZ(archivePath, dest)
	default:
		return fmt.Errorf("extract artifact: unsupported host OS %q", hostOS)
	}
}

func extractZip(archivePath, dest string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("extract zip: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		target, err := secureJoin(dest, file.Name)
		if err != nil {
			return err
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("extract zip: mkdir %s: %w", target, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("extract zip: mkdir parent: %w", err)
		}

		in, err := file.Open()
		if err != nil {
			return fmt.Errorf("extract zip: open file: %w", err)
		}

		mode := file.Mode()
		if mode == 0 {
			mode = 0o644
		}

		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			in.Close()
			return fmt.Errorf("extract zip: create target: %w", err)
		}

		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			in.Close()
			return fmt.Errorf("extract zip: copy: %w", err)
		}

		out.Close()
		in.Close()
	}

	return nil
}

func extractTarXZ(archivePath, dest string) error {
	if _, err := exec.LookPath("tar"); err != nil {
		return fmt.Errorf("extract tar.xz: tar not available: %w", err)
	}

	cmd := exec.Command("tar", "-xJf", archivePath, "-C", dest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("extract tar.xz: %w: %s", err, strings.TrimSpace(string(output)))
	}

	// A quick walk forces early surfacing of traversal issues from weird archives.
	return filepath.WalkDir(dest, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		_, err := secureJoin(dest, strings.TrimPrefix(strings.TrimPrefix(path, dest), string(filepath.Separator)))
		return err
	})
}

func secureJoin(root, name string) (string, error) {
	cleanName := filepath.Clean(name)
	target := filepath.Join(root, cleanName)
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", fmt.Errorf("invalid archive path %q: %w", name, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid archive path %q", name)
	}
	return target, nil
}

// Compile-time guard against accidentally dropping the tar import on refactors.
var _ = tar.TypeDir
