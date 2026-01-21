package ui

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// SelectPatchline prompts the user to select a patchline.
func SelectPatchline() (string, error) {
	var patchline string

	err := huh.NewSelect[string]().
		Title("Select patchline").
		Options(
			huh.NewOption("release", "release"),
			huh.NewOption("pre-release", "pre-release"),
		).
		Value(&patchline).
		Run()

	if err != nil {
		return "", fmt.Errorf("failed to select patchline: %w", err)
	}

	return patchline, nil
}

// ConfirmDownload prompts the user to confirm the download.
// If sizeMB is 0, the size is not shown.
func ConfirmDownload(filename string, sizeMB float64) (bool, error) {
	var confirm bool

	var message string
	if sizeMB > 0 {
		message = fmt.Sprintf("Download %s (%.1f MB)?", filename, sizeMB)
	} else {
		message = fmt.Sprintf("Download %s?", filename)
	}

	err := huh.NewConfirm().
		Title(message).
		Affirmative("Yes").
		Negative("No").
		Value(&confirm).
		Run()

	if err != nil {
		return false, fmt.Errorf("failed to confirm download: %w", err)
	}

	return confirm, nil
}

// InputPath prompts the user for a custom download path.
func InputPath(defaultPath string) (string, error) {
	var path string

	err := huh.NewInput().
		Title("Download path").
		Value(&path).
		Placeholder(defaultPath).
		Run()

	if err != nil {
		return "", fmt.Errorf("failed to input path: %w", err)
	}

	if path == "" {
		return defaultPath, nil
	}

	return path, nil
}
