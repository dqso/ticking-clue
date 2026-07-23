//go:build !js

package core

// yieldToBrowser is a no-op outside wasm: real threads run the
// loading goroutine in parallel with the render loop.
func yieldToBrowser() {}
