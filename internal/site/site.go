package site

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type Metadata struct {
	Title, Description, IconURL string
}

var client = &http.Client{Timeout: 8 * time.Second}

func Fetch(rawURL string) (Metadata, error) {
	res, err := client.Get(rawURL)
	if err != nil {
		return Metadata{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Metadata{}, fmt.Errorf("website returned %s", res.Status)
	}
	base := res.Request.URL
	root, err := html.Parse(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return Metadata{}, err
	}
	var m Metadata
	var title []string
	var ogTitle string
	var siteName string
	var icons []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			attrs := make(map[string]string, len(n.Attr))
			for _, a := range n.Attr {
				attrs[strings.ToLower(a.Key)] = strings.TrimSpace(a.Val)
			}
			switch strings.ToLower(n.Data) {
			case "title":
				if n.FirstChild != nil {
					title = append(title, n.FirstChild.Data)
				}
			case "meta":
				key := strings.ToLower(attrs["name"])
				if key == "" {
					key = strings.ToLower(attrs["property"])
				}
				if (key == "description" || key == "og:description") && m.Description == "" {
					m.Description = attrs["content"]
				}
				if key == "og:title" && ogTitle == "" {
					ogTitle = attrs["content"]
				}
				if key == "og:site_name" && siteName == "" {
					siteName = attrs["content"]
				}
			case "link":
				rel := strings.Fields(strings.ToLower(attrs["rel"]))
				if attrs["href"] != "" && (contains(rel, "icon") || contains(rel, "apple-touch-icon")) {
					icons = append(icons, attrs["href"])
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	if t := strings.TrimSpace(strings.Join(title, "")); t != "" {
		m.Title = t
	} else if ogTitle != "" {
		m.Title = ogTitle
	} else {
		m.Title = siteName
	}
	m.Title = strings.Join(strings.Fields(m.Title), " ")
	m.Description = strings.Join(strings.Fields(m.Description), " ")
	for _, icon := range icons {
		u, err := base.Parse(icon)
		if err == nil && (u.Scheme == "https" || u.Scheme == "http") && strings.EqualFold(u.Host, base.Host) {
			m.IconURL = u.String()
			break
		}
	}
	if m.IconURL == "" {
		m.IconURL = base.ResolveReference(&url.URL{Path: "/favicon.ico"}).String()
	}
	return m, nil
}

func DownloadPNG(rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("icon returned %s", res.Status)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if len(b) < 24 || !bytes.Equal(b[:8], []byte("\x89PNG\r\n\x1a\n")) {
		return nil, fmt.Errorf("website icon is not a PNG")
	}
	config, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("invalid website PNG icon: %w", err)
	}
	if config.Width == 0 || config.Width != config.Height {
		return nil, fmt.Errorf("website icon must be square (got %dx%d)", config.Width, config.Height)
	}
	return b, nil
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
