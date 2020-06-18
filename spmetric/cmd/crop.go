package main

import (
	"github.com/mitroadmaps/gomapinfer/common"

	"flag"
	"fmt"
	"strconv"
	"strings"
)

var (
	inFname = flag.String("in", "in.graph", "input filename")
	outFname = flag.String("out", "out.graph", "output filename")
	rectStr = flag.String("rect", "", "e.g. 500,500,1000,1000")
	originStr = flag.String("origin", "", "empty for no origin, e.g. 34,34 to convert to meters with origin")
)

func main() {
	flag.Parse()
	graph, err := common.ReadGraph(*inFname)
	if err != nil {
		panic(err)
	}
	graph.MakeBidirectional()

	if *originStr != "" {
		parts := strings.Split(*originStr, ",")
		x, _ := strconv.ParseFloat(parts[0], 64)
		y, _ := strconv.ParseFloat(parts[1], 64)
		origin := common.Point{x, y}
		graph.LonLatToMeters(origin)
	}

	parts := strings.Split(*rectStr, ",")
	x1, _ := strconv.Atoi(parts[0])
	y1, _ := strconv.Atoi(parts[1])
	x2, _ := strconv.Atoi(parts[2])
	y2, _ := strconv.Atoi(parts[3])
	r := common.Rectangle{
		common.Point{float64(x1), float64(y1)},
		common.Point{float64(x2), float64(y2)},
	}

	oldBounds := graph.Bounds()
	oldCount := len(graph.Edges)
	graph = graph.GetSubgraphInRect(r)
	fmt.Printf("cropped from bounds=%v, count=%d to bounds=%v, count=%d\n", oldBounds, oldCount, graph.Bounds(), len(graph.Edges))
	if err := graph.Write(*outFname); err != nil {
		panic(err)
	}
}
