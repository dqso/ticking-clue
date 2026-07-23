package main

import (
	"log"
	"maps"
	"slices"

	pb "github.com/dqso/ticking-clue/proto/gen"
)

// stats accumulates counters while the export is being parsed.
type stats struct {
	nodes int
	edges map[pb.EdgeType]int
}

func newStats() *stats {
	return &stats{edges: make(map[pb.EdgeType]int)}
}

func (s *stats) addNode() {
	s.nodes++
}

func (s *stats) addEdge(t pb.EdgeType) {
	s.edges[t]++
}

// print logs node count and edge counts per link type.
func (s *stats) print(name string) {
	total := 0
	for _, n := range s.edges {
		total += n
	}
	log.Printf("%s: %d nodes, %d edges", name, s.nodes, total)

	// sort by enum value to keep the output stable
	for _, t := range slices.Sorted(maps.Keys(s.edges)) {
		log.Printf("  %s: %d", t, s.edges[t])
	}
}
