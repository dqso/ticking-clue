package core

import (
	"image/color"

	pb "github.com/dqso/ticking-clue/proto/gen"
)

// Colors of the game scene. The look is a purple mind map drawn on a ruled
// notebook page: gray cloud interiors under a darker purple outline.
var (
	paperColor     = color.NRGBA{R: 0xf6, G: 0xf5, B: 0xef, A: 0xff}
	paperRule      = color.NRGBA{R: 0xc2, G: 0xdf, B: 0xdb, A: 0xff}
	cloudStroke    = color.NRGBA{R: 0x6a, G: 0x6a, B: 0x72, A: 0xff} // gray cloud outline
	cloudInterior  = color.NRGBA{R: 0xd7, G: 0xd6, B: 0xda, A: 0xff} // gray, lighter than the outline
	cloudTextColor = color.NRGBA{R: 0x33, G: 0x33, B: 0x3a, A: 0xff}
	skippedColor   = color.NRGBA{R: 0x8c, G: 0x86, B: 0x98, A: 0xff}

	// Arrow colors: neutral by default, switched to the relation palette
	// below once the "reveal colors" hint is bought.
	arrowColor    = color.NRGBA{R: 0x7a, G: 0x76, B: 0x86, A: 0xff}
	synonymColor  = color.NRGBA{R: 0x2f, G: 0x9e, B: 0x54, A: 0xff} // green
	antonymColor  = color.NRGBA{R: 0xd6, G: 0x3a, B: 0x3a, A: 0xff} // red
	hyponymyColor = color.NRGBA{R: 0x2f, G: 0x6f, B: 0xd6, A: 0xff} // blue: hyponymy
	meronymyColor = color.NRGBA{R: 0xe0, G: 0x82, B: 0x1e, A: 0xff} // orange: meronymy
	coordColor    = color.NRGBA{R: 0x8a, G: 0x4f, B: 0xc0, A: 0xff} // purple: coordinate term
	derivedColor  = color.NRGBA{R: 0xc0, G: 0x4f, B: 0x9a, A: 0xff} // magenta: derived
	guessColor    = color.NRGBA{R: 0x8a, G: 0x86, B: 0x94, A: 0xff}
	promptColor   = color.NRGBA{R: 0xb2, G: 0xae, B: 0xbc, A: 0xff} // guess prompt, lighter
	timerColor    = color.NRGBA{R: 0x3a, G: 0x2a, B: 0x4a, A: 0xff}
	timerLowColor = color.NRGBA{R: 0xd6, G: 0x2a, B: 0x2a, A: 0xff}
	rewardColor   = color.NRGBA{R: 0x2f, G: 0x9e, B: 0x54, A: 0xff}
	penaltyColor  = color.NRGBA{R: 0xd6, G: 0x2a, B: 0x2a, A: 0xff}

	overlayColor     = color.NRGBA{R: 0x1a, G: 0x14, B: 0x24, A: 0xcc}
	overlayTextColor = color.NRGBA{R: 0xf4, G: 0xf2, B: 0xf8, A: 0xff}

	// Hint cells: light squares so the dark icons and cost read clearly.
	hintIconColor = color.NRGBA{R: 0x2a, G: 0x2a, B: 0x2a, A: 0xff}
	hintCostColor = color.NRGBA{R: 0x9a, G: 0x22, B: 0x22, A: 0xff}

	// Miss list: a yellow sticky note holding the words that were too far or
	// unknown, drawn down the left edge of the screen.
	stickerColor  = color.NRGBA{R: 0xfd, G: 0xe7, B: 0x8a, A: 0xff}
	stickerEdge   = color.NRGBA{R: 0xd9, G: 0xbf, B: 0x55, A: 0xff}
	listTextColor = color.NRGBA{R: 0x40, G: 0x38, B: 0x18, A: 0xff}

	// Colors note: a white sticky note dropped on the sheet when the "reveal
	// colors" hint is bought. Collapsed it shows crossed arrows; expanded it
	// holds the arrow legend.
	colorsNoteColor = color.NRGBA{R: 0xfb, G: 0xfb, B: 0xf9, A: 0xff}
	colorsNoteEdge  = color.NRGBA{R: 0xcc, G: 0xc9, B: 0xd2, A: 0xff}
)

// edgeColor maps a relation type to its arrow color. Hypernym and hyponym share
// one color (hyponymy), as do holonym and meronym (meronymy). Relations without
// a dedicated color (RELATED, HAS_*, unspecified) fall back to neutral.
func edgeColor(t pb.EdgeType) color.NRGBA {
	switch t {
	case pb.EdgeType_SYNONYM:
		return synonymColor
	case pb.EdgeType_ANTONYM:
		return antonymColor
	case pb.EdgeType_HYPERNYM, pb.EdgeType_HYPONYM:
		return hyponymyColor
	case pb.EdgeType_HOLONYM, pb.EdgeType_MERONYM:
		return meronymyColor
	case pb.EdgeType_COORDINATE_TERM:
		return coordColor
	case pb.EdgeType_DERIVED:
		return derivedColor
	default:
		return arrowColor
	}
}

// edgeName returns the canonical name of a relation. Hypernym and hyponym are
// one relation ("hyponymy"), as are holonym and meronym ("meronymy").
func edgeName(t pb.EdgeType) string {
	switch t {
	case pb.EdgeType_SYNONYM:
		return "synonym"
	case pb.EdgeType_ANTONYM:
		return "antonym"
	case pb.EdgeType_HYPERNYM, pb.EdgeType_HYPONYM:
		return "hyponymy"
	case pb.EdgeType_HOLONYM, pb.EdgeType_MERONYM:
		return "meronymy"
	case pb.EdgeType_COORDINATE_TERM:
		return "coordinate term"
	case pb.EdgeType_DERIVED:
		return "derived"
	default:
		return "related"
	}
}
