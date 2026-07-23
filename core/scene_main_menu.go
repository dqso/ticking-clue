package core

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type MainMenuScene struct{}

func newMainMenuScene() *MainMenuScene { return &MainMenuScene{} }

func (s *MainMenuScene) Update(g *Game) error {
	return nil
}

func (s *MainMenuScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.White)
}
