package core

// shortestPath returns the nodes of a shortest path from src to dst following
// outgoing edges, searching no deeper than maxDepth. The result starts with
// src and ends with dst; it is nil when dst is unreachable within maxDepth.
// The graph distance is len(path)-1. Bounded by depth and a visited set, so
// it is cheap enough to run on every guess.
func (g *Graph) shortestPath(src, dst *Node, maxDepth int) []*Node {
	if src == nil || dst == nil {
		return nil
	}
	if src == dst {
		return []*Node{src}
	}
	// prev maps a node id to the node it was first reached from.
	prev := map[int64]*Node{src.ID: nil}
	frontier := []*Node{src}
	for depth := 1; depth <= maxDepth && len(frontier) > 0; depth++ {
		var next []*Node
		for _, n := range frontier {
			for _, l := range n.Links {
				to := l.To
				if _, ok := prev[to.ID]; ok {
					continue
				}
				prev[to.ID] = n
				if to == dst {
					return reconstructPath(prev, dst)
				}
				next = append(next, to)
			}
		}
		frontier = next
	}
	return nil
}

// reconstructPath walks the prev map back from dst to the root (nil) and
// returns the nodes in forward order.
func reconstructPath(prev map[int64]*Node, dst *Node) []*Node {
	var rev []*Node
	for cur := dst; cur != nil; cur = prev[cur.ID] {
		rev = append(rev, cur)
	}
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return rev
}
