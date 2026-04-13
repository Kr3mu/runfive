package runtimepath

import (
	"os"
	"path/filepath"
)

// Resolve returns a path rooted beside the running executable.
// If the executable path can't be determined, it falls back to the
// current working directory.
func Resolve(name string) string {
	baseDir, err := executableDir()
	if err != nil {
		return name
	}
	return filepath.Join(baseDir, name)
}

func executableDir() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exePath), nil
}
