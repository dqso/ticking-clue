package core

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// loadResult is what the background loading goroutine returns.
type loadResult struct {
	graph *Graph
	err   error
}

type LoadingScene struct {
	result chan loadResult
}

func newLoadingScene() *LoadingScene { return &LoadingScene{} }

func (s *LoadingScene) Update(g *Game) error {
	if s.result == nil {
		// Parse the embedded graph in a goroutine to keep frames flowing.
		s.result = make(chan loadResult, 1)
		go func() {
			graph, err := LoadGraph()
			s.result <- loadResult{graph: graph, err: err}
		}()
		return nil
	}
	select {
	case res := <-s.result:
		if res.err != nil {
			return res.err
		}
		g.graph = res.graph
		g.Pop()
		g.Push(newMainMenuScene(g.graph))
	default:
		// Still loading.
	}
	return nil
}

func (s *LoadingScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
}
