package models

import (
	"time"
)

// APIKey represents an authenticated API Key for external integrations, POS terminals, or mobile clients.
type APIKey struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`        // e.g. "POS Terminal 1", "Mobile App"
	KeyHash    string     `json:"key_hash"`    // SHA-256 hash of raw secret key
	KeyPrefix  string     `json:"key_prefix"`  // e.g. "tch_live_a1b2..."
	Role       string     `json:"role"`        // e.g. "admin", "staff", "client"
	IsActive   bool       `json:"is_active"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}
