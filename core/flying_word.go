package core

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/ebitenui/ebitenui/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	flyingWordFontSize = 28
	flyingWordSpeedX   = 2.0
	flyingWordSpeedY   = 1.5
	// flyingWordMaxOmega limits the spin (radians per tick) so the word
	// stays readable after energetic corner hits.
	flyingWordMaxOmega = 0.05
	// flyingWordRestitution is 1: bounces keep kinetic energy, the word
	// never slows down.
	flyingWordRestitution = 1.0
	// flyingLetterFade is the alpha lost per tick by burst letters
	// (full fade in 1.5 seconds at 60 TPS).
	flyingLetterFade = 1.0 / 90
	// flyingLetterKick is the extra radial speed letters get in a burst.
	flyingLetterKick = 1.5
	// Chain length limits: every word gets its own random maximum of
	// words to show before it bursts anyway.
	flyingWordMinChain = 15
	flyingWordMaxChain = 100
)

// flyingWords is a group of words flying on menu backgrounds.
// The group is shared between scenes so words keep flying on transitions.
type flyingWords struct {
	graph *Graph
	words []*flyingWord
}

func newFlyingWords(graph *Graph, count int) *flyingWords {
	ws := &flyingWords{graph: graph}
	for range count {
		ws.createWord()
	}
	return ws
}

// createWord adds one more word with a random start to the group.
func (ws *flyingWords) createWord() {
	if f := newFlyingWord(ws.graph); f != nil {
		ws.words = append(ws.words, f)
	}
}

// createWordAt adds a word that starts flying from the given point.
func (ws *flyingWords) createWordAt(x, y float64) {
	if f := newFlyingWord(ws.graph); f != nil {
		f.cx, f.cy = x, y
		f.placed = true
		ws.words = append(ws.words, f)
	}
}

// handleClick adds a word at the cursor when the player clicks the free
// background, not a UI widget (buttons set input.UIHovered).
func (ws *flyingWords) handleClick() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !input.UIHovered {
		x, y := ebiten.CursorPosition()
		ws.createWordAt(float64(x), float64(y))
	}
}

func (ws *flyingWords) update(screenW, screenH float64) {
	// Words finished bursting are removed from the group.
	kept := ws.words[:0]
	for _, f := range ws.words {
		f.update(screenW, screenH)
		if !f.done {
			kept = append(kept, f)
		}
	}
	ws.words = kept
	// The screen never stays empty: the last word is replaced.
	if len(ws.words) == 0 {
		ws.createWord()
	}
}

func (ws *flyingWords) draw(screen *ebiten.Image) {
	for _, f := range ws.words {
		f.draw(screen)
	}
}

// flyingLetter is one letter of an exploded word. It ignores walls and
// slowly fades out.
type flyingLetter struct {
	ch     string
	x, y   float64
	vx, vy float64
	angle  float64
	omega  float64
	alpha  float64
}

// flyingWord is a decorative word bouncing on the menu background.
// It is a rotating rigid rectangle: wall hits apply an impulse at the
// touching corner, so energy moves between motion and spin like in real
// physics. On every bounce the word changes to an unused word from the
// node's links and gets a new color. When no unused link is left or the
// chain grew too long, the word bursts into letters and disappears.
type flyingWord struct {
	graph *Graph
	node  *Node
	face  text.Face
	// used holds words already shown, they are never picked again.
	used map[string]struct{}
	// maxChain is this word's own limit of shown words before a burst.
	maxChain int
	clr      color.NRGBA

	// Center of mass position and linear velocity (pixels per tick).
	cx, cy float64
	vx, vy float64
	// Rotation angle and angular velocity (radians per tick).
	angle, omega float64
	// Text rectangle size of the current word.
	w, h float64

	// letters is non-nil while the word is bursting.
	letters []*flyingLetter
	// done reports that the burst has finished and the word
	// should be removed from its group.
	done bool

	placed bool // position was initialized inside the screen
}

// newFlyingWord returns nil when there is nothing to show.
func newFlyingWord(graph *Graph) *flyingWord {
	if graph == nil {
		return nil
	}
	node := graph.Random()
	if node == nil {
		return nil
	}
	f := &flyingWord{
		graph:    graph,
		face:     newFace(flyingWordFontSize),
		used:     make(map[string]struct{}),
		maxChain: flyingWordMinChain + rand.IntN(flyingWordMaxChain-flyingWordMinChain+1),
		clr:      randomTextColor(),
		vx:       flyingWordSpeedX,
		vy:       flyingWordSpeedY,
		omega:    (rand.Float64() - 0.5) * 0.04,
	}
	if rand.IntN(2) == 0 {
		f.vx = -f.vx
	}
	if rand.IntN(2) == 0 {
		f.vy = -f.vy
	}
	f.setNode(node)
	return f
}

// randomTextColor returns a random color bright enough for the dark
// menu background.
func randomTextColor() color.NRGBA {
	return color.NRGBA{
		R: uint8(100 + rand.IntN(156)),
		G: uint8(100 + rand.IntN(156)),
		B: uint8(100 + rand.IntN(156)),
		A: 0xff,
	}
}

func (f *flyingWord) setNode(n *Node) {
	f.node = n
	f.used[n.Word] = struct{}{}
	f.w, f.h = text.Measure(n.Word, f.face, 0)
}

// next switches to a random unused linked word. The word bursts into
// letters when every link is already used or the chain grew too long.
func (f *flyingWord) next() {
	if len(f.used) > f.maxChain {
		f.startBurst()
		return
	}
	candidates := make([]*Node, 0, len(f.node.Links))
	for _, l := range f.node.Links {
		if l.To.Word == "" {
			continue
		}
		if _, ok := f.used[l.To.Word]; ok {
			continue
		}
		candidates = append(candidates, l.To)
	}
	if len(candidates) == 0 {
		f.startBurst()
		return
	}
	f.setNode(candidates[rand.IntN(len(candidates))])
}

// startBurst replaces the word with letters flying away from the word
// center: each letter keeps the rigid body point velocity at its spot
// and gets an extra radial kick.
func (f *flyingWord) startBurst() {
	sin, cos := math.Sincos(f.angle)
	f.letters = f.letters[:0]
	offset := -f.w / 2
	for _, r := range f.node.Word {
		ch := string(r)
		adv := text.Advance(ch, f.face)
		// Letter center in the local text frame, on the center line.
		lx := offset + adv/2
		offset += adv
		if ch == " " {
			continue
		}
		// Offset from the center of mass in world space.
		rx, ry := lx*cos, lx*sin
		// Radial direction away from the center (random for the middle).
		dir := math.Atan2(ry, rx)
		if lx == 0 {
			dir = rand.Float64() * 2 * math.Pi
		}
		kick := flyingLetterKick * (0.7 + 0.6*rand.Float64())
		f.letters = append(f.letters, &flyingLetter{
			ch: ch,
			x:  f.cx + rx,
			y:  f.cy + ry,
			// Point velocity of the body (v + omega x r) plus the kick.
			vx:    f.vx - f.omega*ry + kick*math.Cos(dir),
			vy:    f.vy + f.omega*rx + kick*math.Sin(dir),
			angle: f.angle,
			omega: (rand.Float64() - 0.5) * 0.2,
			alpha: 1,
		})
	}
}

// corners returns the four rectangle corners in screen space.
func (f *flyingWord) corners() [4][2]float64 {
	sin, cos := math.Sincos(f.angle)
	hw, hh := f.w/2, f.h/2
	local := [4][2]float64{{-hw, -hh}, {hw, -hh}, {hw, hh}, {-hw, hh}}
	var res [4][2]float64
	for i, p := range local {
		res[i][0] = f.cx + p[0]*cos - p[1]*sin
		res[i][1] = f.cy + p[0]*sin + p[1]*cos
	}
	return res
}

func (f *flyingWord) update(screenW, screenH float64) {
	if !f.placed {
		// First update: drop the word at a random spot inside the screen,
		// keeping a margin for the rotating rectangle.
		margin := math.Hypot(f.w, f.h) / 2
		f.cx = margin + rand.Float64()*max(screenW-2*margin, 1)
		f.cy = margin + rand.Float64()*max(screenH-2*margin, 1)
		f.placed = true
	}

	if f.letters != nil {
		f.updateBurst()
		return
	}

	f.cx += f.vx
	f.cy += f.vy
	f.angle += f.omega

	if f.collideWalls(screenW, screenH) {
		f.clr = randomTextColor()
		f.next()
	}
}

// updateBurst moves and fades the letters; when all of them are gone
// the word is marked as done and gets removed from its group.
func (f *flyingWord) updateBurst() {
	alive := false
	for _, l := range f.letters {
		l.x += l.vx
		l.y += l.vy
		l.angle += l.omega
		l.alpha -= flyingLetterFade
		if l.alpha > 0 {
			alive = true
		}
	}
	if alive {
		return
	}
	f.letters = nil
	f.done = true
}

// collideWalls checks every wall against the rectangle corners and applies
// a rigid body impulse at the deepest touching corner. Reports whether any
// bounce happened.
func (f *flyingWord) collideWalls(screenW, screenH float64) bool {
	// Walls as inward normals with penetration of a point.
	walls := [4]struct {
		nx, ny float64
		pen    func(x, y float64) float64
	}{
		{1, 0, func(x, _ float64) float64 { return -x }},           // left
		{-1, 0, func(x, _ float64) float64 { return x - screenW }}, // right
		{0, 1, func(_, y float64) float64 { return -y }},           // top
		{0, -1, func(_, y float64) float64 { return y - screenH }}, // bottom
	}

	corners := f.corners()
	bounced := false
	for _, wall := range walls {
		// Find the deepest corner behind this wall.
		deepest, maxPen := -1, 0.0
		for i, c := range corners {
			if p := wall.pen(c[0], c[1]); p > maxPen {
				deepest, maxPen = i, p
			}
		}
		if deepest < 0 {
			continue
		}
		// Push the body back inside the screen.
		f.cx += wall.nx * maxPen
		f.cy += wall.ny * maxPen
		if f.applyImpulse(corners[deepest], wall.nx, wall.ny) {
			bounced = true
		}
	}
	return bounced
}

// applyImpulse performs a frictionless rigid body collision response at
// contact point p with wall normal (nx, ny). Reports whether an impulse
// was applied (the corner was moving into the wall).
func (f *flyingWord) applyImpulse(p [2]float64, nx, ny float64) bool {
	// Contact point offset from the center of mass.
	rx, ry := p[0]-f.cx, p[1]-f.cy
	// Velocity of the contact point: v + omega x r.
	pvx := f.vx - f.omega*ry
	pvy := f.vy + f.omega*rx
	vn := pvx*nx + pvy*ny
	if vn >= 0 {
		// The corner already moves away from the wall.
		return false
	}
	// Moment of inertia of a rectangle with unit mass.
	inertia := (f.w*f.w + f.h*f.h) / 12
	rCrossN := rx*ny - ry*nx
	j := -(1 + flyingWordRestitution) * vn / (1 + rCrossN*rCrossN/inertia)
	f.vx += j * nx
	f.vy += j * ny
	f.omega += rCrossN * j / inertia
	f.omega = min(max(f.omega, -flyingWordMaxOmega), flyingWordMaxOmega)
	return true
}

func (f *flyingWord) draw(screen *ebiten.Image) {
	if f.letters != nil {
		f.drawBurst(screen)
		return
	}
	op := &text.DrawOptions{}
	// Rotate around the text center, then move to the body position.
	op.GeoM.Translate(-f.w/2, -f.h/2)
	op.GeoM.Rotate(f.angle)
	op.GeoM.Translate(f.cx, f.cy)
	op.ColorScale.ScaleWithColor(f.clr)
	// Linear filter smooths the rotated glyph edges.
	op.Filter = ebiten.FilterLinear
	text.Draw(screen, f.node.Word, f.face, op)
}

func (f *flyingWord) drawBurst(screen *ebiten.Image) {
	for _, l := range f.letters {
		if l.alpha <= 0 {
			continue
		}
		lw, lh := text.Measure(l.ch, f.face, 0)
		op := &text.DrawOptions{}
		op.GeoM.Translate(-lw/2, -lh/2)
		op.GeoM.Rotate(l.angle)
		op.GeoM.Translate(l.x, l.y)
		op.ColorScale.ScaleWithColor(f.clr)
		op.ColorScale.ScaleAlpha(float32(l.alpha))
		op.Filter = ebiten.FilterLinear
		text.Draw(screen, l.ch, f.face, op)
	}
}
