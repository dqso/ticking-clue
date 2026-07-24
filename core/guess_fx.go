package core

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// flyDuration is how long a guessed word takes to fly to its place.
	flyDuration = 550 * time.Millisecond
	// flyMissTargetY is where a missed word flies to before it drops into the
	// list (near the top of the sticky note).
	flyMissTargetY = 60.0

	// Miss list (left sticky note) metrics.
	listFontPx      = 20.0
	listPad         = 14.0 // inner padding around the words
	listWordGap     = 10.0 // extra gap between two words
	listScrollSpeed = 28.0 // pixels per wheel notch
	listButtonH     = 40.0 // collapse button strip at the bottom
	listIconSize    = 60.0 // collapsed square sticker side
)

// flyGuess is a submitted word animating from the input to its destination: a
// reward flies to node's cloud, a penalty flies into the miss list.
type flyGuess struct {
	word  string
	lines []string
	kind  guessResult
	node  *Node // reward target; nil for a miss
	t     float64
	// fromCenter starts the flight at the hidden cloud (used by the link hint)
	// instead of at the input line.
	fromCenter bool
}

// guessFX groups the flying guesses and the miss list, so the scene keeps a
// single field for all of it.
type guessFX struct {
	// flyers are the guessed words animating to their place or into the list.
	flyers []*flyGuess
	// missed is the sorted, unknown or too-far words shown in the list.
	missed []string
	// scroll is the vertical scroll offset of the expanded list.
	scroll float64
	// missedSet is the same words as a set, for O(1) "already rejected" checks;
	// a word joins it the moment it is rejected, even before it lands.
	missedSet map[string]bool

	// collapsed folds the list into a small square sticker that lives on the
	// sheet (world coords sx, sy are its center relative to the graph center,
	// in zoom-1 units, so it pans and zooms with the clouds). It can be dragged,
	// clouds push it outward when they need its spot, and a click expands it.
	collapsed  bool
	iconPlaced bool
	sx, sy     float64
	// Drag bookkeeping for the collapsed sticker.
	iconDragging           bool
	iconMoved              bool
	iconGrabDX, iconGrabDY float64
	iconPressX, iconPressY int
	// captured marks that the current press is handled by the list, so the
	// graph pan/drag ignores it.
	captured bool
}

// submitGuess resolves the current guess. A word already on the graph or
// already rejected is declined with feedback and no time change; anything else
// goes to the round and flies to its place or into the miss list.
func (s *GameScene) submitGuess() {
	word := strings.TrimSpace(strings.ToLower(s.round.guess))
	if word != "" && word != s.round.hidden.Word && s.alreadyKnown(word) {
		s.round.guess = ""
		s.round.setFeedback(guessKnown, "already tried", 0)
		return
	}
	s.startFlyer(s.round.submit(), false)
}

// alreadyKnown reports that the word is shown on the graph or already rejected,
// both O(1) lookups.
func (s *GameScene) alreadyKnown(word string) bool {
	if s.fx.missedSet[word] {
		return true
	}
	if n := s.round.graph.ByWord(word); n != nil {
		_, ok := s.round.revealedSet[n.ID]
		return ok
	}
	return false
}

// startFlyer begins the fly animation for a submitted guess (rewards and
// misses only; wins and empty inputs animate nothing). fromCenter starts the
// flight at the hidden cloud instead of the input line.
func (s *GameScene) startFlyer(res guessOutcome, fromCenter bool) {
	if res.kind != guessReward && res.kind != guessPenalty {
		return
	}
	if res.kind == guessPenalty {
		if s.fx.missedSet == nil {
			s.fx.missedSet = make(map[string]bool)
		}
		s.fx.missedSet[res.word] = true // known at once, even while flying
	}
	s.fx.flyers = append(s.fx.flyers, &flyGuess{
		word:       res.word,
		lines:      wrapWord(res.word),
		kind:       res.kind,
		node:       res.node,
		fromCenter: fromCenter,
	})
}

// updateFlyers advances every flying word; a landed miss drops into the list.
func (s *GameScene) updateFlyers() {
	if len(s.fx.flyers) == 0 {
		return
	}
	dt := float64(tickDuration()) / float64(flyDuration)
	kept := s.fx.flyers[:0]
	for _, f := range s.fx.flyers {
		f.t += dt
		if f.t < 1 {
			kept = append(kept, f)
			continue
		}
		if f.kind == guessPenalty {
			s.addMissed(f.word)
		}
	}
	s.fx.flyers = kept
}

// animatingHits is the set of node ids whose cloud is still flying, so the
// graph does not draw them twice (the flyer draws them meanwhile).
func (s *GameScene) animatingHits() map[int64]bool {
	var m map[int64]bool
	for _, f := range s.fx.flyers {
		if f.kind == guessReward && f.node != nil {
			if m == nil {
				m = make(map[int64]bool)
			}
			m[f.node.ID] = true
		}
	}
	return m
}

// drawFlyers draws each flying word as a cloud moving toward its destination.
func (s *GameScene) drawFlyers(screen *ebiten.Image, l gameLayout, pos []nodePos) {
	if len(s.fx.flyers) == 0 {
		return
	}
	face := l.linkFace()
	inputX, inputY := l.cx, l.h-56 // where the typed guess sits
	for _, f := range s.fx.flyers {
		startX, startY := inputX, inputY
		if f.fromCenter {
			startX, startY = l.cx, l.cy // hint clouds grow out of the center
		}
		tx, ty := s.missTarget(l)
		if f.kind == guessReward {
			if idx, ok := s.round.revealedSet[f.node.ID]; ok && idx < len(pos) {
				tx, ty = pos[idx].x, pos[idx].y
			} else {
				tx, ty = startX, startY
			}
		}
		e := easeOutCubic(f.t)
		x, y := lerp(startX, tx, e), lerp(startY, ty, e)
		sz := cloudSizeFor(f.lines, face, linkMinHalfWidth*l.zoom)
		// Reward flyers reuse the node id so the shape matches the landed cloud.
		seed := hashWord(f.word)
		if f.node != nil {
			seed = f.node.ID
		}
		drawCloudShape(screen, x, y, sz.hx, sz.hy, cloudStrokeWidth(sz.hy), seed)
		// Hint flyers (fromCenter) mask the hidden word; self-guessed ones don't.
		var mask map[string]struct{}
		if f.fromCenter {
			mask = s.round.hiddenTokens
		}
		drawCloudWord(screen, f.word, face, x, y, cloudTextColor, mask)
	}
}

// missTarget is where a missed word flies to: the collapsed sticker on the
// sheet, or the top of the expanded left list.
func (s *GameScene) missTarget(l gameLayout) (float64, float64) {
	if s.fx.collapsed {
		return s.iconScreenPos(l)
	}
	return math.Max(s.listWidth()/2, flyMissTargetY), flyMissTargetY
}

// iconScreenPos is the collapsed sticker's center on screen (its world position
// projected through the current pan and zoom).
func (s *GameScene) iconScreenPos(l gameLayout) (float64, float64) {
	return l.cx + s.fx.sx*l.zoom, l.cy + s.fx.sy*l.zoom
}

// iconHalf is the collapsed sticker's half side on screen (it scales with zoom
// like the clouds).
func (s *GameScene) iconHalf(l gameLayout) float64 {
	return listIconSize * l.zoom / 2
}

// addMissed inserts a word into the sorted miss list, skipping duplicates.
func (s *GameScene) addMissed(word string) {
	i := sort.SearchStrings(s.fx.missed, word)
	if i < len(s.fx.missed) && s.fx.missed[i] == word {
		return
	}
	s.fx.missed = append(s.fx.missed, "")
	copy(s.fx.missed[i+1:], s.fx.missed[i:])
	s.fx.missed[i] = word
}

// listFace is the font of the miss list.
func listFace() text.Face { return newFace(listFontPx) }

// listWidth is the sticky note width: the widest wrapped word line plus padding.
func (s *GameScene) listWidth() float64 {
	if len(s.fx.missed) == 0 {
		return 0
	}
	face := listFace()
	maxW := 0.0
	for _, w := range s.fx.missed {
		for _, ln := range wrapWord(w) {
			if lw, _ := text.Measure(ln, face, 0); lw > maxW {
				maxW = lw
			}
		}
	}
	return maxW + 2*listPad
}

// listContentHeight is the total height of all words in the list.
func (s *GameScene) listContentHeight() float64 {
	lh := faceLineSpacing(listFace())
	h := listPad
	for _, w := range s.fx.missed {
		h += float64(len(wrapWord(w)))*lh + listWordGap
	}
	return h + listPad
}

// drawList draws the miss list, either as the full sticky note or, when
// collapsed, as a small square sticker.
func (s *GameScene) drawList(screen *ebiten.Image, l gameLayout) {
	if len(s.fx.missed) == 0 {
		return
	}
	if s.fx.collapsed {
		s.drawListIcon(screen, l)
		return
	}
	s.drawExpandedList(screen, l)
}

// drawExpandedList draws the yellow sticky note with the sorted, struck-through
// words, a scroll offset, and the collapse button pinned at the bottom.
func (s *GameScene) drawExpandedList(screen *ebiten.Image, l gameLayout) {
	face := listFace()
	w := s.listWidth()
	vector.FillRect(screen, 0, 0, fx(w), fx(l.h), stickerColor, false)
	vector.StrokeLine(screen, fx(w), 0, fx(w), fx(l.h), 2, stickerEdge, false)

	lh := faceLineSpacing(face)
	viewH := l.h - listButtonH // words stop above the button
	y := listPad - s.fx.scroll
	for _, word := range s.fx.missed {
		for _, ln := range wrapWord(word) {
			if y+lh >= 0 && y <= viewH {
				op := &text.DrawOptions{}
				op.GeoM.Translate(listPad, y)
				op.ColorScale.ScaleWithColor(listTextColor)
				op.Filter = ebiten.FilterLinear
				text.Draw(screen, ln, face, op)
				// Strike the line through its middle.
				lw, _ := text.Measure(ln, face, 0)
				my := y + lh*0.45
				vector.StrokeLine(screen, fx(listPad), fx(my), fx(listPad+lw), fx(my), 1.5, listTextColor, true)
			}
			y += lh
		}
		y += listWordGap
	}
	s.drawCollapseButton(screen, w, l.h)
}

// drawCollapseButton draws the bottom strip that folds the list into a sticker.
func (s *GameScene) drawCollapseButton(screen *ebiten.Image, w, h float64) {
	by := h - listButtonH
	vector.FillRect(screen, 0, fx(by), fx(w), fx(listButtonH), stickerEdge, false)
	// A small square glyph: "fold into a square sticker".
	cx, cy, sq := w/2, by+listButtonH/2, 16.0
	vector.StrokeRect(screen, fx(cx-sq/2), fx(cy-sq/2), fx(sq), fx(sq), 2, listTextColor, true)
}

// inCollapseButton reports whether (mx, my) is over the collapse button.
func (s *GameScene) inCollapseButton(mx, my int, l gameLayout) bool {
	return float64(mx) <= s.listWidth() && float64(my) >= l.h-listButtonH
}

// drawListIcon draws the collapsed square sticker on the sheet, scaled by zoom.
func (s *GameScene) drawListIcon(screen *ebiten.Image, l gameLayout) {
	cx, cy := s.iconScreenPos(l)
	h := s.iconHalf(l)
	x, y, side := cx-h, cy-h, 2*h
	vector.FillRect(screen, fx(x), fx(y), fx(side), fx(side), stickerColor, false)
	vector.StrokeRect(screen, fx(x), fx(y), fx(side), fx(side), fx(2*l.zoom), stickerEdge, true)
	// Three short struck lines hint at the crossed-out words inside.
	for i := 0; i < 3; i++ {
		ly := y + (0.3+float64(i)*0.2)*side
		vector.StrokeLine(screen, fx(x+0.2*side), fx(ly), fx(x+0.8*side), fx(ly), fx(2*l.zoom), listTextColor, true)
	}
}

// inIcon reports whether (mx, my) is over the collapsed square sticker.
func (s *GameScene) inIcon(mx, my int, l gameLayout) bool {
	cx, cy := s.iconScreenPos(l)
	h := s.iconHalf(l)
	return math.Abs(float64(mx)-cx) <= h && math.Abs(float64(my)-cy) <= h
}

// handleListInput drives the collapse button, the sticker drag, and the click
// that expands it. It sets fx.captured so the graph ignores that press.
func (s *GameScene) handleListInput(g *Game) {
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		s.fx.iconDragging = false
		s.fx.captured = false
	}
	if len(s.fx.missed) == 0 {
		return
	}
	mx, my := ebiten.CursorPosition()
	l := computeLayout(g.screenWidth, g.screenHeight, s.panX, s.panY, s.zoom)
	justPressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	justReleased := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)

	if s.fx.collapsed {
		if justPressed && s.inIcon(mx, my, l) {
			cx, cy := s.iconScreenPos(l)
			s.fx.iconDragging, s.fx.captured, s.fx.iconMoved = true, true, false
			s.fx.iconPressX, s.fx.iconPressY = mx, my
			s.fx.iconGrabDX = float64(mx) - cx
			s.fx.iconGrabDY = float64(my) - cy
		}
		if s.fx.iconDragging && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			if absInt(mx-s.fx.iconPressX) > 3 || absInt(my-s.fx.iconPressY) > 3 {
				s.fx.iconMoved = true
			}
			// Store the sticker in world coords so it stays put on the sheet.
			s.fx.sx = (float64(mx) - s.fx.iconGrabDX - l.cx) / l.zoom
			s.fx.sy = (float64(my) - s.fx.iconGrabDY - l.cy) / l.zoom
		}
		if justReleased && !s.fx.iconMoved && s.inIcon(mx, my, l) {
			s.fx.collapsed = false // a click (no drag) expands the list
		}
		return
	}

	// Expanded: a press anywhere on the sticker is captured; the button folds.
	if justPressed && float64(mx) <= s.listWidth() {
		s.fx.captured = true
		if s.inCollapseButton(mx, my, l) {
			s.fx.collapsed = true
			s.placeIcon(l)
		}
	}
}

// placeIcon puts the collapsed sticker on the sheet the first time it folds:
// near the lower-left corner, converted to world coords so it pans and zooms.
func (s *GameScene) placeIcon(l gameLayout) {
	if s.fx.iconPlaced {
		return
	}
	s.fx.sx = (120 - l.cx) / l.zoom
	s.fx.sy = (l.h - 120 - l.cy) / l.zoom
	s.fx.iconPlaced = true
}

// resolveSticker pushes the collapsed sticker outward (away from the graph
// center) until it no longer overlaps the central cloud or any revealed cloud,
// so the clouds keep their spots and the sticker yields room to them.
func (s *GameScene) resolveSticker(l gameLayout, pos []nodePos) {
	if !s.fx.collapsed || len(s.fx.missed) == 0 || s.fx.iconDragging {
		return
	}
	cx, cy := s.iconScreenPos(l)
	r := s.iconHalf(l) * 1.15 // treat the square as a slightly larger circle
	dx, dy := cx-l.cx, cy-l.cy
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		dx, dy, dist = 0, 1, 1 // undefined direction: push straight down
	}
	ux, uy := dx/dist, dy/dist
	centralR := l.centralSize(s.round).hx
	step := math.Max(6, r*0.3)
	for iter := 0; iter < 400; iter++ {
		hit := circlesOverlap(cx, cy, r, l.cx, l.cy, centralR)
		for i := 0; !hit && i < len(pos); i++ {
			ri := l.linkSize(s.round.revealed[i].node.Word).hx
			hit = circlesOverlap(cx, cy, r, pos[i].x, pos[i].y, ri)
		}
		if !hit {
			break
		}
		cx, cy = cx+ux*step, cy+uy*step
	}
	s.fx.sx = (cx - l.cx) / l.zoom
	s.fx.sy = (cy - l.cy) / l.zoom
}

// circlesOverlap reports whether two circles (with a small gap) intersect.
func circlesOverlap(x0, y0, r0, x1, y1, r1 float64) bool {
	const gap = 10.0
	dx, dy := x0-x1, y0-y1
	min := r0 + r1 + gap
	return dx*dx+dy*dy < min*min
}

// updateListScroll scrolls the expanded list with the wheel over it.
func (s *GameScene) updateListScroll(l gameLayout) {
	if len(s.fx.missed) == 0 || s.fx.collapsed {
		return
	}
	mx, _ := ebiten.CursorPosition()
	if float64(mx) > s.listWidth() {
		return
	}
	if _, dy := ebiten.Wheel(); dy != 0 {
		s.fx.scroll -= dy * listScrollSpeed
	}
	max := math.Max(0, s.listContentHeight()-(l.h-listButtonH))
	s.fx.scroll = clampF(s.fx.scroll, 0, max)
}

// easeOutCubic eases a 0..1 progress so the flight slows as it lands.
func easeOutCubic(t float64) float64 {
	t = clampF(t, 0, 1)
	u := 1 - t
	return 1 - u*u*u
}

// lerp is a linear interpolation between a and b.
func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// clampF clamps v into [lo, hi].
func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// absInt is the absolute value of an int.
func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// hashWord is a small stable hash used to seed a missed word's cloud shape.
func hashWord(w string) int64 {
	var h uint64 = 1469598103934665603
	for _, c := range w {
		h ^= uint64(c)
		h *= 1099511628211
	}
	return int64(h >> 1)
}
