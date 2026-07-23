package core

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png" // register the PNG decoder for the logos
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/dqso/ticking-clue/assets"
)

const (
	// loadingMinTime is how long the loading screen with the logos
	// stays visible even when the graph loads faster.
	loadingMinTime = time.Second
	// loadingFadeTime is how long the logos fade out before the menu.
	loadingFadeTime = 500 * time.Millisecond
)

// loadResult is what the background loading goroutine returns.
type loadResult struct {
	graph *Graph
	err   error
}

type LoadingScene struct {
	result chan loadResult
	// progress receives the loading progress (0..1) from the goroutine.
	progress chan float64
	// percent is the last consumed progress value, shown by the bar.
	percent float64
	// started is when the scene became visible.
	started time.Time
	// loaded keeps the result until the show time and the fade pass.
	loaded *loadResult
	// fadeStart is when the logos started to fade out.
	fadeStart time.Time
	// gmtk and ebitengine are the jam and engine logos.
	gmtk       *ebiten.Image
	ebitengine *ebiten.Image
}

func newLoadingScene() *LoadingScene {
	return &LoadingScene{
		gmtk:       decodeLogo(assets.GMTK26LogoPNG),
		ebitengine: decodeLogo(assets.EbitengineLogoPNG),
	}
}

// decodeLogo decodes an embedded PNG. Logos are build-time assets,
// so a broken one is a programmer error and panics.
func decodeLogo(data []byte) *ebiten.Image {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	return ebiten.NewImageFromImage(img)
}

func (s *LoadingScene) Update(g *Game) error {
	if s.result == nil {
		// First update: remember the start time and parse the embedded
		// graph in a goroutine to keep frames flowing.
		s.started = time.Now()
		s.result = make(chan loadResult, 1)
		s.progress = make(chan float64, 1)
		go func() {
			graph, err := LoadGraph(s.progress)
			s.result <- loadResult{graph: graph, err: err}
		}()
		return nil
	}
	// Consume one progress value per tick, freeing the buffer
	// for the next report from the goroutine.
	select {
	case p := <-s.progress:
		s.percent = p
	default:
	}
	if s.loaded == nil {
		select {
		case res := <-s.result:
			s.loaded = &res
		default:
			// Still loading.
			return nil
		}
	}
	if s.loaded.err != nil {
		return s.loaded.err
	}
	// Keep the logos fully visible for at least loadingMinTime.
	if time.Since(s.started) < loadingMinTime {
		return nil
	}
	if s.fadeStart.IsZero() {
		s.fadeStart = time.Now()
	}
	if time.Since(s.fadeStart) < loadingFadeTime {
		return nil
	}
	g.graph = s.loaded.graph
	g.Pop()
	g.Push(newMainMenuScene(g.graph))
	return nil
}

// alpha is the current opacity of the loading screen: 1 before the
// fade starts, then it goes down to 0 during loadingFadeTime.
func (s *LoadingScene) alpha() float32 {
	if s.fadeStart.IsZero() {
		return 1
	}
	f := float32(time.Since(s.fadeStart).Seconds() / loadingFadeTime.Seconds())
	return max(0, 1-f)
}

func (s *LoadingScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
	alpha := s.alpha()
	b := screen.Bounds()
	w, h := float64(b.Dx()), float64(b.Dy())
	// The jam logo takes the upper part, the engine logo the lower one.
	drawFitted(screen, s.gmtk, w/2, h*0.4, w*0.55, h*0.5, alpha)
	drawFitted(screen, s.ebitengine, w/2, h*0.8, w*0.25, h*0.22, alpha)
	// The bar is shown only while the graph is still loading.
	if s.loaded == nil {
		s.drawProgressBar(screen, w, h, alpha)
	}
}

// drawProgressBar draws a thin loading bar near the bottom: a dark
// track and a light fill sized by the current progress.
func (s *LoadingScene) drawProgressBar(screen *ebiten.Image, w, h float64, alpha float32) {
	const barH = 6
	barW := w * 0.3
	x, y := float32((w-barW)/2), float32(h*0.92)
	track := fadedColor(color.NRGBA{R: 0x30, G: 0x30, B: 0x40, A: 0xff}, alpha)
	fill := fadedColor(uiTextColor, alpha)
	vector.FillRect(screen, x, y, float32(barW), barH, track, false)
	vector.FillRect(screen, x, y, float32(barW*s.percent), barH, fill, false)
}

// fadedColor multiplies the color's alpha channel by a.
func fadedColor(c color.NRGBA, a float32) color.NRGBA {
	c.A = uint8(float32(c.A) * a)
	return c
}

// drawFitted draws img centered at (cx, cy), scaled to fit the
// maxW x maxH box keeping the aspect ratio.
func drawFitted(screen, img *ebiten.Image, cx, cy, maxW, maxH float64, alpha float32) {
	ib := img.Bounds()
	iw, ih := float64(ib.Dx()), float64(ib.Dy())
	scale := min(maxW/iw, maxH/ih)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(cx-iw*scale/2, cy-ih*scale/2)
	op.ColorScale.ScaleAlpha(alpha)
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(img, op)
}
