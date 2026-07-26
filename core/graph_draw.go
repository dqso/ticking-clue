package core

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// paperGridCell is the size of one square of the notebook grid.
const paperGridCell = 34.0

// arrowCloudGap is the clearance (in screen pixels at zoom 1) left between an
// arrow end and the cloud outline, so the arrow never touches the puffs.
const arrowCloudGap = 2.0

// fx is a short float64->float32 helper for the vector package calls.
func fx(v float64) float32 { return float32(v) }

// drawPaper fills the squared notebook background, scrolled with the pan.
func drawPaper(screen *ebiten.Image, l gameLayout) {
	screen.Fill(paperColor)
	// Offset the first grid line by the pan, wrapped into one cell, so the
	// grid appears to move under the words.
	cell := paperGridCell * l.zoom
	startX := math.Mod(l.panX, cell)
	for x := startX; x < l.w; x += cell {
		vector.StrokeLine(screen, fx(x), 0, fx(x), fx(l.h), 1, paperRule, false)
	}
	startY := math.Mod(l.panY, cell)
	for y := startY; y < l.h; y += cell {
		vector.StrokeLine(screen, 0, fx(y), fx(l.w), fx(y), 1, paperRule, false)
	}
}

// drawGraphArrows draws an arrow from each revealed node's parent (the center
// or another node) outward to the node, marking skipped path vertices. Nodes in
// skip are still flying in, so their arrow is drawn only once they land.
func drawGraphArrows(screen *ebiten.Image, r *round, l gameLayout, pos []nodePos, skip map[int64]bool) {
	centralSize := l.centralSize(r)
	for i, rn := range r.revealed {
		if skip[rn.node.ID] {
			continue
		}
		childSize := l.linkSize(rn.node.Word)
		var px, py float64
		var pSize cloudSize
		var pSeed int64
		if rn.parent < 0 {
			px, py, pSize, pSeed = l.cx, l.cy, centralSize, r.hidden.ID
		} else {
			px, py = pos[rn.parent].x, pos[rn.parent].y
			pSize = l.linkSize(r.revealed[rn.parent].node.Word)
			pSeed = r.revealed[rn.parent].node.ID
		}
		dx, dy := pos[i].x-px, pos[i].y-py
		dist := math.Hypot(dx, dy)
		if dist < 1 {
			continue
		}
		ux, uy := dx/dist, dy/dist
		// Stop the arrow at each cloud's lumpy outline (plus a small gap), so its
		// ends sit clear of the puffs instead of vanishing under them. The child
		// outline is crossed by the ray coming back toward its center (-u).
		gap := arrowCloudGap * l.zoom
		tailLen := cloudRayDist(cloudOutline(pSize.hx, pSize.hy, pSeed), ux, uy) + gap
		headLen := cloudRayDist(cloudOutline(childSize.hx, childSize.hy, rn.node.ID), -ux, -uy) + gap
		tailX, tailY := px+tailLen*ux, py+tailLen*uy
		headX, headY := pos[i].x-headLen*ux, pos[i].y-headLen*uy
		// Skip when the two clouds nearly touch and leave no room for an arrow.
		if (headX-tailX)*ux+(headY-tailY)*uy <= 8*l.zoom {
			continue
		}
		// Fade the arrow with the graph distance of its far end, so distant
		// relations look fainter than close ones.
		alpha := arrowAlpha(len(rn.path) - 1)
		if r.colorsShown {
			// The colors hint reveals each relation by its color and its arrow
			// shape (double head, cup, square, or tick).
			drawRelationArrow(screen, tailX, tailY, headX, headY, rn.edge, fadeColor(edgeColor(rn.edge), alpha), l.zoom)
		} else {
			// Before the hint: plain gray lines with no heads, so no relation
			// direction or type is revealed.
			drawShaft(screen, tailX, tailY, headX, headY, fadeColor(arrowColor, alpha), l.zoom)
		}
		if rn.skipped > 0 {
			drawSkippedNodes(screen, tailX, tailY, headX, headY, rn.skipped, l.zoom, alpha)
		}
	}
}

// drawGraphClouds draws every revealed word as a small cloud with the word.
// Nodes in skip are still flying in and are drawn by the flyer instead.
func drawGraphClouds(screen *ebiten.Image, r *round, l gameLayout, pos []nodePos, skip map[int64]bool) {
	face := l.linkFace()
	for i, rn := range r.revealed {
		if skip[rn.node.ID] {
			continue
		}
		sz := l.linkSize(rn.node.Word)
		drawCloudShape(screen, pos[i].x, pos[i].y, sz.hx, sz.hy, cloudStrokeWidth(sz.hy), rn.node.ID)
		// Only hint-revealed clouds mask the hidden word; self-guessed ones show
		// every token.
		var mask map[string]struct{}
		if rn.byHint {
			mask = r.hiddenTokens
		}
		drawCloudWord(screen, rn.node.Word, face, pos[i].x, pos[i].y, cloudTextColor, mask)
	}
}

// drawCloud draws the central hidden word cloud on top of the arrow tails.
func drawCloud(screen *ebiten.Image, r *round, l gameLayout) {
	sz := l.centralSize(r)
	drawCloudShape(screen, l.cx, l.cy, sz.hx, sz.hy, cloudStrokeWidth(sz.hy), r.hidden.ID)
	face := l.centralFace()
	if centralMasked(r) {
		// Boxed letters (one per letter), separators shown as themselves,
		// revealed letters drawn inside their box.
		drawMaskedWord(screen, wrapWord(r.hidden.Word), face, l.cx, l.cy, cloudTextColor, r.revealedLetters)
		return
	}
	drawCloudText(screen, centralLines(r), face, l.cx, l.cy, cloudTextColor)
}

// drawArrow draws a thin solid arrow (shaft plus head) from tail to tip.
// w scales the shaft and head thickness (1 at zoom 1).
func drawArrow(dst *ebiten.Image, x0, y0, x1, y1 float64, clr color.Color, w float64) {
	shaftW := 3.0 * w
	headW := 10.0 * w
	headLen := 12.0 * w
	dx, dy := x1-x0, y1-y0
	length := math.Hypot(dx, dy)
	if length < 1 {
		return
	}
	ux, uy := dx/length, dy/length // along the arrow
	nx, ny := -uy, ux              // perpendicular
	hl := math.Min(headLen, length*0.8)
	bx, by := x1-ux*hl, y1-uy*hl // head base center
	sw, hw := shaftW/2, headW/2
	p := &vector.Path{}
	p.MoveTo(fx(x0+nx*sw), fx(y0+ny*sw))
	p.LineTo(fx(bx+nx*sw), fx(by+ny*sw))
	p.LineTo(fx(bx+nx*hw), fx(by+ny*hw))
	p.LineTo(fx(x1), fx(y1))
	p.LineTo(fx(bx-nx*hw), fx(by-ny*hw))
	p.LineTo(fx(bx-nx*sw), fx(by-ny*sw))
	p.LineTo(fx(x0-nx*sw), fx(y0-ny*sw))
	p.Close()
	fillPath(dst, p, clr)
}

// drawSkippedNodes draws count small empty nodes evenly along the arrow to
// stand for the hidden vertices skipped on that path, faded by alpha with it.
func drawSkippedNodes(dst *ebiten.Image, x0, y0, x1, y1 float64, count int, zoom, alpha float64) {
	rad := fx(7 * zoom)
	fill := fadeColor(cloudInterior, alpha)
	stroke := fadeColor(skippedColor, alpha)
	for k := 1; k <= count; k++ {
		t := float64(k) / float64(count+1)
		px, py := x0+(x1-x0)*t, y0+(y1-y0)*t
		vector.FillCircle(dst, fx(px), fx(py), rad, fill, true)
		vector.StrokeCircle(dst, fx(px), fx(py), rad, fx(2*zoom), stroke, true)
	}
}

// arrowAlpha fades an arrow with the graph distance of its far end, down to a
// floor so it never fully disappears.
func arrowAlpha(dist int) float64 {
	return math.Max(0.3, 1.0-0.16*float64(dist-1))
}

// fadeColor returns c with its alpha scaled by a (0..1).
func fadeColor(c color.NRGBA, a float64) color.NRGBA {
	c.A = uint8(clampF(float64(c.A)*a, 0, 255))
	return c
}

// fillPath fills a vector path with a flat color.
func fillPath(dst *ebiten.Image, p *vector.Path, clr color.Color) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, p, nil, op)
}

// drawTextCentered draws s centered at (cx, cy).
func drawTextCentered(dst *ebiten.Image, s string, face text.Face, cx, cy float64, clr color.Color) {
	w, h := text.Measure(s, face, 0)
	op := &text.DrawOptions{}
	op.GeoM.Translate(cx-w/2, cy-h/2)
	op.ColorScale.ScaleWithColor(clr)
	op.Filter = ebiten.FilterLinear
	text.Draw(dst, s, face, op)
}

// drawTextLeft draws s left-aligned with its top-left corner at (x, y).
func drawTextLeft(dst *ebiten.Image, s string, face text.Face, x, y float64, clr color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	op.Filter = ebiten.FilterLinear
	text.Draw(dst, s, face, op)
}
