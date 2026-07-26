package core

import (
	"log"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/input"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// minStartLinks is the minimum number of links the starting word must have,
// so the round always has enough directions to explore.
const minStartLinks = 5

// Zoom bounds and per-button-press step for the graph view.
const (
	zoomMin  = 0.4
	zoomMax  = 2.5
	zoomStep = 1.2
)

// GameScene is the gameplay scene: a mind map of the hidden word with a hint
// column and a countdown. All of it is custom drawing except the hint cells,
// which are the scene's only ebitenui widgets.
type GameScene struct {
	round *round
	ui    *ebitenui.UI

	// panX / panY scroll the sheet; the player drags the background to move.
	panX, panY float64
	// dragging tracks a background drag and its last cursor position.
	dragging     bool
	dragX, dragY int
	// dragCloud is the revealed node being dragged (-1 when none); grabDX/DY
	// keep the cursor offset from that cloud's center in screen space.
	dragCloud      int
	grabDX, grabDY float64
	// zoom scales the whole graph around its center; the HUD +/- buttons
	// change it within [zoomMin, zoomMax].
	zoom float64

	// fx holds the flying guesses and the miss list (state and logic live in
	// guess_fx.go).
	fx guessFX
	// hints holds the hint cells so used-up ones can be removed (see hint.go).
	hints hintUI
	// colorsNote is the white sticker dropped on the sheet when the colors hint
	// is bought (state and logic live in hint_colors_note.go).
	colorsNote colorsNote
}

func newGameScene(graph *Graph, start *Node) *GameScene {
	log.Printf("hidden word: %s", start.Word)
	s := &GameScene{round: newRound(graph, start), dragCloud: -1, zoom: 1}
	s.ui = &ebitenui.UI{Container: s.buildHintUI()}
	return s
}

// buildHintUI lays out the hint column (see hint.go) and the zoom controls.
func (s *GameScene) buildHintUI() *widget.Container {
	root := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	root.AddChild(s.hintColumn())

	// Zoom controls in the bottom-right corner.
	zoom := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(20)),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionEnd,
			VerticalPosition:   widget.AnchorLayoutPositionEnd,
		})),
	)
	zoom.AddChild(newZoomButton("-", func() { s.zoomBy(1 / zoomStep) }))
	zoom.AddChild(newZoomButton("+", func() { s.zoomBy(zoomStep) }))
	root.AddChild(zoom)
	return root
}

// newZoomButton builds a small square +/- button for the zoom controls.
func newZoomButton(label string, onClick func()) *widget.Button {
	return widget.NewButton(
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(48, 48)),
		widget.ButtonOpts.Image(newButtonImage()),
		widget.ButtonOpts.Text(label, facePtr(28), &widget.ButtonTextColor{Idle: uiTextColor}),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			onClick()
		}),
	)
}

// zoomBy multiplies the zoom by f, clamped to the allowed range.
func (s *GameScene) zoomBy(f float64) {
	s.zoom = min(zoomMax, max(zoomMin, s.zoom*f))
}

// repeatKey reports a key press once, then repeatedly while it stays held,
// so backspace deletes several characters when kept down.
func repeatKey(k ebiten.Key) bool {
	d := inpututil.KeyPressDuration(k)
	return d == 1 || (d >= 30 && (d-30)%3 == 0)
}

func (s *GameScene) Update(g *Game) error {
	s.ui.Update()
	// Drop hint cells whose hint is used up (one-time hints, or all links shown).
	s.refreshHints()

	if s.round.state != roundPlaying {
		// Round is over: Enter or Escape returns to the main menu.
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
			inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) ||
			inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.Pop()
		}
		return nil
	}

	// The miss list gets first pick of the pointer (it draws on top), then the
	// colors note, then the graph pans and drags clouds.
	s.handleListInput(g)
	s.updateColorsNote(g)
	s.handleMouse(g)
	// Scroll the miss list when the wheel turns over it, and let the clouds
	// push the collapsed sticker out of their way.
	l := computeLayout(g.screenWidth, g.screenHeight, s.panX, s.panY, s.zoom)
	s.updateListScroll(l)
	s.resolveSticker(l, computeNodePositions(s.round, l))
	// Advance the flying guesses.
	s.updateFlyers()

	// Typing builds the current guess.
	for _, ch := range ebiten.AppendInputChars(nil) {
		s.round.typeRune(ch)
	}
	if repeatKey(ebiten.KeyBackspace) {
		s.round.backspace()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		s.submitGuess()
	}
	// Esc opens a modal dialog asking to leave to the main menu.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.Push(newPauseScene(s, "Pause", "Exit to the main menu?", func(g *Game) error {
			g.Pop() // close the dialog
			g.Pop() // leave the game scene, back to the main menu
			return nil
		}))
		return nil
	}

	s.round.update()
	return nil
}

// handleMouse handles left-button dragging: grabbing a cloud moves that cloud,
// grabbing the empty background scrolls the sheet. A drag never starts over the
// hint cells, so clicking a cell is left to ebitenui.
func (s *GameScene) handleMouse(g *Game) {
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		s.dragging = false
		s.dragCloud = -1
		return
	}
	mx, my := ebiten.CursorPosition()
	l := computeLayout(g.screenWidth, g.screenHeight, s.panX, s.panY, s.zoom)

	// Continue an ongoing drag.
	if s.dragCloud >= 0 {
		// Store the cloud relative to the center in zoom-1 units, so it pans
		// and zooms together with the rest of the graph.
		s.round.pinCloud(s.dragCloud,
			(float64(mx)-s.grabDX-l.cx)/l.zoom,
			(float64(my)-s.grabDY-l.cy)/l.zoom)
		return
	}
	if s.dragging {
		s.panX += float64(mx - s.dragX)
		s.panY += float64(my - s.dragY)
		s.dragX, s.dragY = mx, my
		return
	}

	// A fresh press: decide what it grabs (never over the hint cells or when
	// the miss list already captured this press).
	if input.UIHovered || s.fx.captured || s.colorsNote.captured {
		return
	}
	pos := computeNodePositions(s.round, l)
	if idx := cloudAt(s.round, l, pos, float64(mx), float64(my)); idx >= 0 {
		s.dragCloud = idx
		s.grabDX = float64(mx) - pos[idx].x
		s.grabDY = float64(my) - pos[idx].y
		return
	}
	s.dragging = true
	s.dragX, s.dragY = mx, my
}

func (s *GameScene) Draw(screen *ebiten.Image) {
	l := computeLayout(screen.Bounds().Dx(), screen.Bounds().Dy(), s.panX, s.panY, s.zoom)
	drawPaper(screen, l)

	// Arrows first, then the word clouds, then the central cloud on top.
	pos := computeNodePositions(s.round, l)
	skip := s.animatingHits()
	drawGraphArrows(screen, s.round, l, pos, skip)
	drawGraphClouds(screen, s.round, l, pos, skip)
	drawCloud(screen, s.round, l)

	// Flying guesses on top of the graph, then the colors note (the crossed-
	// arrows sticker), then the miss list sticky note on top of it.
	s.drawFlyers(screen, l, pos)
	s.drawColorsNote(screen, l)
	s.drawList(screen, l)

	// The hint cells are the only ebitenui part of this scene.
	s.ui.Draw(screen)

	// HUD text on top of everything.
	drawTimer(screen, s.round, l)
	drawGuess(screen, s.round, l)
	if s.round.state != roundPlaying {
		drawEndOverlay(screen, s.round, l)
	}
}
