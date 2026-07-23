//go:build windows

package core

import "os/exec"

// openURL opens the url in the default browser on Windows.
// rundll32 with FileProtocolHandler is the standard way to open a URL.
func openURL(url string) {
	// Errors are ignored: failing to open a link must not crash the game.
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}
