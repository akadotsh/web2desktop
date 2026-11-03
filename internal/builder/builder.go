package builder

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func Build(workdir, output, name, version string, verbose bool) ([]string, error) {
	if runtime.GOOS != "darwin" {
		return nil, errors.New("MyGo app packaging is currently supported on macOS only")
	}
	args := []string{"run", "github.com/egoist/mygo/cmd/mygo@v0.7.4", "build", "-platform", "darwin/" + runtime.GOARCH, "-o", "build"}
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
	buildDir := filepath.Join(workdir, "build", "darwin-"+runtime.GOARCH)
	entries, err := os.ReadDir(buildDir)
	if err != nil {
		return nil, fmt.Errorf("cannot find MyGo artifacts in %s: %w", buildDir, err)
	}
	var artifacts []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".app") || strings.HasSuffix(entry.Name(), ".dmg") {
			artifacts = append(artifacts, filepath.Join(buildDir, entry.Name()))
		}
	}
	if len(artifacts) < 2 {
		return nil, fmt.Errorf("expected .app and .dmg artifacts in %s", buildDir)
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return nil, err
	}
	for _, src := range artifacts {
		dst := filepath.Join(output, filepath.Base(src))
		if _, err := os.Lstat(dst); err == nil {
			return nil, fmt.Errorf("refusing to replace existing artifact %s", dst)
		}
	}
	var copied []string
	for _, src := range artifacts {
		dst := filepath.Join(output, filepath.Base(src))
		tmp := dst + ".web2desktop-tmp"
		if err := copyPath(src, tmp); err != nil {
			for _, p := range copied {
				_ = os.RemoveAll(p)
			}
			_ = os.RemoveAll(tmp)
			return nil, err
		}
		if err := os.Rename(tmp, dst); err != nil {
			_ = os.RemoveAll(tmp)
			for _, p := range copied {
				_ = os.RemoveAll(p)
			}
			return nil, err
		}
		copied = append(copied, dst)
	}
	return copied, nil
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
