package config_test

import (
	"os"
	"testing"

	"teachar.in/config"
)

func TestConfigDefaults(t *testing.T) {
	// Ensure env vars are cleared for test
	os.Unsetenv("APP_NAME")
	os.Unsetenv("APP_HOST")
	os.Unsetenv("APP_PORT")
	os.Unsetenv("APP_ENV")

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
}

func TestConfigCustomEnv(t *testing.T) {
	t.Setenv("APP_NAME", "custom-app")
	t.Setenv("APP_HOST", "127.0.0.1")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("APP_ENV", "production")

	cfg, err := config.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AppName != "custom-app" {
		t.Errorf("expected custom-app, got %s", cfg.AppName)
	}
	if cfg.Host != "127.0.0.1" {
		t.Errorf("expected 127.0.0.1, got %s", cfg.Host)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected 9090, got %s", cfg.Port)
	}
	if cfg.Env != "production" {
		t.Errorf("expected production, got %s", cfg.Env)
	}
}
