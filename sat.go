package main


import (
	//"../common"
	"github.com/mitroadmaps/gomapinfer/common"

	"bytes"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"net/http"
	"time"
)

const URL = "https://maps.googleapis.com/maps/api/staticmap?center=%f,%f&zoom=%d&size=576x576&maptype=satellite&key=%s"

/*
	To use Bing Static Map Maker instead of Google services,

	Comment off the above URL (line 17) and add the following line to the code.
	
	const URL = "https://dev.virtualearth.net/REST/V1/Imagery/Map/Aerial/%f,%f/%d?mapSize=576,576&format=png&key=%s"
*/

// Returns a 512x512 image centered at the point using the specified zoom level and API key
func GetSatelliteImage(point common.Point, zoom int, key string) image.Image {
	url := fmt.Sprintf(URL, point.Y, point.X, zoom, key)
	//fmt.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "image/png" {
		var errdesc string
		if resp.Header.Get("Content-Type") != "image/png" {
			if bytes, err := ioutil.ReadAll(resp.Body); err == nil {
				errdesc = string(bytes)
			}
		}
		if resp.StatusCode == 500 {
			fmt.Printf("warning: got 500 (errdesc=%s) on %s (retrying later)\n", errdesc, url)
			time.Sleep(time.Minute)
			return GetSatelliteImage(point, zoom, key)
		} else {
			panic(fmt.Errorf("got ssssssssstatus code %d (errdesc=%s)", resp.StatusCode, errdesc))
		}
	}
	imBytes, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		panic(err)
	}
	im, err := png.Decode(bytes.NewReader(imBytes))
	if err != nil {
		panic(err)
	}

	// crop out logo
	cropped := image.NewRGBA(image.Rect(0, 0, 512, 512))
	for i := 0; i < 512; i++ {
		for j := 0; j < 512; j++ {
			cropped.Set(i, j, im.At(i + 32, j + 32))
		}
	}

	return cropped
}
