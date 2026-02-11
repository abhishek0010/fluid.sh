package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigDir_ExplicitOverride(t *testing.T) {
	// Test that FLUID_CONFIG_DIR takes precedence
	t.Setenv("FLUID_CONFIG_DIR", "/custom/config/dir")
	
	dir, err := GetConfigDir()
	require.NoError(t, err)
	assert.Equal(t, "/custom/config/dir", dir)
}

func TestGetConfigDir_XDG(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping XDG test on Windows")
	}

	// Unset FLUID_CONFIG_DIR to test XDG behavior
	os.Unsetenv("FLUID_CONFIG_DIR")
	
	t.Run("with XDG_CONFIG_HOME set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/custom/xdg/config")
		
		dir, err := GetConfigDir()
		require.NoError(t, err)
		assert.Equal(t, "/custom/xdg/config/fluid", dir)
	})
	
	t.Run("without XDG_CONFIG_HOME", func(t *testing.T) {
		os.Unsetenv("XDG_CONFIG_HOME")
		
		dir, err := GetConfigDir()
		require.NoError(t, err)
		
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		
		expected := filepath.Join(home, ".config", "fluid")
		assert.Equal(t, expected, dir)
	})
}

func TestGetConfigDir_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows test on non-Windows OS")
	}

	// Unset FLUID_CONFIG_DIR to test Windows behavior
	os.Unsetenv("FLUID_CONFIG_DIR")
	
	t.Run("with APPDATA set", func(t *testing.T) {
		t.Setenv("APPDATA", "C:\\Users\\TestUser\\AppData\\Roaming")
		
		dir, err := GetConfigDir()
		require.NoError(t, err)
		assert.Equal(t, "C:\\Users\\TestUser\\AppData\\Roaming\\fluid", dir)
	})
	
	t.Run("without APPDATA", func(t *testing.T) {
		// This test is tricky on real Windows since APPDATA is usually set
		// We'll just verify it doesn't error
		dir, err := GetConfigDir()
		require.NoError(t, err)
		assert.Contains(t, dir, "fluid")
	})
}

func TestGetConfigDir_CrossPlatform(t *testing.T) {
	// Test that the function returns a valid path on any OS
	os.Unsetenv("FLUID_CONFIG_DIR")
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Unsetenv("APPDATA")
	
	dir, err := GetConfigDir()
	require.NoError(t, err)
	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, "fluid")
}
