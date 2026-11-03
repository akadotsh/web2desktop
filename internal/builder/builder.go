package builder

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Platform struct {
	OS, Arch string
}

func ParsePlatforms(raw string) ([]Platform, error) {
	var platforms []Platform
	seen := make(map[string]bool)
	for _, target := range strings.Split(raw, ",") {
		parts := strings.Split(strings.TrimSpace(target), "/")
		if len(parts) != 2 || !supportedPlatform(parts[0], parts[1]) {
			return nil, fmt.Errorf("unsupported platform %q", target)
		}
		platform := Platform{OS: parts[0], Arch: parts[1]}
		if !seen[platform.String()] {
			platforms = append(platforms, platform)
			seen[platform.String()] = true
		}
	}
	if len(platforms) == 0 {
		return nil, errors.New("at least one platform is required")
	}
	return platforms, nil
}

func (p Platform) String() string { return p.OS + "/" + p.Arch }

func (p Platform) Directory() string { return p.OS + "-" + p.Arch }

func supportedPlatform(osName, arch string) bool {
	switch osName {
	case "darwin":
		return arch == "amd64" || arch == "arm64" || arch == "universal"
	case "windows", "linux":
		return arch == "amd64" || arch == "arm64"
	default:
		return false
	}
}

func Build(workdir, output string, platforms []Platform, verbose bool) ([]string, error) {
	targets := make([]string, len(platforms))
	for i, platform := range platforms {
		targets[i] = platform.String()
	}
	args := []string{"run", "github.com/egoist/mygo/cmd/mygo@v0.7.4", "build", "-platform", strings.Join(targets, ","), "-o", "build"}
	cmd := exec.Command("go", args...)
	cmd.Dir = workdir
	var logs bytes.Buffer
	if verbose {
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	} else {
		cmd.Stdout, cmd.Stderr = &logs, &logs
	}
	if err := cmd.Run(); err != nil {
		if !verbose && logs.Len() > 0 {
			return nil, fmt.Errorf("MyGo build failed: %w\n%s", err, tail(logs.String(), 30))
		}
		return nil, fmt.Errorf("MyGo build failed: %w", err)
	}
	var artifacts []string
	if err := os.MkdirAll(output, 0755); err != nil {
		return nil, err
	}
	for _, platform := range platforms {
		dst := filepath.Join(output, platform.Directory())
		if _, err := os.Lstat(dst); err == nil {
			return nil, fmt.Errorf("refusing to replace existing artifact %s", dst)
		}
	}
	var copied []string
	for _, platform := range platforms {
		buildDir := filepath.Join(workdir, "build", platform.Directory())
		entries, err := os.ReadDir(buildDir)
		if err != nil {
			for _, path := range copied {
				_ = os.RemoveAll(path)
			}
			return nil, fmt.Errorf("cannot find MyGo artifacts in %s: %w", buildDir, err)
		}
		dst := filepath.Join(output, platform.Directory())
		tmp := dst + ".web2desktop-tmp"
		if err := copyPath(buildDir, tmp); err != nil {
			for _, path := range copied {
				_ = os.RemoveAll(path)
			}
			_ = os.RemoveAll(tmp)
			return nil, err
		}
		if err := os.Rename(tmp, dst); err != nil {
			for _, path := range copied {
				_ = os.RemoveAll(path)
			}
			_ = os.RemoveAll(tmp)
			return nil, err
		}
		copied = append(copied, dst)
		for _, entry := range entries {
			artifacts = append(artifacts, filepath.Join(dst, entry.Name()))
		}
	}
	return artifacts, nil
}

func tail(s string, maxLines int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n")
}

func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	}
	if info.IsDir() {
		if err := os.Mkdir(dst, info.Mode()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
