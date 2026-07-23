package main

import (
	"log"

	"github.com/dqso/ticking-clue/core"
	"github.com/hajimehoshi/ebiten/v2"
)

// Set via -ldflags "-X main.debug=true -X main.version=...".
var (
	debug   = "false"
	version = "dev"
)

func main() {
	const (
		screenWidth  = 800
		screenHeight = 600
	)

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Ticking Clue")
	if err := ebiten.RunGame(core.NewGame(debug == "true", version, screenWidth, screenHeight)); err != nil {
		log.Fatal(err)
	}
}
