//go:build !js

package core

import (
	"os"
	"path/filepath"
)

// settingsPath returns the file the settings are stored in, inside the user's
// OS config directory (e.g. ~/Library/Application Support/ticking-clue on
// macOS). ok is false when that directory cannot be resolved.
func settingsPath() (path string, ok bool) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(dir, "ticking-clue", storageKey+".json"), true
}

// readSettingsBlob loads the saved settings file. ok is false when there is no
// file yet or its path cannot be resolved.
func readSettingsBlob() (blob []byte, ok bool) {
	path, ok := settingsPath()
	if !ok {
		return nil, false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return b, true
}

// writeSettingsBlob saves the settings blob, creating the config directory if
// needed. Errors are ignored: a failed save must not crash the game.
func writeSettingsBlob(blob []byte) {
	path, ok := settingsPath()
	if !ok {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(path, blob, 0o644)
}
