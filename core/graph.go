package core

import (
	"fmt"
	"math/rand/v2"

	"google.golang.org/protobuf/proto"

	"github.com/dqso/ticking-clue/assets"
	pb "github.com/dqso/ticking-clue/proto/gen"
)

// Link is one outgoing relation of a node. Keeping the edge type on the
// link (instead of nine separate slices) lets path search algorithms
// iterate all neighbors uniformly and filter by type with a simple check.
type Link struct {
	Type pb.EdgeType
	To   *Node
}

// Node is a lemma vertex with its outgoing links (adjacency list).
type Node struct {
	ID    int64
	Word  string
	Links []Link
}

// Neighbors returns nodes connected by any of the given edge types.
// With no arguments it returns all neighbors.
func (n *Node) Neighbors(types ...pb.EdgeType) []*Node {
	res := make([]*Node, 0, len(n.Links))
	for _, l := range n.Links {
		if len(types) == 0 {
			res = append(res, l.To)
			continue
		}
		for _, t := range types {
			if l.Type == t {
				res = append(res, l.To)
				break
			}
		}
	}
	return res
}

// Graph is the lemma graph loaded from assets/graph.pb.
type Graph struct {
	byID   map[int64]*Node
	byWord map[string]*Node
}

// LoadGraph parses the embedded protobuf and builds the in-memory graph.
// progress, when not nil, receives the loading progress from 0 to 1.
// Intermediate values are dropped when the reader is busy; the final 1
// is always delivered, so the send blocks until it is consumed.
func LoadGraph(progress chan<- float64) (*Graph, error) {
	report := func(p float64) {
		if progress == nil {
			return
		}
		if p >= 1 {
			// The final 100% always reaches the reader.
			progress <- p
			return
		}
		select {
		case progress <- p:
		default:
			// The previous value is not consumed yet, skip this one.
		}
		// Let the wasm build render a frame between work chunks.
		yieldToBrowser()
	}
	report(0)
	var src pb.Graph
	if err := proto.Unmarshal(assets.GraphPB, &src); err != nil {
		return nil, fmt.Errorf("unmarshal graph.pb: %w", err)
	}
	// Unmarshal is roughly half of the whole work.
	report(0.5)
	return buildGraph(&src, report), nil
}

// reportEvery limits how often the build loops report their progress.
const reportEvery = 4096

func buildGraph(src *pb.Graph, report func(float64)) *Graph {
	g := &Graph{
		byID:   make(map[int64]*Node, len(src.GetNodes())),
		byWord: make(map[string]*Node, len(src.GetNodes())),
	}
	nodes := src.GetNodes()
	for i, n := range nodes {
		node := &Node{ID: n.GetId(), Word: n.GetWord()}
		g.byID[node.ID] = node
		// On duplicate words the first node wins.
		if _, ok := g.byWord[node.Word]; !ok && node.Word != "" {
			g.byWord[node.Word] = node
		}
		if i%reportEvery == 0 {
			// Nodes take the 0.5..0.7 part of the progress.
			report(0.5 + 0.2*float64(i)/float64(len(nodes)))
		}
	}
	edges := src.GetEdges()
	for i, e := range edges {
		from, to := g.byID[e.GetFrom()], g.byID[e.GetTo()]
		if from == nil || to == nil {
			// Skip edges pointing to missing nodes.
			continue
		}
		from.Links = append(from.Links, Link{Type: e.GetType(), To: to})
		if i%reportEvery == 0 {
			// Edges take the 0.7..1.0 part of the progress.
			report(0.7 + 0.3*float64(i)/float64(len(edges)))
		}
	}
	report(1)
	return g
}

// Node returns a node by its id or nil.
func (g *Graph) Node(id int64) *Node {
	return g.byID[id]
}

// ByWord returns a node by its word or nil.
func (g *Graph) ByWord(word string) *Node {
	return g.byWord[word]
}

// Len returns the number of nodes.
func (g *Graph) Len() int {
	return len(g.byID)
}

// RandomLinked returns a uniformly random node with a non-empty word
// and more than min links, or nil when there is no such node.
// Uses reservoir sampling over a linear scan, so call it rarely.
func (g *Graph) RandomLinked(min int) *Node {
	var res *Node
	seen := 0
	for _, n := range g.byWord {
		if len(n.Links) <= min {
			continue
		}
		// Each matching node replaces the result with probability 1/seen,
		// which keeps the choice uniform without collecting a slice.
		seen++
		if rand.IntN(seen) == 0 {
			res = n
		}
	}
	return res
}

// Random returns a random node with a non-empty word,
// or nil when the graph has none. Linear scan, so call it rarely.
func (g *Graph) Random() *Node {
	if len(g.byWord) == 0 {
		return nil
	}
	skip := rand.IntN(len(g.byWord))
	for _, n := range g.byWord {
		if skip == 0 {
			return n
		}
		skip--
	}
	return nil
}
