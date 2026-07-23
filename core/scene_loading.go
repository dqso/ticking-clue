package core

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type LoadingScene struct{}

func newLoadingScene() *LoadingScene { return &LoadingScene{} }

func (s *LoadingScene) Update(g *Game) error {
	g.Pop()
	g.Push(newMainMenuScene())
	return nil
}

func (s *LoadingScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
}
