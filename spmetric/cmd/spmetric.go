package main

import (
	".."

	"fmt"
	"os"
)

func main() {
	fmt.Println("reading truth graph")
	truth, err := spmetric.ReadNodePathsGraph(os.Args[1])
	if err != nil {
		panic(err)
	}
	fmt.Println("reading inferred graph")
	graph, err := spmetric.ReadNodePathsGraph(os.Args[2])
	if err != nil {
		panic(err)
	}

	/*for {
		path := spmetric.SamplePath(truth)
		var maxDistance float64 = 0
		for i := 0; i < len(path) - 1; i++ {
			distance := path[i].Point.Distance(path[i + 1].Point)
			if distance > maxDistance {
				maxDistance = distance
			}
		}
		fmt.Println(maxDistance)
	}*/

	fmt.Println("running metric")
	avg1, stddev1, coverage1 := spmetric.SPMetric(truth, graph, "d1/", false)
	avg2, stddev2, coverage2 := spmetric.SPMetric(graph, truth, "d2/", true)
	fmt.Printf("truth -> inferred: avg=%.0f, stddev=%.0f, coverage=%.1f\n", avg1, stddev1, coverage1)
	fmt.Printf("inferred -> truth: avg=%.0f, stddev=%.0f, coverage=%.1f\n", avg2, stddev2, coverage2)
}
