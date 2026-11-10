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

func Build(workdir, output string, platforms []Platform, verbose, overwrite bool) ([]string, error) {
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
	if err := os.MkdirAll(output, 0755); err != nil {
		return nil, err
	}
	buildDirs := make(map[string]string, len(platforms))
	for _, platform := range platforms {
		buildDir := filepath.Join(workdir, "build", platform.Directory())
		if _, err := os.ReadDir(buildDir); err != nil {
			return nil, fmt.Errorf("cannot find MyGo artifacts in %s: %w", buildDir, err)
		}
		buildDirs[platform.Directory()] = buildDir
		dst := filepath.Join(output, platform.Directory())
		if _, err := os.Lstat(dst); err == nil && !overwrite {
			return nil, fmt.Errorf("artifact already exists at %s; pass --overwrite to replace it", dst)
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	stageRoot, err := os.MkdirTemp(output, ".web2desktop-stage-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(stageRoot)
	var artifacts []string
	for _, platform := range platforms {
		buildDir := buildDirs[platform.Directory()]
		entries, err := os.ReadDir(buildDir)
		if err != nil {
			return nil, fmt.Errorf("cannot find MyGo artifacts in %s: %w", buildDir, err)
		}
		dst := filepath.Join(output, platform.Directory())
		staged := filepath.Join(stageRoot, platform.Directory())
		if err := copyPath(buildDir, staged); err != nil {
			return nil, err
		}
		for _, entry := range entries {
			artifacts = append(artifacts, filepath.Join(dst, entry.Name()))
		}
	}
	type backup struct{ dir, path string }
	backups := make(map[string]backup)
	var touched []string
	restore := func() error {
		var restoreErr error
		for i := len(touched) - 1; i >= 0; i-- {
			dst := touched[i]
			if err := os.RemoveAll(dst); err != nil && restoreErr == nil {
				restoreErr = err
			}
			if old, ok := backups[dst]; ok {
				if err := os.Rename(old.path, dst); err != nil && restoreErr == nil {
					restoreErr = fmt.Errorf("restore %s from %s: %w", dst, old.path, err)
				}
				if restoreErr == nil {
					_ = os.RemoveAll(old.dir)
				}
			}
		}
		return restoreErr
	}
	for _, platform := range platforms {
		dst := filepath.Join(output, platform.Directory())
		if _, err := os.Lstat(dst); err == nil {
			if !overwrite {
				cause := fmt.Errorf("artifact already exists at %s; pass --overwrite to replace it", dst)
				if restoreErr := restore(); restoreErr != nil {
					return nil, fmt.Errorf("%w (rollback failed: %v)", cause, restoreErr)
				}
				return nil, cause
			}
			backupDir, err := os.MkdirTemp(output, ".web2desktop-backup-")
			if err != nil {
				if restoreErr := restore(); restoreErr != nil {
					return nil, fmt.Errorf("%w (rollback failed: %v)", err, restoreErr)
				}
				return nil, err
			}
			old := backup{dir: backupDir, path: filepath.Join(backupDir, "previous")}
			if err := os.Rename(dst, old.path); err != nil {
				_ = os.RemoveAll(backupDir)
				if restoreErr := restore(); restoreErr != nil {
					return nil, fmt.Errorf("%w (rollback failed: %v)", err, restoreErr)
				}
				return nil, err
			}
			backups[dst] = old
			touched = append(touched, dst)
		} else if !errors.Is(err, os.ErrNotExist) {
			if restoreErr := restore(); restoreErr != nil {
				return nil, fmt.Errorf("%w (rollback failed: %v)", err, restoreErr)
			}
			return nil, err
		}
		staged := filepath.Join(stageRoot, platform.Directory())
		if err := os.Rename(staged, dst); err != nil {
			if restoreErr := restore(); restoreErr != nil {
				return nil, fmt.Errorf("install %s: %w (rollback failed: %v)", dst, err, restoreErr)
			}
			return nil, fmt.Errorf("install %s: %w", dst, err)
		}
		if len(touched) == 0 || touched[len(touched)-1] != dst {
			touched = append(touched, dst)
		}
	}
	for _, old := range backups {
		if err := os.RemoveAll(old.dir); err != nil {
			return artifacts, fmt.Errorf("artifacts installed, but could not remove previous artifact backup %s: %w", old.dir, err)
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
