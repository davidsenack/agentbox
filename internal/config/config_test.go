package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Check VM defaults
	if cfg.VM.CPUs != 4 {
		t.Errorf("expected 4 CPUs, got %d", cfg.VM.CPUs)
	}
	if cfg.VM.Memory != "4GiB" {
		t.Errorf("expected 4GiB memory, got %s", cfg.VM.Memory)
	}
	if cfg.VM.Disk != "30GiB" {
		t.Errorf("expected 30GiB disk, got %s", cfg.VM.Disk)
	}

	// Check network defaults
	if cfg.Network.ProxyPort != 3128 {
		t.Errorf("expected proxy port 3128, got %d", cfg.Network.ProxyPort)
	}
	if len(cfg.Network.InjectAuth) == 0 {
		t.Error("expected at least one auth injection config")
	}

	// Check secrets defaults
	if len(cfg.Secrets.RedactPatterns) == 0 {
		t.Error("expected at least one redact pattern")
	}
	if len(cfg.Secrets.AllowedEnvVars) == 0 {
		t.Error("expected at least one allowed env var")
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "agentbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save default config
	cfg := DefaultConfig()
	cfg.VM.CPUs = 8 // Modify to verify round-trip

	if err := Save(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, "agentbox.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load and verify
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.VM.CPUs != 8 {
		t.Errorf("expected 8 CPUs after load, got %d", loaded.VM.CPUs)
	}
	if loaded.VM.Memory != cfg.VM.Memory {
		t.Errorf("memory mismatch: expected %s, got %s", cfg.VM.Memory, loaded.VM.Memory)
	}
}

func TestExists(t *testing.T) {
	// Create temp directory with config
	tmpDir, err := os.MkdirTemp("", "agentbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Should not exist initially
	if Exists(tmpDir) {
		t.Error("config should not exist before saving")
	}

	// Save config
	if err := Save(tmpDir, DefaultConfig()); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Should exist now
	if !Exists(tmpDir) {
		t.Error("config should exist after saving")
	}
}

func TestAuthInjection(t *testing.T) {
	cfg := DefaultConfig()

	// Check that Anthropic auth injection is configured
	found := false
	for _, auth := range cfg.Network.InjectAuth {
		if auth.Host == "api.anthropic.com" {
			found = true
			if auth.Header != "x-api-key" {
				t.Errorf("expected x-api-key header, got %s", auth.Header)
			}
			if auth.Env != "ANTHROPIC_API_KEY" {
				t.Errorf("expected ANTHROPIC_API_KEY env, got %s", auth.Env)
			}
		}
	}
	if !found {
		t.Error("expected api.anthropic.com in auth injection config")
	}
}
