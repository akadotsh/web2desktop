package config

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

const DefaultVersion = "0.1.0"

var (
	identifierPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*(\.[A-Za-z][A-Za-z0-9]*)+$`)
	versionPattern    = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)([-+][0-9A-Za-z.-]+)?$`)
	schemePattern     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)
)

type Config struct {
	URL, Name, Identifier, Version, Icon, Output string
	Width, Height                                int
	Dev, Verbose, KeepWorkdir                    bool
}

func (c *Config) Validate() error {
	if !schemePattern.MatchString(c.URL) && !strings.HasPrefix(c.URL, "//") {
		c.URL = "https://" + c.URL
	}
	u, err := url.ParseRequestURI(c.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return errors.New("URL must be an absolute http or https URL")
	}
	if u.User != nil {
		return errors.New("URL must not contain credentials")
	}
	if c.Name == "" {
		c.Name = DeriveName(u.Hostname())
	}
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("name cannot be empty")
	}
	if c.Version == "" {
		c.Version = DefaultVersion
	}
	if !versionPattern.MatchString(c.Version) {
		return fmt.Errorf("invalid semantic version %q", c.Version)
	}
	if c.Width < 320 || c.Width > 10000 {
		return errors.New("width must be between 320 and 10000")
	}
	if c.Height < 240 || c.Height > 10000 {
		return errors.New("height must be between 240 and 10000")
	}
	if c.Identifier == "" {
		name := strings.ReplaceAll(Slug(c.Name), "-", "")
		if name == "" {
			name = "app"
		}
		c.Identifier = "com.web2desktop." + name
	}
	if !identifierPattern.MatchString(c.Identifier) {
		return fmt.Errorf("invalid bundle identifier %q", c.Identifier)
	}
	if c.Icon != "" {
		abs, err := filepath.Abs(c.Icon)
		if err != nil {
			return err
		}
		c.Icon = abs
	}
	if c.Output == "" {
		c.Output = "./dist"
	}
	return nil
}

func Slug(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			lastDash = false
		} else if b.Len() > 0 && !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func DeriveName(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) > 2 && parts[0] == "www" {
		parts = parts[1:]
	}
	if len(parts) > 1 {
		host = strings.Join(parts[:len(parts)-1], ".")
	} else {
		host = parts[0]
	}
	host = strings.ReplaceAll(host, "-", " ")
	return strings.ToUpper(host[:1]) + host[1:]
}
