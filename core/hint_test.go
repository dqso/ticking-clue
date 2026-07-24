package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCeilMinute(t *testing.T) {
	tests := []struct {
		in, want time.Duration
	}{
		{0, time.Minute},                     // never free
		{-5 * time.Second, time.Minute},      // negatives round up to one minute
		{30 * time.Second, time.Minute},      // below a minute rounds up
		{60 * time.Second, time.Minute},      // exact minute stays
		{61 * time.Second, 2 * time.Minute},  // just over rounds up
		{150 * time.Second, 3 * time.Minute}, // 2:30 -> 3:00
		{180 * time.Second, 3 * time.Minute}, // exact stays
	}
	for _, tc := range tests {
		assert.Equalf(t, tc.want, ceilMinute(tc.in), "ceilMinute(%s)", tc.in)
	}
}

// newLetterRound builds a round detailed just enough for the letter hint: a
// hidden word, the remaining time, and the letters already opened.
func newLetterRound(word string, remaining time.Duration, revealed ...rune) *round {
	r := &round{
		hidden:          &Node{Word: word},
		remaining:       remaining,
		revealedLetters: map[rune]bool{},
	}
	for _, ch := range revealed {
		r.revealedLetters[ch] = true
	}
	return r
}

func TestLetterHintCost(t *testing.T) {
	tests := []struct {
		name      string
		word      string
		remaining time.Duration
		revealed  []rune
		want      time.Duration
	}{
		// egg: 2 distinct letters, ceilMinute = 180s -> 180*(0+1)/2 = 90s.
		{"egg fresh", "egg", 3 * time.Minute, nil, 90 * time.Second},
		// coffee: 4 distinct letters, ceilMinute = 600s -> 600*(0+1)/4 = 150s.
		{"coffee fresh", "coffee", 10 * time.Minute, nil, 150 * time.Second},
		// one letter open -> 600*(1+1)/4 = 300s (dearer than fresh).
		{"coffee one open", "coffee", 10 * time.Minute, []rune{'c'}, 300 * time.Second},
		// remaining rounds up: 2:01 -> 3:00, same as egg fresh.
		{"egg rounds remaining up", "egg", 121 * time.Second, nil, 90 * time.Second},
	}
	for _, tc := range tests {
		r := newLetterRound(tc.word, tc.remaining, tc.revealed...)
		assert.Equalf(t, tc.want, r.letterHintCost(), "letterHintCost %s", tc.name)
	}
}

// TestLetterHintCostRises checks the price goes up as more of the word is open,
// so the hint never only gets cheaper.
func TestLetterHintCostRises(t *testing.T) {
	r := newLetterRound("coffee", 10*time.Minute)
	fresh := r.letterHintCost()
	r.revealedLetters['c'] = true
	after := r.letterHintCost()
	assert.Greater(t, after, fresh)
}

// TestLetterHintCap checks the letter hint is capped at half the distinct
// letters, so the word can never be bought out.
func TestLetterHintCap(t *testing.T) {
	// coffee: 4 distinct -> at most 2 opens.
	c := newLetterRound("coffee", 10*time.Minute)
	assert.True(t, c.canOpenLetter())
	c.revealedLetters['c'] = true
	assert.True(t, c.canOpenLetter())
	c.revealedLetters['o'] = true
	assert.False(t, c.canOpenLetter(), "two of four distinct letters open -> capped")

	// egg: 2 distinct -> at most 1 open.
	e := newLetterRound("egg", 3*time.Minute)
	assert.True(t, e.canOpenLetter())
	e.revealedLetters['g'] = true
	assert.False(t, e.canOpenLetter(), "one of two distinct letters open -> capped")
}

// TestUseLetterHintRespectsCap opens letters through the real code path until it
// stops, and checks it never opened more than half the distinct letters.
func TestUseLetterHintRespectsCap(t *testing.T) {
	r := newLetterRound("coffee", 100*time.Hour) // plenty of time so it never runs out
	r.lengthShown = true
	for i := 0; r.canOpenLetter(); i++ {
		if i > 10 {
			t.Fatal("cap is not enforced: opened too many letters")
		}
		r.useLetterHint()
	}
	// coffee has 4 distinct letters, so at most 2 may be open.
	assert.Len(t, r.revealedLetters, 2)
}
