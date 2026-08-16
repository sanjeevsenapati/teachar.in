package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sort"
	"time"

	"teachar.in/models"
	"teachar.in/repository"
)

type SecurityService struct {
	securityRepo repository.SecurityRepository
}

func NewSecurityService(securityRepo repository.SecurityRepository) *SecurityService {
	return &SecurityService{securityRepo: securityRepo}
}

// GenerateAPIKey creates a new API key with a raw secret string "tch_live_<hex>".
// Returns the saved APIKey model and the raw key string (displayed ONCE to superadmin).
func (s *SecurityService) GenerateAPIKey(ctx context.Context, name, role string, expiryDays int) (*models.APIKey, string, error) {
	if name == "" {
		return nil, "", fmt.Errorf("API key name is required")
	}
	if role == "" {
		role = "admin"
	}

	// Generate 32 cryptographically secure random bytes
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, "", fmt.Errorf("failed generating random bytes: %w", err)
	}

	rawKey := "tch_live_" + hex.EncodeToString(randomBytes)
	keyPrefix := rawKey[:16] + "..."

	// Compute SHA-256 hash of the raw key
	hashBytes := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hashBytes[:])

	now := time.Now()
	var expiresAt *time.Time
	if expiryDays > 0 {
		exp := now.AddDate(0, 0, expiryDays)
		expiresAt = &exp
	}

	apiKey := models.APIKey{
		Name:      name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Role:      role,
		IsActive:  true,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	savedKey, err := s.securityRepo.SaveAPIKey(ctx, apiKey)
	if err != nil {
		return nil, "", err
	}
	return savedKey, rawKey, nil
}

// ValidateAPIKey verifies an incoming raw secret key header string.
func (s *SecurityService) ValidateAPIKey(ctx context.Context, rawKey string) (*models.APIKey, error) {
	if rawKey == "" {
		return nil, errors.New("empty API key")
	}

	hashBytes := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hashBytes[:])

	key, err := s.securityRepo.GetAPIKeyByHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}

	_ = s.securityRepo.UpdateAPIKeyLastUsed(ctx, key.ID)

	return key, nil
}

func (s *SecurityService) GetAllAPIKeys(ctx context.Context) ([]models.APIKey, error) {
	keys, err := s.securityRepo.GetAllAPIKeys(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].CreatedAt.After(keys[j].CreatedAt)
	})
	return keys, nil
}

func (s *SecurityService) RevokeAPIKey(ctx context.Context, id int64) error {
	return s.securityRepo.RevokeAPIKey(ctx, id)
}

// GenerateSelfSignedCert generates a local TLS X.509 certificate and private key using standard library.
func GenerateSelfSignedCert(certPath, keyPath string) error {
	if err := os.MkdirAll(filepath.Dir(certPath), 0755); err != nil {
		return fmt.Errorf("failed creating certificate directory: %w", err)
	}

	// Check if cert files already exist
	if _, cErr := os.Stat(certPath); cErr == nil {
		if _, kErr := os.Stat(keyPath); kErr == nil {
			return nil // Certs already present
		}
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed generating ECDSA key: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed generating serial number: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"TEACHAR.in Development Security"},
			CommonName:   "localhost",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "127.0.0.1"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed creating certificate: %w", err)
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("failed creating cert.pem file: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("failed encoding PEM cert: %w", err)
	}

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed creating key.pem file: %w", err)
	}
	defer keyOut.Close()

	b, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed marshaling EC key: %w", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}); err != nil {
		return fmt.Errorf("failed encoding PEM key: %w", err)
	}

	return nil
}
