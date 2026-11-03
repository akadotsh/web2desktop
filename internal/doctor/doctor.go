package doctor

import (
	"fmt"
	"os/exec"
	"runtime"
)

func Run() error {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("Go is not installed or is not on PATH")
	}
	fmt.Printf("Go executable: %s\n", goPath)
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("native packaging currently supports macOS only (detected %s/%s)", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "arm64" && runtime.GOARCH != "amd64" {
		return fmt.Errorf("unsupported macOS architecture %s", runtime.GOARCH)
	}
	for _, tool := range []string{"xcrun", "hdiutil"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("%s is required; install Xcode Command Line Tools", tool)
		}
		fmt.Printf("%s: available\n", tool)
	}
	cmd := exec.Command("go", "run", "github.com/egoist/mygo/cmd/mygo@v0.7.4", "--help")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("MyGo v0.7.4 is unavailable: %s", string(out))
	}
	fmt.Println("MyGo v0.7.4: available (Go can download the pinned toolchain and module)")
	return nil
}
