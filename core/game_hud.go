package core

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// timerLowThreshold is when the countdown turns red.
const timerLowThreshold = 15 * time.Second

// drawTimer draws the countdown MM:SS in the top-right corner.
func drawTimer(screen *ebiten.Image, r *round, l gameLayout) {
	face := newFace(32)
	txt := formatMMSS(r.remaining)
	clr := timerColor
	if r.remaining <= timerLowThreshold {
		clr = timerLowColor
	}
	w, _ := text.Measure(txt, face, 0)
	op := &text.DrawOptions{}
	op.GeoM.Translate(l.w-16-w, 14)
	op.ColorScale.ScaleWithColor(clr)
	op.Filter = ebiten.FilterLinear
	text.Draw(screen, txt, face, op)
}

// formatMMSS renders a non-negative duration as MM:SS.
func formatMMSS(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d / time.Second)
	return fmt.Sprintf("%02d:%02d", total/60, total%60)
}

// drawGuess draws the guess input just to the left of the timer (right-aligned
// against it): a prompt until the first character is typed, then the typed word
// with a caret, and the last guess result flashing just under it.
func drawGuess(screen *ebiten.Image, r *round, l gameLayout) {
	const rightPad, gap = 16.0, 16.0
	// Right edge to align to: left of the timer, which reserves a fixed width
	// (measured from "00:00") so it never jitters as the digits change.
	timerFace := newFace(32)
	reserveW, timerH := text.Measure("00:00", timerFace, 0)
	rightEdge := l.w - rightPad - reserveW - gap

	// Prompt while empty (no caret), the typed word plus caret once typing.
	content, face, clr := r.guess+"|", newFace(26), guessColor
	if r.guess == "" {
		content, face, clr = "Use the keyboard to guess the word", newFace(18), promptColor
	}
	cw, ch := text.Measure(content, face, 0)
	drawTextLeft(screen, content, face, rightEdge-cw, 14+(timerH-ch)/2, clr)

	// Feedback right under the input, right-aligned to the same edge.
	if r.flash > 0 {
		if fb := feedbackText(r); fb != "" {
			clr := rewardColor
			switch r.lastResult {
			case guessPenalty:
				clr = penaltyColor
			case guessKnown:
				clr = guessColor
			}
			fb18 := newFace(18)
			fw, _ := text.Measure(fb, fb18, 0)
			drawTextLeft(screen, fb, fb18, rightEdge-fw, 14+timerH+2, clr)
		}
	}
}

// feedbackText is the reason plus the signed time change shown after a guess.
func feedbackText(r *round) string {
	switch r.lastResult {
	case guessReward:
		return r.lastReason + "  +" + formatMMSS(r.lastDelta)
	case guessPenalty:
		return r.lastReason + "  -" + formatMMSS(-r.lastDelta)
	case guessKnown:
		return r.lastReason
	default:
		return ""
	}
}

// drawEndOverlay dims the screen and shows the round result.
func drawEndOverlay(screen *ebiten.Image, r *round, l gameLayout) {
	vector.FillRect(screen, 0, 0, fx(l.w), fx(l.h), overlayColor, false)
	title := "Time's up!"
	if r.state == roundWon {
		title = "You win!"
	}
	drawTextCentered(screen, title, newFace(48), l.w/2, l.h/2-40, overlayTextColor)
	drawTextCentered(screen, "The word was: "+r.hidden.Word, newFace(26), l.w/2, l.h/2+8, cloudInterior)
	drawTextCentered(screen, "Press Enter to return to the menu", newFace(20), l.w/2, l.h/2+52, guessColor)
}
