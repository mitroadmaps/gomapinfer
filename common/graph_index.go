package common

type GraphGridIndex struct {
	idx *GridIndex
	graph *Graph
}

func (idx GraphGridIndex) Search(rect Rectangle) []*Edge {
	edgeIDs := idx.idx.Search(rect)
	edges := make([]*Edge, len(edgeIDs))
	for i, edgeID := range edgeIDs {
		edges[i] = idx.graph.Edges[edgeID]
	}
	return edges
}

func (graph *Graph) GridIndex(gridSize float64) GraphGridIndex {
	idx := NewGridIndex(gridSize)
	for _, edge := range graph.Edges {
		idx.Insert(edge.ID, edge.Segment().Bounds())
	}
	return GraphGridIndex{idx, graph}
}
