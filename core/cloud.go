package core

import (
	"image/color"
	"math"
	"math/rand/v2"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// A cloud wraps its word as a flat oval. cloudPadYRatio is the top and
	// bottom padding, cloudPadXRatio the extra side padding, both measured in
	// units of the word height so the padding scales with the font and zoom.
	cloudPadYRatio = 0.5
	cloudPadXRatio = 0.3
	// wrapMinLineChars is how many characters a cloud line must already hold
	// before a separator (space or hyphen) is allowed to break it.
	wrapMinLineChars = 10
	// centralMinHalfWidth / linkMinHalfWidth keep short words inside readable
	// clouds by giving the cloud a minimum half-width.
	centralMinHalfWidth = 60.0
	linkMinHalfWidth    = 40.0

	// Cloud outline thickness: cloudStrokeRatio of the cloud half-height,
	// clamped to the [cloudStrokeMin, cloudStrokeMax] pixel range.
	cloudStrokeRatio = 0.07
	cloudStrokeMin   = 3.0
	cloudStrokeMax   = 8.0
)

// cloudSize is the half-width (hx) and half-height (hy) of a cloud in pixels.
// A cloud is a flat oval so it follows the shape of a one-line word.
type cloudSize struct{ hx, hy float64 }

// ellipseEdge returns the distance from the center of an ellipse with half
// axes (hx, hy) to its outline along the unit direction (ux, uy). It is used
// to start and stop arrows right at the cloud outline in any direction.
func ellipseEdge(hx, hy, ux, uy float64) float64 {
	dx, dy := ux/hx, uy/hy
	d := math.Hypot(dx, dy)
	if d < 1e-9 {
		return hx
	}
	return 1 / d
}

// centralMasked reports that the central cloud shows the hidden word as boxes
// (one per letter): the letter-count hint is bought but the round is not won.
func centralMasked(r *round) bool {
	return r.state != roundWon && r.lengthShown
}

// centralLines are the display lines of the central plain-text cloud: the real
// word once won, or a single question mark while the letter count is unknown.
// The masked (boxes) state is handled separately, see drawMaskedWord.
func centralLines(r *round) []string {
	if r.state == roundWon {
		return wrapWord(r.hidden.Word)
	}
	return []string{"?"}
}

// wrapWord splits a lemma into cloud lines at its separators (space, hyphen).
// A separator only breaks the line once the line already holds at least
// wrapMinLineChars characters; otherwise it stays inline (a space as a space, a
// hyphen as a hyphen). When a line breaks, a space is dropped but a hyphen is
// kept at the end of the line. Spaces have priority over hyphens: a hyphen break
// is deferred when a space still comes before the next hyphen further along.
func wrapWord(word string) []string {
	type tok struct {
		text string
		sep  rune // ' ', '-', or 0 for the last token
	}
	var toks []tok
	var b strings.Builder
	for _, ch := range word {
		if ch == ' ' || ch == '-' {
			toks = append(toks, tok{b.String(), ch})
			b.Reset()
			continue
		}
		b.WriteRune(ch)
	}
	toks = append(toks, tok{b.String(), 0})

	// spaceBeforeHyphen reports whether a space separator appears before the
	// next hyphen when scanning the tokens from index `from`.
	spaceBeforeHyphen := func(from int) bool {
		for _, t := range toks[from:] {
			switch t.sep {
			case ' ':
				return true
			case '-':
				return false
			}
		}
		return false
	}

	var lines []string
	var cur strings.Builder
	for i, t := range toks {
		cur.WriteString(t.text)
		if t.sep == '-' {
			cur.WriteRune('-') // a hyphen always stays on its line
		}
		if t.sep == 0 {
			break
		}
		if len([]rune(cur.String())) < wrapMinLineChars {
			if t.sep == ' ' {
				cur.WriteRune(' ') // keep the space inline
			}
			continue
		}
		// The line is long enough to break here, but prefer a space: defer a
		// hyphen break when a space still comes before the next hyphen.
		if t.sep == '-' && spaceBeforeHyphen(i+1) {
			continue
		}
		lines = append(lines, cur.String())
		cur.Reset()
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	if len(lines) == 0 {
		return []string{word}
	}
	return lines
}

// faceLineSpacing is the baseline-to-baseline distance of a face, used both to
// measure and to draw multi-line cloud text.
func faceLineSpacing(face text.Face) float64 {
	m := face.Metrics()
	return m.HAscent + m.HDescent + m.HLineGap
}

// cloudSizeFromBox turns a text box (w, h) into a cloud half-size, adding
// comfortable padding on every side. The padding is measured in lineHeight, so
// it stays even no matter how many lines there are; the half-width is never
// smaller than minHalfWidth.
func cloudSizeFromBox(w, h, lineHeight, minHalfWidth float64) cloudSize {
	hy := h/2 + lineHeight*cloudPadYRatio
	// capRoom is the least horizontal space so the text corner (w/2, h/2) stays
	// inside the rounded end cap of radius hy. It is much smaller than hy, so
	// the side padding is driven by cloudPadXRatio, not by the cap.
	capRoom := hy - math.Sqrt(math.Max(0, hy*hy-(h/2)*(h/2)))
	hx := w/2 + capRoom + lineHeight*cloudPadXRatio
	return cloudSize{hx: math.Max(hx, minHalfWidth), hy: hy}
}

// cloudSizeFor returns the cloud half-size that wraps the given text lines.
func cloudSizeFor(lines []string, face text.Face, minHalfWidth float64) cloudSize {
	ls := faceLineSpacing(face)
	w, h := text.Measure(strings.Join(lines, "\n"), face, ls)
	return cloudSizeFromBox(w, h, ls, minHalfWidth)
}

// cloudStrokeWidth is the outline thickness for a cloud of the given half
// height, clamped so it stays thin on small clouds and readable on big ones.
func cloudStrokeWidth(hy float64) float64 {
	return math.Max(cloudStrokeMin, math.Min(cloudStrokeMax, hy*cloudStrokeRatio))
}

// stadiumOutline maps an arc-length position s along the outline of a stadium
// (a rectangle 2*span wide with semicircular caps of radius hy) to a point and
// its outward unit normal. The bumps of a cloud are scattered along this
// outline, so every side of the cloud gets its own random puffs.
func stadiumOutline(s, span, hy float64) (x, y, nx, ny float64) {
	edge := 2 * span    // length of the top (or bottom) straight edge
	arc := math.Pi * hy // length of one semicircular cap
	switch {
	case s < edge: // top edge, left to right
		return -span + s, hy, 0, 1
	case s < edge+arc: // right cap, top to bottom
		ang := math.Pi/2 - (s-edge)/hy
		nx, ny = math.Cos(ang), math.Sin(ang)
		return span + hy*nx, hy * ny, nx, ny
	case s < 2*edge+arc: // bottom edge, right to left
		return span - (s - edge - arc), -hy, 0, -1
	default: // left cap, bottom to top
		ang := -math.Pi/2 - (s-2*edge-arc)/hy
		nx, ny = math.Cos(ang), math.Sin(ang)
		return -span + hy*nx, hy * ny, nx, ny
	}
}

// cloudPuff is one circle of a cloud: its center offset from the cloud center
// and its radius.
type cloudPuff struct{ dx, dy, r float64 }

// cloudPuffs builds the circles that make up a cloud: a row of circles along the
// x axis (a "stadium") fills the wide interior, and random puffs scattered along
// the whole outline form the lumpy, irregular edge. The seed keeps the puffs
// stable between frames but different from cloud to cloud. Both the drawing and
// the outline path (cloudOutline) use this, so what is drawn and where arrows
// stop always agree.
func cloudPuffs(hx, hy float64, seed int64) []cloudPuff {
	rng := rand.New(rand.NewPCG(uint64(seed)+1, 0x9e3779b97f4a7c15))
	var puffs []cloudPuff
	// Stadium core: circles of radius hy stepping along x fill the interior
	// with a flat oval span the full height of the cloud.
	span := math.Max(0, hx-hy)
	steps := int(span/(hy*0.5)) + 1
	for i := 0; i <= steps; i++ {
		t := -span + 2*span*float64(i)/float64(steps)
		puffs = append(puffs, cloudPuff{t, 0, hy})
	}
	// Outline puffs: walk the whole stadium outline at a roughly constant
	// spacing (so wider clouds get more puffs) and drop a circle at each step.
	// The position jitters along the outline and each puff sits partly inside
	// the body, so it bulges out by a random amount for a lumpy silhouette.
	perim := 2*(2*span) + 2*math.Pi*hy
	nb := int(perim / (hy * 0.85))
	if nb < 8 {
		nb = 8
	}
	for i := 0; i < nb; i++ {
		// Even step plus a strong jitter so puffs never look lined up.
		u := (float64(i) + rng.Float64()*0.9 - 0.45) / float64(nb)
		u -= math.Floor(u) // wrap into [0, 1)
		bx, by, nx, ny := stadiumOutline(u*perim, span, hy)
		r := hy * (0.5 + rng.Float64()*0.7)      // 0.5..1.2 of the height
		inset := hy * (0.25 + rng.Float64()*0.3) // pull the center into the body
		puffs = append(puffs, cloudPuff{bx - nx*inset, by - ny*inset, r})
	}
	return puffs
}

// drawCloudShape draws a flat, word-shaped cloud from its puffs. Every circle is
// drawn once in the outline color, then a smaller one in the interior color,
// leaving a ring of width strokeW.
func drawCloudShape(dst *ebiten.Image, cx, cy, hx, hy, strokeW float64, seed int64) {
	puffs := cloudPuffs(hx, hy, seed)
	for _, p := range puffs {
		vector.FillCircle(dst, fx(cx+p.dx), fx(cy+p.dy), fx(p.r), cloudStroke, true)
	}
	for _, p := range puffs {
		vector.FillCircle(dst, fx(cx+p.dx), fx(cy+p.dy), fx(p.r-strokeW), cloudInterior, true)
	}
}

// cloudOutlineSamples is how many directions trace a cloud's rough outline path.
const cloudOutlineSamples = 48

// puffsReach returns how far the puff silhouette extends from the cloud center
// along the unit direction (ux, uy): the point where a ray from the center
// leaves the union of the puffs. Used only to trace the outline path.
func puffsReach(puffs []cloudPuff, ux, uy float64) float64 {
	best := 0.0
	for _, p := range puffs {
		cd := p.dx*ux + p.dy*uy                // puff center projected on the ray
		perp2 := p.dx*p.dx + p.dy*p.dy - cd*cd // squared distance from ray to center
		disc := p.r*p.r - perp2
		if disc <= 0 {
			continue // the ray misses this puff
		}
		if t := cd + math.Sqrt(disc); t > best {
			best = t
		}
	}
	return best
}

// cloudOutline traces a rough closed outline path of a cloud: at evenly spaced
// directions it samples how far the puffs reach, giving a lumpy polygon that
// follows the cloud's silhouette. Arrows stop where they cross this path, so
// their ends sit just outside the cloud instead of vanishing under a puff.
// Points are in cloud-local coords (relative to the cloud center).
func cloudOutline(hx, hy float64, seed int64) [][2]float64 {
	puffs := cloudPuffs(hx, hy, seed)
	pts := make([][2]float64, cloudOutlineSamples)
	for k := range pts {
		a := 2 * math.Pi * float64(k) / float64(cloudOutlineSamples)
		ux, uy := math.Cos(a), math.Sin(a)
		d := puffsReach(puffs, ux, uy)
		pts[k] = [2]float64{d * ux, d * uy}
	}
	return pts
}

// cloudRayDist returns the distance from the cloud center to where a ray leaving
// it along (ux, uy) crosses the outline path. When several edges are hit (a
// concave dip between puffs), the farthest crossing is kept so the arrow clears
// the whole silhouette.
func cloudRayDist(outline [][2]float64, ux, uy float64) float64 {
	best := 0.0
	n := len(outline)
	for i := 0; i < n; i++ {
		ax, ay := outline[i][0], outline[i][1]
		bx, by := outline[(i+1)%n][0], outline[(i+1)%n][1]
		ex, ey := bx-ax, by-ay
		det := ex*uy - ey*ux
		if math.Abs(det) < 1e-9 {
			continue // ray parallel to this edge
		}
		// Solve center + t*(ux,uy) = A + s*(B-A) for t >= 0 and s in [0, 1].
		t := (ex*ay - ey*ax) / det
		s := (ux*ay - uy*ax) / det
		if t >= 0 && s >= 0 && s <= 1 && t > best {
			best = t
		}
	}
	return best
}

// drawCloudText draws the given lines centered at (cx, cy), one under another.
func drawCloudText(dst *ebiten.Image, lines []string, face text.Face, cx, cy float64, clr color.Color) {
	joined := strings.Join(lines, "\n")
	ls := faceLineSpacing(face)
	_, h := text.Measure(joined, face, ls)
	op := &text.DrawOptions{}
	op.GeoM.Translate(cx, cy-h/2)
	op.LineSpacing = ls
	op.PrimaryAlign = text.AlignCenter // center every line horizontally
	op.ColorScale.ScaleWithColor(clr)
	op.Filter = ebiten.FilterLinear
	text.Draw(dst, joined, face, op)
}

const (
	// maskBoxRatio is a letter box side as a fraction of the line spacing.
	maskBoxRatio = 0.62
	// maskLineGapRatio is the gap between masked lines, in box-side units.
	maskLineGapRatio = 0.45
	// maskStrokeRatio is the box outline thickness, in box-side units.
	maskStrokeRatio = 0.12
	// maskSpaceFactor widens a space between boxed letters so words separate
	// clearly (the font's own space is a bit tight here).
	maskSpaceFactor = 2.0
)

// isSeparator reports whether ch is a cloud line separator (space or hyphen).
func isSeparator(ch rune) bool { return ch == ' ' || ch == '-' }

// separatorAdvance is the horizontal room a separator takes between boxes: its
// font advance, but a space is widened by maskSpaceFactor.
func separatorAdvance(ch rune, face text.Face) float64 {
	adv, _ := text.Measure(string(ch), face, 0)
	if ch == ' ' {
		adv *= maskSpaceFactor
	}
	return adv
}

// maskedLineWidth is the pixel width of one masked line: a box per letter and
// the separator advance for each separator.
func maskedLineWidth(line string, face text.Face, box float64) float64 {
	w := 0.0
	for _, ch := range line {
		if isSeparator(ch) {
			w += separatorAdvance(ch, face)
			continue
		}
		w += box
	}
	return w
}

// maskedCloudSize is the cloud half-size for the hidden word shown as boxes.
// The lines come from the same wrapWord as the plain word, so the shape matches.
func maskedCloudSize(lines []string, face text.Face, minHalfWidth float64) cloudSize {
	box := faceLineSpacing(face) * maskBoxRatio
	gap := box * maskLineGapRatio
	maxW := 0.0
	for _, ln := range lines {
		if w := maskedLineWidth(ln, face, box); w > maxW {
			maxW = w
		}
	}
	n := float64(len(lines))
	h := n*box + (n-1)*gap
	return cloudSizeFromBox(maxW, h, box+gap, minHalfWidth)
}

// drawMaskedWord draws the hidden word as outlined boxes (one per letter),
// centered at (cx, cy). A letter present in revealed is drawn inside its box;
// separators keep their own glyph between the boxes.
func drawMaskedWord(dst *ebiten.Image, lines []string, face text.Face, cx, cy float64, clr color.Color, revealed map[rune]bool) {
	box := faceLineSpacing(face) * maskBoxRatio
	gap := box * maskLineGapRatio
	stroke := math.Max(1.5, box*maskStrokeRatio)
	letterFace := newFace(box * 0.8) // revealed letters fit inside the box
	n := float64(len(lines))
	y := cy - (n*box+(n-1)*gap)/2 // top of the block
	for _, ln := range lines {
		x := cx - maskedLineWidth(ln, face, box)/2
		yc := y + box/2 // vertical center of this line
		for _, ch := range ln {
			if isSeparator(ch) {
				adv := separatorAdvance(ch, face)
				if ch == '-' {
					drawTextCentered(dst, "-", face, x+adv/2, yc, clr)
				}
				x += adv
				continue
			}
			vector.StrokeRect(dst, fx(x), fx(yc-box/2), fx(box), fx(box), fx(stroke), clr, true)
			if revealed[ch] {
				drawTextCentered(dst, string(ch), letterFace, x+box/2, yc, clr)
			}
			x += box
		}
		y += box + gap
	}
}

// hiddenStopwords are tokens left visible inside hint clouds: common particles
// and prepositions that would not give the hidden word away.
var hiddenStopwords = map[string]struct{}{
	"back": {}, "out": {}, "up": {}, "in": {}, "down": {},
	"across": {}, "on": {}, "along": {}, "away": {}, "over": {},
	"for": {}, "with": {}, "to": {}, "of": {}, "off": {},
	"after": {}, "about": {}, "through": {}, "the": {}, "by": {},
	"aside": {},
}

// isTokenSep reports a token separator (space or hyphen).
func isTokenSep(r rune) bool { return r == ' ' || r == '-' }

// significantTokens splits a word into tokens (by space or hyphen), drops the
// stopwords, and returns the rest as a set. These are the tokens that must be
// masked when they show up inside a hint cloud.
func significantTokens(word string) map[string]struct{} {
	res := map[string]struct{}{}
	for _, tok := range strings.FieldsFunc(word, isTokenSep) {
		tok = strings.ToLower(tok)
		if _, stop := hiddenStopwords[tok]; !stop {
			res[tok] = struct{}{}
		}
	}
	return res
}

// lineToken is one token of a wrapped line plus the separator that follows it
// (' ', '-', or 0 at the end of the line).
type lineToken struct {
	text string
	sep  rune
}

// tokenizeLine splits a single wrapped line into its tokens and separators.
func tokenizeLine(line string) []lineToken {
	var toks []lineToken
	var b strings.Builder
	for _, ch := range line {
		if isTokenSep(ch) {
			toks = append(toks, lineToken{b.String(), ch})
			b.Reset()
			continue
		}
		b.WriteRune(ch)
	}
	return append(toks, lineToken{b.String(), 0})
}

// maskWord replaces every masked token (a significant token of the hidden word)
// with a single "<?>", keeping the separators; other tokens stay as they are.
func maskWord(word string, mask map[string]struct{}) string {
	var b strings.Builder
	for _, tk := range tokenizeLine(word) {
		if _, masked := mask[strings.ToLower(tk.text)]; masked {
			b.WriteString("<?>")
		} else {
			b.WriteString(tk.text)
		}
		if tk.sep != 0 {
			b.WriteRune(tk.sep)
		}
	}
	return b.String()
}

// drawCloudWord draws a hint cloud's word centered at (cx, cy), with every token
// that also appears in the hidden word replaced by "?".
func drawCloudWord(dst *ebiten.Image, word string, face text.Face, cx, cy float64, clr color.Color, mask map[string]struct{}) {
	drawCloudText(dst, wrapWord(maskWord(word, mask)), face, cx, cy, clr)
}
