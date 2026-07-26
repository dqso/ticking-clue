package core

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Win effect tuning: a firework "salute" launched from the bottom that bursts
// into colorful paint splats piling up over the whole graph.
const (
	// winLaunchTicks is how long new rockets keep launching after the win.
	winLaunchTicks = 200
	// winLaunchEvery is the tick period between rocket launches.
	winLaunchEvery = 5
	// winRocketSpeed is the base upward speed of a rising rocket (px per tick).
	winRocketSpeed = 13.0
)

// paintPalette are the bright paint colors a rocket can burst into.
var paintPalette = []color.NRGBA{
	{R: 0xff, G: 0x45, B: 0x45, A: 0xff}, // red
	{R: 0xff, G: 0x9f, B: 0x1c, A: 0xff}, // orange
	{R: 0xff, G: 0xe0, B: 0x66, A: 0xff}, // yellow
	{R: 0x2e, G: 0xc4, B: 0xb6, A: 0xff}, // teal
	{R: 0x3a, G: 0x86, B: 0xff, A: 0xff}, // blue
	{R: 0x83, G: 0x38, B: 0xec, A: 0xff}, // purple
	{R: 0xff, G: 0x5d, B: 0xa2, A: 0xff}, // pink
	{R: 0x38, G: 0xb0, B: 0x00, A: 0xff}, // green
}

// paintRocket is one rising firework: it climbs to targetY, then bursts into a
// paint splat of its color.
type paintRocket struct {
	x, y    float64
	vx, vy  float64
	targetY float64
	clr     color.NRGBA
}

// winFX is the victory effect: rockets rising from the bottom that burst into
// paint splats accumulated on a screen-sized layer drawn over the graph.
type winFX struct {
	active bool
	// paint is the accumulated splat layer, kept between frames so paint piles up.
	paint   *ebiten.Image
	pw, ph  int
	rockets []paintRocket
	ticks   int
	// endTick is the tick the last rocket burst; the paint is held, then faded
	// out a few seconds later. Zero means the salute is still going.
	endTick int
	// alpha is the paint layer opacity, dropped from 1 to 0 during the fade.
	alpha float64
	// done stays true once the paint has fully faded, so the salute never
	// restarts for the rest of the won round.
	done bool
}

// start begins the effect.
func (e *winFX) start() {
	e.active = true
	e.ticks = 0
	e.endTick = 0
	e.alpha = 1
	e.rockets = e.rockets[:0]
}

// ensureLayer (re)creates the paint layer when the screen size changes, so the
// splats always match the current resolution.
func (e *winFX) ensureLayer(w, h int) {
	if e.paint != nil && e.pw == w && e.ph == h {
		return
	}
	e.paint = ebiten.NewImage(w, h)
	e.pw, e.ph = w, h
}

// update launches rockets, moves them, and bursts each one into the paint layer
// when it reaches its target height.
func (e *winFX) update(l gameLayout) {
	if !e.active {
		return
	}
	e.ensureLayer(int(l.w), int(l.h))
	e.ticks++
	// Launch new rockets for the first stretch of the effect.
	if e.ticks <= winLaunchTicks && e.ticks%winLaunchEvery == 0 {
		e.launch(l)
	}
	// Move rockets; burst those that reached their target height.
	kept := e.rockets[:0]
	for _, r := range e.rockets {
		r.x += r.vx
		r.y += r.vy
		if r.y <= r.targetY {
			e.burst(r)
			continue
		}
		kept = append(kept, r)
	}
	e.rockets = kept
	// Once the salute is over, hold the paint, then fade it out; when it is fully
	// faded the salute stops for good and never pours again this round.
	if e.ticks > winLaunchTicks && len(e.rockets) == 0 {
		if e.endTick == 0 {
			e.endTick = e.ticks
		}
		fade := e.ticks - e.endTick - 180 // hold ~3s, then fade over ~2s
		switch {
		case fade < 0:
			e.alpha = 1
		case fade < 120:
			e.alpha = 1 - float64(fade)/120
		default:
			e.paint.Clear()
			e.active = false
			e.done = true
		}
	}
}

// launch adds one rocket rising from the bottom at a random x toward a random
// height in the upper part of the screen.
func (e *winFX) launch(l gameLayout) {
	e.rockets = append(e.rockets, paintRocket{
		x:       rand.Float64() * l.w,
		y:       l.h + 10,
		vx:      (rand.Float64() - 0.5) * 3,
		vy:      -winRocketSpeed * (0.8 + 0.4*rand.Float64()),
		targetY: l.h * (0.05 + 0.9*rand.Float64()),
		clr:     paintPalette[rand.IntN(len(paintPalette))],
	})
}

// burst stamps a paint splat of the rocket's color onto the layer: a big central
// blob, satellite blobs, and a few small droplets flung around it.
func (e *winFX) burst(r paintRocket) {
	base := 40.0 + rand.Float64()*40 // central blob radius
	vector.FillCircle(e.paint, fx(r.x), fx(r.y), fx(base), r.clr, true)
	// Satellite blobs pressed around the center.
	for i := 0; i < 6+rand.IntN(6); i++ {
		ang := rand.Float64() * 2 * math.Pi
		dist := base * (0.4 + rand.Float64())
		rad := base * (0.25 + 0.4*rand.Float64())
		vector.FillCircle(e.paint, fx(r.x+math.Cos(ang)*dist), fx(r.y+math.Sin(ang)*dist), fx(rad), r.clr, true)
	}
	// Tiny droplets flung farther out.
	for i := 0; i < 8+rand.IntN(8); i++ {
		ang := rand.Float64() * 2 * math.Pi
		dist := base * (1.2 + rand.Float64()*1.5)
		rad := base * (0.06 + 0.12*rand.Float64())
		vector.FillCircle(e.paint, fx(r.x+math.Cos(ang)*dist), fx(r.y+math.Sin(ang)*dist), fx(rad), r.clr, true)
	}
}

// draw renders the paint layer over the graph, then the rising rockets on top.
func (e *winFX) draw(screen *ebiten.Image, l gameLayout) {
	if !e.active {
		return
	}
	e.ensureLayer(int(l.w), int(l.h))
	// The paint layer fades out at the end via its alpha.
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(float32(e.alpha))
	screen.DrawImage(e.paint, op)
	// Rising rockets: a bright head with a short fading trail below it.
	for _, r := range e.rockets {
		for t := 0; t < 4; t++ {
			c := r.clr
			c.A = uint8(0xff - t*0x38)
			vector.FillCircle(screen,
				fx(r.x-r.vx*float64(t)*0.6), fx(r.y-r.vy*float64(t)*0.6),
				fx(4-float64(t)*0.7), c, true)
		}
	}
}
