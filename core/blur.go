package core

import "github.com/hajimehoshi/ebiten/v2"

// Blur settings: a box-blur repeated several times looks close to a
// gaussian blur. Bigger radius or more passes give a stronger effect.
const (
	blurRadius = 8
	blurPasses = 3
)

// blurred returns a new image with a blurred copy of src.
// The work happens on CPU over raw RGBA pixels, so it is meant for
// one-time use (like a frozen frame), not for every frame.
func blurred(src *ebiten.Image) *ebiten.Image {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	buf := make([]byte, 4*w*h)
	src.ReadPixels(buf)
	boxBlur(buf, w, h, blurRadius, blurPasses)
	out := ebiten.NewImage(w, h)
	out.WritePixels(buf)
	return out
}

// boxBlur blurs the RGBA buffer in place. Each pass runs a horizontal
// and then a vertical blur, so the filter stays separable and fast.
func boxBlur(pix []byte, w, h, radius, passes int) {
	if radius < 1 || w == 0 || h == 0 {
		return
	}
	tmp := make([]byte, len(pix))
	for range passes {
		boxBlurH(pix, tmp, w, h, radius)
		boxBlurV(tmp, pix, w, h, radius)
	}
}

// boxBlurH writes into dst the horizontal average of src with window
// 2*r+1. A sliding sum keeps the cost independent of the radius, and
// edge pixels are repeated at the borders.
func boxBlurH(src, dst []byte, w, h, r int) {
	win := 2*r + 1
	for y := range h {
		row := y * w * 4
		for c := range 4 {
			sum := 0
			for k := -r; k <= r; k++ {
				sum += int(src[row+clampInt(k, 0, w-1)*4+c])
			}
			for x := range w {
				dst[row+x*4+c] = byte(sum / win)
				sum += int(src[row+clampInt(x+r+1, 0, w-1)*4+c]) -
					int(src[row+clampInt(x-r, 0, w-1)*4+c])
			}
		}
	}
}

// boxBlurV is the vertical version of boxBlurH.
func boxBlurV(src, dst []byte, w, h, r int) {
	win := 2*r + 1
	for x := range w {
		col := x * 4
		for c := range 4 {
			sum := 0
			for k := -r; k <= r; k++ {
				sum += int(src[clampInt(k, 0, h-1)*w*4+col+c])
			}
			for y := range h {
				dst[y*w*4+col+c] = byte(sum / win)
				sum += int(src[clampInt(y+r+1, 0, h-1)*w*4+col+c]) -
					int(src[clampInt(y-r, 0, h-1)*w*4+col+c])
			}
		}
	}
}

// clampInt keeps v inside the [lo, hi] range.
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
