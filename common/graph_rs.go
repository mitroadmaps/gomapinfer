package common

import (
	"fmt"
)

type RoadSegment struct {
	ID int
	Edges []*Edge
	EdgeDistances []float64
}

func NewRoadSegment(edges []*Edge) (RoadSegment, error) {
	edgeDistances := []float64{0}
	var cur *Node
	for _, edge := range edges {
		if cur == nil {
			edgeDistances = append(edgeDistances, edge.Segment().Length())
			cur = edge.Dst
		} else if cur != edge.Src {
			return RoadSegment{}, fmt.Errorf("edges do not form a path")
		} else {
			edgeDistances = append(edgeDistances, edgeDistances[len(edgeDistances) - 1] + edge.Segment().Length())
			cur = edge.Dst
		}
	}
	return RoadSegment{
		Edges: edges,
		EdgeDistances: edgeDistances,
	}, nil
}

func (rs RoadSegment) Src() *Node {
	return rs.Edges[0].Src
}

func (rs RoadSegment) Dst() *Node {
	return rs.Edges[len(rs.Edges) - 1].Dst
}

func (rs RoadSegment) Length() float64 {
	return rs.EdgeDistances[len(rs.EdgeDistances) - 1]
}

func (rs RoadSegment) DistanceToIndex(t float64) int {
	for i, edge := range rs.Edges {
		t -= edge.Segment().Length()
		if t <= 0 {
			return i
		}
	}
	return len(rs.Edges) - 1
}

func (rs RoadSegment) DistanceToEdge(t float64) *Edge {
	return rs.Edges[rs.DistanceToIndex(t)]
}

func (rs RoadSegment) DistanceOfEdge(edge *Edge) float64 {
	for i, other := range rs.Edges {
		if edge == other {
			return rs.EdgeDistances[i]
		}
	}
	return -1
}

func (rs RoadSegment) PosAtFactor(t float64) EdgePos {
	idx := rs.DistanceToIndex(t)
	edge := rs.Edges[idx]
	position := t - rs.EdgeDistances[idx]
	if position > edge.Segment().Length() {
		position = edge.Segment().Length()
	}
	return EdgePos{edge, position}
}

func (rs RoadSegment) ClosestPos(p Point) EdgePos {
	var bestPos EdgePos
	var bestDistance float64 = -1
	for _, edge := range rs.Edges {
		pos := edge.ClosestPos(p)
		distance := pos.Point().Distance(p)
		if bestDistance == -1 || distance < bestDistance {
			bestPos = pos
			bestDistance = distance
		}
	}
	return bestPos
}

// Get road segments, i.e., sequences of edges between junctions.
// Junctions are vertices where the number of incident edges != 2.
// This function only works for bidirectional graphs.
func (graph *Graph) GetRoadSegments() []RoadSegment {
	var roadSegments []RoadSegment
	seenEdges := make(map[int]bool)

	// Incorporate a new road segment with the specified starting edge.
	// If expectLoop is false, this should start from a junction or dead end node.
	// If expectLoop is true, initialEdge can be an arbitrary edge along an isolated loop.
	incorporate := func(initialEdge *Edge, expectLoop bool) {
		edges := []*Edge{initialEdge}
		seenEdges[initialEdge.ID] = true
		prevEdge := initialEdge
		for {
			if len(prevEdge.Dst.Out) != 2 {
				if expectLoop {
					panic(fmt.Errorf("ran into junction vertex while following loop"))
				}
				break
			}
			var nextEdge *Edge
			if prevEdge.Dst.Out[0].Dst != prevEdge.Src {
				nextEdge = prevEdge.Dst.Out[0]
			} else if prevEdge.Dst.Out[1].Dst != prevEdge.Src {
				nextEdge = prevEdge.Dst.Out[1]
			} else {
				panic(fmt.Errorf("got stuck following edge"))
			}
			if nextEdge == initialEdge {
				if !expectLoop {
					panic(fmt.Errorf("unexpectedly entered loop"))
				}
				break
			}
			edges = append(edges, nextEdge)
			seenEdges[nextEdge.ID] = true
			prevEdge = nextEdge
		}
		rs, err := NewRoadSegment(edges)
		if err != nil {
			panic(err)
		}
		rs.ID = len(roadSegments)
		roadSegments = append(roadSegments, rs)
	}

	for _, node := range graph.Nodes {
		if len(node.Out) == 2 {
			continue
		}
		for _, edge := range node.Out {
			incorporate(edge, false)
		}
	}
	// if there is a loop that is disconnected from the rest of the graph, it is not added above
	// we now add these loops
	for _, edge := range graph.Edges {
		if seenEdges[edge.ID] {
			continue
		}
		if len(edge.Src.Out) != 2 || len(edge.Dst.Out) != 2 {
			panic(fmt.Errorf("missing edge does not look like a loop"))
		}
		incorporate(edge, true)
	}
	return roadSegments
}

func (graph *Graph) GetRoadSegmentGraph() (*Graph, map[int]RoadSegment, map[int]*Node) {
	roadSegments := graph.GetRoadSegments()
	nodeMap := make(map[int]*Node)
	edgeToSegment := make(map[int]RoadSegment)
	g := &Graph{}
	for _, rs := range roadSegments {
		for _, node := range []*Node{rs.Src(), rs.Dst()} {
			if nodeMap[node.ID] == nil {
				nodeMap[node.ID] = g.AddNode(node.Point)
			}
		}
		edge := g.AddEdge(nodeMap[rs.Src().ID], nodeMap[rs.Dst().ID])
		edgeToSegment[edge.ID] = rs
	}
	return g, edgeToSegment, nodeMap
}
