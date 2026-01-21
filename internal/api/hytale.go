package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
)

const (
	// API endpoints
	accountDataBaseURL   = "https://account-data.hytale.com/game-assets"
	downloaderVersionURL = "https://downloader.hytale.com/version.json"
)

// HytaleAPI provides access to Hytale's APIs.
type HytaleAPI struct {
	client *Client
	logger zerolog.Logger
}

// NewHytaleAPI creates a new Hytale API client.
func NewHytaleAPI(logger zerolog.Logger) *HytaleAPI {
	return &HytaleAPI{
		client: NewClient(logger),
		logger: logger,
	}
}

// DownloaderVersion represents the official downloader version info.
type DownloaderVersion struct {
	Latest string `json:"latest"`
}

// VersionInfo represents game version information.
type VersionInfo struct {
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	SHA256      string `json:"sha256"`
}

// SignedURLResponse represents a response containing a signed URL.
type SignedURLResponse struct {
	URL string `json:"url"`
}

// GetDownloaderVersion fetches the official downloader version.
func (h *HytaleAPI) GetDownloaderVersion(ctx context.Context) (*DownloaderVersion, error) {
	h.logger.Debug().Msg("Fetching downloader version")

	data, err := h.client.GetJSON(ctx, downloaderVersionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch downloader version: %w", err)
	}

	var version DownloaderVersion
	if err := json.Unmarshal(data, &version); err != nil {
		return nil, fmt.Errorf("failed to parse downloader version: %w", err)
	}

	h.logger.Debug().Str("version", version.Latest).Msg("Downloader version fetched")
	return &version, nil
}

// GetVersionInfo fetches game version info for a patchline.
func (h *HytaleAPI) GetVersionInfo(ctx context.Context, accessToken, patchline string) (*VersionInfo, error) {
	h.logger.Debug().Str("patchline", patchline).Msg("Fetching version info")

	// Step 1: Get signed URL for version.json
	versionURL := fmt.Sprintf("%s/version/%s.json", accountDataBaseURL, patchline)

	data, err := h.client.GetJSONWithAuth(ctx, versionURL, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get version URL: %w", err)
	}

	var signedURL SignedURLResponse
	if err := json.Unmarshal(data, &signedURL); err != nil {
		return nil, fmt.Errorf("failed to parse signed URL response: %w", err)
	}

	// Step 2: Fetch version info from R2 storage
	versionData, err := h.client.GetJSON(ctx, signedURL.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch version info: %w", err)
	}

	var versionInfo VersionInfo
	if err := json.Unmarshal(versionData, &versionInfo); err != nil {
		return nil, fmt.Errorf("failed to parse version info: %w", err)
	}

	h.logger.Debug().
		Str("version", versionInfo.Version).
		Str("sha256", versionInfo.SHA256).
		Msg("Version info fetched")

	return &versionInfo, nil
}

// GetDownloadURL fetches the signed download URL for the game files.
func (h *HytaleAPI) GetDownloadURL(ctx context.Context, accessToken, downloadPath string) (string, error) {
	h.logger.Debug().Str("path", downloadPath).Msg("Fetching download URL")

	buildURL := fmt.Sprintf("%s/%s", accountDataBaseURL, downloadPath)

	data, err := h.client.GetJSONWithAuth(ctx, buildURL, accessToken)
	if err != nil {
		return "", fmt.Errorf("failed to get download URL: %w", err)
	}

	var signedURL SignedURLResponse
	if err := json.Unmarshal(data, &signedURL); err != nil {
		return "", fmt.Errorf("failed to parse download URL response: %w", err)
	}

	h.logger.Debug().Msg("Download URL fetched")
	return signedURL.URL, nil
}
