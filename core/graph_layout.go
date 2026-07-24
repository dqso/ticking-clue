package core

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	// hintColumnWidth is the space reserved on the right for the hint cells.
	hintColumnWidth = 150.0
	// ringStep is the extra radius (in base-radius units) added per depth
	// level, so deeper guessed words sit on wider rings.
	ringStep = 0.75
)

// gameLayout holds the derived geometry for one frame.
type gameLayout struct {
	w, h   float64
	cx, cy float64 // center of the hidden cloud, shifted by the pan
	radius float64 // radius of the first ring of words (already zoomed)
	panX   float64 // scrolling offset of the sheet
	panY   float64
	zoom   float64 // graph zoom factor around the center
}

func computeLayout(screenW, screenH int, panX, panY, zoom float64) gameLayout {
	w, h := float64(screenW), float64(screenH)
	playW := w - hintColumnWidth
	return gameLayout{
		w:      w,
		h:      h,
		cx:     playW/2 + panX,
		cy:     h/2 + panY,
		radius: math.Min(playW, h) * 0.28 * zoom,
		panX:   panX,
		panY:   panY,
		zoom:   zoom,
	}
}

// linkFace / centralFace are the word fonts scaled by the zoom.
func (l gameLayout) linkFace() text.Face    { return newFace(20 * l.zoom) }
func (l gameLayout) centralFace() text.Face { return newFace(34 * l.zoom) }

// linkSize / centralSize are the cloud sizes (already zoomed) that fit a word,
// used for both drawing and layout so everything stays consistent. Both wrap
// the word into lines the same way the drawing does.
func (l gameLayout) linkSize(word string) cloudSize {
	return cloudSizeFor(wrapWord(word), l.linkFace(), linkMinHalfWidth*l.zoom)
}

func (l gameLayout) centralSize(r *round) cloudSize {
	min := centralMinHalfWidth * l.zoom
	if centralMasked(r) {
		return maskedCloudSize(wrapWord(r.hidden.Word), l.centralFace(), min)
	}
	return cloudSizeFor(centralLines(r), l.centralFace(), min)
}

// nodePos is the on-screen center of a revealed word.
type nodePos struct{ x, y float64 }

// computeNodePositions turns each node's fixed angle and depth into a screen
// position. Angles are assigned once on reveal, so nodes keep their place when
// new ones appear. Nodes are placed in reveal order and each is pushed farther
// from the center along its angle until it no longer overlaps the central
// cloud or any earlier node; comparing only against already placed nodes keeps
// the earlier ones from moving.
func computeNodePositions(r *round, l gameLayout) []nodePos {
	n := len(r.revealed)
	pos := make([]nodePos, n)
	// Spacing uses the cloud half-width, which is the wider axis of the oval.
	scales := make([]float64, n)
	for i, rn := range r.revealed {
		scales[i] = l.linkSize(rn.node.Word).hx
	}
	centralScale := l.centralSize(r).hx
	step := math.Max(24*l.zoom, l.radius*0.2)

	for i, rn := range r.revealed {
		if rn.pinned {
			// The player placed this one; px, py are stored relative to the
			// center in zoom-1 units, so it follows the pan and the zoom.
			pos[i] = nodePos{l.cx + rn.px*l.zoom, l.cy + rn.py*l.zoom}
			continue
		}
		cos, sin := math.Cos(rn.angle), math.Sin(rn.angle)
		// The ring follows the true graph distance (path length), so distant
		// relatives fly straight out far from the center, not just one ring.
		dist := len(rn.path) - 1
		if dist < 1 {
			dist = 1
		}
		ring := l.radius * (1 + ringStep*float64(dist-1))
		maxRing := ring + l.radius*8 // safety cap against a runaway loop
		for {
			x, y := l.cx+ring*cos, l.cy+ring*sin
			if ring >= maxRing || !cloudsOverlap(x, y, scales[i], l, pos, scales, centralScale, i) {
				pos[i] = nodePos{x, y}
				break
			}
			ring += step
		}
	}
	return pos
}

// cloudAt returns the index of the topmost revealed cloud under (mx, my), or
// -1 when the point is on the empty background. The central cloud is not
// draggable, so it is not considered.
func cloudAt(r *round, l gameLayout, pos []nodePos, mx, my float64) int {
	best := -1
	for i, rn := range r.revealed {
		sz := l.linkSize(rn.node.Word)
		// Point-in-ellipse test with the cloud half axes.
		dx, dy := (mx-pos[i].x)/sz.hx, (my-pos[i].y)/sz.hy
		if dx*dx+dy*dy <= 1 {
			best = i // later nodes are drawn on top, so they win the hit
		}
	}
	return best
}

// cloudsOverlap reports whether a cloud of the given scale centered at (x, y)
// would touch the central cloud or any of the first count placed nodes.
func cloudsOverlap(x, y, scale float64, l gameLayout, pos []nodePos, scales []float64, centralScale float64, count int) bool {
	const cloudGap = 10.0
	touches := func(ox, oy, oScale float64) bool {
		minDist := scale + oScale + cloudGap
		dx, dy := x-ox, y-oy
		return dx*dx+dy*dy < minDist*minDist
	}
	if touches(l.cx, l.cy, centralScale) {
		return true
	}
	for j := 0; j < count; j++ {
		if touches(pos[j].x, pos[j].y, scales[j]) {
			return true
		}
	}
	return false
}
