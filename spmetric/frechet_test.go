package spmetric

import (
	"github.com/mitroadmaps/gomapinfer/common"

	"math"
	"testing"
)

func TestComputeFrechetDistance(t *testing.T) {
	f := func(a []common.Point, b []common.Point, expected float64) {
		d := ComputeFrechetDistance(a, b)
		if math.Abs(d - expected) > 0.001 {
			t.Errorf("d(%v, %v) got %v expected %v", a, b, d, expected)
		}
	}

	// single point tests
	f([]common.Point{{1, 1}}, []common.Point{{1, 1}}, 0)
	f([]common.Point{{1, 1}}, []common.Point{{3, 1}}, 2)

	// loop path against single point
	path := []common.Point{
		{0, 0},
		{1, 0},
		{1, 1},
		{0, 1},
		{0, 0},
	}
	f(path, []common.Point{{0, 0}}, math.Sqrt(2))
	f(path, []common.Point{{0.5, 0.5}}, math.Sqrt(2) / 2)

	// two paths, one with detour
	directPath := []common.Point{
		{0, 0},
		{1, 0},
		{1, 1},
		{2, 1},
	}
	detourPath := []common.Point{
		{0, 0},
		{1, 0},
		{1, 1},
		{1, 2},
		{2, 2},
		{2, 1},
	}
	f(directPath, directPath, 0)
	f(directPath, detourPath, 1)
	f(detourPath, directPath, 1)

	// two paths, one loops back
	straightPath := []common.Point{
		{0, 0},
		{2, 0},
		{4, 0},
	}
	loopPath := []common.Point{
		{0, 0},
		{4, 0},
		{0, 0},
		{4, 0},
	}
	f(loopPath, loopPath, 0)
	f(straightPath, loopPath, 2)
}

func TestGetClosestPath(t *testing.T) {
	graph := &common.Graph{}
	v11 := graph.AddNode(common.Point{1, 1})
	v12 := graph.AddNode(common.Point{1, 2})
	v31 := graph.AddNode(common.Point{3, 1})
	v32 := graph.AddNode(common.Point{3, 2})
	v51 := graph.AddNode(common.Point{5, 1})
	v52 := graph.AddNode(common.Point{5, 2})
	graph.AddBidirectionalEdge(v11, v12)
	graph.AddBidirectionalEdge(v11, v31)
	graph.AddBidirectionalEdge(v31, v32)
	graph.AddBidirectionalEdge(v31, v51)
	graph.AddBidirectionalEdge(v32, v52)
	graph.AddBidirectionalEdge(v51, v52)

	radius := 10.0

	f := func(inPath []common.Point, expected []*common.Node, d float64) {
		outPath, gotD := GetClosestPath(graph, inPath, radius)
		var outNodes []*common.Node
		outNodes = append(outNodes, outPath.Start.Edge.Src)
		outNodes = append(outNodes, outPath.Path...)
		outNodes = append(outNodes, outPath.End.Edge.Dst)
		if len(outNodes) != len(expected) {
			t.Errorf("GetClosestPath(%v) got %v expected %v", inPath, outNodes, expected)
			return
		}
		var points []common.Point
		for i := range outNodes {
			if outNodes[i] != expected[i] {
				t.Errorf("GetClosestPath(%v) got %v expected %v", inPath, outNodes, expected)
				return
			}
			points = append(points, outNodes[i].Point)
		}
		if math.Abs(gotD - d) > 0.001 {
			t.Errorf("GetClosestPath(%v) got distance %v expected %v", inPath, gotD, d)
			return
		}
	}

	f([]common.Point{
		{1, 2},
		{3, 2},
	}, []*common.Node{
		v12,
		v11,
		v31,
		v32,
	}, 1)

	f([]common.Point{
		{0.8, 2.2},
		{0.8, 0.8},
		{3.2, 0.8},
		{2.8, 2.2},
	}, []*common.Node{
		v12,
		v11,
		v31,
		v32,
	}, math.Sqrt(2) / 5)
}
