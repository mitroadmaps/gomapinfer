package common

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/qedus/osmpbf"
)

var HIGHWAY_BLACKLIST []string = []string{
	"pedestrian",
	"footway",
	"bridleway",
	"steps",
	"path",
	"sidewalk",
	"cycleway",
	"proposed",
	"construction",
	"bus_stop",
	"crossing",
	"elevator",
	"emergency_access_point",
	"escape",
	"give_way",
}

func IsOSMBlacklisted(highway string) bool {
	for _, x := range HIGHWAY_BLACKLIST {
		if highway == x {
			return true
		}
	}
	return false
}

func IsOSMBlacklistedWithList(highway string, blacklist []string) bool {
	for _, x := range blacklist {
		if highway == x {
			return true
		}
	}
	return false
}

func LoadOSM(path string, bounds Rectangle) (*Graph, error) {
	graphs, err := LoadOSMMultiple(path, []Rectangle{bounds}, OSMOptions{})
	if err != nil {
		return nil, err
	} else {
		return graphs[0], nil
	}
}

type OSMOptions struct {
	Verbose bool
	EdgeWidths []map[int]float64
	NoParking bool
	NoTunnels bool
	LayerEdges []map[int]bool
	EdgeTags []map[int]map[string]string
	NodeTags []map[int]map[string]string
	OneWay bool
	OnlyMotorways bool
	MotorwayEdges []map[int]bool
	TunnelEdges []map[int]bool
	Bytes []byte
	CustomBlacklist []string
	IncludeRailway bool
}

/*func LoadOSMMultiple(path string, regions []Rectangle, options OSMOptions) ([]*Graph, error) {
	graphs := make([]*Graph, len(regions))
	for i := range graphs {
		graphs[i] = &Graph{}
	}
	vertexIDMaps := make([]map[int64]*Node, len(regions))
	for i := range vertexIDMaps {
		vertexIDMaps[i] = make(map[int64]*Node)
	}
	vertexRegionMap := make(map[int64][]int)

	// do two passes through the OSM data:
	//  1) collect vertices in the bounds
	//  2) collect edges from the OSM ways
	process := func(f func(v interface{})) error {
		var d *osmpbf.Decoder
		if options.Bytes == nil {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("error opening %s: %v", path, err)
			}
			defer file.Close()
			d = osmpbf.NewDecoder(file)
		} else {
			d = osmpbf.NewDecoder(bytes.NewBuffer(options.Bytes))
		}
		d.SetBufferSize(osmpbf.MaxBlobSize)
		nthreads := runtime.GOMAXPROCS(-1)
		if nthreads > 1 {
			nthreads = 1
		}
		d.Start(nthreads)
		for {
			if v, err := d.Decode(); err == io.EOF {
				break
			} else if err != nil {
				return fmt.Errorf("decode error: %v", err)
			} else {
				f(v)
			}
		}
		return nil
	}

	var count int64 = 0
	vertexStartTime := time.Now()
	err := process(func(v interface{}) {
		switch v := v.(type) {
		case *osmpbf.Node:
			point := Point{v.Lon, v.Lat}
			for i := range regions {
				if regions[i].Contains(point) {
					vertexIDMaps[i][v.ID] = graphs[i].AddNode(point)
					vertexRegionMap[v.ID] = append(vertexRegionMap[v.ID], i)
				}
			}
			count++
			if options.Verbose && count % 10000000 == 0 {
				fmt.Printf("finished %dM vertices (%d/sec)\n", count / 1000000, count / int64(time.Now().Sub(vertexStartTime).Seconds() + 1))
			}
		}
	})
	if err != nil {
		return nil, err
	}

	count = 0
	err = process(func(v interface{}) {
		switch v := v.(type) {
		case *osmpbf.Way:
			highway, ok := v.Tags["highway"]
			if !ok || IsOSMBlacklisted(highway) {
				return
			} else if len(v.NodeIDs) < 2 {
				return
			}

			if options.NoParking {
				if v.Tags["amenity"] == "parking" || v.Tags["service"] == "parking_aisle" {
					return
				} else if v.Tags["service"] == "driveway" {
					return
				}
			}
			isTunnel := (len(v.Tags["layer"]) >= 2 && v.Tags["layer"][0] == '-') || v.Tags["tunnel"] == "yes"
			if options.NoTunnels && isTunnel {
				return
			}
			isMotorway := v.Tags["highway"] == "motorway" || v.Tags["highway"] == "trunk"
			if options.OnlyMotorways && !isMotorway {
				return
			}

			// determine oneway, 0 for no, 1 for forward, -1 for reverse
			oneway := 0
			if options.OneWay {
				// (1) if oneway tag is set, use that exclusively
				//     (note that this overrides (2) since some ways can have motorway but
				//      use tag set to "no" to disable oneway)
				// (2) based on other tags that are default oneway
				if v.Tags["oneway"] != "" {
					if v.Tags["oneway"] == "yes" || v.Tags["oneway"] == "1" {
						oneway = 1
					} else if v.Tags["oneway"] == "-1" {
						oneway = -1
					}
				} else if v.Tags["highway"] == "motorway" || v.Tags["junction"] == "roundabout" {
					oneway = 1
				}
			}

			type RegionEdge struct {
				Edge *Edge
				RegionID int
			}

			var wayEdges []RegionEdge
			var lastVertexID int64 = v.NodeIDs[0]
			for _, vertexID := range v.NodeIDs[1:] {
				for _, regionID := range vertexRegionMap[vertexID] {
					node1 := vertexIDMaps[regionID][lastVertexID]
					node2 := vertexIDMaps[regionID][vertexID]
					if node1 != nil && node2 != nil {
						if oneway == 0 {
							edge := graphs[regionID].AddBidirectionalEdge(node1, node2)
							wayEdges = append(
								wayEdges,
								RegionEdge{edge[0], regionID},
								RegionEdge{edge[1], regionID},
							)
						} else if oneway == 1 {
							edge := graphs[regionID].AddEdge(node1, node2)
							wayEdges = append(wayEdges, RegionEdge{edge, regionID})
						} else if oneway == -1 {
							edge := graphs[regionID].AddEdge(node2, node1)
							wayEdges = append(wayEdges, RegionEdge{edge, regionID})
						} else {
							panic(fmt.Errorf("invalid oneway %d", oneway))
						}
					}
				}
				lastVertexID = vertexID
			}

			if len(options.EdgeWidths) > 0 {
				var width float64
				if val, ok := v.Tags["lanes"]; ok {
					lanes, _ := strconv.ParseFloat(strings.Split(val, ";")[0], 64)
					if lanes == 1 {
						width = 6.6
					} else {
						width = lanes * 3.7
					}
				} else if val, ok := v.Tags["width"]; ok {
					width, _ = strconv.ParseFloat(strings.Fields(strings.Split(val, ";")[0])[0], 64)
				} else {
					width = 6.6
				}
				for _, redge := range wayEdges {
					options.EdgeWidths[redge.RegionID][redge.Edge.ID] = width
				}
			}

			if len(options.LayerEdges) > 0 && v.Tags["layer"] != "" {
				for _, redge := range wayEdges {
					options.LayerEdges[redge.RegionID][redge.Edge.ID] = true
				}
			}

			if len(options.EdgeTags) > 0 {
				for _, redge := range wayEdges {
					options.EdgeTags[redge.RegionID][redge.Edge.ID] = v.Tags
				}
			}

			if len(options.MotorwayEdges) > 0 && isMotorway {
				for _, redge := range wayEdges {
					options.MotorwayEdges[redge.RegionID][redge.Edge.ID] = true
				}
			}

			if len(options.TunnelEdges) > 0 && isTunnel {
				for _, redge := range wayEdges {
					options.TunnelEdges[redge.RegionID][redge.Edge.ID] = true
				}
			}

			count++
			if options.Verbose && count % 100000 == 0 {
				fmt.Printf("finished %dK ways\n", count / 1000)
			}
		}
	})
	if err != nil {
		return nil, err
	}

	return graphs, nil
}*/


func DecodeOSM(path string, options OSMOptions, f func(v interface{})) error {
	var d *osmpbf.Decoder
	if options.Bytes == nil {
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("error opening %s: %v", path, err)
		}
		defer file.Close()
		d = osmpbf.NewDecoder(file)
	} else {
		d = osmpbf.NewDecoder(bytes.NewBuffer(options.Bytes))
	}
	d.SetBufferSize(osmpbf.MaxBlobSize)
	nthreads := runtime.GOMAXPROCS(-1)
	/*if nthreads > 1 {
		nthreads = 1
	}*/
	d.Start(nthreads)
	for {
		if v, err := d.Decode(); err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("decode error: %v", err)
		} else {
			f(v)
		}
	}
	return nil
}

func LoadOSMMultiple(path string, regions []Rectangle, options OSMOptions) ([]*Graph, error) {
	return LoadOSMMultiple2(path, regions, options)
}

const OSM_INDEX_SCALE = 2

// New version improves performance when there are many bounding boxes.
func LoadOSMMultiple2(path string, regions []Rectangle, options OSMOptions) ([]*Graph, error) {
	graphs := make([]*Graph, len(regions))
	for i := range graphs {
		graphs[i] = &Graph{}
	}
	vertexIDMaps := make([]map[int64]*Node, len(regions))
	for i := range vertexIDMaps {
		vertexIDMaps[i] = make(map[int64]*Node)
	}
	vertexRegionMap := make(map[int64][]int)

	// binary grid index over cells that we are interested in for the regions
	regionIndex := make(map[[2]int][]int)
	for regionID, region := range regions {
		sx := int(region.Min.X * OSM_INDEX_SCALE)
		sy := int(region.Min.Y * OSM_INDEX_SCALE)
		ex := int(region.Max.X * OSM_INDEX_SCALE)
		ey := int(region.Max.Y * OSM_INDEX_SCALE)
		for x := sx; x <= ex; x++ {
			for y := sy; y <= ey; y++ {
				regionIndex[[2]int{x, y}] = append(regionIndex[[2]int{x, y}], regionID)
			}
		}
	}

	// do two passes through the OSM data:
	//  1) collect vertices in the bounds
	//  2) collect edges from the OSM ways

	var count int64 = 0
	vertexStartTime := time.Now()
	err := DecodeOSM(path, options, func(v interface{}) {
		switch v := v.(type) {
		case *osmpbf.Node:
			point := Point{v.Lon, v.Lat}
			x, y := int(point.X * OSM_INDEX_SCALE), int(point.Y * OSM_INDEX_SCALE)

			type RegionVertex struct {
				Vertex *Node
				RegionID int
			}
			var nodeVertices []RegionVertex

			for _, regionID := range regionIndex[[2]int{x, y}] {
				if regions[regionID].Contains(point) {
					vertex := graphs[regionID].AddNode(point)
					vertexIDMaps[regionID][v.ID] = vertex
					vertexRegionMap[v.ID] = append(vertexRegionMap[v.ID], regionID)
					nodeVertices = append(nodeVertices, RegionVertex{
						Vertex: vertex,
						RegionID: regionID,
					})
				}
			}
			count++
			if len(options.NodeTags) > 0 {
				for _, rvertex := range nodeVertices {
					options.NodeTags[rvertex.RegionID][rvertex.Vertex.ID] = v.Tags
				}
			}
			if options.Verbose && count % 10000000 == 0 {
				fmt.Printf("finished %dM vertices (%d/sec)\n", count / 1000000, count / int64(time.Now().Sub(vertexStartTime).Seconds() + 1))
			}
		}
	})
	if err != nil {
		return nil, err
	}

	blacklist := HIGHWAY_BLACKLIST
	if options.CustomBlacklist != nil {
		blacklist = options.CustomBlacklist
	}

	count = 0
	err = DecodeOSM(path, options, func(v interface{}) {
		switch v := v.(type) {
		case *osmpbf.Way:
			highway, ok := v.Tags["highway"]
			if !ok && options.IncludeRailway {
				_, ok = v.Tags["railway"]
			}
			if !ok || IsOSMBlacklistedWithList(highway, blacklist) {
				return
			} else if len(v.NodeIDs) < 2 {
				return
			}

			if options.NoParking {
				if v.Tags["amenity"] == "parking" || v.Tags["service"] == "parking_aisle" {
					return
				} else if v.Tags["service"] == "driveway" {
					return
				}
			}
			isTunnel := (len(v.Tags["layer"]) >= 2 && v.Tags["layer"][0] == '-') || v.Tags["tunnel"] == "yes"
			if options.NoTunnels && isTunnel {
				return
			}
			isMotorway := v.Tags["highway"] == "motorway" || v.Tags["highway"] == "trunk"
			if options.OnlyMotorways && !isMotorway {
				return
			}

			// determine oneway, 0 for no, 1 for forward, -1 for reverse
			oneway := 0
			if options.OneWay {
				// (1) if oneway tag is set, use that exclusively
				//     (note that this overrides (2) since some ways can have motorway but
				//      use tag set to "no" to disable oneway)
				// (2) based on other tags that are default oneway
				if v.Tags["oneway"] != "" {
					if v.Tags["oneway"] == "yes" || v.Tags["oneway"] == "1" {
						oneway = 1
					} else if v.Tags["oneway"] == "-1" {
						oneway = -1
					}
				} else if v.Tags["highway"] == "motorway" || v.Tags["junction"] == "roundabout" {
					oneway = 1
				}
			}

			type RegionEdge struct {
				Edge *Edge
				RegionID int
			}

			var wayEdges []RegionEdge
			var lastVertexID int64 = v.NodeIDs[0]
			for _, vertexID := range v.NodeIDs[1:] {
				for _, regionID := range vertexRegionMap[vertexID] {
					node1 := vertexIDMaps[regionID][lastVertexID]
					node2 := vertexIDMaps[regionID][vertexID]
					if node1 != nil && node2 != nil {
						if oneway == 0 {
							edge := graphs[regionID].AddBidirectionalEdge(node1, node2)
							wayEdges = append(
								wayEdges,
								RegionEdge{edge[0], regionID},
								RegionEdge{edge[1], regionID},
							)
						} else if oneway == 1 {
							edge := graphs[regionID].AddEdge(node1, node2)
							wayEdges = append(wayEdges, RegionEdge{edge, regionID})
						} else if oneway == -1 {
							edge := graphs[regionID].AddEdge(node2, node1)
							wayEdges = append(wayEdges, RegionEdge{edge, regionID})
						} else {
							panic(fmt.Errorf("invalid oneway %d", oneway))
						}
					}
				}
				lastVertexID = vertexID
			}

			if len(options.EdgeWidths) > 0 {
				var width float64
				if val, ok := v.Tags["lanes"]; ok {
					lanes, _ := strconv.ParseFloat(strings.Split(val, ";")[0], 64)
					if lanes == 1 {
						width = 6.6
					} else {
						width = lanes * 3.7
					}
				} else if val, ok := v.Tags["width"]; ok {
					width, _ = strconv.ParseFloat(strings.Fields(strings.Split(val, ";")[0])[0], 64)
				} else {
					width = 6.6
				}
				for _, redge := range wayEdges {
					options.EdgeWidths[redge.RegionID][redge.Edge.ID] = width
				}
			}

			if len(options.LayerEdges) > 0 && v.Tags["layer"] != "" {
				for _, redge := range wayEdges {
					options.LayerEdges[redge.RegionID][redge.Edge.ID] = true
				}
			}

			if len(options.EdgeTags) > 0 {
				for _, redge := range wayEdges {
					options.EdgeTags[redge.RegionID][redge.Edge.ID] = v.Tags
				}
			}

			if len(options.MotorwayEdges) > 0 && isMotorway {
				for _, redge := range wayEdges {
					options.MotorwayEdges[redge.RegionID][redge.Edge.ID] = true
				}
			}

			if len(options.TunnelEdges) > 0 && isTunnel {
				for _, redge := range wayEdges {
					options.TunnelEdges[redge.RegionID][redge.Edge.ID] = true
				}
			}

			count++
			if options.Verbose && count % 100000 == 0 {
				fmt.Printf("finished %dK ways\n", count / 1000)
			}
		}
	})
	if err != nil {
		return nil, err
	}

	return graphs, nil
}
