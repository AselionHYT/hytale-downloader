// Package download handles file downloads with parallel chunks and verification.
package download

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// VerifyChecksum verifies the SHA256 checksum of a file.
func VerifyChecksum(filepath, expectedHash string) error {
	if expectedHash == "" {
		return nil // No checksum to verify
	}

	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

// checksumWriter wraps a writer and computes SHA256 on the fly.
type checksumWriter struct {
	writer io.Writer
	hasher io.Writer
	sum    func() []byte
}

// newChecksumWriter creates a writer that also computes SHA256.
func newChecksumWriter(w io.Writer) *checksumWriter {
	h := sha256.New()
	return &checksumWriter{
		writer: w,
		hasher: h,
		sum:    func() []byte { return h.Sum(nil) },
	}
}

func (c *checksumWriter) Write(p []byte) (n int, err error) {
	n, err = c.writer.Write(p)
	if n > 0 {
		c.hasher.Write(p[:n])
	}
	return
}

func (c *checksumWriter) Sum() string {
	return hex.EncodeToString(c.sum())
}
