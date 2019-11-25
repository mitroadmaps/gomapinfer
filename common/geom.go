package common

import (
	"math"
	"math/rand"
	"sort"
)

type Boundable interface {
	Bounds() Rectangle
}

type Point struct {
	X float64
	Y float64
}

func (point Point) LonLatToMeters(origin Point) Point {
	return Point{
		X: 111111 * math.Cos(origin.Y * math.Pi / 180) * (point.X - origin.X),
		Y: 111111 * (point.Y - origin.Y),
	}
}

// Converts from meters back to longitude/latitude
// origin should be the same point passed to LonLatToMeters (origin should be longitude/latitude)
func (point Point) MetersToLonLat(origin Point) Point {
	return Point{
		X: point.X / 111111 / math.Cos(origin.Y * math.Pi / 180) + origin.X,
		Y: point.Y / 111111 + origin.Y,
	}
}

func (point Point) Rectangle() Rectangle {
	return point.RectangleTol(0)
}

func (point Point) Bounds() Rectangle {
	return point.Rectangle()
}

func (point Point) RectangleTol(tol float64) Rectangle {
	t := Point{tol, tol}
	return Rectangle{
		Min: point.Sub(t),
		Max: point.Add(t),
	}
}

func (point Point) Dot(other Point) float64 {
	return point.X * other.X + point.Y * other.Y
}

func (point Point) Magnitude() float64 {
	return math.Sqrt(point.X * point.X + point.Y * point.Y)
}

func (point Point) Distance(other Point) float64 {
	dx := point.X - other.X
	dy := point.Y - other.Y
	return math.Sqrt(dx * dx + dy * dy)
}

func (point Point) Add(other Point) Point {
	return Point{point.X + other.X, point.Y + other.Y}
}

func (point Point) Sub(other Point) Point {
	return Point{point.X - other.X, point.Y - other.Y}
}

func (point Point) Scale(f float64) Point {
	return Point{f * point.X, f * point.Y}
}

func (point Point) MulPairwise(other Point) Point {
	return Point{point.X * other.X, point.Y * other.Y}
}

func (point Point) AngleTo(other Point) float64 {
	s := point.Dot(other) / point.Magnitude() / other.Magnitude()
	s = math.Max(-1, math.Min(1, s))
	angle := math.Acos(s)
	if angle > math.Pi {
		return 2 * math.Pi - angle
	} else {
		return angle
	}
}

func (point Point) SignedAngle(other Point) float64 {
	return math.Atan2(other.Y, other.X) - math.Atan2(point.Y, point.X)
}

// computes the z-coordinate of the cross product, assuming that
// both points are on the z=0 plane
func (point Point) Cross(other Point) float64 {
	return point.X * other.Y - point.Y * other.X
}

type Segment struct {
	Start Point
	End Point
}

func (segment Segment) Length() float64 {
	return segment.Start.Distance(segment.End)
}

func (segment Segment) Project(point Point, normalized bool) float64 {
	l := segment.Length()
	if l == 0 {
		return 0
	}
	t := point.Sub(segment.Start).Dot(segment.End.Sub(segment.Start)) / l / l
	t = math.Max(0, math.Min(1, t))
	if !normalized {
		t *= l
	}
	return t
}

func (segment Segment) ProjectPoint(point Point) Point {
	t := segment.Project(point, true)
	return segment.PointAtFactor(t, true)
}

func (segment Segment) PointAtFactor(factor float64, normalized bool) Point {
	if segment.Length() == 0 {
		return segment.Start
	}

	if !normalized {
		factor = factor / segment.Length()
	}
	return segment.Start.Add(segment.End.Sub(segment.Start).Scale(factor))
}

func (segment Segment) ProjectWithWidth(point Point, width float64) Point {
	proj := segment.ProjectPoint(point)
	d := point.Sub(proj)
	if d.Magnitude() < 1 {
		return point
	} else {
		d = d.Scale(1 / d.Magnitude())
		return proj.Add(d)
	}
}

func (segment Segment) Distance(point Point) float64 {
	p := segment.ProjectPoint(point)
	return p.Distance(point)
}

func (segment Segment) Vector() Point {
	return segment.End.Sub(segment.Start)
}

func (segment Segment) AngleTo(other Segment) float64 {
	return segment.Vector().AngleTo(other.Vector())
}

func (segment Segment) Bounds() Rectangle {
	return segment.Start.Rectangle().Extend(segment.End)
}

// 2D implementation of "On fast computation of distance between line segments" (V. Lumelsky)
func (segment Segment) DistanceToSegment(other Segment) float64 {
	d1 := segment.Vector()
	d2 := other.Vector()
	d12 := other.Start.Sub(segment.Start)

	r := d1.Dot(d2)
	s1 := d1.Dot(d12)
	s2 := d2.Dot(d12)
	mag1 := d1.Dot(d1)
	mag2 := d2.Dot(d2)

	if mag1 == 0 && mag2 == 0 {
		return segment.Start.Distance(other.Start)
	} else if mag1 == 0 {
		return other.Distance(segment.Start)
	} else if mag2 == 0 {
		return segment.Distance(other.Start)
	}

	denominator := mag1 * mag2 - r * r
	var t, u float64
	if denominator != 0 {
		t = (s1 * mag2 - s2 * r) / denominator
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
	}
	u = (t * r - s2) / mag2
	if u < 0 || u > 1 {
		if u < 0 {
			u = 0
		} else if u > 1 {
			u = 1
		}
		t = (u * r + s1) / mag1
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
	}
	dx := d1.X * t - d2.X * u - d12.X
	dy := d1.Y * t - d2.Y * u - d12.Y
	return math.Sqrt(dx * dx + dy * dy)
}

func (segment Segment) Line() Line {
	return Line{segment.Start, segment.End}
}

// from https://github.com/paulmach/go.geo/blob/master/line.go
func (segment Segment) Intersection(other Segment) *Point {
	d1 := segment.Vector()
	d2 := other.Vector()
	d12 := other.Start.Sub(segment.Start)

	den := d1.Y * d2.X - d1.X * d2.Y
	u1 := d1.X * d12.Y - d1.Y * d12.X
	u2 := d2.X * d12.Y - d2.Y * d12.X

	if den == 0 {
		// collinear
		if u1 == 0 && u2 == 0 {
			return &segment.Start
		} else {
			return nil
		}
	}

	if u1 / den < 0 || u1 / den > 1 || u2 / den < 0 || u2 / den > 1 {
		return nil
	}

	p := segment.PointAtFactor(u2 / den, true)
	return &p
}

// sample the segment discretely at some frequency (in terms of distance between points)
func (segment Segment) Sample(d float64) []Point {
	points := []Point{segment.Start}
	cur := segment.Start
	for cur.Distance(segment.End) > d {
		vector := segment.End.Sub(cur)
		vector = vector.Scale(d / vector.Magnitude())
		cur = cur.Add(vector)
		points = append(points, cur)
	}
	points = append(points, segment.End)
	return points
}

type Line struct {
	A Point
	B Point
}

func (line Line) ProjectPoint(point Point) Point {
	vector := line.B.Sub(line.A)
	t := point.Sub(line.A).Dot(vector) / vector.Magnitude() / vector.Magnitude()
	return line.A.Add(vector.Scale(t))
}

type Rectangle struct {
	Min Point
	Max Point
}

var EmptyRectangle Rectangle = Rectangle{
	Min: Point{math.Inf(1), math.Inf(1)},
	Max: Point{math.Inf(-1), math.Inf(-1)},
}

func Rect(sx, sy, ex, ey float64) Rectangle {
	return Rectangle{
		Point{sx, sy},
		Point{ex, ey},
	}
}

func (rect Rectangle) Extend(point Point) Rectangle {
	return Rectangle{
		Min: Point{
			X: math.Min(rect.Min.X, point.X),
			Y: math.Min(rect.Min.Y, point.Y),
		},
		Max: Point{
			X: math.Max(rect.Max.X, point.X),
			Y: math.Max(rect.Max.Y, point.Y),
		},
	}
}

func (rect Rectangle) ExtendRect(other Rectangle) Rectangle {
	return rect.Extend(other.Min).Extend(other.Max)
}

func (rect Rectangle) Contains(point Point) bool {
	return point.X >= rect.Min.X && point.X <= rect.Max.X && point.Y >= rect.Min.Y && point.Y <= rect.Max.Y
}

func (rect Rectangle) ContainsRect(other Rectangle) bool {
	return rect.Contains(other.Min) && rect.Contains(other.Max)
}

func (rect Rectangle) Lengths() Point {
	return rect.Max.Sub(rect.Min)
}

func (rect Rectangle) AddTol(tol float64) Rectangle {
	return Rectangle{
		Min: Point{
			X: rect.Min.X - tol,
			Y: rect.Min.Y - tol,
		},
		Max: Point{
			X: rect.Max.X + tol,
			Y: rect.Max.Y + tol,
		},
	}
}

func (rect Rectangle) Bounds() Rectangle {
	return rect
}

func (rect Rectangle) Intersects(other Rectangle) bool {
	return rect.Max.Y >= other.Min.Y && other.Max.Y >= rect.Min.Y && rect.Max.X >= other.Min.X && other.Max.X >= rect.Min.X
}

func (rect Rectangle) Diagonal() float64 {
	return rect.Min.Distance(rect.Max)
}

func (rect Rectangle) Center() Point {
	return rect.Min.Add(rect.Max).Scale(0.5)
}

func (rect Rectangle) Intersection(other Rectangle) Rectangle {
	intersection := Rectangle{
		Min: Point{math.Max(rect.Min.X, other.Min.X), math.Max(rect.Min.Y, other.Min.Y)},
		Max: Point{math.Min(rect.Max.X, other.Max.X), math.Min(rect.Max.Y, other.Max.Y)},
	}
	if intersection.Max.X <= intersection.Min.X {
		intersection.Max.X = intersection.Min.X
	}
	if intersection.Max.Y <= intersection.Min.Y {
		intersection.Max.Y = intersection.Min.Y
	}
	return intersection
}

func (rect Rectangle) Area() float64 {
	return (rect.Max.X - rect.Min.X) * (rect.Max.Y - rect.Min.Y)
}

func (rect Rectangle) ToPolygon() Polygon {
	return Polygon{
		rect.Min,
		Point{rect.Min.X, rect.Max.Y},
		rect.Max,
		Point{rect.Max.X, rect.Min.Y},
	}
}

type Polygon []Point

func (poly Polygon) Segments() []Segment {
	var segments []Segment
	for i := range poly {
		cur := poly[i]
		next := poly[(i+1)%len(poly)]
		segments = append(segments, Segment{cur, next})
	}
	return segments
}

func (poly Polygon) Bounds() Rectangle {
	r := EmptyRectangle
	for _, p := range poly {
		r = r.Extend(p)
	}
	return r
}

// Ray casting algorithm (https://stackoverflow.com/questions/217578/how-can-i-determine-whether-a-2d-point-is-within-a-polygon).
// We count the number of polygon segments that a ray from outside the polygon to p intersects.
// even -> p is outside, odd -> p is inside
func (poly Polygon) Contains(p Point) bool {
	bounds := poly.Bounds()
	if !bounds.Contains(p) {
		return false
	}
	// try a few times to get a rayStart that doesn't come close to any polygon point
	segments := poly.Segments()
	lengths := poly.Bounds().Lengths()
	magnitude := lengths.Magnitude()
	threshold := lengths.Scale(1/100).Magnitude()
	// smallBounds contains some padding, bigBounds contains more padding
	smallBounds := poly.Bounds().AddTol(threshold)
	bigBounds := poly.Bounds().AddTol(magnitude)
	sampleRayStart := func() Point {
		for {
			p := Point{bigBounds.Lengths().X * rand.Float64(), bigBounds.Lengths().Y * rand.Float64()}
			p = p.Add(bigBounds.Min)
			if !smallBounds.Contains(p) {
				return p
			}
		}
	}
	var ray Segment
	for i := 0; i < 5; i++ {
		rayStart := sampleRayStart()
		vector := rayStart.Sub(p)
		vector = vector.Scale(10*magnitude / vector.Magnitude())
		// not actually a ray
		ray = Segment{p, p.Add(vector)}
		good := true
		for _, p := range poly {
			if ray.Distance(p) < threshold {
				good = false
			}
		}
		if good {
			break
		}
	}
	// count intersections
	var count int = 0
	for _, segment := range segments {
		if segment.Intersection(ray) != nil {
			count++
		}
	}
	return count % 2 == 1
}

func (poly Polygon) Distance(p Point) float64 {
	if poly.Contains(p) {
		return 0
	}
	segments := poly.Segments()
	var minDistance = segments[0].Distance(p)
	for _, segment := range segments[1:] {
		minDistance = math.Min(minDistance, segment.Distance(p))
	}
	return minDistance
}

func (poly Polygon) SegmentIntersections(segment Segment) []Point {
	var intersections []Point
	for _, polySegment := range poly.Segments() {
		intersection := polySegment.Intersection(segment)
		if intersection != nil {
			intersections = append(intersections, *intersection)
		}
	}
	return intersections
}

// from https://github.com/Ch3ck/algo/blob/master/convexHull/convexHull.go
func GetConvexHull(points []Point) Polygon {
	if len(points) == 0 {
		return Polygon{}
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].X < points[j].X || (points[i].X == points[j].X && points[i].Y < points[j].Y)
	})

	hull := Polygon{points[0]}
	count := 1

	crossProduct := func(o Point, a Point, b Point) float64 {
		return a.Sub(o).Cross(b.Sub(o))
	}

	// find the lower hull
	for i := 1; i < len(points); i++ {
		// remove points which are not part of the lower hull
		for count > 1 && crossProduct(hull[count-2], hull[count-1], points[i]) < 0 {
			count--
			hull = hull[:count]
		}

		// add a new better point than the removed ones
		hull = append(hull, points[i])
		count++
	}

	// our base counter for the upper hull
	count0 := count

	// find the upper hull
	for i := len(points) - 2; i >= 0; i-- {
		// remove points which are not part of the upper hull
		for count-count0 > 0 && crossProduct(hull[count-2], hull[count-1], points[i]) < 0 {
			count--
			hull = hull[:count]
		}

		// add a new better point than the removed ones
		hull = append(hull, points[i])
		count++
	}

	return hull[0:len(hull)-1]
}
