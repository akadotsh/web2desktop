package generator

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

func makeDefaultIcon() ([]byte, error) {
	img := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	navy := color.NRGBA{R: 23, G: 36, B: 63, A: 255}
	white := color.NRGBA{R: 248, G: 250, B: 255, A: 255}
	blue := color.NRGBA{R: 55, G: 119, B: 232, A: 255}
	ink := color.NRGBA{R: 34, G: 56, B: 91, A: 255}
	fillRoundRect(img, 0, 0, 256, 256, 48, navy)
	fillRoundRect(img, 24, 27, 232, 229, 24, white)
	fillRoundRect(img, 36, 39, 220, 83, 14, ink)
	fillCircle(img, 55, 61, 5, color.NRGBA{R: 255, G: 107, B: 107, A: 255})
	fillCircle(img, 73, 61, 5, color.NRGBA{R: 255, G: 195, B: 75, A: 255})
	fillCircle(img, 91, 61, 5, color.NRGBA{R: 82, G: 203, B: 143, A: 255})
	fillCircle(img, 128, 153, 50, blue)
	drawEllipse(img, 128, 153, 21, 48, white)
	drawEllipse(img, 128, 153, 46, 20, white)
	return encodePNG(img)
}

func encodePNG(img image.Image) ([]byte, error) {
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func fillRoundRect(img *image.NRGBA, x0, y0, x1, y1, radius int, c color.NRGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			cx, cy := x, y
			if cx < x0+radius {
				cx = x0 + radius
			}
			if cx >= x1-radius {
				cx = x1 - radius - 1
			}
			if cy < y0+radius {
				cy = y0 + radius
			}
			if cy >= y1-radius {
				cy = y1 - radius - 1
			}
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= radius*radius {
				img.SetNRGBA(x, y, c)
			}
		}
	}
}

func fillCircle(img *image.NRGBA, cx, cy, radius int, c color.NRGBA) {
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= radius*radius {
				img.SetNRGBA(x, y, c)
			}
		}
	}
}

func drawEllipse(img *image.NRGBA, cx, cy, rx, ry int, c color.NRGBA) {
	for angle := 0.0; angle < 2*math.Pi; angle += 0.004 {
		x := cx + int(math.Round(float64(rx)*math.Cos(angle)))
		y := cy + int(math.Round(float64(ry)*math.Sin(angle)))
		img.SetNRGBA(x, y, c)
		img.SetNRGBA(x+1, y, c)
		img.SetNRGBA(x, y+1, c)
	}
}
