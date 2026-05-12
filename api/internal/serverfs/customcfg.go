package serverfs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// customCfgFile is the operator-owned cfg slot that lives next to the
	// generated server.cfg. Created empty on first reload and never
	// overwritten by the panel afterwards.
	customCfgFile = "custom.cfg"

	// CustomCfgMaxBytes caps PUT payloads at 256 KiB. fxserver itself does
	// not impose a meaningful limit but anything beyond this size strongly
	// suggests the operator is putting data in the wrong file (rules of
	// thumb: convars + ACEs + a handful of starts ≪ 10 KiB). Exported so
	// the HTTP layer can surface the cap to the UI without a magic number.
	CustomCfgMaxBytes = 256 * 1024
)

// customCfgHeader is written exactly once, when the panel first creates the
// file. Subsequent edits round-trip the user's content verbatim — including
// the case where the operator deletes the header.
const customCfgHeader = `# runfive — custom server.cfg overrides
#
# This file is auto-loaded by the generated server.cfg via:
#     exec "configurations/custom.cfg"
#
# The panel will NEVER overwrite this file. Reloads of server.toml leave it
# untouched. Anything you write here runs AFTER the panel-managed
# hostname / port / maxclients / license / build directives, and BEFORE the
# resources listed under [resources.ensure] in server.toml are started.
#
# Typical contents:
#     set sv_scriptHookAllowed 0
#     add_ace group.admin command allow
#     add_principal identifier.steam:0000000000000000 group.admin
#     start my_custom_resource
`

// CustomCfg is the on-disk snapshot of a server's custom.cfg.
type CustomCfg struct {
	// Content is the raw file body, exactly as it lives on disk.
	Content string
	// SizeBytes is len(Content) at the time of read.
	SizeBytes int
	// UpdatedAt is the last modification time of custom.cfg, or zero if the
	// file was just created by ensureCustomCfgExists this tick.
	UpdatedAt time.Time
}

// ErrCustomCfgTooLarge is returned when a PUT body exceeds CustomCfgMaxBytes.
var ErrCustomCfgTooLarge = fmt.Errorf("custom.cfg payload exceeds %d bytes", CustomCfgMaxBytes)

// ErrCustomCfgNullByte is returned when the body contains a NUL byte. fxserver
// uses C-style string handling for cfg parsing and a stray NUL truncates
// everything that follows, which is a footgun we'd rather reject at the
// boundary than silently honour.
var ErrCustomCfgNullByte = errors.New("custom.cfg must not contain NUL bytes")

// customCfgPath resolves the on-disk path for the given server directory.
func customCfgPath(serverDir string) string {
	return filepath.Join(serverDir, configurationsDir, customCfgFile)
}

// ensureCustomCfgExists creates an empty custom.cfg with the standard header
// when one is missing. Idempotent: if the file is already present (even
// empty) the call is a no-op. The atomic write pattern matches
// writeGeneratedServerCfg so a crash mid-write can never leave a partial
// file behind.
func ensureCustomCfgExists(serverDir string) error {
	path := customCfgPath(serverDir)
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("stat custom.cfg: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create configurations dir: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(customCfgHeader), 0o600); err != nil {
		return fmt.Errorf("write custom.cfg: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("finalize custom.cfg: %w", err)
	}
	return nil
}

// readCustomCfg returns the current snapshot. A missing file is reported as
// an empty body rather than an error — callers can decide whether to seed it
// via ensureCustomCfgExists or simply render nothing.
func readCustomCfg(serverDir string) (CustomCfg, error) {
	path := customCfgPath(serverDir)
	//nolint:gosec // path is rooted at the registry and never user-supplied.
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return CustomCfg{}, nil
		}
		return CustomCfg{}, fmt.Errorf("read custom.cfg: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return CustomCfg{}, fmt.Errorf("stat custom.cfg: %w", err)
	}
	return CustomCfg{
		Content:   string(data),
		SizeBytes: len(data),
		UpdatedAt: info.ModTime(),
	}, nil
}

// writeCustomCfg replaces the file body with content. The caller is expected
// to have validated permissions; this layer enforces size and content-safety
// only. Atomic via temp-file + rename so a concurrent reader never sees a
// partial write.
func writeCustomCfg(serverDir, content string) (CustomCfg, error) {
	if len(content) > CustomCfgMaxBytes {
		return CustomCfg{}, ErrCustomCfgTooLarge
	}
	if strings.IndexByte(content, 0) >= 0 {
		return CustomCfg{}, ErrCustomCfgNullByte
	}

	path := customCfgPath(serverDir)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return CustomCfg{}, fmt.Errorf("create configurations dir: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
		return CustomCfg{}, fmt.Errorf("write custom.cfg: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return CustomCfg{}, fmt.Errorf("finalize custom.cfg: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return CustomCfg{}, fmt.Errorf("stat custom.cfg after write: %w", err)
	}
	return CustomCfg{
		Content:   content,
		SizeBytes: len(content),
		UpdatedAt: info.ModTime(),
	}, nil
}
