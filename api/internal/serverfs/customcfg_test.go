package serverfs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReload_SeedsEmptyCustomCfg(t *testing.T) {
	reg := newTestRegistry(t, baseConfig("Alpha", 30120))

	path := filepath.Join(reg.rootDir, "Alpha", configurationsDir, customCfgFile)
	data, err := os.ReadFile(path) //nolint:gosec // path is rooted at t.TempDir() and built from internal constants.
	if err != nil {
		t.Fatalf("expected custom.cfg to be seeded on first reload, got err: %v", err)
	}
	if !strings.HasPrefix(string(data), "# runfive") {
		t.Fatalf("expected default header on seeded custom.cfg, got: %q", string(data))
	}
}

func TestReload_DoesNotOverwriteExistingCustomCfg(t *testing.T) {
	reg := newTestRegistry(t, baseConfig("Alpha", 30120))
	path := filepath.Join(reg.rootDir, "Alpha", configurationsDir, customCfgFile)

	const userBody = "# my edits\nset sv_scriptHookAllowed 0\nadd_ace group.admin command allow\n"
	if err := os.WriteFile(path, []byte(userBody), 0o600); err != nil {
		t.Fatalf("seed user body: %v", err)
	}

	if err := reg.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	got, err := os.ReadFile(path) //nolint:gosec // path is rooted at t.TempDir() and built from internal constants.
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != userBody {
		t.Fatalf("Reload overwrote operator-owned custom.cfg.\nwant:\n%s\ngot:\n%s", userBody, string(got))
	}
}

func TestRenderServerCfg_ContainsCustomCfgExec(t *testing.T) {
	cfg := baseConfig("Alpha", 30120)
	rendered, warnings := renderServerCfg(&cfg, "")
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if !strings.Contains(rendered, `exec "configurations/custom.cfg"`) {
		t.Fatalf("server.cfg missing custom.cfg exec line:\n%s", rendered)
	}

	// exec must precede ensure, so resources see the final convar set.
	execIdx := strings.Index(rendered, `exec "configurations/custom.cfg"`)
	ensureIdx := strings.Index(rendered, "ensure chat")
	if execIdx < 0 || ensureIdx < 0 {
		t.Fatalf("missing expected directives:\n%s", rendered)
	}
	if execIdx > ensureIdx {
		t.Fatalf("exec line must appear before ensure block, got exec=%d ensure=%d\n%s", execIdx, ensureIdx, rendered)
	}
}

func TestRenderServerCfg_ExecLineEmittedEvenWithoutEnsures(t *testing.T) {
	cfg := baseConfig("Alpha", 30120)
	cfg.Resources.Ensure = nil
	rendered, _ := renderServerCfg(&cfg, "")
	if !strings.Contains(rendered, `exec "configurations/custom.cfg"`) {
		t.Fatalf("exec line should be emitted regardless of ensure list:\n%s", rendered)
	}
}

func TestWriteCustomCfg_RoundTrip(t *testing.T) {
	reg := newTestRegistry(t, baseConfig("Alpha", 30120))

	const body = "set sv_hostname \"override\"\n"
	written, err := reg.WriteCustomCfg("Alpha", body)
	if err != nil {
		t.Fatalf("WriteCustomCfg: %v", err)
	}
	if written.Content != body || written.SizeBytes != len(body) {
		t.Fatalf("unexpected write snapshot: %#v", written)
	}

	read, err := reg.ReadCustomCfg("Alpha")
	if err != nil {
		t.Fatalf("ReadCustomCfg: %v", err)
	}
	if read.Content != body {
		t.Fatalf("round-trip mismatch.\nwant: %q\ngot:  %q", body, read.Content)
	}
}

func TestWriteCustomCfg_RejectsTooLarge(t *testing.T) {
	reg := newTestRegistry(t, baseConfig("Alpha", 30120))

	oversized := strings.Repeat("x", CustomCfgMaxBytes+1)
	if _, err := reg.WriteCustomCfg("Alpha", oversized); err == nil {
		t.Fatal("expected size-cap error")
	}
}

func TestWriteCustomCfg_RejectsNullByte(t *testing.T) {
	reg := newTestRegistry(t, baseConfig("Alpha", 30120))

	if _, err := reg.WriteCustomCfg("Alpha", "set foo bar\x00\n"); err == nil {
		t.Fatal("expected NUL-byte rejection")
	}
}

func TestWriteCustomCfg_UnknownServer(t *testing.T) {
	reg := newTestRegistry(t)
	if _, err := reg.WriteCustomCfg("ghost", "anything"); err == nil {
		t.Fatal("expected error for unknown server")
	}
}

func TestReadCustomCfg_MissingFileReturnsEmptySnapshot(t *testing.T) {
	reg := newTestRegistry(t, baseConfig("Alpha", 30120))
	path := filepath.Join(reg.rootDir, "Alpha", configurationsDir, customCfgFile)
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove seeded custom.cfg: %v", err)
	}

	snap, err := reg.ReadCustomCfg("Alpha")
	if err != nil {
		t.Fatalf("ReadCustomCfg: %v", err)
	}
	if snap.Content != "" || snap.SizeBytes != 0 {
		t.Fatalf("expected empty snapshot for missing file, got %#v", snap)
	}
}

func TestCustomCfg_NotWatchedByReload(t *testing.T) {
	// custom.cfg lives under configurations/, which is not added to the
	// fsnotify watch list — but Reload itself must still be idempotent
	// against operator edits there.
	reg := newTestRegistry(t, baseConfig("Alpha", 30120))
	path := filepath.Join(reg.rootDir, "Alpha", configurationsDir, customCfgFile)

	for i := 0; i < 3; i += 1 {
		if err := os.WriteFile(path, []byte("set foo bar\n"), 0o600); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
		if err := reg.Reload(); err != nil {
			t.Fatalf("reload %d: %v", i, err)
		}
		got, err := os.ReadFile(path) //nolint:gosec // path is rooted at t.TempDir() and built from internal constants.
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		if string(got) != "set foo bar\n" {
			t.Fatalf("iteration %d: panel rewrote custom.cfg: %q", i, string(got))
		}
	}
}
