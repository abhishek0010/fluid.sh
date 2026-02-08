package sshkeys

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aspectrr/fluid.sh/fluid/internal/sshca"
)

// testCA creates a real CA for testing.
// Returns the CA and a cleanup function.
func testCA(t *testing.T) (*sshca.CA, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "sshkeys-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	keyPath := filepath.Join(tempDir, "test_ca")

	// Generate CA keypair.
	if err := sshca.GenerateCA(keyPath, "test-ca"); err != nil {
		_ = os.RemoveAll(tempDir)
		t.Fatalf("failed to generate CA: %v", err)
	}

	cfg := sshca.Config{
		CAKeyPath:             keyPath,
		CAPubKeyPath:          keyPath + ".pub",
		WorkDir:               filepath.Join(tempDir, "work"),
		DefaultTTL:            5 * time.Minute,
		MaxTTL:                10 * time.Minute,
		DefaultPrincipals:     []string{"sandbox"},
		EnforceKeyPermissions: false, // Disable for tests
	}

	ca, err := sshca.NewCA(cfg)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		t.Fatalf("failed to create CA: %v", err)
	}

	if err := ca.Initialize(context.Background()); err != nil {
		_ = os.RemoveAll(tempDir)
		t.Fatalf("failed to initialize CA: %v", err)
	}

	return ca, func() {
		_ = os.RemoveAll(tempDir)
	}
}

func TestNewKeyManager(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "keymanager-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := Config{
		KeyDir:          tempDir,
		CertificateTTL:  5 * time.Minute,
		RefreshMargin:   30 * time.Second,
		DefaultUsername: "sandbox",
	}

	km, err := NewKeyManager(ca, cfg, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer func() { _ = km.Close() }()

	if km.ca == nil {
		t.Error("CA is nil")
	}
	if km.cfg.KeyDir != tempDir {
		t.Errorf("KeyDir mismatch: got %s, want %s", km.cfg.KeyDir, tempDir)
	}
}

func TestNewKeyManager_NilCA(t *testing.T) {
	_, err := NewKeyManager(nil, Config{}, nil)
	if err == nil {
		t.Error("expected error for nil CA")
	}
}

func TestNewKeyManager_DefaultConfig(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "keymanager-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Empty config should use defaults.
	km, err := NewKeyManager(ca, Config{KeyDir: tempDir}, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer func() { _ = km.Close() }()

	defaults := DefaultConfig()
	if km.cfg.CertificateTTL != defaults.CertificateTTL {
		t.Errorf("CertificateTTL mismatch: got %v, want %v", km.cfg.CertificateTTL, defaults.CertificateTTL)
	}
	if km.cfg.RefreshMargin != defaults.RefreshMargin {
		t.Errorf("RefreshMargin mismatch: got %v, want %v", km.cfg.RefreshMargin, defaults.RefreshMargin)
	}
	if km.cfg.DefaultUsername != defaults.DefaultUsername {
		t.Errorf("DefaultUsername mismatch: got %s, want %s", km.cfg.DefaultUsername, defaults.DefaultUsername)
	}
}

func TestGetCredentials_GeneratesNewKeys(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "keymanager-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	km, err := NewKeyManager(ca, Config{KeyDir: tempDir, CertificateTTL: 5 * time.Minute}, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer func() { _ = km.Close() }()

	ctx := context.Background()
	creds, err := km.GetCredentials(ctx, "SBX-123", "sandbox")
	if err != nil {
		t.Fatalf("GetCredentials failed: %v", err)
	}

	// Check credentials.
	if creds.SandboxID != "SBX-123" {
		t.Errorf("SandboxID mismatch: got %s, want SBX-123", creds.SandboxID)
	}
	if creds.Username != "sandbox" {
		t.Errorf("Username mismatch: got %s, want sandbox", creds.Username)
	}
	if creds.PrivateKeyPath == "" {
		t.Error("PrivateKeyPath is empty")
	}
	if creds.CertificatePath == "" {
		t.Error("CertificatePath is empty")
	}
	if creds.ValidUntil.IsZero() {
		t.Error("ValidUntil is zero")
	}

	// Check files exist.
	if _, err := os.Stat(creds.PrivateKeyPath); os.IsNotExist(err) {
		t.Error("private key file does not exist")
	}
	if _, err := os.Stat(creds.CertificatePath); os.IsNotExist(err) {
		t.Error("certificate file does not exist")
	}
}

func TestGetCredentials_ReturnsCached(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "keymanager-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	km, err := NewKeyManager(ca, Config{KeyDir: tempDir, CertificateTTL: 5 * time.Minute}, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer func() { _ = km.Close() }()

	ctx := context.Background()

	// First call generates.
	creds1, err := km.GetCredentials(ctx, "SBX-123", "sandbox")
	if err != nil {
		t.Fatalf("GetCredentials (1) failed: %v", err)
	}

	// Second call should return cached.
	creds2, err := km.GetCredentials(ctx, "SBX-123", "sandbox")
	if err != nil {
		t.Fatalf("GetCredentials (2) failed: %v", err)
	}

	// Should be the same credentials.
	if creds1.PrivateKeyPath != creds2.PrivateKeyPath {
		t.Error("expected cached credentials to be returned")
	}
}

func TestGetCredentials_RegeneratesOnExpiry(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "keymanager-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	km, err := NewKeyManager(ca, Config{
		KeyDir:         tempDir,
		CertificateTTL: 5 * time.Minute,
		RefreshMargin:  30 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer func() { _ = km.Close() }()

	ctx := context.Background()
	// First call generates.
	creds1, err := km.GetCredentials(ctx, "SBX-123", "sandbox")
	if err != nil {
		t.Fatalf("GetCredentials (1) failed: %v", err)
	}

	// Simulate time passing by modifying cached credentials to be expired.
	km.mu.Lock()
	for key, creds := range km.credentials {
		creds.ValidUntil = time.Now().Add(-1 * time.Minute) // Already expired
		km.credentials[key] = creds
	}
	km.mu.Unlock()

	// Second call should regenerate.
	creds2, err := km.GetCredentials(ctx, "SBX-123", "sandbox")
	if err != nil {
		t.Fatalf("GetCredentials (2) failed: %v", err)
	}

	// ValidUntil should be different (new certificate was issued).
	// Note: paths are the same because sandbox ID is the same, but the
	// certificate content and expiry time will be different.
	if creds2.ValidUntil.Before(time.Now()) {
		t.Error("expected new credentials with valid expiry after regeneration")
	}
	// New expiry should be after the old (expired) one.
	if !creds2.ValidUntil.After(creds1.ValidUntil) {
		t.Error("expected new credentials to have later expiry than expired ones")
	}
}

func TestGetCredentials_DefaultUsername(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "keymanager-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	km, err := NewKeyManager(ca, Config{
		KeyDir:          tempDir,
		DefaultUsername: "myuser",
	}, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer func() { _ = km.Close() }()

	ctx := context.Background()

	// Empty username should use default.
	creds, err := km.GetCredentials(ctx, "SBX-123", "")
	if err != nil {
		t.Fatalf("GetCredentials failed: %v", err)
	}

	if creds.Username != "myuser" {
		t.Errorf("Username mismatch: got %s, want myuser", creds.Username)
	}
}

func TestGetCredentials_ConcurrentSafety(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "keymanager-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	km, err := NewKeyManager(ca, Config{KeyDir: tempDir}, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer func() { _ = km.Close() }()

	ctx := context.Background()
	sandboxID := "SBX-CONCURRENT"

	// Launch multiple goroutines requesting the same sandbox's credentials.
	var wg sync.WaitGroup
	results := make(chan *Credentials, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			creds, err := km.GetCredentials(ctx, sandboxID, "sandbox")
			if err != nil {
				errors <- err
				return
			}
			results <- creds
		}()
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check for errors.
	for err := range errors {
		t.Errorf("GetCredentials error: %v", err)
	}

	// All results should have the same private key path (cached).
	var firstPath string
	for creds := range results {
		if firstPath == "" {
			firstPath = creds.PrivateKeyPath
		} else if creds.PrivateKeyPath != firstPath {
			t.Error("concurrent calls returned different credentials")
		}
	}
}

func TestCleanupSandbox_RemovesFiles(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "keymanager-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	km, err := NewKeyManager(ca, Config{KeyDir: tempDir}, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer func() { _ = km.Close() }()

	ctx := context.Background()

	// Generate credentials.
	creds, err := km.GetCredentials(ctx, "SBX-CLEANUP", "sandbox")
	if err != nil {
		t.Fatalf("GetCredentials failed: %v", err)
	}

	// Verify files exist.
	if _, err := os.Stat(creds.PrivateKeyPath); os.IsNotExist(err) {
		t.Fatal("private key file should exist")
	}

	// Cleanup.
	if err := km.CleanupSandbox(ctx, "SBX-CLEANUP"); err != nil {
		t.Fatalf("CleanupSandbox failed: %v", err)
	}

	// Verify files are gone.
	if _, err := os.Stat(creds.PrivateKeyPath); !os.IsNotExist(err) {
		t.Error("private key file should be deleted")
	}
	sandboxDir := km.sandboxKeyDir("SBX-CLEANUP")
	if _, err := os.Stat(sandboxDir); !os.IsNotExist(err) {
		t.Error("sandbox key directory should be deleted")
	}

	// Verify cache is cleared.
	km.mu.RLock()
	if len(km.credentials) > 0 {
		t.Error("credentials should be cleared from cache")
	}
	km.mu.RUnlock()
}

func TestCleanupSandbox_EmptySandboxID(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "keymanager-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	km, err := NewKeyManager(ca, Config{KeyDir: tempDir}, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer func() { _ = km.Close() }()

	err = km.CleanupSandbox(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty sandboxID")
	}
}

func TestKeyFilePermissions(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "keymanager-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	km, err := NewKeyManager(ca, Config{KeyDir: tempDir}, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer func() { _ = km.Close() }()

	ctx := context.Background()
	creds, err := km.GetCredentials(ctx, "SBX-PERM", "sandbox")
	if err != nil {
		t.Fatalf("GetCredentials failed: %v", err)
	}

	// Check private key permissions.
	info, err := os.Stat(creds.PrivateKeyPath)
	if err != nil {
		t.Fatalf("failed to stat private key: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("private key has wrong permissions: %o, expected 0600", perm)
	}
}

func TestCredentials_IsExpired(t *testing.T) {
	tests := []struct {
		name       string
		validUntil time.Time
		margin     time.Duration
		want       bool
	}{
		{
			name:       "not expired",
			validUntil: time.Now().Add(10 * time.Minute),
			margin:     30 * time.Second,
			want:       false,
		},
		{
			name:       "expired",
			validUntil: time.Now().Add(-1 * time.Minute),
			margin:     30 * time.Second,
			want:       true,
		},
		{
			name:       "within margin",
			validUntil: time.Now().Add(20 * time.Second),
			margin:     30 * time.Second,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Credentials{ValidUntil: tt.validUntil}
			if got := c.IsExpired(tt.margin); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSourceVMCredentials_GeneratesNewKeys(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "keymanager-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	km, err := NewKeyManager(ca, Config{KeyDir: tempDir, CertificateTTL: 5 * time.Minute}, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer func() { _ = km.Close() }()

	ctx := context.Background()
	creds, err := km.GetSourceVMCredentials(ctx, "golden-ubuntu-22.04")
	if err != nil {
		t.Fatalf("GetSourceVMCredentials failed: %v", err)
	}

	// Check credentials.
	if creds.SandboxID != "golden-ubuntu-22.04" {
		t.Errorf("SandboxID mismatch: got %s, want golden-ubuntu-22.04", creds.SandboxID)
	}
	if creds.Username != "fluid-readonly" {
		t.Errorf("Username mismatch: got %s, want fluid-readonly", creds.Username)
	}
	if creds.PrivateKeyPath == "" {
		t.Error("PrivateKeyPath is empty")
	}
	if creds.CertificatePath == "" {
		t.Error("CertificatePath is empty")
	}
	if creds.ValidUntil.IsZero() {
		t.Error("ValidUntil is zero")
	}

	// Check files exist.
	if _, err := os.Stat(creds.PrivateKeyPath); os.IsNotExist(err) {
		t.Error("private key file does not exist")
	}
	if _, err := os.Stat(creds.CertificatePath); os.IsNotExist(err) {
		t.Error("certificate file does not exist")
	}
}

func TestGetSourceVMCredentials_ReturnsCached(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "keymanager-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	km, err := NewKeyManager(ca, Config{KeyDir: tempDir, CertificateTTL: 5 * time.Minute}, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer func() { _ = km.Close() }()

	ctx := context.Background()

	// First call generates.
	creds1, err := km.GetSourceVMCredentials(ctx, "golden-ubuntu-22.04")
	if err != nil {
		t.Fatalf("GetSourceVMCredentials (1) failed: %v", err)
	}

	// Second call should return cached.
	creds2, err := km.GetSourceVMCredentials(ctx, "golden-ubuntu-22.04")
	if err != nil {
		t.Fatalf("GetSourceVMCredentials (2) failed: %v", err)
	}

	// Should be the same credentials.
	if creds1.PrivateKeyPath != creds2.PrivateKeyPath {
		t.Error("expected cached credentials to be returned")
	}
	if creds1.ValidUntil != creds2.ValidUntil {
		t.Error("expected cached credentials to have same expiry")
	}
}

func TestGetSourceVMCredentials_RegeneratesOnExpiry(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "keymanager-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	km, err := NewKeyManager(ca, Config{
		KeyDir:         tempDir,
		CertificateTTL: 5 * time.Minute,
		RefreshMargin:  30 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer func() { _ = km.Close() }()

	ctx := context.Background()
	// First call generates.
	creds1, err := km.GetSourceVMCredentials(ctx, "golden-ubuntu-22.04")
	if err != nil {
		t.Fatalf("GetSourceVMCredentials (1) failed: %v", err)
	}

	// Simulate time passing by modifying cached credentials to be expired.
	km.mu.Lock()
	for key, creds := range km.credentials {
		if key == "sourcevm:golden-ubuntu-22.04:fluid-readonly" {
			creds.ValidUntil = time.Now().Add(-1 * time.Minute) // Already expired
			km.credentials[key] = creds
		}
	}
	km.mu.Unlock()

	// Second call should regenerate.
	creds2, err := km.GetSourceVMCredentials(ctx, "golden-ubuntu-22.04")
	if err != nil {
		t.Fatalf("GetSourceVMCredentials (2) failed: %v", err)
	}

	// ValidUntil should be different (new certificate was issued).
	if creds2.ValidUntil.Before(time.Now()) {
		t.Error("expected new credentials with valid expiry after regeneration")
	}
	// New expiry should be after the old (expired) one.
	if !creds2.ValidUntil.After(creds1.ValidUntil) {
		t.Error("expected new credentials to have later expiry than expired ones")
	}
}

func TestGetSourceVMCredentials_FilesystemLayout(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "keymanager-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	km, err := NewKeyManager(ca, Config{KeyDir: tempDir}, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer func() { _ = km.Close() }()

	ctx := context.Background()

	testCases := []struct {
		name            string
		sourceVMName    string
		expectedDirName string
	}{
		{
			name:            "simple name",
			sourceVMName:    "ubuntu-22.04",
			expectedDirName: "sourcevm-ubuntu-22.04",
		},
		{
			name:            "name with dots",
			sourceVMName:    "golden.ubuntu.22.04",
			expectedDirName: "sourcevm-golden.ubuntu.22.04",
		},
		{
			name:            "name with hyphens",
			sourceVMName:    "my-golden-vm",
			expectedDirName: "sourcevm-my-golden-vm",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			creds, err := km.GetSourceVMCredentials(ctx, tc.sourceVMName)
			if err != nil {
				t.Fatalf("GetSourceVMCredentials failed: %v", err)
			}

			// Verify the directory structure.
			expectedDir := filepath.Join(tempDir, tc.expectedDirName)
			expectedKeyPath := filepath.Join(expectedDir, "key")
			expectedCertPath := filepath.Join(expectedDir, "key-cert.pub")

			if creds.PrivateKeyPath != expectedKeyPath {
				t.Errorf("PrivateKeyPath mismatch: got %s, want %s", creds.PrivateKeyPath, expectedKeyPath)
			}
			if creds.CertificatePath != expectedCertPath {
				t.Errorf("CertificatePath mismatch: got %s, want %s", creds.CertificatePath, expectedCertPath)
			}

			// Verify directory exists with correct permissions.
			info, err := os.Stat(expectedDir)
			if err != nil {
				t.Fatalf("failed to stat directory: %v", err)
			}
			if !info.IsDir() {
				t.Error("expected directory, got file")
			}
			perm := info.Mode().Perm()
			if perm != 0o700 {
				t.Errorf("directory has wrong permissions: %o, expected 0700", perm)
			}

			// Verify private key has correct permissions.
			keyInfo, err := os.Stat(creds.PrivateKeyPath)
			if err != nil {
				t.Fatalf("failed to stat private key: %v", err)
			}
			keyPerm := keyInfo.Mode().Perm()
			if keyPerm != 0o600 {
				t.Errorf("private key has wrong permissions: %o, expected 0600", keyPerm)
			}
		})
	}
}

func TestGetSourceVMCredentials_EmptySourceVMName(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "keymanager-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	km, err := NewKeyManager(ca, Config{KeyDir: tempDir}, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer func() { _ = km.Close() }()

	_, err = km.GetSourceVMCredentials(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty sourceVMName")
	}
}

func TestGetSourceVMCredentials_ConcurrentSafety(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "keymanager-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	km, err := NewKeyManager(ca, Config{KeyDir: tempDir}, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer func() { _ = km.Close() }()

	ctx := context.Background()
	sourceVMName := "golden-concurrent"

	// Launch multiple goroutines requesting the same source VM's credentials.
	var wg sync.WaitGroup
	results := make(chan *Credentials, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			creds, err := km.GetSourceVMCredentials(ctx, sourceVMName)
			if err != nil {
				errors <- err
				return
			}
			results <- creds
		}()
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check for errors.
	for err := range errors {
		t.Errorf("GetSourceVMCredentials error: %v", err)
	}

	// All results should have the same private key path (cached).
	var firstPath string
	for creds := range results {
		if firstPath == "" {
			firstPath = creds.PrivateKeyPath
		} else if creds.PrivateKeyPath != firstPath {
			t.Error("concurrent calls returned different credentials")
		}
	}
}

// TestSanitizeVMName tests the VM name sanitization function.
func TestSanitizeVMName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple alphanumeric",
			input:    "ubuntu2204",
			expected: "ubuntu2204",
		},
		{
			name:     "with dots",
			input:    "ubuntu.22.04",
			expected: "ubuntu.22.04",
		},
		{
			name:     "with hyphens",
			input:    "my-golden-vm",
			expected: "my-golden-vm",
		},
		{
			name:     "with underscores",
			input:    "test_vm_123",
			expected: "test_vm_123",
		},
		{
			name:     "path traversal with ..",
			input:    "../../../etc/passwd",
			expected: ".._.._.._etc_passwd",
		},
		{
			name:     "path traversal with /",
			input:    "/etc/passwd",
			expected: "_etc_passwd",
		},
		{
			name:     "path traversal complex",
			input:    "../../vm/../config",
			expected: ".._.._vm_.._config",
		},
		{
			name:     "special characters",
			input:    "vm@#$%^&*()name",
			expected: "vm_________name",
		},
		{
			name:     "spaces",
			input:    "my vm name",
			expected: "my_vm_name",
		},
		{
			name:     "mixed valid and invalid",
			input:    "vm-123.test/path",
			expected: "vm-123.test_path",
		},
		{
			name:     "unicode characters",
			input:    "vm-名前",
			expected: "vm-__", // Each unicode character is replaced with underscore
		},
		{
			name:     "backslash (Windows path)",
			input:    "C:\\Users\\test",
			expected: "C__Users_test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeVMName(tc.input)
			if result != tc.expected {
				t.Errorf("sanitizeVMName(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestSourceVMCredentialsPathTraversal tests that path traversal attacks are prevented.
func TestSourceVMCredentialsPathTraversal(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "sshkeys-traversal-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := DefaultConfig()
	cfg.KeyDir = tmpDir
	cfg.CertificateTTL = 5 * time.Minute

	km, err := NewKeyManager(ca, cfg, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer km.Close()

	ctx := context.Background()

	// Test cases that attempt path traversal
	testCases := []struct {
		name         string
		vmName       string
		shouldCreate bool // whether the directory should be created
	}{
		{
			name:         "path traversal with ..",
			vmName:       "../../etc/passwd",
			shouldCreate: true, // should create sanitized directory
		},
		{
			name:         "absolute path",
			vmName:       "/tmp/evil",
			shouldCreate: true,
		},
		{
			name:         "complex traversal",
			vmName:       "../../../root/.ssh/id_rsa",
			shouldCreate: true,
		},
		{
			name:         "normal name",
			vmName:       "ubuntu-22.04",
			shouldCreate: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			creds, err := km.GetSourceVMCredentials(ctx, tc.vmName)
			if err != nil {
				t.Fatalf("GetSourceVMCredentials failed: %v", err)
			}

			// Verify that the private key path is within the expected directory
			if !strings.HasPrefix(creds.PrivateKeyPath, tmpDir) {
				t.Errorf("private key path %q is not within base directory %q", creds.PrivateKeyPath, tmpDir)
			}

			// Verify that the certificate path is within the expected directory
			if !strings.HasPrefix(creds.CertificatePath, tmpDir) {
				t.Errorf("certificate path %q is not within base directory %q", creds.CertificatePath, tmpDir)
			}

			// Verify that files were actually created
			if _, err := os.Stat(creds.PrivateKeyPath); os.IsNotExist(err) {
				t.Errorf("private key file does not exist: %s", creds.PrivateKeyPath)
			}
			if _, err := os.Stat(creds.CertificatePath); os.IsNotExist(err) {
				t.Errorf("certificate file does not exist: %s", creds.CertificatePath)
			}

			// Verify that the parent directory name contains the sanitized VM name
			parentDir := filepath.Base(filepath.Dir(creds.PrivateKeyPath))
			expectedPrefix := "sourcevm-"
			if !strings.HasPrefix(parentDir, expectedPrefix) {
				t.Errorf("parent directory %q does not start with expected prefix %q", parentDir, expectedPrefix)
			}

			// Verify no path traversal occurred - the directory should be directly under tmpDir
			expectedDir := filepath.Join(tmpDir, parentDir)
			actualDir := filepath.Dir(creds.PrivateKeyPath)
			if actualDir != expectedDir {
				t.Errorf("directory mismatch: got %q, want %q", actualDir, expectedDir)
			}
		})
	}
}

// TestSourceVMCredentialsSanitizationInPath tests that the directory name is sanitized.
func TestSourceVMCredentialsSanitizationInPath(t *testing.T) {
	ca, cleanup := testCA(t)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "sshkeys-sanitize-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := DefaultConfig()
	cfg.KeyDir = tmpDir
	cfg.CertificateTTL = 5 * time.Minute

	km, err := NewKeyManager(ca, cfg, nil)
	if err != nil {
		t.Fatalf("NewKeyManager failed: %v", err)
	}
	defer km.Close()

	ctx := context.Background()

	// Test that the directory name is properly sanitized
	vmName := "../../../evil/../../path"
	creds, err := km.GetSourceVMCredentials(ctx, vmName)
	if err != nil {
		t.Fatalf("GetSourceVMCredentials failed: %v", err)
	}

	// The directory should be named with the sanitized version
	dirName := filepath.Base(filepath.Dir(creds.PrivateKeyPath))
	expectedDirName := "sourcevm-" + sanitizeVMName(vmName)

	if dirName != expectedDirName {
		t.Errorf("directory name = %q, want %q", dirName, expectedDirName)
	}

	// Verify the directory doesn't contain any path separators
	if filepath.Base(dirName) != dirName {
		t.Errorf("directory name %q contains path separators", dirName)
	}
}
