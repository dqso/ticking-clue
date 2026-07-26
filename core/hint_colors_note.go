package core

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	pb "github.com/dqso/ticking-clue/proto/gen"
)

const (
	// colorsNoteIconSize is the collapsed square side (in zoom-1 world units, so
	// it pans and zooms with the clouds, like the miss-list sticker).
	colorsNoteIconSize = 64.0
	// colorsNoteExpandW / colorsNoteExpandH are the expanded sticker size, which
	// holds the arrow legend. The sticker grows right and down from its corner.
	colorsNoteExpandW = 520.0
	colorsNoteExpandH = 360.0
)

// colorsNote is the white sticker handed to the player when the "reveal colors"
// hint is bought. It lives on the sheet in world coords (tx, ty is its top-left
// corner relative to the graph center, in zoom-1 units, so it pans and zooms).
// It can be dragged, and a click (no drag) opens it into a bigger sticker with
// the arrow legend.
type colorsNote struct {
	active   bool // becomes true once the colors hint is bought
	placed   bool // got its initial spot on the sheet
	expanded bool // opened into the legend sticker
	tx, ty   float64

	// Drag bookkeeping, mirroring the miss-list sticker.
	dragging       bool
	moved          bool
	grabDX, grabDY float64
	pressX, pressY int
	// captured marks that the current press is handled by this note, so the
	// miss list and the graph pan/drag both ignore it.
	captured bool

	// legendImg caches the expanded legend rendered once at 1:1. Drawing this
	// picture instead of rebuilding the tiny text every frame avoids the
	// glyph-atlas bleeding that shows up as jpeg-like fringes under zoom.
	legendImg *ebiten.Image
}

// noteSize is the sticker's world size (zoom-1 units), larger while expanded.
func (s *GameScene) noteSize() (float64, float64) {
	if s.colorsNote.expanded {
		return colorsNoteExpandW, colorsNoteExpandH
	}
	return colorsNoteIconSize, colorsNoteIconSize
}

// noteRect is the sticker's screen rectangle (top-left x, y and size), its world
// corner projected through the current pan and zoom.
func (s *GameScene) noteRect(l gameLayout) (x, y, w, h float64) {
	x = l.cx + s.colorsNote.tx*l.zoom
	y = l.cy + s.colorsNote.ty*l.zoom
	if s.colorsNote.expanded {
		// The opened sheet keeps a fixed on-screen size, so it never shrinks and
		// its picture stays sharp regardless of zoom.
		return x, y, colorsNoteExpandW, colorsNoteExpandH
	}
	nw, nh := s.noteSize()
	return x, y, nw * l.zoom, nh * l.zoom
}

// inNote reports whether (mx, my) is over the sticker.
func (s *GameScene) inNote(mx, my int, l gameLayout) bool {
	x, y, w, h := s.noteRect(l)
	return float64(mx) >= x && float64(mx) <= x+w && float64(my) >= y && float64(my) <= y+h
}

// placeColorsNote puts the sticker on the sheet the first time it appears: near
// the top-left corner (clear of the timer, hint column, and zoom controls),
// converted to world coords so it pans and zooms.
func (s *GameScene) placeColorsNote(l gameLayout) {
	s.colorsNote.tx = (40 - l.cx) / l.zoom
	s.colorsNote.ty = (90 - l.cy) / l.zoom
	s.colorsNote.placed = true
}

// updateColorsNote activates the sticker when the colors hint is bought, then
// drives its drag and the click that opens or closes it. It sets captured so
// the miss list and the graph ignore that press.
func (s *GameScene) updateColorsNote(g *Game) {
	n := &s.colorsNote
	if s.round.colorsShown && !n.active {
		n.active = true
	}
	if !n.active {
		return
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		n.dragging = false
		n.captured = false
	}
	l := computeLayout(g.screenWidth, g.screenHeight, s.panX, s.panY, s.zoom)
	if !n.placed {
		s.placeColorsNote(l)
	}
	mx, my := ebiten.CursorPosition()
	justPressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	justReleased := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)

	// Skip if the miss list already grabbed this press: it draws on top, so it
	// keeps the pointer over any overlap.
	if justPressed && s.inNote(mx, my, l) && !s.fx.captured {
		x, y, _, _ := s.noteRect(l)
		n.dragging, n.captured, n.moved = true, true, false
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
	if justReleased && !n.moved && s.inNote(mx, my, l) {
		n.expanded = !n.expanded // a click (no drag) opens or closes the sticker
	}
}

// drawColorsNote draws the white sticker: the crossed-arrows icon while
// collapsed, or the relation legend while expanded.
func (s *GameScene) drawColorsNote(screen *ebiten.Image, l gameLayout) {
	n := &s.colorsNote
	if !n.active {
		return
	}
	x, y, w, h := s.noteRect(l)
	if n.expanded {
		// Blit the cached legend 1:1 at an integer position (no scaling), so the
		// picture is never resampled and stays crisp at any zoom.
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(math.Round(x), math.Round(y))
		screen.DrawImage(s.legendImage(), op)
		return
	}
	vector.FillRect(screen, fx(x), fx(y), fx(w), fx(h), colorsNoteColor, false)
	vector.StrokeRect(screen, fx(x), fx(y), fx(w), fx(h), fx(2*l.zoom), colorsNoteEdge, true)
	drawColorsNoteIcon(screen, x+w/2, y+h/2, w, l.zoom)
}

// legendImage lazily renders the expanded legend once into an offscreen image
// at its natural size, then hands back the cached copy. Rendering the sticker,
// its border and all the small labels a single time (instead of every frame at
// a zoomed, fractional size) keeps the drawing free of the atlas fringes.
func (s *GameScene) legendImage() *ebiten.Image {
	if s.colorsNote.legendImg != nil {
		return s.colorsNote.legendImg
	}
	w, h := colorsNoteExpandW, colorsNoteExpandH
	img := ebiten.NewImage(int(w), int(h))
	vector.FillRect(img, 0, 0, fx(w), fx(h), colorsNoteColor, false)
	// Inset the border by one pixel so the whole stroke stays inside the image.
	vector.StrokeRect(img, 1, 1, fx(w-2), fx(h-2), 2, colorsNoteEdge, true)
	drawColorsNoteLegend(img, 0, 0, w, h, 1)
	s.colorsNote.legendImg = img
	return img
}

// drawColorsNoteIcon draws a green and a red arrow crossing at the center, so
// the collapsed sticker reads as a crosshair of two arrows.
func drawColorsNoteIcon(dst *ebiten.Image, cx, cy, side, zoom float64) {
	p := side * 0.24
	half := side / 2
	x0, x1 := cx-half+p, cx+half-p
	y0, y1 := cy-half+p, cy+half-p
	drawArrow(dst, x0, y1, x1, y0, synonymColor, zoom) // green: bottom-left -> top-right
	drawArrow(dst, x0, y0, x1, y1, antonymColor, zoom) // red: top-left -> bottom-right
}

// colorsLegendWords are the example words of the legend, placed by normalized
// coordinates (0..1) inside the sticker's inner area. "\n" splits a two-line
// label.
var colorsLegendWords = []struct {
	key    string
	label  string
	nx, ny float64
}{
	{"good", "good", 0.30, 0.14},
	{"well", "well", 0.52, 0.14},
	{"bad", "bad", 0.42, 0.30},
	{"walls", "walls", 0.12, 0.31},
	{"room", "room", 0.13, 0.58},
	{"house", "house", 0.45, 0.58},
	{"dirty", "dirty", 0.63, 0.39},
	{"techno", "techno", 0.78, 0.39},
	{"housekeeper", "housekeeper", 0.71, 0.62},
	{"rat", "rat", 0.40, 0.72},
	{"cactusmouse", "cactus\nmouse", 0.50, 0.71},
	{"animal", "animal", 0.65, 0.74},
	{"vole", "vole", 0.11, 0.84},
	{"mouse", "mouse", 0.39, 0.86},
}

// colorsLegendRels are the example relations, each drawn from its parent word to
// its child word with the styled arrow of its type.
var colorsLegendRels = []struct {
	from, to string
	edge     pb.EdgeType
}{
	{"good", "well", pb.EdgeType_SYNONYM},
	{"good", "bad", pb.EdgeType_ANTONYM},
	{"well", "bad", pb.EdgeType_ANTONYM},
	{"mouse", "animal", pb.EdgeType_HYPERNYM},
	{"mouse", "cactusmouse", pb.EdgeType_HYPONYM},
	{"room", "house", pb.EdgeType_HOLONYM},
	{"room", "walls", pb.EdgeType_MERONYM},
	{"mouse", "vole", pb.EdgeType_COORDINATE_TERM},
	{"mouse", "rat", pb.EdgeType_COORDINATE_TERM},
	{"rat", "vole", pb.EdgeType_COORDINATE_TERM},
	{"house", "housekeeper", pb.EdgeType_DERIVED_TO},
	{"house", "techno", pb.EdgeType_RELATED},
	{"bad", "dirty", pb.EdgeType_RELATED},
}

// colorsLegendNotes are the short colored explanations, colored like the
// relation they describe.
var colorsLegendNotes = []struct {
	text   string
	nx, ny float64
	clr    color.NRGBA
}{
	{"Synonyms have a similar meaning.", 0.44, 0.05, synonymColor},
	{"Antonyms are\nthe opposite.", 0.27, 0.24, antonymColor},
	{"Related words are connected in meaning,\nbut not opposite or the same.", 0.76, 0.28, arrowColor},
	{"Walls are a part (meronym) of a room.", 0.36, 0.41, meronymyColor},
	{"And a house is the whole (holonym),\nand a room is a part of it.", 0.37, 0.48, meronymyColor},
	{"Derived words are new words\nformed from old ones.", 0.80, 0.54, derivedColor},
	{"Coordinate terms are\ndifferent words\nat the same level.", 0.24, 0.67, coordColor},
	{"A hyponym is a specific type of something.\nA hypernym is the general group.", 0.74, 0.90, hyponymyColor},
}

// wordAnchor is a legend word's screen center and half extents, used to stop
// arrows right at its text box.
type wordAnchor struct{ cx, cy, hw, hh float64 }

// drawColorsNoteLegend draws the relation diagram inside the expanded sticker:
// example words, the styled colored arrow between each pair, and the short
// colored explanations. Everything is placed by normalized coordinates, so it
// scales with the sticker (and thus with zoom).
func drawColorsNoteLegend(screen *ebiten.Image, x, y, w, h, zoom float64) {
	pad := 10 * zoom
	ix, iy, iw, ih := x+pad, y+pad, w-2*pad, h-2*pad
	at := func(nx, ny float64) (float64, float64) { return ix + nx*iw, iy + ny*ih }
	wordFace := newFace(math.Max(9, 0.050*ih))
	noteFace := newFace(math.Max(8, 0.036*ih))
	arrowW := math.Max(0.6, 0.7*zoom)

	// Measure each word so arrows can stop at its text box.
	anchors := make(map[string]wordAnchor, len(colorsLegendWords))
	lh := faceLineSpacing(wordFace)
	for _, ww := range colorsLegendWords {
		cx, cy := at(ww.nx, ww.ny)
		lines := strings.Split(ww.label, "\n")
		maxW := 0.0
		for _, ln := range lines {
			if lw, _ := text.Measure(ln, wordFace, 0); lw > maxW {
				maxW = lw
			}
		}
		anchors[ww.key] = wordAnchor{cx, cy, maxW / 2, lh * float64(len(lines)) / 2}
	}

	// Arrows first, under the words.
	for _, rel := range colorsLegendRels {
		a, b := anchors[rel.from], anchors[rel.to]
		ux, uy, _, ok := lineDir(a.cx, a.cy, b.cx, b.cy)
		if !ok {
			continue
		}
		gap := 3 * zoom
		ea := ellipseEdge(a.hw+gap, a.hh+gap, ux, uy)
		eb := ellipseEdge(b.hw+gap, b.hh+gap, ux, uy)
		drawRelationArrow(screen, a.cx+ux*ea, a.cy+uy*ea, b.cx-ux*eb, b.cy-uy*eb, rel.edge, edgeColor(rel.edge), arrowW)
	}
	// Words on top of the arrows.
	for _, ww := range colorsLegendWords {
		cx, cy := at(ww.nx, ww.ny)
		drawCenteredLines(screen, strings.Split(ww.label, "\n"), wordFace, cx, cy, cloudTextColor)
	}
	// The colored explanations.
	for _, nt := range colorsLegendNotes {
		cx, cy := at(nt.nx, nt.ny)
		drawCenteredLines(screen, strings.Split(nt.text, "\n"), noteFace, cx, cy, nt.clr)
	}
}

// drawCenteredLines draws several lines centered as a block on (cx, cy).
func drawCenteredLines(dst *ebiten.Image, lines []string, face text.Face, cx, cy float64, clr color.Color) {
	lh := faceLineSpacing(face)
	y := cy - lh*float64(len(lines))/2 + lh/2
	for _, ln := range lines {
		drawTextCentered(dst, ln, face, cx, y, clr)
		y += lh
	}
}
