package core

import (
	"strings"

	pb "github.com/dqso/ticking-clue/proto/gen"
)

// allParts lists every part-of-speech bit from the lowest to the highest, so
// PartsOfSpeech returns the parts of a lemma in a stable order. It holds only
// the POS_* flags; the LEVEL_* flags of the Attributes enum are not parts of
// speech and stay out of this list.
var allParts = []pb.Attributes{
	pb.Attributes_POS_ADJECTIVE,
	pb.Attributes_POS_ADVERB,
	pb.Attributes_POS_CONJUNCTION,
	pb.Attributes_POS_CONTRACTION,
	pb.Attributes_POS_DETERMINER,
	pb.Attributes_POS_INTERJECTION,
	pb.Attributes_POS_NAME,
	pb.Attributes_POS_NOUN,
	pb.Attributes_POS_NUMBER,
	pb.Attributes_POS_PARTICLE,
	pb.Attributes_POS_PHRASE,
	pb.Attributes_POS_PREPOSITION,
	pb.Attributes_POS_PREPOSITIONAL_PHRASE,
	pb.Attributes_POS_PRONOUN,
	pb.Attributes_POS_PROVERB,
	pb.Attributes_POS_VERB,
}

// PartsOfSpeech unpacks the Attributes bitmask into the parts of speech the
// lemma can be, in a stable order.
func (n *Node) PartsOfSpeech() []pb.Attributes {
	var res []pb.Attributes
	for _, p := range allParts {
		if n.Attributes&uint64(p) != 0 {
			res = append(res, p)
		}
	}
	return res
}

// posName is the human-readable English name of a part of speech, used in the
// "parts of speech" hint sticker.
func posName(p pb.Attributes) string {
	switch p {
	case pb.Attributes_POS_ADJECTIVE:
		return "adjective"
	case pb.Attributes_POS_ADVERB:
		return "adverb"
	case pb.Attributes_POS_CONJUNCTION:
		return "conjunction"
	case pb.Attributes_POS_CONTRACTION:
		return "contraction"
	case pb.Attributes_POS_DETERMINER:
		return "determiner"
	case pb.Attributes_POS_INTERJECTION:
		return "interjection"
	case pb.Attributes_POS_NAME:
		return "name"
	case pb.Attributes_POS_NOUN:
		return "noun"
	case pb.Attributes_POS_NUMBER:
		return "number"
	case pb.Attributes_POS_PARTICLE:
		return "particle"
	case pb.Attributes_POS_PHRASE:
		return "phrase"
	case pb.Attributes_POS_PREPOSITION:
		return "preposition"
	case pb.Attributes_POS_PREPOSITIONAL_PHRASE:
		return "prepositional phrase"
	case pb.Attributes_POS_PRONOUN:
		return "pronoun"
	case pb.Attributes_POS_PROVERB:
		return "proverb"
	case pb.Attributes_POS_VERB:
		return "verb"
	default:
		return "word"
	}
}

// article picks "a" or "an" for the given word by its first letter, so the
// hint sentence reads naturally (a noun, an adjective).
func article(word string) string {
	if word == "" {
		return "a"
	}
	switch word[0] {
	case 'a', 'e', 'i', 'o', 'u':
		return "an"
	}
	return "a"
}

// posSentence describes, in simple first-person English (A2-B1), which parts of
// speech the hidden word can be, for the Notes log, e.g. "I learned the word can
// be a verb or a noun".
func posSentence(hidden *Node) string {
	parts := hidden.PartsOfSpeech()
	if len(parts) == 0 {
		return "I checked the part of speech, but it is unknown."
	}
	items := make([]string, len(parts))
	for i, p := range parts {
		name := posName(p)
		items[i] = article(name) + " " + name
	}
	return "I learned the word can be " + strings.Join(items, " or ") + "."
}
