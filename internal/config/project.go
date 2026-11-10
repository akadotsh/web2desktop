package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Project is the portable, user-owned build configuration.
type Project struct {
	URL            string   `json:"url"`
	Name           string   `json:"name"`
	Identifier     string   `json:"identifier"`
	Version        string   `json:"version"`
	Width          int      `json:"width"`
	Height         int      `json:"height"`
	Icon           string   `json:"icon,omitempty"`
	Platforms      []string `json:"platforms"`
	Permissions    []string `json:"permissions,omitempty"`
	AllowedOrigins []string `json:"allowedOrigins,omitempty"`
	AutoName       bool     `json:"autoName,omitempty"`
	Overwrite      bool     `json:"overwrite,omitempty"`
}

func NewProject(c Config, platforms []string) Project {
	return Project{
		URL:            c.URL,
		Name:           c.Name,
		Identifier:     c.Identifier,
		Version:        c.Version,
		Width:          c.Width,
		Height:         c.Height,
		Icon:           c.Icon,
		Platforms:      platforms,
		Permissions:    c.Permissions,
		AllowedOrigins: c.AllowedOrigins,
		AutoName:       c.AutoName,
		Overwrite:      c.Overwrite,
	}
}

func (p Project) BuildConfig() Config {
	return Config{
		URL: p.URL, Name: p.Name, Identifier: p.Identifier, Version: p.Version,
		Width: p.Width, Height: p.Height, Icon: p.Icon, Permissions: p.Permissions,
		AllowedOrigins: p.AllowedOrigins, Overwrite: p.Overwrite,
		AutoName: p.AutoName, IdentifierExplicit: p.Identifier != "" && p.Identifier != Identifier(p.Name),
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
	b = bytes.TrimSpace(b)
	if len(b) == 0 || b[0] != '{' {
		return Project{}, fmt.Errorf("invalid project file: expected a JSON object")
	}
	var p Project
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return Project{}, fmt.Errorf("invalid project file: %w", err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return Project{}, fmt.Errorf("invalid project file: expected one JSON object")
	}
	return p, nil
}

const ProjectFileName = "web2desktop.json"
