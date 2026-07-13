package open 

import (
	"os/exec"
	"fmt"
	"runtime"
)


func Open(filePath string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Windows requires 'cmd /c start' to trigger the default app association
		cmd = exec.Command("cmd", "/c", "start", filePath)
	case "darwin":
		// macOS uses the 'open' command
		cmd = exec.Command("open", filePath)
	case "linux":
		// Linux uses 'xdg-open' for desktop environments
		cmd = exec.Command("xdg-open", filePath)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Run starts the command and waits for it to complete
	return cmd.Run()
}