package core

import (
	"image"

	"github.com/ebitenui/ebitenui"
	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// PauseScene is a modal confirmation dialog drawn over the previous scene.
// It does not replace the scene below it: one frame of that scene is
// captured, blurred and used as the dialog background.
type PauseScene struct {
	ui *ebitenui.UI
	// under is the scene covered by this dialog, used to capture its frame.
	under Scene
	// frame is the blurred snapshot of the scene below, kept downscaled
	// by blurScale and stretched back on draw.
	frame *ebiten.Image
	// frameFor is the screen size the snapshot was captured for.
	frameFor image.Point
	// action is set by button handlers and executed on the next Update.
	action func(g *Game) error
	// onYes is called when the user confirms the dialog.
	onYes func(g *Game) error
}

func newPauseScene(under Scene, title, question string, onYes func(g *Game) error) *PauseScene {
	s := &PauseScene{under: under, onYes: onYes}

	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(eimage.NewNineSliceColor(uiPanelColor)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(16),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(24)),
		)),
	)

	panel.AddChild(widget.NewText(
		widget.TextOpts.Text(title, facePtr(28), uiTextColor),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
	))
	panel.AddChild(widget.NewText(
		widget.TextOpts.Text(question, facePtr(20), uiTextColor),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
	))

	buttons := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(16),
		)),
	)
	buttons.AddChild(newDialogButton("Yes", func() {
		s.action = s.onYes
	}))
	buttons.AddChild(newDialogButton("No", func() {
		s.action = func(g *Game) error {
			g.Pop()
			return nil
		}
	}))
	panel.AddChild(buttons)

	s.ui = &ebitenui.UI{Container: newCenteredRoot(panel)}
	return s
}

// newDialogButton creates a small button used in dialogs (Yes/No).
func newDialogButton(label string, onClick func()) *widget.Button {
	return widget.NewButton(
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(100, 40)),
		widget.ButtonOpts.Image(newButtonImage()),
		widget.ButtonOpts.Text(label, facePtr(20), &widget.ButtonTextColor{
			Idle: uiTextColor,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(8)),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			onClick()
		}),
	)
}

func (s *PauseScene) Update(g *Game) error {
	s.ui.Update()
	// Esc acts like the No button: close the dialog.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.Pop()
		return nil
	}
	if s.action != nil {
		action := s.action
		s.action = nil
		return action(g)
	}
	return nil
}

// blurScale shrinks the captured frame before the CPU blur. Reading
// pixels back from the GPU and blurring them at full size freezes a
// frame (especially in wasm); a small frame costs blurScale^2 times
// less, and stretching it back adds extra smoothing for free.
const blurScale = 4

func (s *PauseScene) Draw(screen *ebiten.Image) {
	s.ensureFrame(screen)
	b, fb := screen.Bounds(), s.frame.Bounds()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(
		float64(b.Dx())/float64(fb.Dx()),
		float64(b.Dy())/float64(fb.Dy()),
	)
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(s.frame, op)
	s.ui.Draw(screen)
}

// ensureFrame captures one frame of the scene below, blurs and dims it.
// The capture is redone when the screen size changes (e.g. window resize).
func (s *PauseScene) ensureFrame(screen *ebiten.Image) {
	b := screen.Bounds()
	if s.frame != nil && s.frameFor == b.Size() {
		return
	}
	if s.frame != nil {
		s.frame.Deallocate()
	}
	s.frameFor = b.Size()

	// Render the scene below at full size, then let the GPU shrink it.
	raw := ebiten.NewImage(b.Dx(), b.Dy())
	s.under.Draw(raw)
	sw, sh := max(b.Dx()/blurScale, 1), max(b.Dy()/blurScale, 1)
	small := ebiten.NewImage(sw, sh)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(sw)/float64(b.Dx()), float64(sh)/float64(b.Dy()))
	op.Filter = ebiten.FilterLinear
	small.DrawImage(raw, op)
	raw.Deallocate()

	s.frame = blurred(small)
	small.Deallocate()
	// Dim the blurred frame once so the dialog stands out.
	vector.FillRect(s.frame, 0, 0, float32(sw), float32(sh), uiOverlayColor, false)
}
