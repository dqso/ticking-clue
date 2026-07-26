package core

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// manualFold maps letters that NFKD does not decompose to their keyboard-
// typeable ASCII form (stroke/bar letters, thorn, eth, eng, and the like).
var manualFold = map[rune]string{
	'ø': "o", 'Ø': "O", 'ł': "l", 'Ł': "L", 'đ': "d", 'Đ': "D",
	'ð': "d", 'Ð': "D", 'þ': "th", 'Þ': "Th", 'ħ': "h", 'Ħ': "H",
	'ŧ': "t", 'Ŧ': "T", 'ı': "i", 'İ': "I", 'ŋ': "n", 'Ŋ': "N",
	'ĸ': "k", 'ƒ': "f", 'ƿ': "w", 'ǝ': "e", 'ȝ': "g",
}

// dropRunes are runes with no sensible ASCII form: a word carrying one cannot be
// typed on a keyboard, so it is dropped from the word index.
var dropRunes = map[rune]bool{
	'ƛ': true, 'Ⅎ': true, 'ⅎ': true, 'Ɂ': true, 'ƹ': true,
}

// asciiFolder decomposes accented letters and strips the combining marks, so
// "café" folds to "cafe".
var asciiFolder = transform.Chain(norm.NFKD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// foldWord returns the lowercase, keyboard-typeable key of a word (accents
// stripped, case folded). ok is false when the word still holds a rune with no
// ASCII form, so it cannot be typed and must be dropped from the index.
func foldWord(word string) (key string, ok bool) {
	var b strings.Builder
	for _, r := range word {
		if dropRunes[r] {
			return "", false
		}
		if s, ok := manualFold[r]; ok {
			b.WriteString(s)
			continue
		}
		b.WriteRune(r)
	}
	res, _, err := transform.String(asciiFolder, b.String())
	if err != nil {
		return "", false
	}
	for _, r := range res {
		if r > unicode.MaxASCII {
			return "", false // a non-ASCII letter outside the maps: drop it
		}
	}
	return strings.ToLower(res), true
}

// sameWord reports whether two words are equal once folded to the typeable key,
// i.e. case- and accent-insensitively (café == Cafe).
func sameWord(a, b string) bool {
	fa, oka := foldWord(a)
	fb, okb := foldWord(b)
	return oka && okb && fa == fb
}
