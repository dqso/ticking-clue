package core

import "github.com/hajimehoshi/ebiten/v2"

type Game struct {
	screenWidth  int
	screenHeight int
	scenes       SceneManager
	debug        debugOverlay
}

func NewGame(debug bool, version string, screenWidth, screenHeight int) *Game {
	g := &Game{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		scenes:       SceneManager{},
		debug:        newDebugOverlay(debug, version),
	}
	g.scenes.Push(newLoadingScene())
	return g
}

func (g *Game) Push(s Scene) {
	g.scenes.Push(s)
}

func (g *Game) Pop() {
	g.scenes.Pop()
}

func (g *Game) Update() error {
	g.debug.update()
	active := g.scenes.Active()
	return active.Update(g)
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.scenes.Active().Draw(screen)
	g.debug.draw(screen)
}

func (g *Game) Layout(outsideWidth int, outsideHeight int) (int, int) {
	g.screenWidth, g.screenHeight = outsideWidth, outsideHeight
	return outsideWidth, outsideHeight
}
