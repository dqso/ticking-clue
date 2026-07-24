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
	hypernymColor = color.NRGBA{R: 0x2f, G: 0x6f, B: 0xd6, A: 0xff} // blue: broader term
	hyponymColor  = color.NRGBA{R: 0x1f, G: 0x9e, B: 0xa8, A: 0xff} // teal: narrower term
	holonymColor  = color.NRGBA{R: 0xe0, G: 0x82, B: 0x1e, A: 0xff} // orange: whole
	meronymColor  = color.NRGBA{R: 0xcf, G: 0xa8, B: 0x1e, A: 0xff} // amber: part
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
)

// edgeColor maps a relation type to its arrow color. Relations without a
// dedicated color (RELATED, HAS_*, unspecified) fall back to neutral.
func edgeColor(t pb.EdgeType) color.NRGBA {
	switch t {
	case pb.EdgeType_SYNONYM:
		return synonymColor
	case pb.EdgeType_ANTONYM:
		return antonymColor
	case pb.EdgeType_HYPERNYM:
		return hypernymColor
	case pb.EdgeType_HYPONYM:
		return hyponymColor
	case pb.EdgeType_HOLONYM:
		return holonymColor
	case pb.EdgeType_MERONYM:
		return meronymColor
	case pb.EdgeType_COORDINATE_TERM:
		return coordColor
	case pb.EdgeType_DERIVED:
		return derivedColor
	default:
		return arrowColor
	}
}
