package imaging

import (
	"fmt"
	"image"
	"image/draw"
	"testing"

	"pgregory.net/rapid"
)

// The resampling and blurring passes were rewritten for speed: the vertical
// resize adds whole rows instead of walking columns, the nearest-neighbour
// downscale copies from an NRGBA source directly, and the horizontal resize and
// both blur passes premultiply a line by its alpha once instead of at every tap.
// The functions below are the passes as they were before, kept as the oracle.
// The vertical resize and the nearest-neighbour downscale must agree with them
// bit for bit; the premultiplied passes round the product a·c·w in another
// order, so they may differ from them by a level of rounding.

func refResizeHorizontal(img image.Image, width int, filter ResampleFilter) *image.NRGBA {
	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, width, src.h))
	weights := precomputeWeights(width, src.w, filter)
	scanLine := make([]uint8, src.w*4)
	for y := 0; y < src.h; y++ {
		src.scan(0, y, src.w, y+1, scanLine)
		for x := range weights {
			var r, g, b, a float64
			for _, w := range weights[x] {
				s := scanLine[w.index*4 : w.index*4+4]
				aw := float64(s[3]) * w.weight
				r += float64(s[0]) * aw
				g += float64(s[1]) * aw
				b += float64(s[2]) * aw
				a += aw
			}
			refStore(dst, x, y, r, g, b, a, 1)
		}
	}
	return dst
}

func refResizeVertical(img image.Image, height int, filter ResampleFilter) *image.NRGBA {
	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, src.w, height))
	weights := precomputeWeights(height, src.h, filter)
	scanLine := make([]uint8, src.h*4)
	for x := 0; x < src.w; x++ {
		src.scan(x, 0, x+1, src.h, scanLine)
		for y := range weights {
			var r, g, b, a float64
			for _, w := range weights[y] {
				s := scanLine[w.index*4 : w.index*4+4]
				aw := float64(s[3]) * w.weight
				r += float64(s[0]) * aw
				g += float64(s[1]) * aw
				b += float64(s[2]) * aw
				a += aw
			}
			refStore(dst, x, y, r, g, b, a, 1)
		}
	}
	return dst
}

func refResizeNearest(img image.Image, width, height int) *image.NRGBA {
	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	dx := float64(src.w) / float64(width)
	dy := float64(src.h) / float64(height)
	for y := 0; y < height; y++ {
		srcY := int((float64(y) + 0.5) * dy)
		for x := 0; x < width; x++ {
			srcX := int((float64(x) + 0.5) * dx)
			src.scan(srcX, srcY, srcX+1, srcY+1, dst.Pix[y*dst.Stride+x*4:y*dst.Stride+x*4+4])
		}
	}
	return dst
}

// refBlur1D blurs along x when horizontal is true and along y otherwise.
func refBlur1D(img image.Image, kernel []float64, horizontal bool) *image.NRGBA {
	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, src.w, src.h))
	radius := len(kernel) - 1
	line, n := src.h, src.w
	if horizontal {
		line, n = src.w, src.h
	}
	scanLine := make([]uint8, line*4)
	for k := 0; k < n; k++ {
		if horizontal {
			src.scan(0, k, src.w, k+1, scanLine)
		} else {
			src.scan(k, 0, k+1, src.h, scanLine)
		}
		for p := 0; p < line; p++ {
			var r, g, b, a, wsum float64
			for q := max(p-radius, 0); q <= min(p+radius, line-1); q++ {
				weight := kernel[absInt(p-q)]
				wsum += weight
				s := scanLine[q*4 : q*4+4]
				wa := float64(s[3]) * weight
				r += float64(s[0]) * wa
				g += float64(s[1]) * wa
				b += float64(s[2]) * wa
				a += wa
			}
			if horizontal {
				refStore(dst, p, k, r, g, b, a, wsum)
			} else {
				refStore(dst, k, p, r, g, b, a, wsum)
			}
		}
	}
	return dst
}

func refStore(dst *image.NRGBA, x, y int, r, g, b, a, wsum float64) {
	if a == 0 {
		return
	}
	d := dst.Pix[y*dst.Stride+x*4 : y*dst.Stride+x*4+4]
	aInv := 1 / a
	d[0] = clamp(r * aInv)
	d[1] = clamp(g * aInv)
	d[2] = clamp(b * aInv)
	d[3] = clamp(a / wsum)
}

// genAlphaNRGBA draws an image up to 24 x 24 whose alpha is often exactly 0 or
// 255, where the premultiplied and the per-tap paths are most likely to part.
func genAlphaNRGBA(t *rapid.T) *image.NRGBA {
	img := genNRGBA(t, "src")
	if rapid.Bool().Draw(t, "large") {
		w := rapid.IntRange(10, 24).Draw(t, "w")
		h := rapid.IntRange(10, 24).Draw(t, "h")
		img = &image.NRGBA{Pix: rapid.SliceOfN(rapid.Byte(), w*h*4, w*h*4).Draw(t, "pix"), Stride: w * 4, Rect: image.Rect(0, 0, w, h)}
	}
	for i := 3; i < len(img.Pix); i += 4 {
		switch rapid.IntRange(0, 3).Draw(t, "alphaKind") {
		case 0:
			img.Pix[i] = 0
		case 1:
			img.Pix[i] = 0xff
		}
	}
	return img
}

func genFilter(t *rapid.T) ResampleFilter {
	filters := []ResampleFilter{Box, Linear, Hermite, MitchellNetravali, CatmullRom, BSpline, Gaussian, Lanczos, Hann, Hamming, Blackman, Bartlett, Welch, Cosine}
	return rapid.SampledFrom(filters).Draw(t, "filter")
}

func TestDifferential_ResizeVerticalMatchesColumnPass(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		img := genAlphaNRGBA(t)
		h := rapid.IntRange(1, 40).Draw(t, "dstH")
		filter := genFilter(t)
		requireSame(t, "resizeVertical", resizeVertical(img, h, filter), refResizeVertical(img, h, filter), 0)
		// A source other than NRGBA goes through the scanner instead.
		rgba := &image.RGBA{Pix: img.Pix, Stride: img.Stride, Rect: img.Rect}
		requireSame(t, "resizeVertical(RGBA)", resizeVertical(rgba, h, filter), refResizeVertical(rgba, h, filter), 0)
	})
}

func TestDifferential_ResizeNearestMatchesScanner(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		img := genAlphaNRGBA(t)
		w := rapid.IntRange(1, 40).Draw(t, "dstW")
		h := rapid.IntRange(1, 40).Draw(t, "dstH")
		requireSame(t, "resizeNearest", resizeNearest(img, w, h), refResizeNearest(img, w, h), 0)
	})
}

func TestDifferential_ResizeHorizontalMatchesPerTapPass(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		img := genAlphaNRGBA(t)
		w := rapid.IntRange(1, 40).Draw(t, "dstW")
		filter := genFilter(t)
		requireSame(t, "resizeHorizontal", resizeHorizontal(img, w, filter), refResizeHorizontal(img, w, filter), 1)
	})
}

func TestDifferential_BlurMatchesPerTapPass(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		img := genAlphaNRGBA(t)
		sigma := rapid.Float64Range(0.1, 4).Draw(t, "sigma")
		radius := int(sigma*3 + 0.999999)
		kernel := make([]float64, radius+1)
		for i := range kernel {
			kernel[i] = gaussianBlurKernel(float64(i), sigma)
		}
		requireSame(t, "blurHorizontal", blurHorizontal(img, kernel), refBlur1D(img, kernel, true), 1)
		requireSame(t, "blurVertical", blurVertical(img, kernel), refBlur1D(img, kernel, false), 1)
	})
}

// TestDifferential_ClampSaturatingMatchesClamp checks that the two forms of
// clamp, one written for predictable values and one for unpredictable ones,
// round and clamp every value alike.
func TestDifferential_ClampSaturatingMatchesClamp(t *testing.T) {
	t.Parallel()
	for _, x := range []float64{0, 0.4999999999999999, 0.5, 1, 254.4, 254.5, 255, 255.5, -0.5, -1, -1e300, 1e300} {
		if a, b := clamp(x), clampSaturating(x); a != b {
			t.Fatalf("x=%v: clamp %d, clampSaturating %d", x, a, b)
		}
	}
	rapid.Check(t, func(t *rapid.T) {
		x := rapid.Float64Range(-1e6, 1e6).Draw(t, "x")
		if a, b := clamp(x), clampSaturating(x); a != b {
			t.Fatalf("x=%v: clamp %d, clampSaturating %d", x, a, b)
		}
	})
}

// TestDifferential_OverlayMatchesDrawOver compares Overlay at opacity 1 with
// image/draw's draw.Over, the Porter-Duff "over" that upstream issue
// disintegration/imaging#164 expected of Paste. Paste replaces the pixels it
// covers, transparent ones included; Overlay is the function that composites.
// draw.Over works on 16-bit premultiplied color and Overlay on 8-bit
// unpremultiplied color, so they may differ by a level of rounding.
func TestDifferential_OverlayMatchesDrawOver(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		bg := genNRGBA(t, "bg")
		img := genNRGBA(t, "img")
		b := bg.Bounds()
		pos := image.Pt(
			rapid.IntRange(b.Min.X-img.Bounds().Dx(), b.Max.X).Draw(t, "x"),
			rapid.IntRange(b.Min.Y-img.Bounds().Dy(), b.Max.Y).Draw(t, "y"),
		)
		want := atOrigin(bg)
		draw.Draw(want, image.Rectangle{Min: pos, Max: pos.Add(img.Bounds().Size())}.Sub(b.Min), img, img.Bounds().Min, draw.Over)
		requireSame(t, fmt.Sprintf("Overlay at %v vs draw.Over", pos), Overlay(bg, img, pos, 1), want, 1)
	})
}
