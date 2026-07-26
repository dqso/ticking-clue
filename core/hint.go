package core

import (
	"image/color"
	"math"
	"math/rand/v2"
	"time"

	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Hint time costs, paid from the remaining time when a hint is bought. The
// letter hint has no fixed cost: it is priced dynamically, see letterHintCost.
// The word-length and arrow-colors hints have no time cost at all: they speed
// up the timer instead (see hintTimeScaleStep).
const (
	hintLinkCost = 45 * time.Second
	hintPosCost  = 1*time.Minute + 30*time.Second
)

// Hint cell sizes: the outer button and the baked icon+cost graphic inside it.
const (
	hintCellSize  = 104
	hintCellInner = 88
)

// hintKind selects which hint a cell triggers and which icon it shows.
type hintKind int

const (
	// hintLengthKind reveals the length of the hidden word (shown as boxes).
	// One-time: its button disappears once bought.
	hintLengthKind hintKind = iota
	// hintLinkKind reveals one random direct link of the hidden word. Its
	// button disappears once every link has been revealed.
	hintLinkKind
	// hintColorsKind colors every arrow by its relation type. One-time: its
	// button disappears once bought.
	hintColorsKind
	// hintPosKind reveals which parts of speech the hidden word can be. One-time:
	// its button disappears once bought, and it is available from the start.
	hintPosKind
	// hintLetterKind opens one random letter of the hidden word (every position
	// of that letter). Its button appears only after hintLengthKind is used,
	// and disappears once every letter has been opened.
	hintLetterKind
)

// Hint cell background colors (light squares on the notebook page).
var (
	hintCellIdle    = color.NRGBA{R: 0xea, G: 0xe8, B: 0xf0, A: 0xff}
	hintCellHover   = color.NRGBA{R: 0xf4, G: 0xf2, B: 0xfa, A: 0xff}
	hintCellPressed = color.NRGBA{R: 0xd6, G: 0xd2, B: 0xe2, A: 0xff}
)

// addTimeScale makes the timer drain faster by one step. Bought "speed" hints
// call this instead of paying a fixed time cost; the effect stacks.
func (r *round) addTimeScale() {
	r.timeScale *= hintTimeScaleStep
}

// useLengthHint reveals the length of the hidden word. It costs no fixed time:
// it speeds up the timer instead (see addTimeScale). Returns the hint's name
// for the Notes log, or "" when it is a no-op.
func (r *round) useLengthHint() string {
	if r.state != roundPlaying || r.lengthShown {
		return ""
	}
	r.lengthShown = true
	r.addTimeScale()
	return "word length"
}

// useColorsHint colors every arrow by its relation type. Like the length hint
// it costs no fixed time and speeds up the timer instead (see addTimeScale).
func (r *round) useColorsHint() string {
	if r.state != roundPlaying || r.colorsShown {
		return ""
	}
	r.colorsShown = true
	r.addTimeScale()
	return "arrow colors"
}

// usePosHint reveals which parts of speech the hidden word can be for a time
// cost and returns the first-person sentence and cost for the Notes log.
func (r *round) usePosHint() (string, time.Duration) {
	if r.state != roundPlaying || r.posShown {
		return "", 0
	}
	r.posShown = true
	r.applyDelta(-hintPosCost)
	return posSentence(r.hidden), hintPosCost
}

// letterCounts returns how many distinct letters the hidden word has and how
// many of them are still closed.
func (r *round) letterCounts() (total, closed int) {
	seen := map[rune]bool{}
	for _, ch := range r.hidden.Word {
		if isSeparator(ch) || seen[ch] {
			continue
		}
		seen[ch] = true
		total++
		if !r.revealedLetters[ch] {
			closed++
		}
	}
	return
}

// canOpenLetter reports whether the letter hint may still open a letter: some
// letter is closed and at most half of the distinct letters are open, so the
// player can never buy out most of the word.
func (r *round) canOpenLetter() bool {
	total, closed := r.letterCounts()
	opened := total - closed
	return closed > 0 && opened < total/2
}

// ceilMinute rounds a duration up to a whole minute (never below one minute).
func ceilMinute(d time.Duration) time.Duration {
	m := d / time.Minute
	if d%time.Minute != 0 {
		m++
	}
	if m < 1 {
		m = 1
	}
	return m * time.Minute
}

// letterHintCost is the price of opening the next letter. It rises with each
// letter already open (so the hint never only gets cheaper), and together with
// the half-word cap (see canOpenLetter) it keeps the word from being bought out:
//
//	cost = ceilMinute(remaining) * (opened + 1) / totalLetters
func (r *round) letterHintCost() time.Duration {
	total, closed := r.letterCounts()
	if total == 0 || closed == 0 {
		return 0
	}
	opened := total - closed
	return ceilMinute(r.remaining) * time.Duration(opened+1) / time.Duration(total)
}

// useLetterHint opens one random still-closed letter of the hidden word (all of
// its positions at once) for the dynamic cost and returns the hint's name and
// cost for the Notes log. It is only usable after the length hint and while the
// half-word cap is not reached.
func (r *round) useLetterHint() (string, time.Duration) {
	if r.state != roundPlaying || !r.lengthShown || !r.canOpenLetter() {
		return "", 0
	}
	seen := map[rune]bool{}
	var candidates []rune
	for _, ch := range r.hidden.Word {
		if isSeparator(ch) || r.revealedLetters[ch] || seen[ch] {
			continue
		}
		seen[ch] = true
		candidates = append(candidates, ch)
	}
	if len(candidates) == 0 {
		return "", 0
	}
	// Price the open before revealing (so "opened" excludes this letter).
	cost := r.letterHintCost()
	r.revealedLetters[candidates[rand.IntN(len(candidates))]] = true
	r.applyDelta(-cost)
	return "open a letter", cost
}

// hasUnrevealedLink reports whether at least one direct neighbor of the hidden
// word is still hidden, i.e. the link hint can still reveal something.
func (r *round) hasUnrevealedLink() bool {
	for _, n := range r.links {
		if _, ok := r.revealedSet[n.ID]; !ok {
			return true
		}
	}
	return false
}

// linkPreferredChance is how often the link hint reveals a neighbor of one of
// the player's enabled levels. The rest of the time it reveals a neighbor of a
// different level, so the map is not perfectly filtered. When only one of the
// two groups has words left, that group is used every time.
const linkPreferredChance = 0.90

// pickLink chooses the neighbor the link hint will reveal next among the still
// hidden ones. With linkPreferredChance it comes from the player's enabled
// levels, otherwise from the other levels; each group is used alone when the
// other is empty. Returns nil when every neighbor is already shown.
func (r *round) pickLink() *Node {
	var preferred, others []*Node
	for _, n := range r.links {
		if _, ok := r.revealedSet[n.ID]; ok {
			continue
		}
		if r.levels[n.MaxLevel()] {
			preferred = append(preferred, n)
		} else {
			others = append(others, n)
		}
	}
	pool := preferred
	switch {
	case len(preferred) == 0:
		pool = others
	case len(others) > 0 && rand.Float64() >= linkPreferredChance:
		pool = others
	}
	if len(pool) == 0 {
		return nil
	}
	return pool[rand.IntN(len(pool))]
}

// ensureNextLink keeps nextLink pointing at a still hidden neighbor, re-rolling
// it when it is missing or has since been revealed by a guess or another hint.
func (r *round) ensureNextLink() {
	if r.nextLink != nil {
		if _, shown := r.revealedSet[r.nextLink.ID]; !shown {
			return
		}
	}
	r.nextLink = r.pickLink()
}

// nextLinkLevel reports the CEFR level of the word the link hint will reveal
// next, so the hint cell can show it. ok is false when no neighbor is left.
func (r *round) nextLinkLevel() (Level, bool) {
	r.ensureNextLink()
	if r.nextLink == nil {
		return 0, false
	}
	return r.nextLink.MaxLevel(), true
}

// useLinkHint reveals the pre-rolled next neighbor (see pickLink) for a time
// cost and returns it so the scene can animate it, then pre-rolls the following
// one for the cell. When every neighbor is already shown it does nothing, costs
// nothing, and returns nil.
func (r *round) useLinkHint() *Node {
	if r.state != roundPlaying {
		return nil
	}
	r.ensureNextLink()
	pick := r.nextLink
	if pick == nil {
		return nil
	}
	r.applyDelta(-hintLinkCost)
	r.revealPath([]*Node{r.hidden, pick}, true)
	r.nextLink = r.pickLink() // pre-roll the next reveal for the cell icon
	return pick
}

// hintUseLength, hintUseLetter, hintUseColors and hintUsePos buy their hint and
// record it in the Notes log, stamped with the timecode as it was before the
// cost was paid.
func (s *GameScene) hintUseLength() {
	t := s.round.remaining
	name := s.round.useLengthHint()
	s.logSpeedHint(t, name)
}

func (s *GameScene) hintUseLetter() {
	t := s.round.remaining
	name, cost := s.round.useLetterHint()
	s.logHint(t, name, cost)
}

func (s *GameScene) hintUseColors() {
	t := s.round.remaining
	name := s.round.useColorsHint()
	s.logSpeedHint(t, name)
}

// hintUsePos buys the parts-of-speech hint and logs the parts it revealed with
// the cost, so the Notes actually show the answer (not just the hint name).
func (s *GameScene) hintUsePos() {
	t := s.round.remaining
	text, cost := s.round.usePosHint()
	if text == "" {
		return
	}
	s.logNote(t, text+" "+signedSeconds(-cost))
}

// hintUseLinkHint buys the "reveal a link" hint, flies the new cloud out from
// the center to its place, and logs it.
func (s *GameScene) hintUseLinkHint() {
	t := s.round.remaining
	if node := s.round.useLinkHint(); node != nil {
		s.startFlyer(guessOutcome{kind: guessReward, word: node.Word, node: node}, true)
		s.logHint(t, "linked word", hintLinkCost)
	}
}

// hintButton is one hint cell plus the test for whether its hint is currently
// available; refreshHints adds or removes the cell as that test changes.
type hintButton struct {
	widget    *widget.Button
	available func() bool
	present   bool
}

// hintUI holds the hint column and its cells so unusable ones can be removed.
// letterGraphic is the letter cell's image, redrawn when its dynamic cost
// changes; letterCost is the price it currently shows. linkGraphic is the link
// cell's image, redrawn when the next reveal's level (linkNote) changes.
type hintUI struct {
	column        *widget.Container
	buttons       []*hintButton
	letterGraphic *ebiten.Image
	letterCost    time.Duration
	linkGraphic   *ebiten.Image
	linkNote      string
}

// hintColumn builds the right-anchored column of hint cells and registers each
// with the test that keeps it on screen.
func (s *GameScene) hintColumn() *widget.Container {
	column := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(16),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(20)),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionEnd,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		})),
	)
	letterCost := s.round.letterHintCost()
	letterGraphic := makeHintCellGraphic(hintLetterKind, letterCost, "")
	linkNote := s.linkLevelNote()
	linkGraphic := makeHintCellGraphic(hintLinkKind, hintLinkCost, linkNote)
	// The length and colors hints show a speed multiplier, not a time cost, so
	// the cost argument is ignored for them (see drawHintCellGraphic).
	length := newHintCell(makeHintCellGraphic(hintLengthKind, 0, ""), s.hintUseLength)
	letter := newHintCell(letterGraphic, s.hintUseLetter)
	link := newHintCell(linkGraphic, s.hintUseLinkHint)
	colors := newHintCell(makeHintCellGraphic(hintColorsKind, 0, ""), s.hintUseColors)
	pos := newHintCell(makeHintCellGraphic(hintPosKind, hintPosCost, ""), s.hintUsePos)
	s.hints = hintUI{
		column:        column,
		letterGraphic: letterGraphic,
		letterCost:    letterCost,
		linkGraphic:   linkGraphic,
		linkNote:      linkNote,
		buttons: []*hintButton{
			// One-time hints go away once bought; the link hint once every
			// neighbor has been revealed; the letter hint appears only after the
			// length hint and goes away once every letter is open.
			{widget: length, available: func() bool { return !s.round.lengthShown }},
			{widget: letter, available: func() bool { return s.round.lengthShown && s.round.canOpenLetter() }},
			{widget: link, available: s.round.hasUnrevealedLink},
			{widget: colors, available: func() bool { return !s.round.colorsShown }},
			// Available from the start; goes away once bought.
			{widget: pos, available: func() bool { return !s.round.posShown }},
		},
	}
	s.refreshHints() // add the cells that are available from the start
	return column
}

// refreshHints keeps each hint cell in the column exactly while its hint is
// available: cells appear when they become usable and are removed once not.
func (s *GameScene) refreshHints() {
	for _, h := range s.hints.buttons {
		switch {
		case h.available() && !h.present:
			s.hints.column.AddChild(h.widget)
			h.present = true
		case !h.available() && h.present:
			s.hints.column.RemoveChild(h.widget)
			h.present = false
		}
	}
	s.updateLetterCost()
	s.updateLinkLevel()
}

// updateLetterCost re-bakes the letter cell's price when it changes. The cost
// only moves at a whole-minute boundary or when a letter is opened, so this
// redraws rarely.
func (s *GameScene) updateLetterCost() {
	if s.hints.letterGraphic == nil {
		return
	}
	cost := s.round.letterHintCost()
	if cost == s.hints.letterCost {
		return
	}
	s.hints.letterCost = cost
	drawHintCellGraphic(s.hints.letterGraphic, hintLetterKind, cost, "")
}

// linkLevelNote is the CEFR label of the word the link hint will reveal next,
// shown on the cell's cloud; empty when no neighbor is left.
func (s *GameScene) linkLevelNote() string {
	if lvl, ok := s.round.nextLinkLevel(); ok {
		return levelLabels[lvl]
	}
	return ""
}

// updateLinkLevel re-bakes the link cell's cloud when the next reveal's level
// changes (after a link is bought, or after the pre-rolled word was guessed and
// re-rolled). It redraws only on an actual change.
func (s *GameScene) updateLinkLevel() {
	if s.hints.linkGraphic == nil {
		return
	}
	note := s.linkLevelNote()
	if note == s.hints.linkNote {
		return
	}
	s.hints.linkNote = note
	drawHintCellGraphic(s.hints.linkGraphic, hintLinkKind, hintLinkCost, note)
}

// newHintCell builds one hint button from a baked graphic, calling onClick when
// pressed. The graphic is kept by the button, so redrawing its contents (for a
// dynamic cost) updates the cell in place.
func newHintCell(graphic *ebiten.Image, onClick func()) *widget.Button {
	return widget.NewButton(
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(hintCellSize, hintCellSize)),
		widget.ButtonOpts.Image(&widget.ButtonImage{
			Idle:    eimage.NewNineSliceColor(hintCellIdle),
			Hover:   eimage.NewNineSliceColor(hintCellHover),
			Pressed: eimage.NewNineSliceColor(hintCellPressed),
		}),
		widget.ButtonOpts.Graphic(&widget.GraphicImage{Idle: graphic}),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			onClick()
		}),
	)
}

// makeHintCellGraphic returns a new image with the hint icon and cost baked in.
// note is only used by the link cell, where it is the next reveal's level.
func makeHintCellGraphic(kind hintKind, cost time.Duration, note string) *ebiten.Image {
	img := ebiten.NewImage(hintCellInner, hintCellInner)
	drawHintCellGraphic(img, kind, cost, note)
	return img
}

// drawHintCellGraphic (re)draws the icon and cost onto img, so a dynamic cost
// (or the link cell's level note) can be refreshed in place without allocating
// a new image.
func drawHintCellGraphic(img *ebiten.Image, kind hintKind, cost time.Duration, note string) {
	img.Clear()
	switch kind {
	case hintLengthKind:
		drawLengthIcon(img)
	case hintLetterKind:
		drawLetterIcon(img)
	case hintLinkKind:
		drawLinkIcon(img, note)
	case hintColorsKind:
		drawColorsIcon(img)
	case hintPosKind:
		drawPosIcon(img)
	}
	face := newFace(16)
	// Speed hints (length, colors) do not cost seconds: they speed up the timer.
	// Their price is a fast-forward glyph plus the stacking multiplier (e.g.
	// ">>×1.5"), drawn in the debuff color like the HUD speed indicator, so it
	// reads as "time runs faster" rather than a time cost.
	if kind == hintLengthKind || kind == hintColorsKind {
		txt := "×" + formatScale(hintTimeScaleStep)
		const iconW, iconH, gap = 13.0, 12.0, 4.0
		tw, _ := text.Measure(txt, face, 0)
		y := float64(hintCellInner) - 24
		left := float64(hintCellInner)/2 - (iconW+gap+tw)/2
		drawFastForward(img, left, y, iconW, iconH, penaltyColor)
		drawTextLeft(img, txt, face, left+iconW+gap, y-2, penaltyColor)
		return
	}
	drawTextCentered(img, "-"+formatMMSS(cost), face,
		float64(hintCellInner)/2, float64(hintCellInner)-13, hintCostColor)
}

// drawLengthIcon draws three adjacent cells (black outline, transparent
// inside) standing for the word length hint.
func drawLengthIcon(dst *ebiten.Image) {
	const sq, stroke = 18.0, 2.0
	x, y := (float64(hintCellInner)-3*sq)/2, 22.0
	for range 3 {
		vector.StrokeRect(dst, fx(x), fx(y), sq, sq, stroke, hintIconColor, false)
		x += sq
	}
}

// drawLetterIcon draws three cells with a letter opened inside the two right
// ones ([ ][e][e]), standing for the "open a letter" hint.
func drawLetterIcon(dst *ebiten.Image) {
	const sq, stroke = 18.0, 2.0
	x, y := (float64(hintCellInner)-3*sq)/2, 22.0
	letterFace := newFace(sq * 0.8)
	for i := range 3 {
		vector.StrokeRect(dst, fx(x), fx(y), sq, sq, stroke, hintIconColor, false)
		if i >= 1 {
			drawTextCentered(dst, "e", letterFace, x+sq/2, y+sq/2, hintIconColor)
		}
		x += sq
	}
}

// drawLinkIcon draws a small cloud with the level of the word that the hint
// will reveal next written inside it (note is empty when no neighbor is left).
func drawLinkIcon(dst *ebiten.Image, note string) {
	cx, cy := float64(hintCellInner)/2, 30.0
	drawCloudShape(dst, cx, cy, 20, 12, 4, 101)
	if note != "" {
		drawTextCentered(dst, note, newFace(13), cx, cy, cloudTextColor)
	}
}

// drawPosIcon draws the two-line label "verb or / ...", standing for the
// "which parts of speech" hint.
func drawPosIcon(dst *ebiten.Image) {
	drawCenteredLines(dst, []string{"verb or", "..."}, newFace(16),
		float64(hintCellInner)/2, 30, hintIconColor)
}

// drawColorsIcon draws two arrows (green and red) leaving one point at the
// bottom for two small clouds on the left and right.
func drawColorsIcon(dst *ebiten.Image) {
	inner := float64(hintCellInner)
	ox, oy := inner/2, inner-24
	drawColorsArrow(dst, ox, oy, 20, 26, 102, synonymColor)
	drawColorsArrow(dst, ox, oy, inner-20, 26, 103, antonymColor)
}

// drawColorsArrow draws a small cloud at (cx, cy) and an arrow from (ox, oy)
// stopping right at the cloud outline.
func drawColorsArrow(dst *ebiten.Image, ox, oy, cx, cy float64, seed int64, clr color.Color) {
	const shx, shy = 16.0, 10.0
	drawCloudShape(dst, cx, cy, shx, shy, 3, seed)
	dx, dy := cx-ox, cy-oy
	d := math.Hypot(dx, dy)
	if d < 1 {
		return
	}
	// Stop the arrow right at the small cloud's oval outline.
	e := ellipseEdge(shx, shy, dx/d, dy/d)
	hx, hy := cx-dx/d*e, cy-dy/d*e
	drawArrow(dst, ox, oy, hx, hy, clr, 1)
}
