package core

import (
	"fmt"
	"image/color"
	"strconv"
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
	w, h := text.Measure(txt, face, 0)
	op := &text.DrawOptions{}
	op.GeoM.Translate(l.w-16-w, 14)
	op.ColorScale.ScaleWithColor(clr)
	op.Filter = ebiten.FilterLinear
	text.Draw(screen, txt, face, op)

	// Speed debuff badge right under the timer while a speed hint sped time up:
	// a fast-forward glyph plus the current stacking multiplier, in red, so it
	// reads as "time runs faster", not "more time".
	if r.timeScale > 1 {
		drawSpeedDebuff(screen, r.timeScale, l.w-16, 14+h+4)
	}
}

// drawSpeedDebuff draws the fast-forward icon and "×N" multiplier right-aligned
// with its right edge at (rightX) and its top at (topY).
func drawSpeedDebuff(screen *ebiten.Image, scale float64, rightX, topY float64) {
	const iconW, iconH, gap = 15.0, 14.0, 5.0
	face := newFace(18)
	txt := "×" + formatScale(scale)
	tw, _ := text.Measure(txt, face, 0)
	startX := rightX - tw - gap - iconW
	drawFastForward(screen, startX, topY, iconW, iconH, penaltyColor)
	drawTextLeft(screen, txt, face, startX+iconW+gap, topY-1, penaltyColor)
}

// drawFastForward draws two right-pointing filled triangles (a fast-forward /
// "faster" glyph) inside the box [x, y, w, h].
func drawFastForward(dst *ebiten.Image, x, y, w, h float64, clr color.Color) {
	half := w / 2
	for i := 0; i < 2; i++ {
		ox := x + float64(i)*half
		p := &vector.Path{}
		p.MoveTo(fx(ox), fx(y))
		p.LineTo(fx(ox+half), fx(y+h/2))
		p.LineTo(fx(ox), fx(y+h))
		p.Close()
		fillPath(dst, p, clr)
	}
}

// formatScale renders a time-speed multiplier without trailing zeros, e.g.
// 1.5 -> "1.5", 2.25 -> "2.25".
func formatScale(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
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

	// Prompt while empty (no caret), the typed word plus caret once typing. Once
	// the round is over, the same line tells the player how to leave.
	content, face, clr := r.guess+"|", newFace(26), guessColor
	switch {
	case r.state != roundPlaying:
		content, face, clr = "Press Esc to return to the menu", newFace(18), promptColor
	case r.guess == "":
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

// drawTimeUp shows the loss message in big letters near the bottom of the
// screen. Unlike a modal overlay it does not dim the field: after a loss the
// graph stays visible and interactive, like after a win (the answer is revealed
// in the central cloud). The wording depends on how the round was lost.
func drawTimeUp(screen *ebiten.Image, r *round, l gameLayout) {
	msg := "Time's up!"
	if r.surrendered {
		msg = "You gave up!"
	}
	drawTextCentered(screen, msg, newFace(64), l.w/2, l.h-72, penaltyColor)
}
