package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ProgressBar renders a terminal progress bar.
type ProgressBar struct {
	width      int
	downloaded int64
	total      int64
	speed      float64
	startTime  time.Time
}

// NewProgressBar creates a new progress bar.
func NewProgressBar(width int) *ProgressBar {
	return &ProgressBar{
		width:     width,
		startTime: time.Now(),
	}
}

// Update updates the progress bar state.
func (p *ProgressBar) Update(downloaded, total int64, speed float64) {
	p.downloaded = downloaded
	p.total = total
	p.speed = speed
}

// Render returns the progress bar string.
func (p *ProgressBar) Render() string {
	if p.total == 0 {
		return ""
	}

	percent := float64(p.downloaded) / float64(p.total)
	filled := int(float64(p.width) * percent)
	empty := p.width - filled

	// Build progress bar
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	// Format sizes
	downloadedMB := float64(p.downloaded) / 1024 / 1024
	totalMB := float64(p.total) / 1024 / 1024

	// Format speed
	speedStr := formatSpeed(p.speed)

	// Calculate ETA
	eta := p.calculateETA()

	// Color the bar
	barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))
	percentStyle := lipgloss.NewStyle().Bold(true)

	return fmt.Sprintf(
		"  %s %s\n\n  %.1f MB / %.1f MB   %s %s   ETA: %s",
		barStyle.Render(bar),
		percentStyle.Render(fmt.Sprintf("%3.0f%%", percent*100)),
		downloadedMB,
		totalMB,
		SymbolDownload,
		speedStr,
		eta,
	)
}

// RenderCompact returns a single-line progress bar.
func (p *ProgressBar) RenderCompact() string {
	if p.total == 0 {
		return ""
	}

	percent := float64(p.downloaded) / float64(p.total)
	filled := int(float64(p.width) * percent)
	empty := p.width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	downloadedMB := float64(p.downloaded) / 1024 / 1024
	totalMB := float64(p.total) / 1024 / 1024
	speedStr := formatSpeed(p.speed)

	return fmt.Sprintf(
		"\r  %s %3.0f%% | %.1f/%.1f MB | %s %s",
		bar,
		percent*100,
		downloadedMB,
		totalMB,
		SymbolDownload,
		speedStr,
	)
}

func (p *ProgressBar) calculateETA() string {
	if p.speed <= 0 || p.downloaded <= 0 {
		return "--:--"
	}

	remaining := p.total - p.downloaded
	seconds := float64(remaining) / p.speed

	if seconds > 3600 {
		return fmt.Sprintf("%dh%02dm", int(seconds)/3600, (int(seconds)%3600)/60)
	}
	return fmt.Sprintf("%02d:%02d", int(seconds)/60, int(seconds)%60)
}

func formatSpeed(bytesPerSec float64) string {
	if bytesPerSec >= 1024*1024*1024 {
		return fmt.Sprintf("%.1f GB/s", bytesPerSec/1024/1024/1024)
	}
	if bytesPerSec >= 1024*1024 {
		return fmt.Sprintf("%.1f MB/s", bytesPerSec/1024/1024)
	}
	if bytesPerSec >= 1024 {
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/1024)
	}
	return fmt.Sprintf("%.0f B/s", bytesPerSec)
}
