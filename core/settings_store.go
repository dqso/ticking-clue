package core

import "encoding/json"

// storageKey names the persisted settings blob: a file name on desktop and a
// localStorage key in wasm (see the platform-specific store files).
const storageKey = "ticking-clue-settings"

// persistedSettings is the on-disk shape of the saved options. It is kept
// separate from Settings so the storage format stays explicit and versioned.
type persistedSettings struct {
	// Levels are the enabled CEFR levels, ordered A1..C2.
	Levels []bool `json:"levels"`
}

// loadSettings restores the saved options, falling back to the defaults when
// nothing is stored yet or the stored blob cannot be read. It never fails: bad
// or missing data just yields fresh default settings.
func loadSettings() *Settings {
	blob, ok := readSettingsBlob()
	if !ok || len(blob) == 0 {
		return newSettings()
	}
	var p persistedSettings
	if err := json.Unmarshal(blob, &p); err != nil {
		return newSettings()
	}
	s := newSettings()
	// Copy only as many flags as both sides have, so an older or newer file
	// (with a different level count) still loads without panicking.
	for i := 0; i < len(p.Levels) && i < len(s.Levels); i++ {
		s.Levels[i] = p.Levels[i]
	}
	return s
}

// save persists the current options. Errors are ignored on purpose: failing to
// save must never crash the game, exactly like openURL.
func (s *Settings) save() {
	p := persistedSettings{Levels: s.Levels[:]}
	blob, err := json.Marshal(p)
	if err != nil {
		return
	}
	writeSettingsBlob(blob)
}
