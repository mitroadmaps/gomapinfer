package googlemaps

import (
	"../common"

	"math"
)

const ORIGIN_SHIFT = 2 * math.Pi * 6378137 / 2.0

// convert latitude/longitude to Spherical Mercator EPSG:900913
// source: http://gis.stackexchange.com/questions/46729/corner-coordinates-of-google-static-map-tile
func LonLatToMeters(lonLat common.Point) common.Point {
	mx := lonLat.X * ORIGIN_SHIFT / 180.0
	my := math.Log(math.Tan((90 + lonLat.Y) * math.Pi / 360.0)) / (math.Pi / 180.0)
	my = my * ORIGIN_SHIFT / 180.0
	return common.Point{mx, my}
}

func MetersToLonLat(meters common.Point) common.Point {
	lon := (meters.X / ORIGIN_SHIFT) * 180.0
	lat := (meters.Y / ORIGIN_SHIFT) * 180.0
	lat = 180 / math.Pi * (2 * math.Atan(math.Exp(lat * math.Pi / 180.0)) - math.Pi / 2.0)
	return common.Point{lon, lat}
}

func GetMetersPerPixel(zoom int) float64 {
	return 2 * math.Pi * 6378137 / math.Exp2(float64(zoom)) / 256
}

func LonLatToPixel(p common.Point, origin common.Point, zoom int) common.Point {
	p = LonLatToMeters(p).Sub(LonLatToMeters(origin))
	p = p.Scale(1 / GetMetersPerPixel(zoom)) // get pixel coordinates
	p = common.Point{p.X, -p.Y} // invert y axis to correspond to sat image orientation
	p = p.Add(common.Point{256, 256}) // sat image is offset a bit due to picking centers
	return p
}

func PixelToLonLat(p common.Point, origin common.Point, zoom int) common.Point {
	p = p.Sub(common.Point{256, 256})
	p = common.Point{p.X, -p.Y}
	p = p.Scale(GetMetersPerPixel(zoom))
	p = MetersToLonLat(p.Add(LonLatToMeters(origin)))
	return p
}

func LonLatToMapboxTile(lonLat common.Point, zoom int) [2]int {
	n := math.Exp2(float64(zoom))
	xtile := int((lonLat.X + 180) / 360 * n)
	ytile := int((1 - math.Log(math.Tan(lonLat.Y * math.Pi / 180) + (1 / math.Cos(lonLat.Y * math.Pi / 180))) / math.Pi) / 2 * n)
	return [2]int{xtile, ytile}
}

func LonLatToMapbox(lonLat common.Point, zoom int, originTile [2]int) common.Point {
	n := math.Exp2(float64(zoom))
	x := (lonLat.X + 180) / 360 * n
	y := (1 - math.Log(math.Tan(lonLat.Y * math.Pi / 180) + (1 / math.Cos(lonLat.Y * math.Pi / 180))) / math.Pi) / 2 * n
	xoff := x - float64(originTile[0])
	yoff := y - float64(originTile[1])
	return common.Point{xoff, yoff}.Scale(256)
}

func MapboxToLonLat(p common.Point, zoom int, originTile [2]int) common.Point {
	n := math.Exp2(float64(zoom))
	p = p.Scale(1.0/256).Add(common.Point{float64(originTile[0]), float64(originTile[1])})
	x := p.X * 360 / n - 180
	y := math.Atan(math.Sinh(math.Pi * (1 - 2 * p.Y / n)))
	y = y * 180 / math.Pi
	return common.Point{x, y}
}
