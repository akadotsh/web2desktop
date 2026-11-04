package config

import (
	"path/filepath"
	"testing"
)

func TestProjectRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "example.web2desktop.json")
	want := Project{
		URL: "https://example.com", Name: "Example", Identifier: "com.example.app",
		Version: "1.2.3", Width: 1200, Height: 800, Platforms: []string{"darwin/arm64", "linux/amd64"},
	}
	if err := WriteProject(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadProject(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != want.URL || got.Name != want.Name || got.Identifier != want.Identifier || got.Version != want.Version || got.Width != want.Width || got.Height != want.Height {
		t.Fatalf("loaded project = %#v, want %#v", got, want)
	}
	if len(got.Platforms) != 2 || got.Platforms[1] != "linux/amd64" {
		t.Fatalf("platforms = %#v", got.Platforms)
	}
}

func TestValidateDerivesIdentifierFromMultiWordName(t *testing.T) {
	c := Config{URL: "https://example.com", Name: "Example App", Width: 1200, Height: 800}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if got, want := c.Identifier, "com.web2desktop.exampleapp"; got != want {
		t.Fatalf("identifier = %q, want %q", got, want)
	}
}
