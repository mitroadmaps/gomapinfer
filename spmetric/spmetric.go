package spmetric

import (
	"github.com/mitroadmaps/gomapinfer/common"

	"fmt"
	"math"
	"math/rand"
	"runtime"
)

const SP_ITERATIONS = 100
const SP_RADIUS = 512
const SP_MAX_DISTANCE = 2048

func SamplePath(graph *common.Graph) []*common.Node {
	var node *common.Node
	bounds := graph.Bounds().AddTol(-512)
	for i := 0; true; i++ {
		if i % 50 == 0 {
			bounds = bounds.AddTol(64)
		}
		node = graph.Nodes[rand.Intn(len(graph.Nodes))]
		if len(node.Out) > 0 && bounds.Contains(node.Point) {
			break
		}
	}

	result := graph.ShortestPath(node, common.ShortestPathParams{
		MaxDistance: SP_MAX_DISTANCE,
	})

	var dstIDs []int
	for dstID := range result.Backpointers {
		if !result.Remaining[dstID] && dstID != node.ID {
			dstIDs = append(dstIDs, dstID)
		}
	}
	dst := graph.Nodes[dstIDs[rand.Intn(len(dstIDs))]]
	return result.GetPathTo(dst)
}

func SPMetric(a NodePathsGraph, b NodePathsGraph, prefix string, showA bool) (float64, float64, float64) {
	f := func(iter int) (float64, bool) {
		aPath := SamplePath(a.Graph)
		aPoints := make([]common.Point, len(aPath))
		for i := range aPoints {
			aPoints[i] = aPath[i].Point
		}
		bPath, distance := GetClosestPath(b, aPoints, SP_RADIUS)
		if bPath == nil {
			return 0, false
		}

		if true {
			var bPoints []common.Point
			bPoints = append(bPoints, bPath.Start.Point())
			for _, node := range bPath.Path {
				bPoints = append(bPoints, node.Point)
			}
			bPoints = append(bPoints, bPath.End.Point())

			var boundables []common.Boundable
			boundables = append(boundables, common.EmbeddedImage{
				Src: common.Point{-4096, -8192},
				Dst: common.Point{4096, 0},
				Image: "./14-chicago.png",
			})
			if showA {
				boundables = append(boundables, common.ColoredBoundable{b.Graph, "yellow"})
				boundables = append(boundables, common.ColoredBoundable{a.Graph, "blue"})
			} else {
				boundables = append(boundables, common.ColoredBoundable{a.Graph, "yellow"})
				boundables = append(boundables, common.ColoredBoundable{b.Graph, "blue"})
			}
			r := common.EmptyRectangle
			for i := 0; i < len(aPoints) - 1; i++ {
				segment := common.Segment{aPoints[i], aPoints[i + 1]}
				boundables = append(boundables, common.WidthBoundable{common.ColoredBoundable{segment, "green"}, 5})
				r = r.Extend(aPoints[i])
			}
			for i := 0; i < len(bPoints) - 1; i++ {
				segment := common.Segment{bPoints[i], bPoints[i + 1]}
				boundables = append(boundables, common.WidthBoundable{common.ColoredBoundable{segment, "red"}, 5})
				r = r.Extend(bPoints[i])
			}
			if err := common.CreateSVG(fmt.Sprintf("%s%d.svg", prefix, iter), [][]common.Boundable{boundables}, common.SVGOptions{
				Bounds: r.AddTol(128),
				Unflip: true,
			}); err != nil {
				panic(err)
			}
		}

		return distance, true
	}

	type Result struct {
		Distances []float64
		InvalidCount int
	}

	inCh := make(chan int)
	doneCh := make(chan Result)

	nthreads := runtime.NumCPU()
	for i := 0; i < nthreads; i++ {
		go func() {
			var result Result
			for iter := range inCh {
				distance, valid := f(iter)
				if valid {
					result.Distances = append(result.Distances, distance)
				} else {
					result.InvalidCount++
				}
			}
			doneCh <- result
		}()
	}

	for iter := 0; iter < SP_ITERATIONS; iter++ {
		inCh <- iter
	}
	close(inCh)

	var result Result
	for i := 0; i < nthreads; i++ {
		threadResult := <- doneCh
		result.Distances = append(result.Distances, threadResult.Distances...)
		result.InvalidCount += threadResult.InvalidCount
	}

	coverage := float64(len(result.Distances)) / float64(len(result.Distances) + result.InvalidCount)
	var sum float64
	for _, d := range result.Distances {
		sum += d
	}
	avg := sum / float64(len(result.Distances))
	var sumSqErr float64
	for _, d := range result.Distances {
		sumSqErr += (d - avg) * (d - avg)
	}
	stddev := math.Sqrt(sumSqErr / float64(len(result.Distances) - 1))
	return avg, stddev, coverage
}
