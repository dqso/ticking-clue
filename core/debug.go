package core

import (
	"fmt"
	"image/color"

	"github.com/ebitenui/ebitenui"
	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const debugFontSize = 12

var debugFaceCache *text.GoTextFace

func debugFace() *text.GoTextFace {
	if debugFaceCache == nil {
		debugFaceCache = newFace(debugFontSize)
	}
	return debugFaceCache
}

type debugOverlay struct {
	enabled bool

	version string

	ui   *ebitenui.UI
	text *widget.Text
}

func newDebugOverlay(enabled bool, version string) debugOverlay {
	face := text.Face(debugFace())

	label := widget.NewText(
		widget.TextOpts.Text("", &face, color.RGBA{R: 0x3d, G: 0xff, B: 0x88, A: 0xff}),
	)

	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(eimage.NewNineSliceColor(color.RGBA{A: 0xaa})),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(8)),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionStart,
		})),
	)
	panel.AddChild(label)

	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(4)),
		)),
	)
	root.AddChild(panel)

	return debugOverlay{
		enabled: enabled,
		version: version,
		ui:      &ebitenui.UI{Container: root},
		text:    label,
	}
}

func (d *debugOverlay) update() {
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		d.enabled = !d.enabled
	}
	if !d.enabled {
		return
	}

	d.text.Label = fmt.Sprintf("Version: %s\nFPS: %.1f\nTPS: %.1f",
		d.version, ebiten.ActualFPS(), ebiten.ActualTPS())
	d.ui.Update()
}

func (d *debugOverlay) draw(dst *ebiten.Image) {
	if !d.enabled {
		return
	}
	d.ui.Draw(dst)
}
