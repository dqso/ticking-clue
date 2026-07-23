package core

import (
	"image/color"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type OptionsScene struct {
	ui *ebitenui.UI
	// back is set by the Back button and handled on the next Update.
	back bool
}

func newOptionsScene() *OptionsScene {
	s := &OptionsScene{}

	content := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(12),
		)),
	)

	content.AddChild(widget.NewText(
		widget.TextOpts.Text("Options", facePtr(36), uiTextColor),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
	))

	content.AddChild(newMenuButton("Back", func() {
		s.back = true
	}))

	s.ui = &ebitenui.UI{Container: newCenteredRoot(content)}
	return s
}

func (s *OptionsScene) Update(g *Game) error {
	s.ui.Update()
	// Esc acts like the Back button.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.back = true
	}
	if s.back {
		s.back = false
		g.Pop()
	}
	return nil
}

func (s *OptionsScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 0x18, G: 0x18, B: 0x24, A: 0xff})
	s.ui.Draw(screen)
}
