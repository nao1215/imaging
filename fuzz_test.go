package imaging

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fuzzMaxPixels caps the pixel count of any image a fuzz target is willing to
// decode or produce. Image headers are free to claim billions of pixels in a
// handful of bytes, and the decoders allocate what the header says, so without
// a cap the fuzzer spends its time exhausting memory instead of finding bugs.
const fuzzMaxPixels = 1 << 20

// fuzzEncodedSeeds returns a tiny image encoded in every format imaging can
// write, so the decode fuzzer starts from valid files of each kind.
func fuzzEncodedSeeds(tb testing.TB) [][]byte {
	tb.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for i := range img.Pix {
		img.Pix[i] = uint8(i * 17)
	}
	formats := []Format{JPEG, PNG, GIF, TIFF, BMP}
	seeds := make([][]byte, 0, len(formats))
	for _, format := range formats {
		var buf bytes.Buffer
		if err := Encode(&buf, img, format); err != nil {
			tb.Fatalf("encode %v: %v", format, err)
		}
		seeds = append(seeds, buf.Bytes())
	}
	return seeds
}

// FuzzDecode feeds arbitrary bytes to Decode, with and without
// AutoOrientation, which is what Open does with the contents of a file.
//
// Properties:
//   - neither path panics;
//   - both paths agree on whether the bytes are an image;
//   - ReadOrientation only ever reports one of the nine defined values;
//   - auto-orientation keeps the pixel count and swaps width and height
//     exactly for the orientations that rotate by a quarter turn.
func FuzzDecode(f *testing.F) {
	for _, seed := range fuzzEncodedSeeds(f) {
		f.Add(seed)
		f.Add(seed[:len(seed)/2])
	}
	for i := 0; i <= 8; i++ {
		data, err := os.ReadFile(filepath.Join("testdata", "orientation_"+string(rune('0'+i))+".jpg"))
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	f.Add([]byte{})
	f.Add([]byte{0xff, 0xd8, 0xff, 0xe1, 0x00, 0x10, 'E', 'x', 'i', 'f', 0, 0, 'M', 'M', 0, 0x2a, 0xff, 0xff, 0xff, 0xff})

	f.Fuzz(func(t *testing.T, data []byte) {
		orient := ReadOrientation(bytes.NewReader(data))
		if orient < OrientationUnspecified || orient > OrientationRotate90 {
			t.Fatalf("ReadOrientation returned %d, want 0..8", orient)
		}

		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			// Still exercise the decoders on bytes whose header does not
			// parse: they must fail cleanly, not panic.
			if _, err := Decode(bytes.NewReader(data)); err == nil {
				t.Fatal("Decode accepted bytes that DecodeConfig rejected")
			}
			if _, err := Decode(bytes.NewReader(data), AutoOrientation(true)); err == nil {
				t.Fatal("Decode with AutoOrientation accepted bytes that DecodeConfig rejected")
			}
			return
		}
		if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > fuzzMaxPixels || cfg.Height > fuzzMaxPixels ||
			cfg.Width*cfg.Height > fuzzMaxPixels {
			t.Skip("image header claims too many pixels for a fuzz run")
		}

		plain, plainErr := Decode(bytes.NewReader(data))
		oriented, orientedErr := Decode(bytes.NewReader(data), AutoOrientation(true))
		if (plainErr == nil) != (orientedErr == nil) {
			t.Fatalf("Decode error %v but Decode with AutoOrientation error %v", plainErr, orientedErr)
		}
		if plainErr != nil {
			return
		}

		pw, ph := plain.Bounds().Dx(), plain.Bounds().Dy()
		ow, oh := oriented.Bounds().Dx(), oriented.Bounds().Dy()
		switch orient {
		case OrientationTranspose, OrientationRotate270, OrientationTransverse, OrientationRotate90:
			if ow != ph || oh != pw {
				t.Fatalf("orientation %d: got %dx%d from a %dx%d image, want width and height swapped", orient, ow, oh, pw, ph)
			}
		default:
			if ow != pw || oh != ph {
				t.Fatalf("orientation %d: got %dx%d from a %dx%d image, want the same size", orient, ow, oh, pw, ph)
			}
		}
	})
}

// FuzzFormatFromFilename checks the parsing of file names and extensions that
// Save uses to pick an encoder.
//
// Properties:
//   - FormatFromFilename is FormatFromExtension applied to filepath.Ext;
//   - an unsupported name yields ErrUnsupportedFormat and the value -1;
//   - the result does not depend on letter case or on a leading dot;
//   - a recognised format's name parses back to the same format.
func FuzzFormatFromFilename(f *testing.F) {
	for _, name := range []string{
		"a.jpg", "a.JPEG", "dir/a.png", "a.gif", "a.tif", "a.TIFF", "a.bmp",
		"a.webp", "noext", ".png", "a.", "a.tar.gz", "dir.jpg/file", "a.Jpg", "ｊｐｇ.ｐｎｇ",
	} {
		f.Add(name)
	}

	f.Fuzz(func(t *testing.T, name string) {
		ext := filepath.Ext(name)
		fromName, errName := FormatFromFilename(name)
		fromExt, errExt := FormatFromExtension(ext)
		if fromName != fromExt || !errors.Is(errName, errExt) {
			t.Fatalf("FormatFromFilename(%q) = %v, %v; FormatFromExtension(%q) = %v, %v", name, fromName, errName, ext, fromExt, errExt)
		}

		format, err := FormatFromExtension(name)
		if err != nil {
			if !errors.Is(err, ErrUnsupportedFormat) || format != -1 {
				t.Fatalf("FormatFromExtension(%q) = %v, %v; want -1, ErrUnsupportedFormat", name, format, err)
			}
			return
		}
		for _, variant := range []string{strings.ToUpper(name), strings.ToLower(name), "." + strings.TrimPrefix(name, ".")} {
			got, err := FormatFromExtension(variant)
			if err != nil || got != format {
				t.Fatalf("FormatFromExtension(%q) = %v, %v; but %q gave %v", variant, got, err, name, format)
			}
		}
		back, err := FormatFromExtension(format.String())
		if err != nil || back != format {
			t.Fatalf("format %v: its name %q parses to %v, %v", format, format.String(), back, err)
		}
	})
}

// fuzzSource builds a small source image. The origin is moved off (0, 0) when
// offset is set, because the geometry helpers work in the image's own
// coordinates and a zero origin would hide mistakes in that arithmetic.
func fuzzSource(w, h int, offset bool) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = uint8(i*31 + 7)
	}
	if !offset {
		return img
	}
	shifted := *img
	shifted.Rect = img.Rect.Add(image.Pt(-5, 11))
	return &shifted
}

func fuzzFilters() []ResampleFilter {
	return []ResampleFilter{
		NearestNeighbor, Box, Linear, Hermite, MitchellNetravali, CatmullRom, BSpline,
		Gaussian, Bartlett, Lanczos, Hann, Hamming, Blackman, Welch, Cosine,
	}
}

// fuzzDim folds an arbitrary int into [-4, 251] so that negative and zero
// sizes are still reached but no output grows beyond a few megabytes.
func fuzzDim(v int) int {
	m := v % 256
	if m < 0 {
		m += 256
	}
	return m - 4
}

// FuzzResizeDimensions drives the resizing and cropping functions with
// arbitrary target sizes, rectangles and angles.
//
// Properties (besides not panicking):
//   - Resize returns the requested size, filling in a zero side from the
//     aspect ratio, and an empty image for a negative or all-zero request;
//   - Fit never exceeds the box and returns the source unchanged when it fits;
//   - Fill and Thumbnail return exactly the requested size;
//   - CropAnchor returns the requested size clipped to the source;
//   - Crop returns the intersection with the source and its first pixel is
//     the source pixel at the intersection's corner;
//   - Rotate returns an image no larger than the source's bounding box at
//     that angle.
func FuzzResizeDimensions(f *testing.F) {
	f.Add(uint8(16), uint8(9), 8, 0, 0, uint8(9), false, 1, 2, 10, 7, 30.0)
	f.Add(uint8(1), uint8(64), 200, 3, 3, uint8(3), true, -3, -3, 3, 3, 90.0)
	f.Add(uint8(100), uint8(100), 50, 150, 0, uint8(0), false, 0, 0, 0, 0, -45.5)
	f.Add(uint8(0), uint8(5), 0, 0, 0, uint8(1), true, 5, 5, 1, 1, math.Inf(1))
	f.Add(uint8(7), uint8(3), -1, 5, 0, uint8(8), false, 100, 100, -100, -100, math.NaN())

	f.Fuzz(func(t *testing.T, sw, sh uint8, w, h int, filterIdx int, anchor uint8, offset bool,
		x0, y0, x1, y1 int, angle float64,
	) {
		srcW, srcH := int(sw%128), int(sh%128)
		w, h = fuzzDim(w), fuzzDim(h)
		filters := fuzzFilters()
		filter := filters[(filterIdx%len(filters)+len(filters))%len(filters)]
		anc := Anchor(anchor % 10) // 9 is not a defined anchor and must act like Center.

		if srcW == 0 || srcH == 0 {
			empty := image.NewNRGBA(image.Rect(0, 0, srcW, srcH))
			for name, got := range map[string]*image.NRGBA{
				"Resize":     Resize(empty, w, h, filter),
				"Fit":        Fit(empty, w, h, filter),
				"Fill":       Fill(empty, w, h, anc, filter),
				"CropAnchor": CropAnchor(empty, w, h, anc),
				"Rotate":     Rotate(empty, angle, color.Black),
			} {
				if !got.Bounds().Empty() {
					t.Fatalf("%s of an empty image returned %v", name, got.Bounds())
				}
			}
			return
		}

		src := fuzzSource(srcW, srcH, offset)
		b := src.Bounds()

		// Resize.
		got := Resize(src, w, h, filter).Bounds()
		switch {
		case w < 0 || h < 0 || w == 0 && h == 0:
			if !got.Empty() {
				t.Fatalf("Resize(%dx%d, %d, %d) = %v, want empty", srcW, srcH, w, h, got)
			}
		default:
			wantW, wantH := w, h
			if wantW == 0 {
				wantW = int(math.Max(1, math.Floor(float64(h)*float64(srcW)/float64(srcH)+0.5)))
			}
			if wantH == 0 {
				wantH = int(math.Max(1, math.Floor(float64(w)*float64(srcH)/float64(srcW)+0.5)))
			}
			if got != image.Rect(0, 0, wantW, wantH) {
				t.Fatalf("Resize(%dx%d, %d, %d) = %v, want %dx%d at the origin", srcW, srcH, w, h, got, wantW, wantH)
			}
		}

		// Fit, Fill and Thumbnail.
		fit := Fit(src, w, h, filter).Bounds()
		fill := Fill(src, w, h, anc, filter).Bounds()
		thumb := Thumbnail(src, w, h, filter).Bounds()
		if w <= 0 || h <= 0 {
			if !fit.Empty() || !fill.Empty() || !thumb.Empty() {
				t.Fatalf("Fit/Fill/Thumbnail(%d, %d) = %v, %v, %v, want empty", w, h, fit, fill, thumb)
			}
		} else {
			if fit.Min != (image.Point{}) || fit.Dx() < 1 || fit.Dy() < 1 || fit.Dx() > w || fit.Dy() > h {
				t.Fatalf("Fit(%dx%d, %d, %d) = %v, want a non-empty image within the box", srcW, srcH, w, h, fit)
			}
			if srcW <= w && srcH <= h && (fit.Dx() != srcW || fit.Dy() != srcH) {
				t.Fatalf("Fit(%dx%d, %d, %d) = %v, want the source size", srcW, srcH, w, h, fit)
			}
			want := image.Rect(0, 0, w, h)
			if fill != want || thumb != want {
				t.Fatalf("Fill/Thumbnail(%dx%d, %d, %d) = %v, %v, want %v", srcW, srcH, w, h, fill, thumb, want)
			}
		}

		// CropAnchor.
		ca := CropAnchor(src, w, h, anc).Bounds()
		if w <= 0 || h <= 0 {
			if !ca.Empty() {
				t.Fatalf("CropAnchor(%dx%d, %d, %d) = %v, want empty", srcW, srcH, w, h, ca)
			}
		} else if ca != image.Rect(0, 0, min(w, srcW), min(h, srcH)) {
			t.Fatalf("CropAnchor(%dx%d, %d, %d, %d) = %v, want %dx%d", srcW, srcH, w, h, anc, ca, min(w, srcW), min(h, srcH))
		}

		// Crop with an arbitrary, possibly inverted or far-off rectangle.
		rect := image.Rectangle{Min: image.Pt(x0, y0), Max: image.Pt(x1, y1)}
		inter := rect.Intersect(b)
		cropped := Crop(src, rect)
		if inter.Empty() {
			if !cropped.Bounds().Empty() {
				t.Fatalf("Crop(%v, %v) = %v, want empty", b, rect, cropped.Bounds())
			}
		} else {
			if cropped.Bounds() != image.Rect(0, 0, inter.Dx(), inter.Dy()) {
				t.Fatalf("Crop(%v, %v) = %v, want %dx%d", b, rect, cropped.Bounds(), inter.Dx(), inter.Dy())
			}
			want := color.NRGBAModel.Convert(src.At(inter.Min.X, inter.Min.Y))
			if gotC := cropped.At(0, 0); gotC != want {
				t.Fatalf("Crop(%v, %v).At(0, 0) = %v, want %v", b, rect, gotC, want)
			}
		}

		// Rotate: bounded by the source's bounding box at any angle.
		rot := Rotate(src, angle, color.Transparent).Bounds()
		if rot.Min != (image.Point{}) {
			t.Fatalf("Rotate(%v) = %v, want an image at the origin", angle, rot)
		}
		limit := int(math.Ceil(math.Hypot(float64(srcW), float64(srcH)))) + 2
		if rot.Dx() > limit || rot.Dy() > limit {
			t.Fatalf("Rotate(%dx%d, %v) = %v, larger than the %d px diagonal", srcW, srcH, angle, rot, limit)
		}
	})
}
