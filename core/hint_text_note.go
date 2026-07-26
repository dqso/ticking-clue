package core

import (
	"fmt"
	"image"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// textNoteIconSize is the collapsed square side (in zoom-1 world units, so it
	// pans and zooms with the clouds, like the colors note).
	textNoteIconSize = 64.0
	// textNoteExpand is the expanded sticker's side; it is square and keeps a
	// fixed on-screen size, so the text stays sharp at any zoom.
	textNoteExpand = 300.0
	// textNotePad is the inner padding of the expanded sticker.
	textNotePad = 16.0
	// textNoteFontPx is the log font size.
	textNoteFontPx = 15.0
	// textNoteEntryGap is the vertical gap between two log entries.
	textNoteEntryGap = 8.0
	// textNoteTimeGap is the gap after the left timecode column.
	textNoteTimeGap = 10.0
	// textNoteScrollSpeed is how far one wheel notch scrolls the log.
	textNoteScrollSpeed = 24.0
	// textNoteMoveSpeed is the lerp factor when the sticker slides to the word.
	textNoteMoveSpeed = 0.18
)

// hintNote is one logged hint: the timecode (remaining time when it was used)
// and a short description of its result. The time is kept as a duration and
// stringified only when drawn.
type hintNote struct {
	time time.Duration
	text string
}

// textNote is the light-pink "Notes" sticker: a running log of every used hint.
// It lives on the sheet in world coords (tx, ty is its top-left corner relative
// to the graph center, in zoom-1 units), can be dragged, and a click (no drag)
// opens it into a scrollable list. It appears once the first hint is logged.
type textNote struct {
	placed   bool
	expanded bool
	tx, ty   float64
	// log is every used hint in order; the sticker shows it top to bottom.
	log []hintNote
	// scroll is the vertical scroll offset of the expanded list.
	scroll float64

	// Drag bookkeeping, mirroring the colors note.
	dragging       bool
	moved          bool
	grabDX, grabDY float64
	pressX, pressY int
	// captured marks that the current press is handled by this note, so the
	// other stickers and the graph pan/drag ignore it.
	captured bool

	// moving slides the sticker to (targetX, targetY) in world coords at the end
	// of a won round, so the notes settle next to the guessed word.
	moving           bool
	movedToWord      bool
	targetX, targetY float64
}

// active reports whether the sticker has anything to show yet.
func (n *textNote) active() bool { return len(n.log) > 0 }

// logNote appends an entry to the Notes log, stamped with the timecode t (the
// remaining time captured before any cost was paid). An empty text is skipped.
func (s *GameScene) logNote(t time.Duration, text string) {
	if text == "" {
		return
	}
	s.textNote.log = append(s.textNote.log, hintNote{time: t, text: text})
}

// logHint records a used hint in first person: the hint's name and its cost.
func (s *GameScene) logHint(t time.Duration, name string, cost time.Duration) {
	if name == "" {
		return
	}
	s.logNote(t, fmt.Sprintf("I used the %q hint. %s", name, signedSeconds(-cost)))
}

// logSpeedHint records a "speed" hint (word length, arrow colors): it costs no
// time but makes the clock run faster, so the note states the new multiplier.
func (s *GameScene) logSpeedHint(t time.Duration, name string) {
	if name == "" {
		return
	}
	s.logNote(t, fmt.Sprintf("I used the %q hint. Time now runs faster (×%s).", name, formatScale(s.round.timeScale)))
}

// logGuess records a submitted guess in first person, with the time change.
func (s *GameScene) logGuess(t time.Duration, out guessOutcome, delta time.Duration) {
	switch out.kind {
	case guessWin:
		s.logNote(t, fmt.Sprintf("I guessed %q. It is right.", out.word))
	case guessReward:
		s.logNote(t, fmt.Sprintf("I guessed %q. It is close. %s", out.word, signedSeconds(delta)))
	case guessPenalty:
		s.logNote(t, fmt.Sprintf("I guessed %q. It is wrong. %s", out.word, signedSeconds(delta)))
	}
}

// logStartNote opens the journal at the full starting time with the free words
// the game gives away. The words are masked like the clouds, so the journal
// never leaks a token of the hidden word.
func (s *GameScene) logStartNote(words []*Node) {
	names := make([]string, len(words))
	for i, w := range words {
		names[i] = maskWord(w.Word, s.round.hiddenTokens)
	}
	s.logNote(s.round.remaining, "Game start: the word is unknown. Given words: "+strings.Join(names, ", ")+".")
}

// signedSeconds formats a signed duration as whole seconds with a kept sign,
// e.g. +25s or -34s (0 gives "0s").
func signedSeconds(d time.Duration) string {
	sec := int(d / time.Second)
	if sec > 0 {
		return fmt.Sprintf("+%ds", sec)
	}
	return fmt.Sprintf("%ds", sec)
}

// textNoteRect is the sticker's screen rectangle (top-left x, y and size), its
// world corner projected through the current pan and zoom.
func (s *GameScene) textNoteRect(l gameLayout) (x, y, w, h float64) {
	x = l.cx + s.textNote.tx*l.zoom
	y = l.cy + s.textNote.ty*l.zoom
	if s.textNote.expanded {
		return x, y, textNoteExpand, textNoteExpand
	}
	return x, y, textNoteIconSize * l.zoom, textNoteIconSize * l.zoom
}

// inTextNote reports whether (mx, my) is over the sticker.
func (s *GameScene) inTextNote(mx, my int, l gameLayout) bool {
	x, y, w, h := s.textNoteRect(l)
	return float64(mx) >= x && float64(mx) <= x+w && float64(my) >= y && float64(my) <= y+h
}

// placeTextNote puts the sticker on the sheet the first time it appears: to the
// right of the colors note's corner (clear of the timer and hint column),
// converted to world coords so it pans and zooms.
func (s *GameScene) placeTextNote(l gameLayout) {
	s.textNote.tx = (120 - l.cx) / l.zoom
	s.textNote.ty = (90 - l.cy) / l.zoom
	s.textNote.placed = true
}

// updateTextNote drives the sticker's slide-to-word, its drag, and the click
// that opens or closes it. The miss list and the colors note get the press
// first (they draw on top), so it skips a press one of them already captured.
func (s *GameScene) updateTextNote(g *Game) {
	n := &s.textNote
	if !n.active() {
		return
	}
	l := computeLayout(g.screenWidth, g.screenHeight, s.panX, s.panY, s.zoom)
	if !n.placed {
		s.placeTextNote(l)
	}
	// Slide toward the guessed word at the end of the round.
	if n.moving {
		n.tx += (n.targetX - n.tx) * textNoteMoveSpeed
		n.ty += (n.targetY - n.ty) * textNoteMoveSpeed
		if math.Hypot(n.targetX-n.tx, n.targetY-n.ty) < 0.5 {
			n.tx, n.ty, n.moving = n.targetX, n.targetY, false
		}
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		n.dragging = false
		n.captured = false
	}
	mx, my := ebiten.CursorPosition()
	justPressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	justReleased := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)

	if justPressed && s.inTextNote(mx, my, l) && !s.fx.captured && !s.colorsNote.captured {
		x, y, _, _ := s.textNoteRect(l)
		// Grabbing the sticker cancels an ongoing slide.
		n.dragging, n.captured, n.moved, n.moving = true, true, false, false
		n.pressX, n.pressY = mx, my
		n.grabDX = float64(mx) - x
		n.grabDY = float64(my) - y
	}
	if n.dragging && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if absInt(mx-n.pressX) > 3 || absInt(my-n.pressY) > 3 {
			n.moved = true
		}
		// Store the corner in world coords so it stays put on the sheet.
		n.tx = (float64(mx) - n.grabDX - l.cx) / l.zoom
		n.ty = (float64(my) - n.grabDY - l.cy) / l.zoom
	}
	if justReleased && !n.moved && s.inTextNote(mx, my, l) {
		n.expanded = !n.expanded // a click (no drag) opens or closes the sticker
	}
}

// updateTextNoteScroll scrolls the expanded log with the wheel over it.
func (s *GameScene) updateTextNoteScroll(l gameLayout) {
	n := &s.textNote
	if !n.active() || !n.expanded {
		return
	}
	mx, my := ebiten.CursorPosition()
	if !s.inTextNote(mx, my, l) {
		return
	}
	if _, dy := ebiten.Wheel(); dy != 0 {
		n.scroll -= dy * textNoteScrollSpeed
	}
	innerH := textNoteExpand - 2*textNotePad
	max := math.Max(0, s.textNoteContentHeight()-innerH)
	n.scroll = clampF(n.scroll, 0, max)
}

// moveNotesToWord slides the sticker next to the central (guessed) word and
// opens it, so the notes are read beside the answer. It runs once per round.
func (s *GameScene) moveNotesToWord(l gameLayout) {
	n := &s.textNote
	if !n.active() || n.movedToWord {
		return
	}
	n.movedToWord = true
	n.expanded = true
	n.placed = true
	// Put the top-left corner just right of the central cloud, centered on it
	// vertically, in world coords so it rides the pan and zoom.
	half := l.centralSize(s.round).hx
	n.targetX = (half + 30) / l.zoom
	n.targetY = (-textNoteExpand / 2) / l.zoom
	n.moving = true
}

// textNoteContentHeight is the total height of every wrapped log entry, used to
// clamp the scroll offset.
func (s *GameScene) textNoteContentHeight() float64 {
	face := newFace(textNoteFontPx)
	lh := faceLineSpacing(face)
	timeW, _ := text.Measure("00:00", face, 0)
	textColW := textNoteExpand - 2*textNotePad - timeW - textNoteTimeGap
	h := 0.0
	for _, e := range s.textNote.log {
		h += float64(len(entryLines(e.text, face, textColW)))*lh + textNoteEntryGap
	}
	return h
}

// drawTextNote draws the pink sticker: the collapsed "Notes" label, or the
// scrollable hint log while expanded.
func (s *GameScene) drawTextNote(screen *ebiten.Image, l gameLayout) {
	n := &s.textNote
	if !n.active() {
		return
	}
	x, y, w, h := s.textNoteRect(l)
	vector.FillRect(screen, fx(x), fx(y), fx(w), fx(h), textNoteColor, false)
	if !n.expanded {
		vector.StrokeRect(screen, fx(x), fx(y), fx(w), fx(h), fx(2*l.zoom), textNoteEdge, true)
		drawTextCentered(screen, "Notes", newFace(w*0.22), x+w/2, y+h/2, textNoteText)
		return
	}
	vector.StrokeRect(screen, fx(x), fx(y), fx(w), fx(h), 2, textNoteEdge, true)

	// Draw the log into the inner area, clipped so nothing spills past the edges.
	innerX, innerY := x+textNotePad, y+textNotePad
	innerW, innerH := w-2*textNotePad, h-2*textNotePad
	clip := screen.SubImage(image.Rect(
		int(innerX), int(innerY),
		int(math.Ceil(innerX+innerW)), int(math.Ceil(innerY+innerH)),
	)).(*ebiten.Image)
	face := newFace(textNoteFontPx)
	lh := faceLineSpacing(face)
	timeW, _ := text.Measure("00:00", face, 0)
	textColW := innerW - timeW - textNoteTimeGap
	cy := innerY - n.scroll
	// Freshest entries on top: walk the log newest to oldest.
	for i := len(n.log) - 1; i >= 0; i-- {
		e := n.log[i]
		lines := entryLines(e.text, face, textColW)
		blockH := float64(len(lines)) * lh
		if cy+blockH >= innerY && cy <= innerY+innerH { // skip entries out of view
			drawTextLeft(clip, formatMMSS(e.time), face, innerX, cy, textNoteTime)
			ty := cy
			for _, ln := range lines {
				drawTextLeft(clip, ln, face, innerX+timeW+textNoteTimeGap, ty, textNoteText)
				ty += lh
			}
		}
		cy += blockH + textNoteEntryGap
	}
}

// entryLines wraps one log entry's text, never returning zero lines.
func entryLines(s string, face text.Face, maxW float64) []string {
	lines := wrapText(s, face, maxW)
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// wrapText breaks s into lines that each fit maxW when drawn with face,
// splitting only on spaces (a single word may overflow if it is too long).
func wrapText(s string, face text.Face, maxW float64) []string {
	var lines []string
	var cur string
	for _, w := range strings.Fields(s) {
		try := w
		if cur != "" {
			try = cur + " " + w
		}
		if lw, _ := text.Measure(try, face, 0); lw > maxW && cur != "" {
			lines = append(lines, cur)
			cur = w
			continue
		}
		cur = try
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}
