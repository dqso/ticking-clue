package core

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// debugOverlay prints version and FPS/TPS info in the corner.
// It is plain text on purpose: an ebitenui UI here would be a second
// UI.Update() per tick, and ebitenui input state is a global singleton
// that allows only one.
type debugOverlay struct {
	enabled bool

	version string
}

func newDebugOverlay(enabled bool, version string) debugOverlay {
	return debugOverlay{
		enabled: enabled,
		version: version,
	}
}

func (d *debugOverlay) update() {
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		d.enabled = !d.enabled
	}
}

func (d *debugOverlay) draw(dst *ebiten.Image) {
	if !d.enabled {
		return
	}
	msg := fmt.Sprintf("Version: %s\nFPS: %.1f\nTPS: %.1f",
		d.version, ebiten.ActualFPS(), ebiten.ActualTPS())
	ebitenutil.DebugPrintAt(dst, msg, 8, 8)
}
