package spmetric

import (
	"github.com/mitroadmaps/gomapinfer/common"

	"math"
)

// Compute the Frechet distance between two paths.
// Paths are represented by the sequence of points along the path. We assume straight
//  line between successive points.
func ComputeFrechetDistance(a []common.Point, b []common.Point) float64 {
	// use dynamic programming algorithm
	// m[i, j] is Frechet distance between a[0:i] and b[0:j]
	// recurrence relation:
	//  m[i, j] = max(min(m[i-1, j], m[i, j-1]), d(i, j))
	m := make([][]float64, len(a))
	for i := range m {
		m[i] = make([]float64, len(b))
	}
	for i := range m {
		for j := range m[i] {
			if i == 0 && j == 0 {
				m[i][j] = a[i].Distance(b[j])
				continue
			}

			var best float64 = math.Inf(1)
			if i > 0 {
				best = math.Min(best, m[i - 1][j])
			}
			if j > 0 {
				best = math.Min(best, m[i][j - 1])
			}
			if i > 0 && j > 0 {
				best = math.Min(best, m[i - 1][j - 1])
			}
			m[i][j] = math.Max(best, a[i].Distance(b[j]))
		}
	}

	return m[len(a) - 1][len(b) - 1]
}

type GraphPath struct {
	Start common.EdgePos
	Path []*common.Node
	End common.EdgePos
}

// Find the path in graph with minimum Frechet distance to the given path, along with the associated Frechet distance.
func GetClosestPath(graph NodePathsGraph, path []common.Point, radius float64) (*GraphPath, float64) {
	// use Viterbi-like dynamic programming algorithm
	// m[i][edge_id] is the minimum Frechet distance for paths ending on edge_id
	// recurrence relation:
	//  m[i][edge_id] = min_{prev_edge_id in m[i - 1]}(
	//      max(
	//          (
	//              (infty if edge_id is not reachable within 4*segment_length from prev_edge_id else 1)
	//              *
	//              frechet_distance(path from prev_edge_id to edge_id, path from i-1 to i),
	//          ),
	//          m[i - 1][prev_edge_id]
	//      )
	//  )
	type Entry struct {
		MaxDistance float64
		EdgePos common.EdgePos
		Nodes []*common.Node

		CurDistance float64
		EndpointDistance float64
	}

	// returns true if a < b (if a is strictly better entry)
	less := func(a *Entry, b *Entry) bool {
		if a.MaxDistance < b.MaxDistance {
			return true
		} else if a.MaxDistance == b.MaxDistance {
			if a.CurDistance < b.CurDistance {
				return true
			} else if a.CurDistance == b.CurDistance {
				if a.EndpointDistance < b.EndpointDistance {
					return true
				} else if a.EndpointDistance == b.EndpointDistance {
					if len(a.Nodes) < len(b.Nodes) {
						return true
					}
				}
			}
		}
		return false
	}

	edgeIndex := graph.Graph.GridIndex(radius * 8)
	entries := make([]map[int]*Entry, len(path))
	backpointers := make([]map[int]int, len(path))

	// fill in initial entries
	entries[0] = make(map[int]*Entry)
	for _, edge := range edgeIndex.Search(path[0].Bounds().AddTol(radius)) {
		distance := edge.Segment().Distance(path[0])
		if distance > radius {
			continue
		}
		entries[0][edge.ID] = &Entry{
			MaxDistance: distance,
			EdgePos: edge.ClosestPos(path[0]),
			CurDistance: distance,
		}
	}

	// apply recurrence relation
	for i := 1; i < len(path); i++ {
		entries[i] = make(map[int]*Entry)
		backpointers[i] = make(map[int]int)

		candidates := edgeIndex.Search(path[i].Bounds().AddTol(radius))
		for _, edge := range candidates {
			if edge.Segment().Distance(path[i]) > radius {
				continue
			}

			for prevEdgeID, prevEntry := range entries[i - 1] {
				// get shortest path from end of previous edge to start of current edge
				// if we stayed on the same edge, shortest path is empty
				var shortestPath []*common.Node
				if prevEntry.EdgePos.Edge != edge {
					if prevEntry.EdgePos.Edge.Dst == edge.Src {
						shortestPath = []*common.Node{edge.Src}
					} else {
						/*result := graph.ShortestPath(prevEntry.EdgePos.Edge.Dst, common.ShortestPathParams{
							MaxDistance: path[i].Distance(path[i - 1]) * 4,
							StopNodes: []*common.Node{edge.Src},
						})
						resultPath := result.GetPathTo(edge.Src)*/
						resultPath := graph.GetShortestPath(prevEntry.EdgePos.Edge.Dst, edge.Src, path[i].Distance(path[i - 1]) * 4)
						if resultPath == nil {
							continue
						}
						shortestPath = append([]*common.Node{prevEntry.EdgePos.Edge.Dst}, resultPath...)
					}
				}

				// compute closest edge position to current path point
				// if we stayed on the same edge, this must be after the previous edge position
				closestPos := edge.ClosestPos(path[i])
				if prevEntry.EdgePos.Edge == edge {
					closestPos.Position = math.Max(closestPos.Position, prevEntry.EdgePos.Position)
				}

				// convert node path to point path
				// include previous edge position and ending edge position in the path
				graphPath := []common.Point{prevEntry.EdgePos.Point()}
				for _, node := range shortestPath {
					graphPath = append(graphPath, node.Point)
				}
				graphPath = append(graphPath, closestPos.Point())

				// compute frechet distance and update entries accordingly
				curDistance := ComputeFrechetDistance(graphPath, path[i-1:i+1])
				maxDistance := math.Max(prevEntry.MaxDistance, curDistance)
				entry := &Entry{
					MaxDistance: maxDistance,
					EdgePos: closestPos,
					Nodes: shortestPath,
					CurDistance: curDistance,
					EndpointDistance: math.Max(prevEntry.EdgePos.Point().Distance(path[i - 1]), closestPos.Point().Distance(path[i])),
				}

				if entries[i][edge.ID] == nil || less(entry, entries[i][edge.ID]) {
					entries[i][edge.ID] = entry
					backpointers[i][edge.ID] = prevEdgeID
				}
			}
		}
	}

	// get best entry from the last map
	var bestEndEntry *Entry
	for _, entry := range entries[len(path) - 1] {
		if bestEndEntry == nil || less(entry, bestEndEntry) {
			bestEndEntry = entry
		}
	}

	if bestEndEntry == nil {
		return nil, 0
	}

	// follow backpointers to extract path
	var nodeSeq []*common.Node
	curEdgeID := bestEndEntry.EdgePos.Edge.ID
	for i := len(path) - 1; i > 0; i-- {
		nodeSeq = append(entries[i][curEdgeID].Nodes, nodeSeq...)
		curEdgeID = backpointers[i][curEdgeID]
	}
	graphPath := &GraphPath{
		Start: entries[0][curEdgeID].EdgePos,
		Path: nodeSeq,
		End: bestEndEntry.EdgePos,
	}

	return graphPath, bestEndEntry.MaxDistance
}
