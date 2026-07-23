package core

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// minStartLinks is the minimum number of links the starting word
// must have, so the round always has enough directions to explore.
const minStartLinks = 5

// GameScene is the gameplay scene. For now it is only a placeholder
// background with an Esc handler.
type GameScene struct {
	// start is the word the round is built around.
	start *Node
}

func newGameScene(start *Node) *GameScene { return &GameScene{start: start} }

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
