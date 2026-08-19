package services_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"teachar.in/repository"
	"teachar.in/services"
)

func TestSecurityService(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	repo, err := repository.NewSQLiteRepository(dbPath, tempDir)
	if err != nil {
		t.Fatalf("failed initializing repo: %v", err)
	}
	defer repo.Close()

	secSvc := services.NewSecurityService(repo)
	ctx := context.Background()

	// 1. Test Generating API Key
	savedKey, rawSecret, err := secSvc.GenerateAPIKey(ctx, "POS Terminal #1", "admin", 30)
	if err != nil {
		t.Fatalf("failed generating API key: %v", err)
	}
	if savedKey.ID == 0 {
		t.Errorf("expected non-zero API Key ID")
	}
	if !strings.HasPrefix(rawSecret, "tch_live_") {
		t.Errorf("expected secret key prefix 'tch_live_', got '%s'", rawSecret)
	}

	// 2. Test Validating Raw Secret Key
	validatedKey, err := secSvc.ValidateAPIKey(ctx, rawSecret)
	if err != nil {
		t.Fatalf("failed validating API key: %v", err)
	}
	if validatedKey.ID != savedKey.ID {
		t.Errorf("expected ID %d, got %d", savedKey.ID, validatedKey.ID)
	}

	// 3. Test Invalid API Key
	_, err = secSvc.ValidateAPIKey(ctx, "tch_live_invalid_secret_key_12345678")
	if err == nil {
		t.Errorf("expected error for invalid API key")
	}

	// 4. Test Revoking API Key
	if err := secSvc.RevokeAPIKey(ctx, savedKey.ID); err != nil {
		t.Fatalf("failed revoking API key: %v", err)
	}

	_, err = secSvc.ValidateAPIKey(ctx, rawSecret)
	if err == nil {
		t.Errorf("expected error when validating revoked API key")
	}
}
