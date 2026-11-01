// Package dirs provides platform-specific directory paths for BookLife configuration and data.
package dirs

import (
	"os"
	"path/filepath"
	"runtime"
)

// ConfigDir returns the platform-specific configuration directory.
// - Linux/Unix: ~/.config/booklife
// - macOS: ~/.config/booklife
// - Windows: %APPDATA%\BookLife
func ConfigDir() (string, error) {
	if runtime.GOOS == "windows" {
		// Windows: %APPDATA%\BookLife
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "BookLife"), nil
	}

	// Linux/macOS: follow XDG Base Directory Specification
	// Prefer XDG_CONFIG_HOME if set, otherwise use ~/.config/booklife
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig != "" {
		return filepath.Join(xdgConfig, "booklife"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "booklife"), nil
}

// DataDir returns the platform-specific data directory.
// - Linux/Unix: ~/.local/share/booklife (XDG_DATA_HOME)
// - macOS: ~/.local/share/booklife
// - Windows: %LOCALAPPDATA%\BookLife
func DataDir() (string, error) {
	if runtime.GOOS == "windows" {
		// Windows: %LOCALAPPDATA%\BookLife
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(localAppData, "BookLife"), nil
	}

	// Linux/macOS: follow XDG Base Directory Specification
	// Prefer XDG_DATA_HOME if set, otherwise use ~/.local/share/booklife
	xdgData := os.Getenv("XDG_DATA_HOME")
	if xdgData != "" {
		return filepath.Join(xdgData, "booklife"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "booklife"), nil
}

// StateDir returns the platform-specific state directory for transient data.
// - Linux: ~/.local/state/booklife (XDG_STATE_HOME)
// - macOS: ~/.local/state/booklife
// - Windows: %LOCALAPPDATA%\BookLife\state
func StateDir() (string, error) {
	if runtime.GOOS == "windows" {
		dataDir, err := DataDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dataDir, "state"), nil
	}

	// Linux/macOS: follow XDG Base Directory Specification
	// Prefer XDG_STATE_HOME if set, otherwise use ~/.local/state/booklife
	xdgState := os.Getenv("XDG_STATE_HOME")
	if xdgState != "" {
		return filepath.Join(xdgState, "booklife"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "booklife"), nil
}

// CacheDir returns the platform-specific cache directory.
// - Linux: ~/.cache/booklife (XDG_CACHE_HOME)
// - macOS: ~/Library/Caches/booklife
// - Windows: %LOCALAPPDATA%\BookLife\cache
func CacheDir() (string, error) {
	if runtime.GOOS == "darwin" {
		// macOS: ~/Library/Caches/booklife
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Caches", "booklife"), nil
	}

	if runtime.GOOS == "windows" {
		// Windows: %LOCALAPPDATA%\BookLife\cache
		dataDir, err := DataDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dataDir, "cache"), nil
	}

	// Linux: follow XDG Base Directory Specification
	// Prefer XDG_CACHE_HOME if set, otherwise use ~/.cache/booklife
	xdgCache := os.Getenv("XDG_CACHE_HOME")
	if xdgCache != "" {
		return filepath.Join(xdgCache, "booklife"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "booklife"), nil
}

// EnsureAll creates all platform-specific directories with appropriate permissions.
func EnsureAll() error {
	dirs := []func() (string, error){
		ConfigDir,
		DataDir,
		StateDir,
		CacheDir,
	}

	for _, getDir := range dirs {
		dir, err := getDir()
		if err != nil {
			return err
		}
		// Create with owner-only permissions (0700)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	return nil
}

// SecureFile creates path if needed and makes it private to the current user.
// Existing files are tightened too, and SQLite gives its -wal and -shm files
// the same mode as the database.
func SecureFile(path string) error {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Chmod(path, 0600)
}
