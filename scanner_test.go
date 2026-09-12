package imaging

import (
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"testing"
)

func TestScanner(t *testing.T) {
	t.Parallel()

	rect := image.Rect(-1, -1, 15, 15)
	colors := palette.Plan9
	testCases := []struct {
		name string
		img  image.Image
	}{
		{
			name: "NRGBA",
			img:  makeNRGBAImage(rect, colors),
		},
		{
			name: "NRGBA64",
			img:  makeNRGBA64Image(rect, colors),
		},
		{
			name: "RGBA",
			img:  makeRGBAImage(rect, colors),
		},
		{
			name: "RGBA64",
			img:  makeRGBA64Image(rect, colors),
		},
		{
			name: "Gray",
			img:  makeGrayImage(rect, colors),
		},
		{
			name: "Gray16",
			img:  makeGray16Image(rect, colors),
		},
		{
			name: "YCbCr-444",
			img:  makeYCbCrImage(rect, colors, image.YCbCrSubsampleRatio444),
		},
		{
			name: "YCbCr-422",
			img:  makeYCbCrImage(rect, colors, image.YCbCrSubsampleRatio422),
		},
		{
			name: "YCbCr-420",
			img:  makeYCbCrImage(rect, colors, image.YCbCrSubsampleRatio420),
		},
		{
			name: "YCbCr-440",
			img:  makeYCbCrImage(rect, colors, image.YCbCrSubsampleRatio440),
		},
		{
			name: "YCbCr-410",
			img:  makeYCbCrImage(rect, colors, image.YCbCrSubsampleRatio410),
		},
		{
			name: "YCbCr-411",
			img:  makeYCbCrImage(rect, colors, image.YCbCrSubsampleRatio411),
		},
		{
			name: "Paletted",
			img:  makePalettedImage(rect, colors),
		},
		{
			name: "Alpha",
			img:  makeAlphaImage(rect, colors),
		},
		{
			name: "Alpha16",
			img:  makeAlpha16Image(rect, colors),
		},
		{
			name: "Generic",
			img:  makeGenericImage(rect, colors),
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := tc.img.Bounds()
			s := newScanner(tc.img)
			for y := r.Min.Y; y < r.Max.Y; y++ {
				buf := make([]byte, r.Dx()*4)
				s.scan(0, y-r.Min.Y, r.Dx(), y+1-r.Min.Y, buf)
				wantBuf := readRow(tc.img, y)
				if !compareBytes(buf, wantBuf, 1) {
					fmt.Println(tc.img)
					t.Fatalf("scan horizontal line (y=%d): got %v want %v", y, buf, wantBuf)
				}
			}
			for x := r.Min.X; x < r.Max.X; x++ {
				buf := make([]byte, r.Dy()*4)
				s.scan(x-r.Min.X, 0, x+1-r.Min.X, r.Dy(), buf)
				wantBuf := readColumn(tc.img, x)
				if !compareBytes(buf, wantBuf, 1) {
					t.Fatalf("scan vertical line (x=%d): got %v want %v", x, buf, wantBuf)
				}
			}
		})
	}
}

func makeYCbCrImage(rect image.Rectangle, colors []color.Color, sr image.YCbCrSubsampleRatio) *image.YCbCr {
	img := image.NewYCbCr(rect, sr)
	j := 0
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			iy := img.YOffset(x, y)
			ic := img.COffset(x, y)
			c := color.NRGBAModel.Convert(colors[j]).(color.NRGBA)
			img.Y[iy], img.Cb[ic], img.Cr[ic] = color.RGBToYCbCr(c.R, c.G, c.B)
			j++
		}
	}
	return img
}

func makeNRGBAImage(rect image.Rectangle, colors []color.Color) *image.NRGBA {
	img := image.NewNRGBA(rect)
	fillDrawImage(img, colors)
	return img
}

func makeNRGBA64Image(rect image.Rectangle, colors []color.Color) *image.NRGBA64 {
	img := image.NewNRGBA64(rect)
	fillDrawImage(img, colors)
	return img
}

func makeRGBAImage(rect image.Rectangle, colors []color.Color) *image.RGBA {
	img := image.NewRGBA(rect)
	fillDrawImage(img, colors)
	return img
}

func makeRGBA64Image(rect image.Rectangle, colors []color.Color) *image.RGBA64 {
	img := image.NewRGBA64(rect)
	fillDrawImage(img, colors)
	return img
}

func makeGrayImage(rect image.Rectangle, colors []color.Color) *image.Gray {
	img := image.NewGray(rect)
	fillDrawImage(img, colors)
	return img
}

func makeGray16Image(rect image.Rectangle, colors []color.Color) *image.Gray16 {
	img := image.NewGray16(rect)
	fillDrawImage(img, colors)
	return img
}

func makePalettedImage(rect image.Rectangle, colors []color.Color) *image.Paletted {
	img := image.NewPaletted(rect, colors)
	fillDrawImage(img, colors)
	return img
}

func makeAlphaImage(rect image.Rectangle, colors []color.Color) *image.Alpha {
	img := image.NewAlpha(rect)
	fillDrawImage(img, colors)
	return img
}

func makeAlpha16Image(rect image.Rectangle, colors []color.Color) *image.Alpha16 {
	img := image.NewAlpha16(rect)
	fillDrawImage(img, colors)
	return img
}

func makeGenericImage(rect image.Rectangle, colors []color.Color) image.Image {
	img := image.NewRGBA(rect)
	fillDrawImage(img, colors)
	type genericImage struct{ *image.RGBA }
	return &genericImage{img}
}

func fillDrawImage(img draw.Image, colors []color.Color) {
	colorsNRGBA := make([]color.NRGBA, len(colors))
	for i, c := range colors {
		nrgba := color.NRGBAModel.Convert(c).(color.NRGBA)
		nrgba.A = uint8(i % 256)
		colorsNRGBA[i] = nrgba
	}
	rect := img.Bounds()
	i := 0
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.Set(x, y, colorsNRGBA[i])
			i++
		}
	}
}

func readRow(img image.Image, y int) []uint8 {
	row := make([]byte, img.Bounds().Dx()*4)
	i := 0
	for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
		c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
		row[i+0] = c.R
		row[i+1] = c.G
		row[i+2] = c.B
		row[i+3] = c.A
		i += 4
	}
	return row
}

func readColumn(img image.Image, x int) []uint8 {
	column := make([]byte, img.Bounds().Dy()*4)
	i := 0
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
		column[i+0] = c.R
		column[i+1] = c.G
		column[i+2] = c.B
		column[i+3] = c.A
		i += 4
	}
	return column
}

// TestScannerMalformedPalette covers palettes that a decoder can hand over but
// that are not the well-formed 256-entry table the scanner used to assume.
// Each case panicked before the palette lookup was made total, and the first
// two panicked inside a goroutine started by parallel, where a caller cannot
// recover: the process died however carefully the call was wrapped.
func TestScannerMalformedPalette(t *testing.T) {
	t.Parallel()

	const (
		size  = 4
		known = 65 // a palette shorter than the 256 values a pixel byte can hold
	)
	shortPalette := make(color.Palette, known)
	for i := range shortPalette {
		shortPalette[i] = color.NRGBA{uint8(i), uint8(255 - i), 0, 0xff}
	}

	testCases := []struct {
		name string
		pal  color.Palette
		pix  uint8
		want color.NRGBA
	}{
		{
			name: "index inside a short palette keeps its colour",
			pal:  shortPalette,
			pix:  known - 1,
			want: color.NRGBA{known - 1, uint8(255 - (known - 1)), 0, 0xff},
		},
		{
			name: "index past a short palette is transparent",
			pal:  shortPalette,
			pix:  70,
			want: color.NRGBA{},
		},
		{
			name: "index into an empty palette is transparent",
			pal:  color.Palette{},
			pix:  0,
			want: color.NRGBA{},
		},
		{
			name: "a nil palette entry is transparent",
			pal:  color.Palette{color.NRGBA{1, 2, 3, 4}, nil},
			pix:  1,
			want: color.NRGBA{},
		},
		{
			name: "a defined entry beside a nil one keeps its colour",
			pal:  color.Palette{color.NRGBA{1, 2, 3, 4}, nil},
			pix:  0,
			want: color.NRGBA{1, 2, 3, 4},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			img := image.NewPaletted(image.Rect(0, 0, size, size), tc.pal)
			for i := range img.Pix {
				img.Pix[i] = tc.pix
			}

			dst := make([]uint8, size*size*4)
			newScanner(img).scan(0, 0, size, size, dst)

			for j := 0; j < len(dst); j += 4 {
				// j steps by 4 through a slice whose length is a multiple of 4.
				got := color.NRGBA{dst[j], dst[j+1], dst[j+2], dst[j+3]} //nolint:gosec // see above
				if got != tc.want {
					t.Fatalf("pixel %d: got %v, want %v", j/4, got, tc.want)
				}
			}
		})
	}
}

// TestScannerMatchesImageAt is the invariant behind the fast paths: whatever
// shortcut a scanner takes for a concrete image type, it has to report the same
// colour the image itself does. It is checked on alpha-premultiplied values
// because that is what is visible; comparing unpremultiplied channels instead
// would divide by a near-zero alpha and flag differences no one can see.
func TestScannerMatchesImageAt(t *testing.T) {
	t.Parallel()

	rects := []image.Rectangle{
		image.Rect(0, 0, 8, 8),
		image.Rect(0, 0, 7, 5),  // odd width and height
		image.Rect(0, 0, 1, 1),  // a single pixel
		image.Rect(3, 2, 11, 9), // an even origin away from zero
		image.Rect(3, 3, 10, 8), // an odd origin, which the chroma paths halve
		image.Rect(1, 1, 6, 4),
	}

	for _, rect := range rects {
		for name, img := range scannerTestImages(rect) {
			t.Run(fmt.Sprintf("%s%v", name, rect), func(t *testing.T) {
				t.Parallel()

				b := img.Bounds()
				w, h := b.Dx(), b.Dy()
				dst := make([]uint8, w*h*4)
				newScanner(img).scan(0, 0, w, h, dst)

				for y := 0; y < h; y++ {
					for x := 0; x < w; x++ {
						want := color.RGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.RGBA)
						j := (y*w + x) * 4
						got := premultiply(color.NRGBA{dst[j], dst[j+1], dst[j+2], dst[j+3]})
						// One unit of slack: the scanner rounds in 8-bit while
						// the model rounds in 16-bit, so the last bit can differ
						// on inputs neither of them got wrong.
						if channelDelta(got, want) > 1 {
							t.Fatalf("at (%d,%d): got %v, want %v", x, y, got, want)
						}
					}
				}
			})
		}
	}
}

// scannerTestImages builds one image per concrete type the scanner special
// cases, plus one it does not, so the default path is covered too.
func scannerTestImages(rect image.Rectangle) map[string]image.Image {
	colors := palette.Plan9
	images := map[string]image.Image{
		"NRGBA":    makeNRGBAImage(rect, colors),
		"NRGBA64":  makeNRGBA64Image(rect, colors),
		"RGBA":     makeRGBAImage(rect, colors),
		"RGBA64":   makeRGBA64Image(rect, colors),
		"Gray":     makeGrayImage(rect, colors),
		"Gray16":   makeGray16Image(rect, colors),
		"Paletted": makePalettedImage(rect, colors),
		"Alpha":    makeAlphaImage(rect, colors),
		"Generic":  makeGenericImage(rect, colors),
	}
	// 411 and 410 have no case of their own in scanYCbCr and fall to the
	// default, which asks the image for the offset. They are here to keep that
	// fallback honest: it is the only chroma path not computed inline.
	for name, ratio := range map[string]image.YCbCrSubsampleRatio{
		"YCbCr444": image.YCbCrSubsampleRatio444,
		"YCbCr422": image.YCbCrSubsampleRatio422,
		"YCbCr420": image.YCbCrSubsampleRatio420,
		"YCbCr440": image.YCbCrSubsampleRatio440,
		"YCbCr411": image.YCbCrSubsampleRatio411,
		"YCbCr410": image.YCbCrSubsampleRatio410,
	} {
		images[name] = makeYCbCrImage(rect, colors, ratio)
	}
	return images
}

// premultiply converts the scanner's unpremultiplied output to the
// alpha-premultiplied form the stdlib colour model reports.
func premultiply(c color.NRGBA) color.RGBA {
	return color.RGBA{
		uint8(uint32(c.R) * uint32(c.A) / 255),
		uint8(uint32(c.G) * uint32(c.A) / 255),
		uint8(uint32(c.B) * uint32(c.A) / 255),
		c.A,
	}
}

// channelDelta returns the largest per-channel difference between two colours.
func channelDelta(a, b color.RGBA) int {
	delta := 0
	for _, pair := range [][2]uint8{{a.R, b.R}, {a.G, b.G}, {a.B, b.B}, {a.A, b.A}} {
		d := int(pair[0]) - int(pair[1])
		if d < 0 {
			d = -d
		}
		if d > delta {
			delta = d
		}
	}
	return delta
}
