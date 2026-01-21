// Package ui provides terminal UI components with styling.
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	successColor   = lipgloss.Color("#10B981") // Green
	errorColor     = lipgloss.Color("#EF4444") // Red
	warningColor   = lipgloss.Color("#F59E0B") // Yellow
	subtleColor    = lipgloss.Color("#6B7280") // Gray
	highlightColor = lipgloss.Color("#3B82F6") // Blue

	// Title box style
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 2)

	// Header style
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	// Success message style
	SuccessStyle = lipgloss.NewStyle().
			Foreground(successColor)

	// Error message style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	// Warning message style
	WarningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	// Subtle/dim text style
	SubtleStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	// Highlight style for important info
	HighlightStyle = lipgloss.NewStyle().
			Foreground(highlightColor).
			Bold(true)

	// Version info style
	VersionStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			Italic(true)

	// URL style
	URLStyle = lipgloss.NewStyle().
			Foreground(highlightColor).
			Underline(true)

	// Success box style
	SuccessBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(successColor).
			Padding(0, 2)

	// Error box style
	ErrorBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(errorColor).
			Padding(0, 2)
)

// Symbols for terminal output
const (
	SymbolSuccess  = "✓"
	SymbolError    = "✗"
	SymbolWarning  = "!"
	SymbolInfo     = "•"
	SymbolArrow    = "→"
	SymbolDownload = "↓"
	SymbolWaiting  = "○"
	SymbolDone     = "●"
)
