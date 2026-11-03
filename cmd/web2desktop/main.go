package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/web2desktop/web2desktop/internal/builder"
	"github.com/web2desktop/web2desktop/internal/config"
	"github.com/web2desktop/web2desktop/internal/doctor"
	"github.com/web2desktop/web2desktop/internal/generator"
)

const cliVersion = "0.1.0"

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
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
	identifier := fs.String("identifier", "", "macOS bundle identifier")
	appVersion := fs.String("app-version", config.DefaultVersion, "generated application version")
	width := fs.Int("width", 1200, "initial window width")
	height := fs.Int("height", 800, "initial window height")
	icon := fs.String("icon", "", "square PNG application icon")
	output := fs.String("output", "./dist", "artifact output directory")
	dev := fs.Bool("dev", false, "launch a development app with inspector")
	verbose := fs.Bool("verbose", false, "show build output")
	keep := fs.Bool("keep-workdir", false, "keep generated project")
	showVersion := fs.Bool("version", false, "print CLI version")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: web2desktop <url> [options]\n       web2desktop doctor\n\nOptions:")
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
	if *dev && runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "development mode currently supports macOS only")
		return 2
	}
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
	platformOutput := filepath.Join(c.Output, "darwin-"+runtime.GOARCH)
	artifacts, err := builder.Build(workdir, platformOutput, c.Name, c.Version, c.Verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, "build:", err)
		return 3
	}
	for _, path := range artifacts {
		size := pathSize(path)
		fmt.Printf("created %s (%s)\n", path, humanSize(size))
	}
	fmt.Printf("platform: darwin-%s; elapsed: %s\n", runtime.GOARCH, time.Since(started).Round(time.Second))
	failed = false
	return 0
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
