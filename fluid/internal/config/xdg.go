package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetConfigDir returns the appropriate configuration directory for the current OS.
// It follows the XDG Base Directory specification on Linux/Unix and uses
// the appropriate Windows directories on Windows.
//
// Priority order:
// 1. Environment variable FLUID_CONFIG_DIR (if set)
// 2. XDG_CONFIG_HOME/fluid (Linux/Unix) or %APPDATA%/fluid (Windows)
// 3. ~/.config/fluid (Linux/Unix) or %USERPROFILE%/AppData/Roaming/fluid (Windows)
// 4. Fallback: ~/.fluid (for backward compatibility)
func GetConfigDir() (string, error) {
	// Allow explicit override via environment variable
	if dir := os.Getenv("FLUID_CONFIG_DIR"); dir != "" {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "windows":
		// On Windows, use %APPDATA%\fluid
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "fluid"), nil
		}
		// Fallback to %USERPROFILE%\AppData\Roaming\fluid
		return filepath.Join(home, "AppData", "Roaming", "fluid"), nil

	case "darwin", "linux", "freebsd", "openbsd", "netbsd":
		// On Unix-like systems, follow XDG Base Directory specification
		if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
			return filepath.Join(xdgConfigHome, "fluid"), nil
		}
		// Default XDG location: ~/.config/fluid
		return filepath.Join(home, ".config", "fluid"), nil

	default:
		// For any other OS, use XDG-style directory
		if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
			return filepath.Join(xdgConfigHome, "fluid"), nil
		}
		return filepath.Join(home, ".config", "fluid"), nil
	}
}
