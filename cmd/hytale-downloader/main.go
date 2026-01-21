// Hytale Downloader - Native macOS ARM64 client for downloading Hytale game files.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/AselionHYT/hytale-downloader/internal/api"
	"github.com/AselionHYT/hytale-downloader/internal/auth"
	"github.com/AselionHYT/hytale-downloader/internal/config"
	"github.com/AselionHYT/hytale-downloader/internal/download"
	"github.com/AselionHYT/hytale-downloader/internal/ui"
	"github.com/AselionHYT/hytale-downloader/pkg/version"
)

func main() {
	// CLI Flags
	patchlineFlag := flag.String("patchline", "", "Patchline to download (release, pre-release)")
	downloadPathFlag := flag.String("download-path", "", "Path to save the downloaded file")
	printVersionFlag := flag.Bool("print-version", false, "Print the current game version and exit")
	showVersionFlag := flag.Bool("version", false, "Show hytale-downloader version")
	debugFlag := flag.Bool("debug", false, "Enable debug logging")
	yesFlag := flag.Bool("yes", false, "Skip confirmation prompts")
	workersFlag := flag.Int("workers", 4, "Number of parallel download workers")
	credentialsFlag := flag.String("credentials", "", "Path to credentials file (for CI/CD)")
	headlessFlag := flag.Bool("headless", false, "Run in headless mode (no interactive prompts, for CI/CD)")
	logoutFlag := flag.Bool("logout", false, "Delete stored credentials and exit")
	flag.Parse()

	// Handle logout flag
	if *logoutFlag {
		handleLogout()
		return
	}

	// Setup logger
	setupLogger(*debugFlag)

	// Handle version flag
	if *showVersionFlag {
		fmt.Println(version.Full())
		return
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\nInterrupted. Cleaning up...")
		cancel()
	}()

	// Run the application
	if err := run(ctx, &appConfig{
		patchline:       *patchlineFlag,
		downloadPath:    *downloadPathFlag,
		printVersion:    *printVersionFlag,
		yes:             *yesFlag,
		workers:         *workersFlag,
		credentialsFile: *credentialsFlag,
		headless:        *headlessFlag,
	}); err != nil {
		fmt.Fprintln(os.Stderr, ui.ErrorStyle.Render(fmt.Sprintf("\n%s Error: %v", ui.SymbolError, err)))
		os.Exit(1)
	}
}

type appConfig struct {
	patchline       string
	downloadPath    string
	printVersion    bool
	yes             bool
	workers         int
	credentialsFile string
	headless        bool
}

func setupLogger(debug bool) {
	level := zerolog.InfoLevel
	if debug {
		level = zerolog.DebugLevel
	}

	zerolog.SetGlobalLevel(level)
	log.Logger = zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "15:04:05",
	}).With().Timestamp().Logger()
}

func run(ctx context.Context, cfg *appConfig) error {
	logger := log.Logger

	// Headless mode implies -yes
	if cfg.headless {
		cfg.yes = true
	}

	// Print header (unless headless)
	if !cfg.headless {
		printHeader()
	}

	// Get paths
	paths, err := config.GetPaths()
	if err != nil {
		return fmt.Errorf("failed to get config paths: %w", err)
	}

	// Initialize API client
	hytaleAPI := api.NewHytaleAPI(logger)

	// Authenticate
	accessToken, err := authenticate(ctx, paths, logger, cfg.credentialsFile, cfg.headless)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Select patchline
	patchline := cfg.patchline
	if patchline == "" {
		if cfg.headless {
			patchline = "release" // Default for headless
			logger.Info().Str("patchline", patchline).Msg("Using default patchline")
		} else {
			var err error
			patchline, err = ui.SelectPatchline()
			if err != nil {
				return err
			}
		}
	}

	// Get version info
	fmt.Printf("\n  %s Fetching %s version...", ui.SymbolWaiting, ui.HighlightStyle.Render(patchline))
	versionInfo, err := hytaleAPI.GetVersionInfo(ctx, accessToken, patchline)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("failed to get version info: %w", err)
	}
	fmt.Printf("\r  %s Version: %s\n", ui.SymbolSuccess, ui.HighlightStyle.Render(versionInfo.Version))

	// Handle print-version flag
	if cfg.printVersion {
		fmt.Printf("  %s SHA256: %s\n", ui.SymbolInfo, versionInfo.SHA256)
		return nil
	}

	// Determine download path
	downloadPath := cfg.downloadPath
	if downloadPath == "" {
		downloadPath = fmt.Sprintf("hytale-server-%s.zip", versionInfo.Version)
	}

	// Get download URL and file size
	fmt.Printf("  %s Preparing download...", ui.SymbolWaiting)
	downloadURL, err := hytaleAPI.GetDownloadURL(ctx, accessToken, versionInfo.DownloadURL)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("failed to get download URL: %w", err)
	}

	// Create downloader and get file info
	downloader := download.NewDownloader(logger, nil)
	downloader.Workers = cfg.workers

	fileSize, _, err := downloader.GetFileInfo(ctx, downloadURL)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("failed to get file info: %w", err)
	}

	fileSizeMB := float64(fileSize) / (1024 * 1024)
	fmt.Printf("\r  %s Ready: %.1f MB\n", ui.SymbolSuccess, fileSizeMB)

	// Confirm download
	if !cfg.yes {
		confirm, err := ui.ConfirmDownload(downloadPath, fileSizeMB)
		if err != nil {
			return err
		}
		if !confirm {
			fmt.Println("\n  Download cancelled.")
			return nil
		}
	}

	// Setup progress bar
	progressBar := ui.NewProgressBar(40)
	downloader.SetProgressCallback(func(p download.Progress) {
		progressBar.Update(p.Downloaded, p.Total, p.Speed)
		fmt.Print(progressBar.RenderCompact())
	})

	// Download
	fmt.Printf("\n  %s Downloading with %d parallel connections...\n\n", ui.SymbolDownload, cfg.workers)

	if err := downloader.Download(ctx, downloadURL, downloadPath, versionInfo.SHA256); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Success!
	fmt.Printf("\n\n  %s Download complete!\n", ui.SuccessStyle.Render(ui.SymbolSuccess))
	fmt.Printf("  %s Checksum verified!\n", ui.SuccessStyle.Render(ui.SymbolSuccess))

	printSuccess(downloadPath)

	return nil
}

func authenticate(ctx context.Context, paths *config.Paths, logger zerolog.Logger, credentialsFile string, headless bool) (string, error) {
	var creds *auth.Credentials
	var err error

	// Priority 1: Environment variable (best for CI/CD)
	if envCreds := auth.LoadCredentialsFromEnv(); envCreds != nil {
		logger.Debug().Msg("Using credentials from HYTALE_REFRESH_TOKEN environment variable")
		creds = envCreds
	}

	// Priority 2: Custom credentials file path
	if creds == nil && credentialsFile != "" {
		creds, err = auth.LoadCredentialsFromFile(credentialsFile)
		if err != nil {
			return "", fmt.Errorf("failed to load credentials from %s: %w", credentialsFile, err)
		}
		logger.Debug().Str("file", credentialsFile).Msg("Using credentials from custom file")
	}

	// Priority 3: Default config location
	if creds == nil {
		creds, err = auth.LoadCredentials(paths)
		if err != nil {
			logger.Debug().Err(err).Msg("No credentials in default location")
		}
	}

	// If we have credentials, try to refresh
	if creds != nil {
		if !headless {
			fmt.Printf("  %s Refreshing authentication...\n", ui.SymbolWaiting)
		}
		refresher := auth.NewTokenRefresher(logger)
		newCreds, err := refresher.RefreshToken(ctx, creds.RefreshToken)
		if err == nil {
			// Save rotated credentials (only if not using env var)
			if auth.LoadCredentialsFromEnv() == nil {
				if saveErr := auth.SaveCredentials(paths, newCreds); saveErr != nil {
					logger.Warn().Err(saveErr).Msg("Failed to save credentials")
				}
			}
			if !headless {
				fmt.Printf("  %s Authenticated!\n", ui.SuccessStyle.Render(ui.SymbolSuccess))
			} else {
				logger.Info().Msg("Authenticated successfully")
			}
			return newCreds.AccessToken, nil
		}
		logger.Debug().Err(err).Msg("Token refresh failed")
	}

	// In headless mode, we cannot do device flow
	if headless {
		return "", fmt.Errorf("no valid credentials found. In headless mode, provide HYTALE_REFRESH_TOKEN env var or -credentials flag")
	}

	// Start device flow (interactive only)
	fmt.Printf("\n  %s No valid credentials. Starting authentication...\n\n", ui.SymbolInfo)

	deviceFlow := auth.NewDeviceFlowAuth(logger, func(verificationURL, userCode string) {
		fmt.Println(ui.TitleStyle.Render("  Authentication Required  "))
		fmt.Println()
		fmt.Printf("  Open this URL: %s\n", ui.URLStyle.Render(verificationURL))
		fmt.Printf("  Or enter code: %s\n", ui.HighlightStyle.Render(userCode))
		fmt.Println()
		fmt.Printf("  %s Waiting for authorization...", ui.SymbolWaiting)
	})

	newCreds, err := deviceFlow.Authenticate(ctx)
	if err != nil {
		return "", err
	}

	// Save credentials
	if saveErr := auth.SaveCredentials(paths, newCreds); saveErr != nil {
		logger.Warn().Err(saveErr).Msg("Failed to save credentials")
	}

	fmt.Printf("\n  %s Authenticated!\n", ui.SuccessStyle.Render(ui.SymbolSuccess))
	return newCreds.AccessToken, nil
}

func printHeader() {
	title := fmt.Sprintf("  Hytale Downloader %s  ", version.Short())
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(title))
	fmt.Println()
}

func printSuccess(downloadPath string) {
	fmt.Println()
	successBox := ui.SuccessBoxStyle.Render(fmt.Sprintf(
		"  Success! Server ready at:\n  %s  ",
		downloadPath,
	))
	fmt.Println(successBox)
	fmt.Println()
}

func handleLogout() {
	paths, err := config.GetPaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s Error: %v\n", ui.SymbolError, err)
		os.Exit(1)
	}

	err = auth.DeleteCredentials(paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s Error deleting credentials: %v\n", ui.SymbolError, err)
		os.Exit(1)
	}

	fmt.Printf("%s Credentials deleted from %s\n", ui.SymbolSuccess, paths.CredentialsFile)
}
