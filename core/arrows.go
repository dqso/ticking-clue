package core

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	pb "github.com/dqso/ticking-clue/proto/gen"
)

// Relation arrows. A symmetric relation is a plain double-headed arrow; a
// directed relation replaces the head with a glyph whose orientation encodes
// the direction, so a relation and its inverse read as mirror images:
//   - hypernym / hyponym: a round cup ("⊂"/"⊃"), open toward the broader word;
//   - holonym / meronym: a square cup ("⊑"/"⊒"), open toward the whole;
//   - derived: a short cross tick at the derived word.

// markerRadius is the size of a directed-relation glyph, scaled by the arrow
// width w like the arrow head.
func markerRadius(w float64) float64 { return 6.5 * w }

// drawRelationArrow draws the connection from a parent word (x0,y0) to a child
// word (x1,y1) styled by its relation type. Any multi-hop or unspecified type
// falls back to a plain single head.
func drawRelationArrow(dst *ebiten.Image, x0, y0, x1, y1 float64, edge pb.EdgeType, clr color.Color, w float64) {
	switch edge {
	case pb.EdgeType_SYNONYM, pb.EdgeType_ANTONYM, pb.EdgeType_COORDINATE_TERM, pb.EdgeType_RELATED:
		drawDoubleArrow(dst, x0, y0, x1, y1, clr, w)
	// One relation each (hyponymy / meronymy), one color; the cup's open side
	// faces the edge direction: hypernym/holonym point to the child, their
	// inverses to the parent.
	case pb.EdgeType_HYPERNYM:
		drawCupArrow(dst, x0, y0, x1, y1, clr, w, false, true)
	case pb.EdgeType_HYPONYM:
		drawCupArrow(dst, x0, y0, x1, y1, clr, w, false, false)
	case pb.EdgeType_HOLONYM:
		drawCupArrow(dst, x0, y0, x1, y1, clr, w, true, true)
	case pb.EdgeType_MERONYM:
		drawCupArrow(dst, x0, y0, x1, y1, clr, w, true, false)
	case pb.EdgeType_DERIVED_TO:
		drawTickArrow(dst, x0, y0, x1, y1, clr, w, true)
	case pb.EdgeType_DERIVED_FROM:
		drawTickArrow(dst, x0, y0, x1, y1, clr, w, false)
	default:
		drawArrow(dst, x0, y0, x1, y1, clr, w)
	}
}

// drawDoubleArrow draws a symmetric arrow with a sharp head on both ends. The
// shaft runs only between the two head bases, so it never pokes past a head or
// blunts a point.
func drawDoubleArrow(dst *ebiten.Image, x0, y0, x1, y1 float64, clr color.Color, w float64) {
	ux, uy, length, ok := lineDir(x0, y0, x1, y1)
	if !ok {
		return
	}
	headLen := math.Min(12*w, length*0.4) // leave room for both heads
	drawShaft(dst, x0+ux*headLen, y0+uy*headLen, x1-ux*headLen, y1-uy*headLen, clr, w)
	drawArrowHead(dst, x1, y1, ux, uy, headLen, clr, w)   // head at one end
	drawArrowHead(dst, x0, y0, -ux, -uy, headLen, clr, w) // head at the other
}

// drawArrowHead fills a triangular head with its tip at (tx,ty) pointing along
// the unit vector (dx,dy), its base headLen behind the tip.
func drawArrowHead(dst *ebiten.Image, tx, ty, dx, dy, headLen float64, clr color.Color, w float64) {
	hw := 10 * w / 2 // half the head width
	nx, ny := -dy, dx
	bx, by := tx-dx*headLen, ty-dy*headLen
	p := &vector.Path{}
	p.MoveTo(fx(tx), fx(ty))
	p.LineTo(fx(bx+nx*hw), fx(by+ny*hw))
	p.LineTo(fx(bx-nx*hw), fx(by-ny*hw))
	p.Close()
	fillPath(dst, p, clr)
}

// lineDir returns the unit vector from (x0,y0) to (x1,y1) and its length; ok is
// false when the two points nearly coincide.
func lineDir(x0, y0, x1, y1 float64) (ux, uy, length float64, ok bool) {
	dx, dy := x1-x0, y1-y0
	length = math.Hypot(dx, dy)
	if length < 1 {
		return 0, 0, 0, false
	}
	return dx / length, dy / length, length, true
}

// drawShaft draws the plain line body of a directed relation arrow.
func drawShaft(dst *ebiten.Image, x0, y0, x1, y1 float64, clr color.Color, w float64) {
	vector.StrokeLine(dst, fx(x0), fx(y0), fx(x1), fx(y1), fx(3*w), clr, true)
}

// drawCupArrow draws a shaft with a cup glyph at one end. square picks a square
// cup (meronymy) over a round one (hyponymy). openOut puts the cup at the child
// end (the edge's "to") opening toward it; false puts it at the parent end (the
// "from") opening toward it. So the cup sits at, and opens toward, the word the
// relation points to; the shaft fills the rest.
func drawCupArrow(dst *ebiten.Image, x0, y0, x1, y1 float64, clr color.Color, w float64, square, openOut bool) {
	ux, uy, length, ok := lineDir(x0, y0, x1, y1)
	if !ok {
		return
	}
	r := markerRadius(w)
	back := math.Min(r*1.6, length*0.5) // how far the cup sits inside its end
	var cx, cy, odx, ody float64
	if openOut {
		cx, cy = x1-ux*back, y1-uy*back // at the child, opening toward it
		odx, ody = ux, uy
	} else {
		cx, cy = x0+ux*back, y0+uy*back // at the parent, opening toward it
		odx, ody = -ux, -uy
	}
	// Meet the shaft at the cup's closed back (opposite its opening, a distance r
	// from the center), so the line stops at the glyph instead of poking through.
	bx, by := cx-odx*r, cy-ody*r
	if openOut {
		drawShaft(dst, x0, y0, bx, by, clr, w)
	} else {
		drawShaft(dst, bx, by, x1, y1, clr, w)
	}
	if square {
		drawSquareCup(dst, cx, cy, odx, ody, r, clr, w)
	} else {
		drawRoundCup(dst, cx, cy, odx, ody, r, clr, w)
	}
}

// drawRoundCup draws a "⊂"-like semicircle centered at (cx,cy) whose opening
// faces (odx,ody).
func drawRoundCup(dst *ebiten.Image, cx, cy, odx, ody, r float64, clr color.Color, w float64) {
	openAng := math.Atan2(ody, odx)
	a0, a1 := openAng+math.Pi/2, openAng+3*math.Pi/2 // the far semicircle
	const seg = 14
	px, py := cx+r*math.Cos(a0), cy+r*math.Sin(a0)
	for i := 1; i <= seg; i++ {
		a := a0 + (a1-a0)*float64(i)/float64(seg)
		nx, ny := cx+r*math.Cos(a), cy+r*math.Sin(a)
		vector.StrokeLine(dst, fx(px), fx(py), fx(nx), fx(ny), fx(2*w), clr, true)
		px, py = nx, ny
	}
}

// drawSquareCup draws a "⊑"-like bracket centered at (cx,cy) whose opening faces
// (odx,ody): a back bar with two arms, plus a second bar next to one arm so it
// reads as the square subset-or-equal sign.
func drawSquareCup(dst *ebiten.Image, cx, cy, odx, ody, r float64, clr color.Color, w float64) {
	qx, qy := -ody, odx // perpendicular to the opening direction
	b1x, b1y := cx-odx*r+qx*r, cy-ody*r+qy*r
	b2x, b2y := cx-odx*r-qx*r, cy-ody*r-qy*r
	t1x, t1y := cx+qx*r, cy+qy*r
	t2x, t2y := cx-qx*r, cy-qy*r
	sw := fx(2 * w)
	vector.StrokeLine(dst, fx(b1x), fx(b1y), fx(b2x), fx(b2y), sw, clr, true) // back
	vector.StrokeLine(dst, fx(b1x), fx(b1y), fx(t1x), fx(t1y), sw, clr, true) // one arm
	vector.StrokeLine(dst, fx(b2x), fx(b2y), fx(t2x), fx(t2y), sw, clr, true) // other arm
	// Double the -q arm with a nearby parallel bar (the "equal" line of ⊑).
	g := r * 0.55
	vector.StrokeLine(dst, fx(b2x-qx*g), fx(b2y-qy*g), fx(t2x-qx*g), fx(t2y-qy*g), sw, clr, true)
}

// drawTickArrow draws a full shaft plus a short cross tick near the derived
// word, marking a derivation. The tick sits at the target end when toEnd is
// true (DERIVED_TO) or at the source end when false (DERIVED_FROM).
func drawTickArrow(dst *ebiten.Image, x0, y0, x1, y1 float64, clr color.Color, w float64, toEnd bool) {
	ux, uy, length, ok := lineDir(x0, y0, x1, y1)
	if !ok {
		return
	}
	drawShaft(dst, x0, y0, x1, y1, clr, w)
	r := markerRadius(w)
	back := math.Min(r, length*0.5)
	tx, ty := x1-ux*back, y1-uy*back
	if !toEnd {
		tx, ty = x0+ux*back, y0+uy*back
	}
	qx, qy := -uy, ux
	vector.StrokeLine(dst, fx(tx+qx*r), fx(ty+qy*r), fx(tx-qx*r), fx(ty-qy*r), fx(2*w), clr, true)
}
