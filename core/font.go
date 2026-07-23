package core

import (
	"bytes"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/dqso/ticking-clue/assets"
)

var faceSource *text.GoTextFaceSource

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(assets.FiraSansTTF))
	if err != nil {
		panic(err)
	}
	faceSource = s
}

func newFace(size float64) *text.GoTextFace {
	return &text.GoTextFace{Source: faceSource, Size: size}
}
