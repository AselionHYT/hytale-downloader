package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const (
	// OAuth endpoints
	deviceAuthURL = "https://oauth.accounts.hytale.com/oauth2/device/auth"
	tokenURL      = "https://oauth.accounts.hytale.com/oauth2/token"

	// OAuth client configuration
	clientID  = "hytale-downloader"
	scope     = "openid offline auth:downloader"
	userAgent = "Go-http-client/2.0"
)

// DeviceAuthResponse represents the response from the device authorization endpoint.
type DeviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// TokenResponse represents the response from the token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// DeviceFlowCallback is called when the user needs to authorize the device.
type DeviceFlowCallback func(verificationURL, userCode string)

// DeviceFlowAuth performs OAuth2 device flow authentication.
type DeviceFlowAuth struct {
	client   *http.Client
	logger   zerolog.Logger
	callback DeviceFlowCallback
}

// NewDeviceFlowAuth creates a new device flow authenticator.
func NewDeviceFlowAuth(logger zerolog.Logger, callback DeviceFlowCallback) *DeviceFlowAuth {
	return &DeviceFlowAuth{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:   logger,
		callback: callback,
	}
}

// Authenticate performs the device flow and returns credentials.
func (d *DeviceFlowAuth) Authenticate(ctx context.Context) (*Credentials, error) {
	d.logger.Debug().Msg("Starting device flow authentication")

	// Step 1: Request device code
	deviceAuth, err := d.requestDeviceCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to request device code: %w", err)
	}

	d.logger.Debug().
		Str("user_code", deviceAuth.UserCode).
		Int("expires_in", deviceAuth.ExpiresIn).
		Msg("Device code received")

	// Notify caller to show the verification URL
	if d.callback != nil {
		d.callback(deviceAuth.VerificationURIComplete, deviceAuth.UserCode)
	}

	// Step 2: Poll for token
	tokenResp, err := d.pollForToken(ctx, deviceAuth)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	d.logger.Debug().Msg("Token received successfully")

	return &Credentials{
		RefreshToken: tokenResp.RefreshToken,
		AccessToken:  tokenResp.AccessToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		TokenType:    tokenResp.TokenType,
	}, nil
}

func (d *DeviceFlowAuth) requestDeviceCode(ctx context.Context) (*DeviceAuthResponse, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("scope", scope)

	req, err := http.NewRequestWithContext(ctx, "POST", deviceAuthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device auth request failed with status %d", resp.StatusCode)
	}

	var deviceAuth DeviceAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceAuth); err != nil {
		return nil, err
	}

	return &deviceAuth, nil
}

func (d *DeviceFlowAuth) pollForToken(ctx context.Context, deviceAuth *DeviceAuthResponse) (*TokenResponse, error) {
	interval := deviceAuth.Interval
	if interval < 5 {
		interval = 5
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	timeout := time.After(time.Duration(deviceAuth.ExpiresIn) * time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("device code expired")
		case <-ticker.C:
			tokenResp, err := d.requestToken(ctx, deviceAuth.DeviceCode)
			if err != nil {
				return nil, err
			}

			switch tokenResp.Error {
			case "":
				// Success!
				return tokenResp, nil
			case "authorization_pending":
				// User hasn't authorized yet, keep polling
				d.logger.Debug().Msg("Waiting for user authorization...")
				continue
			case "slow_down":
				// Server wants us to slow down
				interval += 5
				ticker.Reset(time.Duration(interval) * time.Second)
				continue
			case "expired_token":
				return nil, fmt.Errorf("device code expired")
			case "access_denied":
				return nil, fmt.Errorf("access denied by user")
			default:
				return nil, fmt.Errorf("token error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
			}
		}
	}
}

func (d *DeviceFlowAuth) requestToken(ctx context.Context, deviceCode string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("device_code", deviceCode)
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("scope", scope)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}
