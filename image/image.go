package image

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"strings"
)

func getDecodeFunc(fname string) func(io.Reader) (image.Image, error) {
	lowerFname := strings.ToLower(fname)
	if strings.HasSuffix(lowerFname, ".png") {
		return png.Decode
	} else if strings.HasSuffix(lowerFname, ".jpg") || strings.HasSuffix(lowerFname, ".jpeg") {
		return jpeg.Decode
	} else {
		panic(fmt.Errorf("unknown image extension for file %s", fname))
	}
}

func getEncodeFunc(fname string) func(io.Writer, image.Image) error {
	lowerFname := strings.ToLower(fname)
	if strings.HasSuffix(lowerFname, ".png") {
		return png.Encode
	} else if strings.HasSuffix(lowerFname, ".jpg") || strings.HasSuffix(lowerFname, ".jpeg") {
		return func(w io.Writer, im image.Image) error {
			return jpeg.Encode(w, im, nil)
		}
	} else {
		panic(fmt.Errorf("unknown image extension for file %s", fname))
	}
}

func ReadImage(fname string) [][][3]uint8 {
	file, err := os.Open(fname)
	if err != nil {
		panic(err)
	}
	im, err := getDecodeFunc(fname)(file)
	if err != nil {
		panic(err)
	}
	file.Close()
	rect := im.Bounds()
	pix := make([][][3]uint8, rect.Max.X - rect.Min.X)
	for i := range pix {
		pix[i] = make([][3]uint8, rect.Max.Y - rect.Min.Y)
		for j := range pix[i] {
			r, g, b, _ := im.At(i + rect.Min.X, j + rect.Min.Y).RGBA()
			pix[i][j][0] = uint8(r >> 8)
			pix[i][j][1] = uint8(g >> 8)
			pix[i][j][2] = uint8(b >> 8)
		}
	}
	return pix
}

func ReadGrayImage(fname string) [][]uint8 {
	file, err := os.Open(fname)
	if err != nil {
		panic(err)
	}
	imraw, err := getDecodeFunc(fname)(file)
	if err != nil {
		panic(err)
	}
	file.Close()
	im := imraw.(*image.Gray)
	rect := im.Bounds()
	pix := make([][]uint8, rect.Max.X - rect.Min.X)
	for i := range pix {
		pix[i] = make([]uint8, rect.Max.Y - rect.Min.Y)
		for j := range pix[i] {
			c := im.GrayAt(i + rect.Min.X, j + rect.Min.Y)
			pix[i][j] = c.Y
		}
	}
	return pix
}

func MakeBin(x, y int) [][]bool {
	b := make([][]bool, x)
	for i := range b {
		b[i] = make([]bool, y)
	}
	return b
}

func MakeImage(x, y int, color [3]uint8) [][][3]uint8 {
	im := make([][][3]uint8, x)
	for i := range im {
		im[i] = make([][3]uint8, y)
		for j := range im[i] {
			im[i][j] = color
		}
	}
	return im
}

func MakeGrayImage(x, y int, color uint8) [][]uint8 {
	im := make([][]uint8, x)
	for i := range im {
		im[i] = make([]uint8, y)
		for j := range im[i] {
			im[i][j] = color
		}
	}
	return im
}

func MakeBinLike(o [][]bool) [][]bool {
	return MakeBin(len(o), len(o[0]))
}

func Binarize(pix [][]uint8, threshold uint8) [][]bool {
	b := MakeBin(len(pix), len(pix[0]))
	for i := range pix {
		for j := range pix[i] {
			b[i][j] = pix[i][j] > threshold
		}
	}
	return b
}

func Unbinarize(b [][]bool, value uint8) [][]uint8 {
	pix := make([][]uint8, len(b))
	for i := range b {
		pix[i] = make([]uint8, len(b[i]))
		for j := range b[i] {
			if b[i][j] {
				pix[i][j] = value
			}
		}
	}
	return pix
}

func Clone(b [][]bool) [][]bool {
	n := MakeBinLike(b)
	for i := range b {
		for j := range b[i] {
			n[i][j] = b[i][j]
		}
	}
	return n
}

func Dilate(b [][]bool, amount int) [][]bool {
	n1 := MakeBinLike(b)
	n2 := MakeBinLike(b)
	// dilate b -> n1 along first axis
	for i := range b {
		for j := range b[i] {
			if !b[i][j] {
				continue
			}
			for off := -amount; off <= amount; off++ {
				if i + off < 0 || i + off >= len(b) {
					continue
				}
				n1[i + off][j] = true
			}
		}
	}
	// dilate n1 -> n2 along second axis
	for i := range b {
		for j := range b[i] {
			if !n1[i][j] {
				continue
			}
			for off := -amount; off <= amount; off++ {
				if j + off < 0 || j + off >= len(b[i]) {
					continue
				}
				n2[i][j + off] = true
			}
		}
	}
	return n2
}

func AsImage(pix [][][3]uint8) image.Image {
	im := image.NewNRGBA(image.Rect(0, 0, len(pix), len(pix[0])))
	for i := range pix {
		for j := range pix[i] {
			r, g, b := pix[i][j][0], pix[i][j][1], pix[i][j][2]
			im.SetNRGBA(i, j, color.NRGBA{r, g, b, 255})
		}
	}
	return im
}

func WriteImage(fname string, pix [][][3]uint8) {
	im := image.NewNRGBA(image.Rect(0, 0, len(pix), len(pix[0])))
	for i := range pix {
		for j := range pix[i] {
			r, g, b := pix[i][j][0], pix[i][j][1], pix[i][j][2]
			im.SetNRGBA(i, j, color.NRGBA{r, g, b, 255})
		}
	}
	file, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	if err := getEncodeFunc(fname)(file, im); err != nil {
		panic(err)
	}
	file.Close()
}

func WriteGrayImage(fname string, pix [][]uint8) {
	im := image.NewGray(image.Rect(0, 0, len(pix), len(pix[0])))
	for i := range pix {
		for j := range pix[i] {
			im.SetGray(i, j, color.Gray{pix[i][j]})
		}
	}
	file, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	if err := getEncodeFunc(fname)(file, im); err != nil {
		panic(err)
	}
	file.Close()
}

func Floodfill(b [][]bool, seen [][]bool, i int, j int) [][2]int {
	var members [][2]int
	var f func(i int, j int)
	f = func(i int, j int) {
		if i < 0 || i >= len(b) || j < 0 || j >= len(b[i]) {
			return
		} else if !b[i][j] || seen[i][j] {
			return
		}
		members = append(members, [2]int{i, j})
		seen[i][j] = true
		f(i - 1, j - 1)
		f(i - 1, j)
		f(i - 1, j + 1)
		f(i, j - 1)
		f(i, j + 1)
		f(i + 1, j - 1)
		f(i + 1, j)
		f(i + 1, j + 1)
	}
	f(i, j)
	return members
}

func PointsToRect(points [][2]int) (int, int, int, int) {
	sx := points[0][0]
	sy := points[0][1]
	ex := points[0][0]
	ey := points[0][1]
	for _, p := range points {
		if p[0] < sx {
			sx = p[0]
		}
		if p[0] > ex {
			ex = p[0]
		}
		if p[1] < sy {
			sy = p[1]
		}
		if p[1] > ey {
			ey = p[1]
		}
	}
	return sx, sy, ex, ey
}

func Crop(pix [][][3]uint8, sx, sy, ex, ey int) [][][3]uint8 {
	npix := make([][][3]uint8, ex - sx)
	for i := range npix {
		npix[i] = make([][3]uint8, ey - sy)
		for j := range npix[i] {
			npix[i][j][0] = pix[sx + i][sy + j][0]
			npix[i][j][1] = pix[sx + i][sy + j][1]
			npix[i][j][2] = pix[sx + i][sy + j][2]
		}
	}
	return npix
}

func CropGray(pix [][]uint8, sx, sy, ex, ey int) [][]uint8 {
	npix := make([][]uint8, ex - sx)
	for i := range npix {
		npix[i] = make([]uint8, ey - sy)
		for j := range npix[i] {
			npix[i][j] = pix[sx + i][sy + j]
		}
	}
	return npix
}

func Not(pix [][]bool) [][]bool {
	npix := MakeBinLike(pix)
	for i := range npix {
		for j := range npix[i] {
			npix[i][j] = !pix[i][j]
		}
	}
	return npix
}

func And(pix1 [][]bool, pix2 [][]bool) [][]bool {
	out := MakeBinLike(pix1)
	for i := range out {
		for j := range out[i] {
			out[i][j] = pix1[i][j] && pix2[i][j]
		}
	}
	return out
}

func DrawRect(pix [][][3]uint8, cx int, cy int, side int, color [3]uint8) {
	for i := cx - side; i <= cx + side; i++ {
		for j := cy - side; j <= cy + side; j++ {
			if i < 0 || i >= len(pix) || j < 0 || j >= len(pix[i]) {
				continue
			}
			pix[i][j] = color
		}
	}
}

func FillRectangle(pix [][][3]uint8, sx int, sy int, ex int, ey int, color [3]uint8) {
	for i := sx; i < ex; i++ {
		for j := sy; j < ey; j++ {
			if i < 0 || i >= len(pix) || j < 0 || j >= len(pix[i]) {
				continue
			}
			pix[i][j] = color
		}
	}
}

func DrawRectangle(pix [][][3]uint8, sx int, sy int, ex int, ey int, width int, color [3]uint8) {
	FillRectangle(pix, sx-width, sy, sx+width, ey, color)
	FillRectangle(pix, ex-width, sy, ex+width, ey, color)
	FillRectangle(pix, sx, sy-width, ex, sy+width, color)
	FillRectangle(pix, sx, ey-width, ex, ey+width, color)
}

// draw rectangle, transparent where color = -1
func DrawTransparent(pix [][][3]uint8, cx int, cy int, side int, color [3]int) {
	for i := cx - side; i <= cx + side; i++ {
		for j := cy - side; j <= cy + side; j++ {
			if i < 0 || i >= len(pix) || j < 0 || j >= len(pix[i]) {
				continue
			}
			for k := range color {
				if color[k] == -1 {
					continue
				}
				pix[i][j][k] = uint8(color[k])
			}
		}
	}
}

func ToGray(pix [][][3]uint8) [][]uint8 {
	n := MakeGrayImage(len(pix), len(pix[0]), 0)
	for i := range n {
		for j := range n[i] {
			n[i][j] = 255 - pix[i][j][0]
		}
	}
	return n
}

func FromGray(pix [][]uint8) [][][3]uint8 {
	n := MakeImage(len(pix), len(pix[0]), [3]uint8{0, 0, 0})
	for i := range n {
		for j := range n[i] {
			n[i][j][0] = pix[i][j]
			n[i][j][1] = pix[i][j]
			n[i][j][2] = pix[i][j]
		}
	}
	return n
}
