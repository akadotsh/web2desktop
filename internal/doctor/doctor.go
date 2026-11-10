package doctor

import (
	"fmt"
	"os"
	"os/exec"
)

func Run() error {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("Go is not installed or is not on PATH")
	}
	fmt.Printf("Go executable: %s\n", goPath)
	if out, err := exec.Command("go", "version").CombinedOutput(); err != nil {
		return fmt.Errorf("cannot run Go (%s): %w", string(out), err)
	}
	cmd := exec.Command("go", "run", "github.com/egoist/mygo/cmd/mygo@v0.7.4", "doctor")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("MyGo environment checks failed: %w", err)
	}
	return nil
}
