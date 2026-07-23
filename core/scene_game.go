package core

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// GameScene is the gameplay scene. For now it is only a placeholder
// background with an Esc handler.
type GameScene struct{}

func newGameScene() *GameScene { return &GameScene{} }

func (s *GameScene) Update(g *Game) error {
	// Esc opens a modal dialog asking to leave to the main menu.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.Push(newPauseScene(s, "Pause", "Exit to the main menu?", func(g *Game) error {
			g.Pop() // close the dialog
			g.Pop() // leave the game scene, back to the main menu
			return nil
		}))
	}
	return nil
}

func (s *GameScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 0xe8, G: 0xb4, B: 0xc0, A: 0xff})
}
