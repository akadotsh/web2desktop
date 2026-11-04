package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Project is the portable, user-owned build configuration.
type Project struct {
	URL        string   `json:"url"`
	Name       string   `json:"name"`
	Identifier string   `json:"identifier"`
	Version    string   `json:"version"`
	Width      int      `json:"width"`
	Height     int      `json:"height"`
	Icon       string   `json:"icon,omitempty"`
	Platforms  []string `json:"platforms"`
}

func NewProject(c Config, platforms []string) Project {
	return Project{
		URL:        c.URL,
		Name:       c.Name,
		Identifier: c.Identifier,
		Version:    c.Version,
		Width:      c.Width,
		Height:     c.Height,
		Icon:       c.Icon,
		Platforms:  platforms,
	}
}

func (p Project) BuildConfig() Config {
	return Config{
		URL: p.URL, Name: p.Name, Identifier: p.Identifier, Version: p.Version,
		Width: p.Width, Height: p.Height, Icon: p.Icon,
	}
}

func WriteProject(path string, p Project) error {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	return f.Close()
}

func LoadProject(path string) (Project, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Project{}, err
	}
	var p Project
	if err := json.Unmarshal(b, &p); err != nil {
		return Project{}, fmt.Errorf("invalid project file: %w", err)
	}
	return p, nil
}

const ProjectFileName = "web2desktop.json"
