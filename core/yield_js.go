//go:build js

package core

import "time"

// yieldToBrowser pauses the calling goroutine so the single wasm
// thread can return to the browser event loop and render a frame.
// Without it a busy goroutine starves rendering: wasm goroutines are
// cooperative and only switch on blocking operations.
func yieldToBrowser() {
	time.Sleep(time.Millisecond)
}
