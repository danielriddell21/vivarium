package gui

import "image/color"

var (
	colBackground = color.RGBA{0x12, 0x16, 0x14, 0xff}
	colFood       = color.RGBA{0x3c, 0xb0, 0x43, 0xff}
	colHerbivore  = color.RGBA{0x4f, 0x9d, 0xff, 0xff}
	colCarnivore  = color.RGBA{0xe0, 0x4f, 0x4f, 0xff}
	colSelected   = color.RGBA{0xff, 0xe0, 0x4f, 0xff}
	colText       = color.RGBA{0xe6, 0xe6, 0xe6, 0xff}
	colPanel      = color.RGBA{0x00, 0x00, 0x00, 0xc0}
	colObstacle   = color.RGBA{0x33, 0x37, 0x3b, 0xff}
	colFood2      = color.RGBA{0x3a, 0xa8, 0x9a, 0xff}
)

const (
	graphW      = 280.0
	graphH      = 90.0
	graphMargin = 8.0
)

func nightTint(light float64) color.RGBA {
	t := (light - 0.45) / 0.55 // normalise the usual [0.45,1] range to [0,1]
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	lerp := func(night, day uint8) uint8 { return uint8(float64(night) + (float64(day)-float64(night))*t) }
	return color.RGBA{lerp(0x05, 0x12), lerp(0x07, 0x16), lerp(0x14, 0x14), 0xff}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
