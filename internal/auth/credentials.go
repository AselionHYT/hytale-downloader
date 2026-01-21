// Package auth handles OAuth2 authentication with Hytale's servers.
package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/AselionHYT/hytale-downloader/internal/config"
)

// Credentials represents stored OAuth tokens.
type Credentials struct {
	RefreshToken string    `json:"refresh_token"`
	AccessToken  string    `json:"access_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	TokenType    string    `json:"token_type"`
}

// IsAccessTokenValid checks if the access token is still valid.
func (c *Credentials) IsAccessTokenValid() bool {
	if c.AccessToken == "" {
		return false
	}
	// Consider token invalid 5 minutes before expiry
	return time.Now().Add(5 * time.Minute).Before(c.ExpiresAt)
}

// LoadCredentialsFromEnv loads credentials from environment variable.
// Returns nil if HYTALE_REFRESH_TOKEN is not set.
func LoadCredentialsFromEnv() *Credentials {
	refreshToken := os.Getenv("HYTALE_REFRESH_TOKEN")
	if refreshToken == "" {
		return nil
	}
	return &Credentials{
		RefreshToken: refreshToken,
		TokenType:    "bearer",
	}
}

// LoadCredentialsFromFile loads credentials from a specific file path.
func LoadCredentialsFromFile(filePath string) (*Credentials, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("credentials file not found: %s", filePath)
		}
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	if creds.RefreshToken == "" {
		return nil, fmt.Errorf("credentials file exists but refresh token is empty")
	}

	return &creds, nil
}

// LoadCredentials loads credentials from the config file.
func LoadCredentials(paths *config.Paths) (*Credentials, error) {
	return LoadCredentialsFromFile(paths.CredentialsFile)
}

// SaveCredentials saves credentials to the config file.
func SaveCredentials(paths *config.Paths, creds *Credentials) error {
	if err := paths.EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	if err := os.WriteFile(paths.CredentialsFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}

// DeleteCredentials removes the credentials file.
func DeleteCredentials(paths *config.Paths) error {
	err := os.Remove(paths.CredentialsFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}
	return nil
}
