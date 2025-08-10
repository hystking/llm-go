package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

type Profile struct {
	Provider        string `json:"provider"`
	BaseURL         string `json:"base_url"`
	Model           string `json:"model"`
	MaxTokens       int    `json:"max_tokens"`
	Instructions    string `json:"instructions"`
	Verbosity       string `json:"verbosity"`
	ReasoningEffort string `json:"reasoning_effort"`
	Format          string `json:"format"`
	Only            string `json:"only"`
}

type File struct {
	DefaultProfile string             `json:"default_profile"`
	Profiles       map[string]Profile `json:"profiles"`
}

// Load reads config from the given path (or default) and returns the selected profile.
// Selection precedence:
// 1) explicit profileName arg (if non-empty)
// 2) file's default_profile
// If no config file or no profile found, returns zero Profile and nil error.
func Load(configPath, profileName string) (Profile, error) {
	cfgPath := configPath
	if cfgPath == "" {
		// No path provided; caller is responsible for defaults.
		return Profile{}, nil
	}

	f, err := os.Open(cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Profile{}, nil
		}
		return Profile{}, err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return Profile{}, err
	}

	var file File
	if err := json.Unmarshal(b, &file); err != nil {
		return Profile{}, err
	}

	// choose name
	name := profileName
	if name == "" {
		name = file.DefaultProfile
	}
	if name == "" {
		return Profile{}, nil
	}
	p, ok := file.Profiles[name]
	if !ok {
		return Profile{}, nil
	}
	return p, nil
}

// DefaultPath returns the default config file path following XDG conventions.
// Typically: ${XDG_CONFIG_HOME:-~/.config}/llmx/config.json
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "llmx", "config.json"), nil
}
