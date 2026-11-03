package doctor

import (
	"fmt"
	"os/exec"
)

func Run() error {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("Go is not installed or is not on PATH")
	}
	fmt.Printf("Go executable: %s\n", goPath)
	cmd := exec.Command("go", "run", "github.com/egoist/mygo/cmd/mygo@v0.7.4", "--help")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("MyGo v0.7.4 is unavailable: %s", string(out))
	}
	fmt.Println("MyGo v0.7.4: available")
	return nil
}
