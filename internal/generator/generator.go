package generator

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/*
var files embed.FS

type Options struct {
	Name, URL, Identifier, Version string
	Width, Height                  int
	DevTools                       bool
}

func Generate(dir string, o Options, icon string) error {
	if err := os.MkdirAll(filepath.Join(dir, "resources"), 0755); err != nil {
		return err
	}
	content, err := files.ReadFile("templates/main.go.tmpl")
	if err != nil {
		return err
	}
	t, err := template.New("main").Parse(string(content))
	if err != nil {
		return err
	}
	main, err := os.Create(filepath.Join(dir, "main.go"))
	if err != nil {
		return err
	}
	err = t.Execute(main, o)
	closeErr := main.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	mod := "module generated.local/web2desktop\n\ngo 1.27.3\n\nrequire github.com/egoist/mygo v0.7.4\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0644); err != nil {
		return err
	}
	config := map[string]any{"name": o.Name, "identifier": o.Identifier, "version": o.Version}
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "mygo.json"), append(b, '\n'), 0644); err != nil {
		return err
	}
	if icon != "" {
		return copyIcon(icon, filepath.Join(dir, "resources", "icon.png"))
	}
	return os.WriteFile(filepath.Join(dir, "resources", "icon.png"), defaultIcon, 0644)
}

// copyIcon validates the PNG signature; MyGo performs sizing and format conversion.
func copyIcon(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if len(b) < 24 || string(b[:8]) != "\x89PNG\r\n\x1a\n" {
		return fmt.Errorf("icon must be a PNG file")
	}
	width, height := int(b[16])<<24|int(b[17])<<16|int(b[18])<<8|int(b[19]), int(b[20])<<24|int(b[21])<<16|int(b[22])<<8|int(b[23])
	if width != height {
		return fmt.Errorf("icon must be square (got %dx%d)", width, height)
	}
	return os.WriteFile(dst, b, 0644)
}

var defaultIcon = []byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 73, 72, 68, 82, 0, 0, 0, 1, 0, 0, 0, 1, 8, 6, 0, 0, 0, 31, 21, 196, 137, 0, 0, 0, 13, 73, 68, 65, 84, 120, 156, 99, 248, 207, 192, 240, 31, 0, 5, 0, 1, 255, 137, 153, 61, 29, 0, 0, 0, 0, 73, 69, 78, 68, 174, 66, 96, 130}
