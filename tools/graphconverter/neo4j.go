package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

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
	Word string `json:"word"` // node: lemma text
	Type string `json:"type"` // relationship: link type, e.g. SYNONYM
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
	return &pb.Node{Id: id, Word: e.Properties.Word}, nil
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
