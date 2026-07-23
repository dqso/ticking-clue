//go:build js

package core

import "syscall/js"

// openURL opens the url in a new browser tab. In wasm this is the
// JS window.open call.
func openURL(url string) {
	js.Global().Call("open", url, "_blank")
}
