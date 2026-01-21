package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

const (
	defaultWorkers   = 4
	defaultChunkSize = 50 * 1024 * 1024 // 50MB chunks
	bufferSize       = 1024 * 1024      // 1MB buffer
	userAgent        = "Go-http-client/2.0"
)

// Progress represents download progress information.
type Progress struct {
	Downloaded int64
	Total      int64
	Speed      float64 // bytes per second
}

// ProgressCallback is called with download progress updates.
type ProgressCallback func(Progress)

// Downloader handles parallel chunk downloads.
type Downloader struct {
	Workers    int
	ChunkSize  int64
	logger     zerolog.Logger
	onProgress ProgressCallback
}

// NewDownloader creates a new parallel downloader.
func NewDownloader(logger zerolog.Logger, onProgress ProgressCallback) *Downloader {
	return &Downloader{
		Workers:    defaultWorkers,
		ChunkSize:  defaultChunkSize,
		logger:     logger,
		onProgress: onProgress,
	}
}

// SetProgressCallback sets or updates the progress callback.
func (d *Downloader) SetProgressCallback(callback ProgressCallback) {
	d.onProgress = callback
}

// Download downloads a file using parallel chunks.
func (d *Downloader) Download(ctx context.Context, url, destPath, expectedHash string) error {
	d.logger.Debug().
		Str("url", url).
		Str("dest", destPath).
		Int("workers", d.Workers).
		Msg("Starting download")

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Get file size with HEAD request
	totalSize, supportsRange, err := d.GetFileInfo(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	d.logger.Debug().
		Int64("size", totalSize).
		Bool("supports_range", supportsRange).
		Msg("File info retrieved")

	// Use single-threaded download if server doesn't support range requests
	// or file is smaller than chunk size
	if !supportsRange || totalSize < d.ChunkSize || d.Workers <= 1 {
		d.logger.Info().Msg("Using single-threaded download")
		return d.downloadSingle(ctx, url, destPath, totalSize, expectedHash)
	}

	d.logger.Info().
		Int("workers", d.Workers).
		Msg("Using parallel download")

	return d.downloadParallel(ctx, url, destPath, totalSize, expectedHash)
}

// GetFileInfo retrieves the file size and checks if the server supports range requests.
func (d *Downloader) GetFileInfo(ctx context.Context, url string) (size int64, supportsRange bool, err error) {
	// First try HEAD request
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return 0, false, err
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, false, err
	}
	resp.Body.Close()

	// HEAD worked
	if resp.StatusCode == http.StatusOK {
		supportsRange = resp.Header.Get("Accept-Ranges") == "bytes"
		return resp.ContentLength, supportsRange, nil
	}

	// HEAD failed (403/405), try GET with Range header to probe
	d.logger.Debug().Int("status", resp.StatusCode).Msg("HEAD failed, trying GET with Range")

	req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, false, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Range", "bytes=0-0") // Request just first byte

	resp, err = client.Do(req)
	if err != nil {
		return 0, false, err
	}
	defer resp.Body.Close()

	// Check if server supports range requests
	if resp.StatusCode == http.StatusPartialContent {
		// Parse Content-Range header: "bytes 0-0/TOTAL_SIZE"
		contentRange := resp.Header.Get("Content-Range")
		if contentRange != "" {
			var start, end, total int64
			if _, err := fmt.Sscanf(contentRange, "bytes %d-%d/%d", &start, &end, &total); err == nil {
				return total, true, nil
			}
		}
	}

	// Server doesn't support range, try to get Content-Length from a regular GET
	// But abort immediately - we just need the headers
	if resp.StatusCode == http.StatusOK {
		size = resp.ContentLength
		return size, false, nil
	}

	return 0, false, fmt.Errorf("failed to get file info: status %d", resp.StatusCode)
}

func (d *Downloader) downloadSingle(ctx context.Context, url, destPath string, totalSize int64, expectedHash string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 0} // No timeout for large downloads
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create output file
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Download with progress tracking and checksum
	hasher := sha256.New()
	writer := io.MultiWriter(out, hasher)

	var downloaded int64
	startTime := time.Now()
	buf := make([]byte, bufferSize)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := writer.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)

			if d.onProgress != nil {
				elapsed := time.Since(startTime).Seconds()
				speed := float64(downloaded) / elapsed
				d.onProgress(Progress{
					Downloaded: downloaded,
					Total:      totalSize,
					Speed:      speed,
				})
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	// Verify checksum
	if expectedHash != "" {
		actualHash := hex.EncodeToString(hasher.Sum(nil))
		if actualHash != expectedHash {
			os.Remove(destPath)
			return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
		}
	}

	return nil
}

func (d *Downloader) downloadParallel(ctx context.Context, url, destPath string, totalSize int64, expectedHash string) error {
	// Calculate chunks
	numChunks := (totalSize + d.ChunkSize - 1) / d.ChunkSize
	chunks := make([]chunk, numChunks)

	for i := int64(0); i < numChunks; i++ {
		start := i * d.ChunkSize
		end := start + d.ChunkSize - 1
		if end >= totalSize {
			end = totalSize - 1
		}
		chunks[i] = chunk{
			index: int(i),
			start: start,
			end:   end,
		}
	}

	d.logger.Debug().
		Int64("chunks", numChunks).
		Int64("chunk_size", d.ChunkSize).
		Msg("Chunks calculated")

	// Create temp directory for chunks
	tempDir, err := os.MkdirTemp("", "hytale-download-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Download chunks in parallel
	var downloaded int64
	startTime := time.Now()

	chunkChan := make(chan chunk, len(chunks))
	errChan := make(chan error, d.Workers)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < d.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for c := range chunkChan {
				chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk-%d", c.index))
				if err := d.downloadChunk(ctx, url, chunkPath, c, &downloaded, startTime, totalSize); err != nil {
					errChan <- fmt.Errorf("chunk %d failed: %w", c.index, err)
					return
				}
			}
		}(i)
	}

	// Send chunks to workers
	for _, c := range chunks {
		chunkChan <- c
	}
	close(chunkChan)

	// Wait for completion
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		return err
	}

	// Merge chunks
	d.logger.Debug().Msg("Merging chunks")
	if err := d.mergeChunks(tempDir, destPath, len(chunks)); err != nil {
		return fmt.Errorf("failed to merge chunks: %w", err)
	}

	// Verify checksum
	if expectedHash != "" {
		d.logger.Debug().Msg("Verifying checksum")
		if err := VerifyChecksum(destPath, expectedHash); err != nil {
			os.Remove(destPath)
			return err
		}
	}

	return nil
}

type chunk struct {
	index int
	start int64
	end   int64
}

func (d *Downloader) downloadChunk(ctx context.Context, url, destPath string, c chunk, downloaded *int64, startTime time.Time, totalSize int64) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", c.start, c.end))

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chunk download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	buf := make([]byte, bufferSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			newDownloaded := atomic.AddInt64(downloaded, int64(n))

			if d.onProgress != nil {
				elapsed := time.Since(startTime).Seconds()
				speed := float64(newDownloaded) / elapsed
				d.onProgress(Progress{
					Downloaded: newDownloaded,
					Total:      totalSize,
					Speed:      speed,
				})
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Downloader) mergeChunks(tempDir, destPath string, numChunks int) error {
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	for i := 0; i < numChunks; i++ {
		chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk-%d", i))
		chunk, err := os.Open(chunkPath)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, chunk); err != nil {
			chunk.Close()
			return err
		}
		chunk.Close()
	}

	return nil
}
