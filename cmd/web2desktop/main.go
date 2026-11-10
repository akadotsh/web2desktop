package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/akadotsh/web2desktop/internal/builder"
	"github.com/akadotsh/web2desktop/internal/config"
	"github.com/akadotsh/web2desktop/internal/doctor"
	"github.com/akadotsh/web2desktop/internal/generator"
)

var cliVersion = "0.1.0"

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int { return runWithMode(args, false) }

func runWithMode(args []string, creatingProject bool) int {
	if len(args) > 0 && args[0] == "init" {
		return runWithMode(args[1:], true)
	}
	if len(args) > 0 && args[0] == "build" {
		return runProject(args[1:])
	}
	if len(args) == 1 && args[0] == "doctor" {
		if err := doctor.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "doctor:", err)
			return 2
		}
		return 0
	}
	fs := flag.NewFlagSet("web2desktop", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	name := fs.String("name", "", "application display name")
	identifier := fs.String("identifier", "", "application bundle identifier")
	appVersion := fs.String("app-version", config.DefaultVersion, "generated application version")
	width := fs.Int("width", 1200, "initial window width")
	height := fs.Int("height", 800, "initial window height")
	icon := fs.String("icon", "", "square PNG application icon")
	output := fs.String("output", "./dist", "artifact output directory")
	platform := fs.String("platform", runtime.GOOS+"/"+runtime.GOARCH, "comma-separated build targets")
	projectPath := fs.String("config", "", "project configuration path (init only)")
	dev := fs.Bool("dev", false, "launch a development app with inspector")
	verbose := fs.Bool("verbose", false, "show build output")
	keep := fs.Bool("keep-workdir", false, "keep generated project")
	showVersion := fs.Bool("version", false, "print CLI version")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: web2desktop <url> [options]\n       web2desktop init <url> [options]\n       web2desktop build web2desktop.json\n       web2desktop doctor\n\nOptions:")
		fs.PrintDefaults()
	}
	if len(args) > 0 && args[0] != "--" && (len(args[0]) == 0 || args[0][0] != '-') {
		args = append(append([]string(nil), args[1:]...), args[0])
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if *showVersion {
		fmt.Println(cliVersion)
		return 0
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 1
	}
	c := config.Config{URL: fs.Arg(0), Name: *name, Identifier: *identifier, Version: *appVersion, Width: *width, Height: *height, Icon: *icon, Output: *output, Dev: *dev, Verbose: *verbose, KeepWorkdir: *keep}
	if err := c.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "invalid configuration:", err)
		return 1
	}
	platforms, err := builder.ParsePlatforms(*platform)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid platform:", err)
		return 1
	}
	if *dev && (len(platforms) != 1 || platforms[0].String() != runtime.GOOS+"/"+runtime.GOARCH) {
		fmt.Fprintln(os.Stderr, "development mode only supports the current platform")
		return 1
	}
	if !creatingProject && *projectPath != "" {
		fmt.Fprintln(os.Stderr, "--config is only valid with the init command")
		return 1
	}
	if creatingProject {
		path := *projectPath
		if path == "" {
			path = config.ProjectFileName
		}
		if err := config.WriteProject(path, config.NewProject(c, platformStrings(platforms))); err != nil {
			fmt.Fprintln(os.Stderr, "create project:", err)
			return 1
		}
		fmt.Printf("created %s\n", path)
		return 0
	}
	return build(c, platforms)
}

func runProject(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: web2desktop build web2desktop.json")
		return 1
	}
	p, err := config.LoadProject(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "read project:", err)
		return 1
	}
	c := p.BuildConfig()
	if err := c.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "invalid project:", err)
		return 1
	}
	platforms, err := builder.ParsePlatforms(strings.Join(p.Platforms, ","))
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid project platform:", err)
		return 1
	}
	return build(c, platforms)
}

func build(c config.Config, platforms []builder.Platform) int {
	workdir, err := os.MkdirTemp("", "web2desktop-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 4
	}
	failed := true
	defer func() {
		if !c.KeepWorkdir && !failed {
			_ = os.RemoveAll(workdir)
		} else if c.KeepWorkdir {
			fmt.Fprintln(os.Stderr, "generated project:", workdir)
		} else if failed {
			_ = os.RemoveAll(workdir)
		}
	}()
	err = generator.Generate(workdir, generator.Options{Name: c.Name, URL: c.URL, Identifier: c.Identifier, Version: c.Version, Width: c.Width, Height: c.Height, DevTools: c.Dev}, c.Icon)
	if err != nil {
		fmt.Fprintln(os.Stderr, "generate:", err)
		return 1
	}
	if err := execCommand(workdir, []string{"mod", "tidy"}, c.Verbose)(); err != nil {
		fmt.Fprintln(os.Stderr, "module preparation:", err)
		return 3
	}
	if c.Dev {
		cmdArgs := []string{"run", "github.com/egoist/mygo/cmd/mygo@v0.7.4", "dev"}
		cmd := execCommand(workdir, cmdArgs, c.Verbose)
		if err := cmd(); err != nil {
			fmt.Fprintln(os.Stderr, "MyGo dev:", err)
			return 3
		}
		failed = false
		return 0
	}
	started := time.Now()
	artifacts, err := builder.Build(workdir, c.Output, platforms, c.Verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, "build:", err)
		return 3
	}
	for _, path := range artifacts {
		size := pathSize(path)
		fmt.Printf("created %s (%s)\n", path, humanSize(size))
	}
	fmt.Printf("platforms: %s; elapsed: %s\n", strings.Join(platformNames(platforms), ", "), time.Since(started).Round(time.Second))
	failed = false
	return 0
}

func platformNames(platforms []builder.Platform) []string {
	names := make([]string, len(platforms))
	for i, platform := range platforms {
		names[i] = platform.Directory()
	}
	return names
}

func platformStrings(platforms []builder.Platform) []string {
	names := make([]string, len(platforms))
	for i, platform := range platforms {
		names[i] = platform.String()
	}
	return names
}

func pathSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && info.Mode().IsRegular() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func humanSize(n int64) string {
	if n < 1024*1024 {
		return fmt.Sprintf("%d KB", n/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
}
func execCommand(dir string, args []string, _ bool) func() error {
	return func() error {
		cmd := exec.Command("go", args...)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
}
