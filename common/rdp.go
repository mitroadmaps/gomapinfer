package common

func RDP(points []Point, epsilon float64) []Point {
	segment := Segment{points[0], points[len(points) - 1]}
	var dmax float64 = 0
	index := 0
	for i := 1; i < len(points) - 1; i++ {
		d := segment.Distance(points[i])
		if d > dmax {
			index = i
			dmax = d
		}
	}
	if dmax >= epsilon {
		prefix := RDP(points[:index+1], epsilon)
		suffix := RDP(points[index:], epsilon)
		return append(prefix[:len(prefix)-1], suffix...)
	} else {
		return []Point{points[0], points[len(points)-1]}
	}
}
