package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"teachar.in/config"
)

func TestConfigDefaults(t *testing.T) {
	// Ensure env vars are cleared for test
	os.Unsetenv("APP_NAME")
	os.Unsetenv("APP_HOST")
	os.Unsetenv("APP_PORT")
	os.Unsetenv("APP_ENV")
	os.Unsetenv("CONFIG_PATH")

	cfg, err := config.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AppName != "teachar.in" {
		t.Errorf("expected AppName teachar.in, got %s", cfg.AppName)
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected Host 0.0.0.0, got %s", cfg.Host)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected Port 8080, got %s", cfg.Port)
	}
	if cfg.Env != "development" {
		t.Errorf("expected Env development, got %s", cfg.Env)
	}
	if cfg.LogDir != "logs" {
		t.Errorf("expected LogDir logs, got %s", cfg.LogDir)
	}
}

func TestLoadConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.json")
	jsonContent := `{
		"app_name": "test-teachar",
		"host": "127.0.0.1",
		"port": "9090",
		"env": "staging",
		"enable_tls": true,
		"ssl_cert_file": "custom/cert.pem",
		"ssl_key_file": "custom/key.pem",
		"log_dir": "custom_logs",
		"log_file": "custom_logs/test.log"
	}`

	if err := os.WriteFile(configFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed writing test config file: %v", err)
	}

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		t.Fatalf("failed loading config file: %v", err)
	}

	if cfg.AppName != "test-teachar" {
		t.Errorf("expected test-teachar, got %s", cfg.AppName)
	}
	if cfg.Host != "127.0.0.1" {
		t.Errorf("expected 127.0.0.1, got %s", cfg.Host)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected 9090, got %s", cfg.Port)
	}
	if cfg.EnableTLS != true {
		t.Errorf("expected EnableTLS true, got false")
	}
	if cfg.LogDir != "custom_logs" {
		t.Errorf("expected custom_logs, got %s", cfg.LogDir)
	}
	if cfg.LogFile != "custom_logs/test.log" {
		t.Errorf("expected custom_logs/test.log, got %s", cfg.LogFile)
	}
}
