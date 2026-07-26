//go:build js

package core

import "syscall/js"

// localStorage returns the browser localStorage object, or a null js.Value when
// it is unavailable (e.g. disabled by the browser), so callers can no-op.
func localStorage() js.Value {
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() || ls.IsNull() {
		return js.Null()
	}
	return ls
}

// readSettingsBlob loads the saved settings from localStorage. ok is false when
// there is nothing stored or storage is unavailable.
func readSettingsBlob() (blob []byte, ok bool) {
	ls := localStorage()
	if ls.IsNull() {
		return nil, false
	}
	v := ls.Call("getItem", storageKey)
	if v.IsNull() || v.IsUndefined() {
		return nil, false
	}
	return []byte(v.String()), true
}

// writeSettingsBlob stores the settings blob in localStorage. It is a no-op when
// storage is unavailable.
func writeSettingsBlob(blob []byte) {
	ls := localStorage()
	if ls.IsNull() {
		return
	}
	ls.Call("setItem", storageKey, string(blob))
}
