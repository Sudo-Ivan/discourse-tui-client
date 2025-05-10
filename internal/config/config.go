// Copyright (c) 2025 Sudo-Ivan
// MIT License

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type ColorConfig struct {
	Title    string
	Item     string
	Selected string
	Status   string
	Error    string
}

var DefaultColors = ColorConfig{
	Title:    "#FF4444",
	Item:     "#FF8888",
	Selected: "#FF0000",
	Status:   "#CC0000",
	Error:    "#FF0000",
}

func LoadColors(path string) (ColorConfig, error) {
	colors := DefaultColors
	/* #nosec G304 */
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default colors file
			if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
				return colors, fmt.Errorf("failed to create config directory: %w", err)
			}
			if err := os.WriteFile(path, []byte(fmt.Sprintf("title=%s\nitem=%s\nselected=%s\nstatus=%s\nerror=%s",
				colors.Title, colors.Item, colors.Selected, colors.Status, colors.Error)), 0600); err != nil { //nosec G306
				return colors, fmt.Errorf("failed to write default colors: %w", err)
			}
			return colors, nil
		}
		return colors, fmt.Errorf("failed to read colors file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "title":
			colors.Title = value
		case "item":
			colors.Item = value
		case "selected":
			colors.Selected = value
		case "status":
			colors.Status = value
		case "error":
			colors.Error = value
		}
	}
	return colors, nil
}

var (
	TitleStyle        lipgloss.Style
	ItemStyle         lipgloss.Style
	SelectedItemStyle lipgloss.Style
	StatusStyle       lipgloss.Style
	ErrorStyle        lipgloss.Style
)

func UpdateStyles(colors ColorConfig) {
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colors.Title)).
		PaddingLeft(2)

	ItemStyle = lipgloss.NewStyle().
		PaddingLeft(4).
		Foreground(lipgloss.Color(colors.Item))

	SelectedItemStyle = lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color(colors.Selected))

	StatusStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Status)).
		PaddingLeft(2)

	ErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Error)).
		PaddingLeft(2)
}

func GetInstancesPath() string {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(userConfigDir, "discourse-tui-client", "instances.txt")
}

func SaveInstance(instanceURL string) error {
	path := GetInstancesPath()
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	return os.WriteFile(path, []byte(instanceURL), 0600)
}

func LoadInstance() (string, error) {
	path := GetInstancesPath()
	// #nosec G304
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read instances file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}
