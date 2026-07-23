package core

import (
	"image/color"
	"runtime"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// canExit reports whether quitting the game makes sense on this
// platform. In a browser the tab cannot be closed by the game, the
// canvas would only freeze, so the Exit action is hidden there.
func canExit() bool {
	return runtime.GOOS != "js"
}

type MainMenuScene struct {
	ui *ebitenui.UI
	// words are decorative flying words on the background.
	words *flyingWords
	// action is set by button handlers and executed on the next Update,
	// because handlers have no access to *Game.
	action func(g *Game) error
}

func newMainMenuScene(graph *Graph) *MainMenuScene {
	s := &MainMenuScene{words: newFlyingWords(graph, 1)}

	menu := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(12),
		)),
	)

	menu.AddChild(widget.NewText(
		widget.TextOpts.Text("Ticking Clue", facePtr(36), uiTextColor),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
	))

	menu.AddChild(newMenuButton("New game", func() {
		s.action = func(g *Game) error {
			g.Push(newGameScene())
			return nil
		}
	}))
	menu.AddChild(newMenuButton("Options", func() {
		s.action = func(g *Game) error {
			// The words keep flying on the options background.
			g.Push(newOptionsScene(s.words))
			return nil
		}
	}))
	if canExit() {
		menu.AddChild(newMenuButton("Exit", func() {
			s.action = s.exit
		}))
	}

	s.ui = &ebitenui.UI{Container: newCenteredRoot(menu)}
	return s
}

// exit opens a modal dialog asking to quit the game.
func (s *MainMenuScene) exit(g *Game) error {
	g.Push(newPauseScene(s, "Exit", "Do you really want to exit?", func(g *Game) error {
		return ebiten.Termination
	}))
	return nil
}

func (s *MainMenuScene) Update(g *Game) error {
	s.ui.Update()
	s.words.handleClick()
	s.words.update(float64(g.screenWidth), float64(g.screenHeight))
	// Esc acts like the Exit button.
	if canExit() && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.action = s.exit
	}
	if s.action != nil {
		action := s.action
		s.action = nil
		return action(g)
	}
	return nil
}

func (s *MainMenuScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 0x18, G: 0x18, B: 0x24, A: 0xff})
	s.words.draw(screen)
	s.ui.Draw(screen)
}
