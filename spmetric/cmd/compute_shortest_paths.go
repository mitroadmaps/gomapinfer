package main

import (
	"github.com/mitroadmaps/gomapinfer/common"
	".."

	"encoding/json"
	"fmt"
	"io/ioutil"
	"runtime"
	"os"
)

func main() {
	graph, err := common.ReadGraph(os.Args[1])
	if err != nil {
		panic(err)
	}
	inCh := make(chan int)
	doneCh := make(chan bool)

	getNodePaths := func(node *common.Node) spmetric.NodePaths {
		result := graph.ShortestPath(node, common.ShortestPathParams{
			MaxDistance: 1600,
		})
		np := spmetric.NodePaths{
			Backpointers: result.Backpointers,
			Distances: result.Distances,
		}
		for nodeID := range np.Backpointers {
			if result.Remaining[nodeID] {
				delete(np.Backpointers, nodeID)
			}
		}
		for nodeID := range np.Distances {
			if result.Remaining[nodeID] {
				delete(np.Distances, nodeID)
			}
		}
		return np
	}

	nthreads := runtime.NumCPU()
	for i := 0; i < nthreads; i++ {
		go func() {
			for nodeID := range inCh {
				nodePaths := getNodePaths(graph.Nodes[nodeID])

				bytes, err := json.Marshal(nodePaths)
				if err != nil {
					panic(err)
				}
				if err := ioutil.WriteFile(fmt.Sprintf("%s.sp/%d.sp", os.Args[1], nodeID), bytes, 0644); err != nil {
					panic(err)
				}
			}
			doneCh <- true
		}()
	}

	for nodeID := range graph.Nodes {
		if nodeID % 100 == 0 {
			fmt.Printf("%d/%d\n", nodeID, len(graph.Nodes))
		}
		inCh <- nodeID
	}
	close(inCh)
	for i := 0; i < nthreads; i++ {
		<- doneCh
	}
}
