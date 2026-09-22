# Hytale Downloader

[![Release](https://img.shields.io/github/v/release/AselionHYT/hytale-downloader)](https://github.com/AselionHYT/hytale-downloader/releases)
[![License](https://img.shields.io/github/license/AselionHYT/hytale-downloader)](LICENSE)

A native command-line tool for downloading Hytale game files on macOS and Linux. The official Hytale downloader only supports Windows, so this tool fills the gap for Mac and Linux users.

## Features

- Native macOS ARM64 and Linux AMD64 support
- OAuth2 Device Flow authentication
- **Headless mode for CI/CD pipelines**
- Parallel chunk downloads (up to 4x faster)
- SHA256 checksum verification
- Resume interrupted downloads
- Interactive patchline selection
- Beautiful terminal UI

## Installation

### Homebrew (macOS / Linux)

```bash
brew install AselionHYT/tap/hytale-downloader
```

Or with tap:

```bash
brew tap AselionHYT/tap
brew install hytale-downloader
```

### Manual Download

Download the latest release from the [Releases](https://github.com/AselionHYT/hytale-downloader/releases) page.

### Build from Source

```bash
git clone https://github.com/AselionHYT/hytale-downloader.git
cd hytale-downloader
make build

# Or build for specific platform
make build-darwin-arm64  # macOS ARM64
make build-linux-amd64   # Linux AMD64
make build-all           # All platforms
```

## Usage

### Interactive Mode

Simply run without arguments for an interactive experience:

```bash
hytale-downloader
```

This will:
1. Prompt you to select a patchline (release/pre-release)
2. Authenticate via browser if needed
3. Download the game files with a progress bar

### Command-Line Options

```bash
# Download release version
hytale-downloader -patchline release

# Download pre-release version
hytale-downloader -patchline pre-release

# Custom download path
hytale-downloader -download-path ./server.zip

# Show current game version only
hytale-downloader -print-version

# Skip confirmation prompts
hytale-downloader -yes

# Adjust parallel workers (default: 4)
hytale-downloader -workers 8

# Show version info
hytale-downloader -version

# Enable debug logging
hytale-downloader -debug
```

### Available Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-patchline` | Patchline to download (release, pre-release) | Interactive |
| `-download-path` | Custom path for downloaded file | `hytale-server-{version}.zip` |
| `-print-version` | Print game version and exit | false |
| `-yes` | Skip confirmation prompts | false |
| `-workers` | Number of parallel download workers | 4 |
| `-version` | Show downloader version | false |
| `-debug` | Enable debug logging | false |
| `-headless` | Run without interactive prompts (CI/CD) | false |
| `-credentials` | Path to custom credentials file | Default location |
| `-logout` | Delete stored credentials and exit | false |

## CI/CD / Headless Mode

For automated pipelines, use headless mode with environment variables or credentials file:

### Option 1: Environment Variable (Recommended)

```bash
# Set the refresh token
export HYTALE_REFRESH_TOKEN="ory_rt_your_refresh_token_here"

# Run in headless mode
hytale-downloader -headless -patchline release -download-path ./server.zip
```

### Option 2: Credentials File

```bash
# Use a mounted credentials file
hytale-downloader -headless -credentials /secrets/credentials.json -patchline release
```

### Docker Example

```dockerfile
FROM alpine:latest

# Copy the Linux binary
COPY hytale-downloader-linux-amd64 /usr/local/bin/hytale-downloader
RUN chmod +x /usr/local/bin/hytale-downloader

# Run with environment variable
ENV HYTALE_REFRESH_TOKEN=""
CMD ["hytale-downloader", "-headless", "-patchline", "release", "-download-path", "/out/server.zip"]
```

### GitHub Actions Example

```yaml
jobs:
  download:
    runs-on: ubuntu-latest
    steps:
      - name: Download Hytale Server
        env:
          HYTALE_REFRESH_TOKEN: ${{ secrets.HYTALE_REFRESH_TOKEN }}
        run: |
          curl -L -o hytale-downloader https://github.com/AselionHYT/hytale-downloader/releases/latest/download/hytale-downloader-linux-amd64
          chmod +x hytale-downloader
          ./hytale-downloader -headless -patchline release -download-path server.zip
```

### Getting Your Refresh Token

1. Run the downloader interactively once to authenticate:
   ```bash
   hytale-downloader -print-version
   ```
2. Copy the refresh token from `~/.hytale-downloader/credentials.json`
3. Store it securely in your CI/CD secrets

**Note:** Refresh tokens are rotated on each use. When using environment variables, the new token is not saved. For long-running CI/CD, consider using a credentials file that persists the rotated token.

## Authentication

The downloader uses OAuth2 Device Flow for authentication:

1. On first run, you'll see a URL and code
2. Open the URL in your browser
3. Log in with your Hytale account
4. Enter the code when prompted
5. Credentials are saved to `~/.hytale-downloader/credentials.json`

Credentials are automatically refreshed on subsequent runs.

### Logout

To delete stored credentials:

```bash
hytale-downloader -logout
```

Or manually:

```bash
rm ~/.hytale-downloader/credentials.json
```

## Configuration

Credentials are stored in:
- `~/.hytale-downloader/credentials.json`

## Platforms

| Platform | Architecture | Status |
|----------|--------------|--------|
| macOS | ARM64 (Apple Silicon) | Supported |
| Linux | AMD64 | Supported |
| macOS | AMD64 (Intel) | Not supported |
| Windows | Any | Not supported |

## Requirements

- macOS ARM64 or Linux AMD64
- Valid Hytale account with appropriate access

## License

MIT License - see [LICENSE](LICENSE) for details.

## Disclaimer

This is an unofficial tool and is not affiliated with or endorsed by Hypixel Studios or Riot Games.
