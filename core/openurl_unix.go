//go:build !js && !windows

package core

import (
	"os/exec"
	"runtime"
)

// openURL opens the url in the default browser on macOS and Linux.
func openURL(url string) {
	cmd := "xdg-open"
	if runtime.GOOS == "darwin" {
		cmd = "open"
	}
	// Errors are ignored: failing to open a link must not crash the game.
	_ = exec.Command(cmd, url).Start()
}
