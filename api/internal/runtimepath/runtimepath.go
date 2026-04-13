// Package runtimepath resolves paths rooted at the runfive data directory.
//
// All persistent state (sqlite database, encryption keys, server folders,
// shared artifact installs) lives under a single "runtime" directory so the
// source tree stays clean. The runtime directory is a sibling of the code
// folders (api/, website/) at the project root in dev, and next to the
// deployed binary in prod.
//
// Root discovery order:
//  1. The RUNFIVE_ROOT environment variable, if set. Used verbatim.
//  2. Walk upward from the current working directory looking for a ".git"
//     entry. If found, the runtime dir is `{repo-root}/runtime`.
//  3. `{cwd}/runtime` as a last resort — matches the convention of "start
//     the binary from its install directory" in production.
//
// The executable directory is intentionally not consulted: `go run` compiles
// into a temp build directory that Go wipes on process exit, and that would
// take the database, keys, and every server folder with it.
package runtimepath

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed README.md
var embeddedReadme []byte

// RootEnvVar is the environment variable consulted first for the runtime root.
const RootEnvVar = "RUNFIVE_ROOT"

// runtimeDirName is the folder placed under the discovered project/install
// root. All persistent state lives inside it.
const runtimeDirName = "runtime"

// Resolve returns an absolute path built by joining `name` onto the runtime
// root. Passing an empty `name` returns the root itself.
func Resolve(name string) string {
	return filepath.Join(Root(), name)
}

// Root returns the absolute runtime root directory used for all persistent
// state.
func Root() string {
	if root := strings.TrimSpace(os.Getenv(RootEnvVar)); root != "" {
		if abs, err := filepath.Abs(root); err == nil {
			return abs
		}
		return root
	}

	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Join(".", runtimeDirName)
	}

	if repo := findRepoRoot(cwd); repo != "" {
		return filepath.Join(repo, runtimeDirName)
	}

	return filepath.Join(cwd, runtimeDirName)
}

// EnsureReadme writes the bundled README.md into the runtime root if the file
// does not already exist. Existing READMEs are never overwritten so local
// annotations survive restarts. Missing parent directories are created.
func EnsureReadme() error {
	path := Resolve("README.md")
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("runtimepath: create root dir: %w", err)
	}

	if err := os.WriteFile(path, embeddedReadme, 0o644); err != nil {
		return fmt.Errorf("runtimepath: write README.md: %w", err)
	}

	return nil
}

// findRepoRoot walks upward from `start` looking for a .git entry and returns
// the containing directory, or "" if no marker is found before hitting the
// filesystem root.
func findRepoRoot(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
