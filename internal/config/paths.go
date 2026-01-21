// Package config handles configuration paths and settings.
package config

import (
	"os"
	"path/filepath"
)

const (
	// AppName is the application name used for config directories
	AppName = "hytale-downloader"

	// CredentialsFileName is the name of the credentials file
	CredentialsFileName = "credentials.json"
)

// Paths holds all application paths
type Paths struct {
	ConfigDir       string
	CredentialsFile string
}

// GetPaths returns the application paths for the current user.
// Config directory: ~/.hytale-downloader/
func GetPaths() (*Paths, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(homeDir, "."+AppName)

	return &Paths{
		ConfigDir:       configDir,
		CredentialsFile: filepath.Join(configDir, CredentialsFileName),
	}, nil
}

// EnsureConfigDir creates the config directory if it doesn't exist.
func (p *Paths) EnsureConfigDir() error {
	return os.MkdirAll(p.ConfigDir, 0700)
}
