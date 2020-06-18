package spmetric

import (
	"github.com/mitroadmaps/gomapinfer/common"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

type NodePathsGraph struct {
	Graph *common.Graph
	NodePaths map[int]NodePaths
}

type NodePaths struct {
	Backpointers map[int]int `json:"backpointers"`
	Distances map[int]float64 `json:"distances"`
}

func (g NodePathsGraph) GetShortestPath(src *common.Node, dst *common.Node, maxDistance float64) []*common.Node {
	np := g.NodePaths[src.ID]
	if _, ok := np.Backpointers[dst.ID]; !ok {
		return nil
	} else if np.Distances[dst.ID] > maxDistance {
		return nil
	}

	var reverseSeq []*common.Node
	curNode := dst
	for curNode.ID != src.ID {
		reverseSeq = append(reverseSeq, curNode)
		curNode = g.Graph.Nodes[np.Backpointers[curNode.ID]]
	}
	path := make([]*common.Node, len(reverseSeq))
	for i, node := range reverseSeq {
		path[len(path) - i - 1] = node
	}
	return path
}

func ReadNodePathsGraph(fname string) (NodePathsGraph, error) {
	graph, err := common.ReadGraph(fname)
	if err != nil {
		return NodePathsGraph{}, err
	}
	files, err := ioutil.ReadDir(fname + ".sp")
	if err != nil {
		return NodePathsGraph{}, err
	}
	nodePaths := make(map[int]NodePaths)
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sp") {
			continue
		}
		nodeID, err := strconv.Atoi(strings.Split(file.Name(), ".")[0])
		if err != nil {
			continue
		}
		bytes, err := ioutil.ReadFile(fmt.Sprintf("%s.sp/%d.sp", fname, nodeID))
		if err != nil {
			return NodePathsGraph{}, err
		}
		var np NodePaths
		if err := json.Unmarshal(bytes, &np); err != nil {
			return NodePathsGraph{}, err
		}
		nodePaths[nodeID] = np
	}
	return NodePathsGraph{graph, nodePaths}, nil
}
