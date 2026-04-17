package serverfs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type stubArtifacts struct {
	installed map[string]bool
}

func (s stubArtifacts) IsInstalled(version string) bool {
	return s.installed[version]
}

// noopCipher satisfies FieldCipher without performing real crypto. It is only
// used in update-path tests that exercise license rotation; the encrypted
// blob is opaque to the registry so a deterministic echo is enough.
type noopCipher struct{}

func (noopCipher) Decrypt(ciphertext []byte) ([]byte, error) { return ciphertext, nil }
func (noopCipher) Encrypt(plaintext []byte) ([]byte, error)  { return plaintext, nil }

// newTestRegistry builds a registry rooted in t.TempDir() and pre-populates
// it with the supplied server configs so tests can exercise Update / Delete
// against realistic entries without rebuilding the Create path each time.
func newTestRegistry(t *testing.T, seeds ...ServerConfig) *Registry {
	t.Helper()

	root := t.TempDir()
	artifacts := stubArtifacts{installed: map[string]bool{}}

	for _, s := range seeds {
		artifacts.installed[s.ArtifactVersion] = true
		dir := filepath.Join(root, sanitizeDirName(s.Name))
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir seed: %v", err)
		}
		if err := writeServerConfig(filepath.Join(dir, configFilename), &s); err != nil {
			t.Fatalf("write seed: %v", err)
		}
	}

	reg, err := NewRegistry(root, artifacts, noopCipher{})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	return reg
}

const testArtifactVersion = "9000"

func baseConfig(name string, port int) ServerConfig {
	return ServerConfig{
		Name:            name,
		ArtifactVersion: testArtifactVersion,
		Network: NetworkConfig{
			Port:       port,
			MaxClients: 32,
		},
		Gameplay: GameplayConfig{OneSync: "on"},
		Resources: ResourcesConfig{
			Ensure: []string{"chat"},
		},
	}
}

func TestUpdate_NameRenameKeepsDirectoryID(t *testing.T) {
	reg := newTestRegistry(t, baseConfig("Alpha", 30120))
	id := "Alpha"

	newName := "Alpha Reloaded"
	updated, err := reg.Update(id, &UpdatePatch{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if updated.ID != id {
		t.Fatalf("expected stable ID %q, got %q", id, updated.ID)
	}
	if updated.Name != newName {
		t.Fatalf("expected name %q, got %q", newName, updated.Name)
	}

	if _, err := os.Stat(filepath.Join(reg.rootDir, id, configFilename)); err != nil {
		t.Fatalf("expected config at stable dir, got err: %v", err)
	}
}

func TestUpdate_NameCannotBeClearedOrWhitespace(t *testing.T) {
	reg := newTestRegistry(t, baseConfig("Alpha", 30120))

	empty := ""
	if _, err := reg.Update("Alpha", &UpdatePatch{Name: &empty}); err == nil {
		t.Fatal("expected error clearing name")
	}

	whitespace := "   "
	if _, err := reg.Update("Alpha", &UpdatePatch{Name: &whitespace}); err == nil {
		t.Fatal("expected error for whitespace-only name")
	}
}

func TestUpdate_ArtifactVersionRequiresInstall(t *testing.T) {
	reg := newTestRegistry(t, baseConfig("Alpha", 30120))

	missing := "9001"
	if _, err := reg.Update("Alpha", &UpdatePatch{ArtifactVersion: &missing}); err == nil {
		t.Fatal("expected error for uninstalled artifact")
	}
}

func TestUpdate_PortReallocationAvoidsOwnSlot(t *testing.T) {
	reg := newTestRegistry(t,
		baseConfig("Alpha", 30120),
		baseConfig("Bravo", 30121),
	)

	zero := 0
	updated, err := reg.Update("Alpha", &UpdatePatch{Port: &zero})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if updated.Port == 0 {
		t.Fatal("expected allocator to assign a concrete port")
	}
	if updated.Port == 30121 {
		t.Fatal("allocator picked a port already claimed by another server")
	}
}

func TestUpdate_PortCollisionDowngradesLaterEntry(t *testing.T) {
	reg := newTestRegistry(t,
		baseConfig("Alpha", 30120),
		baseConfig("Bravo", 30121),
	)

	clash := 30120
	_, err := reg.Update("Bravo", &UpdatePatch{Port: &clash})
	// The write succeeds; the cached entry is marked invalid on reload.
	if err == nil {
		if _, ok := reg.Get("Bravo"); ok {
			t.Fatal("expected Bravo to be flagged invalid after port clash")
		}
	}
}

func TestUpdate_OneSyncAllowList(t *testing.T) {
	reg := newTestRegistry(t, baseConfig("Alpha", 30120))

	bad := "ultra"
	if _, err := reg.Update("Alpha", &UpdatePatch{OneSync: &bad}); err == nil {
		t.Fatal("expected error for invalid onesync value")
	}

	good := "legacy"
	if _, err := reg.Update("Alpha", &UpdatePatch{OneSync: &good}); err != nil {
		t.Fatalf("unexpected error for allowed onesync: %v", err)
	}

	disabled := ""
	if _, err := reg.Update("Alpha", &UpdatePatch{OneSync: &disabled}); err != nil {
		t.Fatalf("empty onesync should be accepted (disables CLI arg): %v", err)
	}
}

func TestUpdate_ResourcesEnsureReplacesList(t *testing.T) {
	reg := newTestRegistry(t, baseConfig("Alpha", 30120))

	next := []string{"spawnmanager", "sessionmanager"}
	updated, err := reg.Update("Alpha", &UpdatePatch{ResourcesEnsure: &next})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Round-trip through disk so we also catch the write-read path.
	reread, err := readServerConfig(filepath.Join(reg.rootDir, updated.ID, configFilename))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(reread.Resources.Ensure) != 2 || reread.Resources.Ensure[0] != "spawnmanager" {
		t.Fatalf("unexpected resources list: %#v", reread.Resources.Ensure)
	}

	empty := []string{}
	if _, err := reg.Update("Alpha", &UpdatePatch{ResourcesEnsure: &empty}); err != nil {
		t.Fatalf("empty list should clear: %v", err)
	}
	reread, err = readServerConfig(filepath.Join(reg.rootDir, "Alpha", configFilename))
	if err != nil {
		t.Fatalf("read back after clear: %v", err)
	}
	if len(reread.Resources.Ensure) != 0 {
		t.Fatalf("expected empty resources list, got %#v", reread.Resources.Ensure)
	}
}

func TestUpdate_LicenseKeyLifecycle(t *testing.T) {
	reg := newTestRegistry(t, baseConfig("Alpha", 30120))

	bad := "not-a-license"
	if _, err := reg.Update("Alpha", &UpdatePatch{LicenseKey: &bad}); err == nil {
		t.Fatal("expected rejection of license key without cfxk_ prefix")
	}

	good := "cfxk_xxxxxxx"
	if _, err := reg.Update("Alpha", &UpdatePatch{LicenseKey: &good}); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	cfg, err := readServerConfig(filepath.Join(reg.rootDir, "Alpha", configFilename))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if cfg.License.KeyEncrypted == "" {
		t.Fatal("expected encrypted license to be persisted")
	}

	cleared := ""
	if _, err := reg.Update("Alpha", &UpdatePatch{LicenseKey: &cleared}); err != nil {
		t.Fatalf("clear: %v", err)
	}
	cfg, err = readServerConfig(filepath.Join(reg.rootDir, "Alpha", configFilename))
	if err != nil {
		t.Fatalf("read back after clear: %v", err)
	}
	if cfg.License.KeyEncrypted != "" {
		t.Fatalf("expected cleared license, got %q", cfg.License.KeyEncrypted)
	}
}

func TestUpdate_UnknownServerReturnsError(t *testing.T) {
	reg := newTestRegistry(t)
	name := "ghost"
	if _, err := reg.Update("does-not-exist", &UpdatePatch{Name: &name}); err == nil {
		t.Fatal("expected error for unknown server")
	}
}

func TestDelete_TrashMovesDirectory(t *testing.T) {
	reg := newTestRegistry(t, baseConfig("Alpha", 30120))

	if err := reg.Delete("Alpha", true); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, ok := reg.Get("Alpha"); ok {
		t.Fatal("expected Alpha to be evicted from cache after delete")
	}
	if _, err := os.Stat(filepath.Join(reg.rootDir, "Alpha")); !os.IsNotExist(err) {
		t.Fatalf("expected original dir gone, stat err = %v", err)
	}

	trashed, err := os.ReadDir(filepath.Join(reg.rootDir, trashDirName))
	if err != nil {
		t.Fatalf("read trash dir: %v", err)
	}
	if len(trashed) != 1 || !strings.HasPrefix(trashed[0].Name(), "Alpha.") {
		t.Fatalf("expected Alpha.<timestamp> in trash, got %v", trashed)
	}
}

func TestDelete_PermanentRemovesDirectory(t *testing.T) {
	reg := newTestRegistry(t, baseConfig("Alpha", 30120))

	if err := reg.Delete("Alpha", false); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := os.Stat(filepath.Join(reg.rootDir, "Alpha")); !os.IsNotExist(err) {
		t.Fatalf("expected dir removed, got err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(reg.rootDir, trashDirName)); !os.IsNotExist(err) {
		t.Fatal("permanent delete must not create a trash dir")
	}
}

func TestDelete_TrashDirIgnoredByReload(t *testing.T) {
	reg := newTestRegistry(t, baseConfig("Alpha", 30120))

	if err := reg.Delete("Alpha", true); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Reload explicitly to confirm the trash dir is still invisible on a
	// full rescan, which is what the fsnotify watcher will trigger.
	if err := reg.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if servers, _ := reg.List(); len(servers) != 0 {
		t.Fatalf("expected empty list after trash, got %d entries", len(servers))
	}
}

func TestDelete_UnknownServerReturnsError(t *testing.T) {
	reg := newTestRegistry(t)
	if err := reg.Delete("ghost", true); err == nil {
		t.Fatal("expected error for unknown server")
	}
}
