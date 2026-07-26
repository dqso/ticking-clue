package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	pb "github.com/dqso/ticking-clue/proto/gen"
)

// entity is one line of the apoc.export.json.data output.
// Depending on Type it describes a node or a relationship.
type entity struct {
	Type       string     `json:"type"`
	ID         string     `json:"id"`
	Properties properties `json:"properties"`
	Start      *endpoint  `json:"start"`
	End        *endpoint  `json:"end"`
}

type endpoint struct {
	ID string `json:"id"`
}

type properties struct {
	Word  string   `json:"word"`  // node: lemma text
	Pos   []string `json:"pos"`   // node: parts of speech, e.g. ["adj","noun"]
	Level string   `json:"level"` // node: CEFR level, e.g. "b1"; empty means "c2"
	Type  string   `json:"type"`  // relationship: link type, e.g. SYNONYM
}

// posBits maps a neo4j "pos" value to its Attributes flag bit.
var posBits = map[string]pb.Attributes{
	"adj":         pb.Attributes_POS_ADJECTIVE,
	"adv":         pb.Attributes_POS_ADVERB,
	"conj":        pb.Attributes_POS_CONJUNCTION,
	"contraction": pb.Attributes_POS_CONTRACTION,
	"det":         pb.Attributes_POS_DETERMINER,
	"intj":        pb.Attributes_POS_INTERJECTION,
	"name":        pb.Attributes_POS_NAME,
	"noun":        pb.Attributes_POS_NOUN,
	"num":         pb.Attributes_POS_NUMBER,
	"particle":    pb.Attributes_POS_PARTICLE,
	"phrase":      pb.Attributes_POS_PHRASE,
	"prep":        pb.Attributes_POS_PREPOSITION,
	"prep_phrase": pb.Attributes_POS_PREPOSITIONAL_PHRASE,
	"pron":        pb.Attributes_POS_PRONOUN,
	"proverb":     pb.Attributes_POS_PROVERB,
	"verb":        pb.Attributes_POS_VERB,
}

// levelBits maps a neo4j "level" value to its CEFR flag bit. A missing or
// unknown level is treated as C2 (see levelBit).
var levelBits = map[string]pb.Attributes{
	"a1": pb.Attributes_LEVEL_A1,
	"a2": pb.Attributes_LEVEL_A2,
	"b1": pb.Attributes_LEVEL_B1,
	"b2": pb.Attributes_LEVEL_B2,
	"c1": pb.Attributes_LEVEL_C1,
	"c2": pb.Attributes_LEVEL_C2,
}

// levelBit returns the CEFR flag for a lemma. An empty or unknown level defaults
// to C2, so every node carries exactly one level bit.
func levelBit(level string) pb.Attributes {
	if b, ok := levelBits[strings.ToLower(level)]; ok {
		return b
	}
	return pb.Attributes_LEVEL_C2
}

// readNeo4jGraph parses the JSON lines export into a protobuf graph,
// collecting stats while it reads.
func readNeo4jGraph(r io.Reader) (*pb.Graph, *stats, error) {
	graph := &pb.Graph{}
	st := newStats()

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for line := 1; scanner.Scan(); line++ {
		raw := scanner.Bytes()
		if len(raw) == 0 {
			continue
		}
		var e entity
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil, nil, fmt.Errorf("line %d: %w", line, err)
		}
		switch e.Type {
		case "node":
			node, err := e.toNode()
			if err != nil {
				return nil, nil, fmt.Errorf("line %d: %w", line, err)
			}
			graph.Nodes = append(graph.Nodes, node)
			st.addNode()
		case "relationship":
			edge, err := e.toEdge()
			if err != nil {
				return nil, nil, fmt.Errorf("line %d: %w", line, err)
			}
			graph.Edges = append(graph.Edges, edge)
			st.addEdge(edge.Type)
		default:
			return nil, nil, fmt.Errorf("line %d: unknown entity type %q", line, e.Type)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	return graph, st, nil
}

func (e entity) toNode() (*pb.Node, error) {
	id, err := strconv.ParseInt(e.ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("node id: %w", err)
	}
	attributes, err := posMask(e.Properties.Pos)
	if err != nil {
		return nil, fmt.Errorf("node %s: %w", e.ID, err)
	}
	// Pack the CEFR level as one more bit next to the parts of speech.
	attributes |= uint64(levelBit(e.Properties.Level))
	return &pb.Node{Id: id, Word: e.Properties.Word, Attributes: attributes}, nil
}

// posMask packs the parts of speech of a lemma into a bitmask.
func posMask(pos []string) (uint64, error) {
	var mask uint64
	for _, p := range pos {
		bit, ok := posBits[p]
		if !ok {
			return 0, fmt.Errorf("unknown pos %q", p)
		}
		mask |= uint64(bit)
	}
	return mask, nil
}

func (e entity) toEdge() (*pb.Edge, error) {
	if e.Start == nil || e.End == nil {
		return nil, fmt.Errorf("relationship %s: missing start or end", e.ID)
	}
	id, err := strconv.ParseInt(e.ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("relationship id: %w", err)
	}
	from, err := strconv.ParseInt(e.Start.ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("relationship %s start id: %w", e.ID, err)
	}
	to, err := strconv.ParseInt(e.End.ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("relationship %s end id: %w", e.ID, err)
	}
	edgeType, ok := pb.EdgeType_value[e.Properties.Type]
	if !ok {
		return nil, fmt.Errorf("relationship %s: unknown link type %q", e.ID, e.Properties.Type)
	}
	return &pb.Edge{Id: id, Type: pb.EdgeType(edgeType), From: from, To: to}, nil
}
