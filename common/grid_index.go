package common

import (
	"math"
)

type GridIndex struct {
	gridSize float64
	grid map[[2]int][]int
}

func NewGridIndex(gridSize float64) *GridIndex {
	return &GridIndex{
		gridSize: gridSize,
		grid: make(map[[2]int][]int),
	}
}

func (idx GridIndex) eachCell(rect Rectangle, f func(i int, j int)) {
	for i := int(math.Floor(rect.Min.X / idx.gridSize)); i <= int(math.Floor(rect.Max.X / idx.gridSize)); i++ {
		for j := int(math.Floor(rect.Min.Y / idx.gridSize)); j <= int(math.Floor(rect.Max.Y / idx.gridSize)); j++ {
			f(i, j)
		}
	}
}

func (idx GridIndex) Search(rect Rectangle) []int {
	ids := make(map[int]bool)
	idx.eachCell(rect, func(i int, j int) {
		for _, id := range idx.grid[[2]int{i, j}] {
			ids[id] = true
		}
	})
	var idlist []int
	for id := range ids {
		idlist = append(idlist, id)
	}
	return idlist
}

func (idx GridIndex) Insert(id int, rect Rectangle) {
	idx.eachCell(rect, func(i int, j int) {
		idx.grid[[2]int{i, j}] = append(idx.grid[[2]int{i, j}], id)
	})
}
