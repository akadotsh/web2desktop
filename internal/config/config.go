package config

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
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
	Description                                  string
	Permissions                                  []string
	AllowedOrigins                               []string
	Width, Height                                int
	Dev, Verbose, KeepWorkdir                    bool
	Overwrite                                    bool
	AutoName, IdentifierExplicit                 bool
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
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return errors.New("URL port must be between 1 and 65535")
		}
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
		c.Identifier = Identifier(c.Name)
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
	seen := map[string]bool{}
	for _, permission := range c.Permissions {
		switch permission {
		case "camera", "microphone", "geolocation", "notifications":
		default:
			return fmt.Errorf("unsupported permission %q", permission)
		}
		if seen[permission] {
			return fmt.Errorf("duplicate permission %q", permission)
		}
		seen[permission] = true
	}
	baseOrigin, _ := NormalizeOrigin(u.Scheme + "://" + u.Host)
	seenOrigins := map[string]bool{baseOrigin: true}
	for i, raw := range c.AllowedOrigins {
		origin, err := NormalizeOrigin(raw)
		if err != nil {
			return fmt.Errorf("invalid allowed origin %q: %w", raw, err)
		}
		if seenOrigins[origin] {
			return fmt.Errorf("duplicate allowed origin %q", raw)
		}
		seenOrigins[origin] = true
		c.AllowedOrigins[i] = origin
	}
	return nil
}

func NormalizeOrigin(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Opaque != "" || u.Hostname() == "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(raw, "#") || u.Path != "" && u.Path != "/" {
		return "", errors.New("must be an http or https origin without credentials, path, query, or fragment")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", errors.New("must be an http or https origin")
	}
	port := u.Port()
	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	} else {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return "", errors.New("port must be between 1 and 65535")
		}
		port = strconv.Itoa(n)
	}
	return scheme + "://" + strings.ToLower(u.Hostname()) + ":" + port, nil
}

func Identifier(name string) string {
	name = strings.ReplaceAll(Slug(name), "-", "")
	if name == "" {
		name = "app"
	}
	return "com.web2desktop." + name
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
