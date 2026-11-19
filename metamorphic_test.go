package imaging

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"testing"

	"pgregory.net/rapid"
)

// The tests in this file have no expected image to compare against. Each one
// changes the input in a way whose effect on the output is known -- flipping it,
// moving its origin, storing it as another image type, applying an operation
// before rather than after another -- and checks the output changed exactly
// that way. rapid draws the images, and shrinks a failure to the smallest one.

// genNRGBA draws an image of 1..9 x 1..9 pixels whose bounds start at an
// arbitrary point, so that code assuming Min is (0, 0) is exercised.
func genNRGBA(t *rapid.T, label string) *image.NRGBA {
	w := rapid.IntRange(1, 9).Draw(t, label+".w")
	h := rapid.IntRange(1, 9).Draw(t, label+".h")
	x0 := rapid.IntRange(-4, 4).Draw(t, label+".x0")
	y0 := rapid.IntRange(-4, 4).Draw(t, label+".y0")
	pix := rapid.SliceOfN(rapid.Byte(), w*h*4, w*h*4).Draw(t, label+".pix")
	return &image.NRGBA{Pix: pix, Stride: w * 4, Rect: image.Rect(x0, y0, x0+w, y0+h)}
}

// genOpaque is genNRGBA with every alpha set to 255.
func genOpaque(t *rapid.T, label string) *image.NRGBA {
	img := genNRGBA(t, label)
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 0xff
	}
	return img
}

// atOrigin copies img to an image of the same pixels whose bounds start at (0, 0).
func atOrigin(img *image.NRGBA) *image.NRGBA {
	b := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		copy(dst.Pix[y*dst.Stride:y*dst.Stride+b.Dx()*4], img.Pix[img.PixOffset(b.Min.X, b.Min.Y+y):])
	}
	return dst
}

// diffNRGBA describes the first difference between got and want larger than
// tolerance in any channel, or returns "" when there is none.
//
// With a tolerance, color is compared premultiplied by alpha, which is what
// can be seen of it. Resampling divides the weighted color by the weighted
// alpha, so where the alpha is a rounding away from zero the color is that
// division's noise; premultiplied, it is next to nothing, as it looks. A level
// of rounding in the color and one in the alpha can add up, so premultiplied
// colors may differ by twice the tolerance.
func diffNRGBA(got, want *image.NRGBA, tolerance int) string {
	if got.Bounds().Size() != want.Bounds().Size() {
		return fmt.Sprintf("size %v, want %v", got.Bounds().Size(), want.Bounds().Size())
	}
	g, w := atOrigin(got), atOrigin(want)
	for i := range g.Pix {
		d := int(g.Pix[i]) - int(w.Pix[i])
		limit := tolerance
		if alpha := i - i%4 + 3; tolerance > 0 && i != alpha {
			d = (int(g.Pix[i])*int(g.Pix[alpha]) - int(w.Pix[i])*int(w.Pix[alpha])) / 255
			limit = 2 * tolerance
		}
		if d < -limit || d > limit {
			p, dx := i/4, g.Bounds().Dx()
			return fmt.Sprintf("pixel (%d, %d) channel %d is %d, want %d", p%dx, p/dx, i%4, g.Pix[i], w.Pix[i])
		}
	}
	return ""
}

func requireSame(t *rapid.T, what string, got, want *image.NRGBA, tolerance int) {
	t.Helper()
	if d := diffNRGBA(got, want, tolerance); d != "" {
		t.Fatalf("%s: %s", what, d)
	}
}

type imageOp struct {
	name string
	fn   func(image.Image) *image.NRGBA
}

var geometricOps = []imageOp{
	{"FlipH", FlipH},
	{"FlipV", FlipV},
	{"Transpose", Transpose},
	{"Transverse", Transverse},
	{"Rotate90", Rotate90},
	{"Rotate180", Rotate180},
	{"Rotate270", Rotate270},
}

// pixelOps change each pixel from its own value alone, so they cannot tell
// where a pixel is.
var pixelOps = []imageOp{
	{"Grayscale", Grayscale},
	{"Invert", Invert},
	{"AdjustBrightness(-30)", func(img image.Image) *image.NRGBA { return AdjustBrightness(img, -30) }},
	{"AdjustContrast(40)", func(img image.Image) *image.NRGBA { return AdjustContrast(img, 40) }},
	{"AdjustGamma(0.7)", func(img image.Image) *image.NRGBA { return AdjustGamma(img, 0.7) }},
	{"AdjustSaturation(50)", func(img image.Image) *image.NRGBA { return AdjustSaturation(img, 50) }},
	{"AdjustHue(75)", func(img image.Image) *image.NRGBA { return AdjustHue(img, 75) }},
	{"AdjustSigmoid(0.5, 4)", func(img image.Image) *image.NRGBA { return AdjustSigmoid(img, 0.5, 4) }},
}

// TestMetamorphic_GeometricIdentities checks the group the eight orientations
// form: each operation is undone by its inverse, and the composite ones are the
// composition their names promise.
func TestMetamorphic_GeometricIdentities(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		img := genNRGBA(t, "img")
		want := Clone(img)

		requireSame(t, "FlipH(FlipH(x))", FlipH(FlipH(img)), want, 0)
		requireSame(t, "FlipV(FlipV(x))", FlipV(FlipV(img)), want, 0)
		requireSame(t, "Transpose(Transpose(x))", Transpose(Transpose(img)), want, 0)
		requireSame(t, "Transverse(Transverse(x))", Transverse(Transverse(img)), want, 0)
		requireSame(t, "Rotate270(Rotate90(x))", Rotate270(Rotate90(img)), want, 0)
		requireSame(t, "Rotate90 four times", Rotate90(Rotate90(Rotate90(Rotate90(img)))), want, 0)
		requireSame(t, "Rotate180(Rotate180(x))", Rotate180(Rotate180(img)), want, 0)

		requireSame(t, "Rotate180 vs FlipV(FlipH(x))", Rotate180(img), FlipV(FlipH(img)), 0)
		requireSame(t, "Rotate90 vs Rotate180 then Rotate270", Rotate90(img), Rotate270(Rotate180(img)), 0)
		requireSame(t, "Transpose vs FlipH then Rotate90", Transpose(img), Rotate90(FlipH(img)), 0)
		requireSame(t, "Transverse vs FlipV then Rotate90", Transverse(img), Rotate90(FlipV(img)), 0)
	})
}

// TestMetamorphic_RotateByRightAngles checks Rotate against the exact
// quarter-turn functions, for the angle and for the same angle a whole turn
// away in either direction.
func TestMetamorphic_RotateByRightAngles(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		img := genNRGBA(t, "img")
		turns := rapid.IntRange(-2, 2).Draw(t, "turns")
		bg := color.NRGBA{R: 1, G: 2, B: 3, A: 4}
		for _, tc := range []struct {
			angle float64
			want  *image.NRGBA
		}{
			{0, Clone(img)},
			{90, Rotate90(img)},
			{180, Rotate180(img)},
			{270, Rotate270(img)},
		} {
			angle := tc.angle + float64(turns)*360
			requireSame(t, fmt.Sprintf("Rotate(x, %v)", angle), Rotate(img, angle, bg), tc.want, 0)
		}
	})
}

// TestMetamorphic_PixelOpsCommuteWithGeometry checks that an operation which
// reads one pixel at a time gives the same image whether the image is
// reoriented before or after it.
func TestMetamorphic_PixelOpsCommuteWithGeometry(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		img := genNRGBA(t, "img")
		p := rapid.SampledFrom(pixelOps).Draw(t, "pixelOp")
		g := rapid.SampledFrom(geometricOps).Draw(t, "geometricOp")
		requireSame(t, p.name+" after "+g.name+" vs before it", p.fn(g.fn(img)), g.fn(p.fn(img)), 0)
	})
}

// TestMetamorphic_ResultDoesNotDependOnTheOrigin checks every operation on an
// image whose bounds start away from (0, 0) against the same pixels stored at
// the origin. Where an image starts is not part of what it shows.
func TestMetamorphic_ResultDoesNotDependOnTheOrigin(t *testing.T) {
	t.Parallel()
	ops := append(append([]imageOp{
		{"Clone", Clone},
		{"Blur(0.8)", func(img image.Image) *image.NRGBA { return Blur(img, 0.8) }},
		{"Sharpen(0.8)", func(img image.Image) *image.NRGBA { return Sharpen(img, 0.8) }},
		{"Resize(5, 3, Linear)", func(img image.Image) *image.NRGBA { return Resize(img, 5, 3, Linear) }},
		{"Resize(4, 0, Lanczos)", func(img image.Image) *image.NRGBA { return Resize(img, 4, 0, Lanczos) }},
		{"Resize(3, 7, NearestNeighbor)", func(img image.Image) *image.NRGBA { return Resize(img, 3, 7, NearestNeighbor) }},
		{"Fit(4, 4, Box)", func(img image.Image) *image.NRGBA { return Fit(img, 4, 4, Box) }},
		{"Fill(3, 3, Center, CatmullRom)", func(img image.Image) *image.NRGBA { return Fill(img, 3, 3, Center, CatmullRom) }},
		{"Rotate(30)", func(img image.Image) *image.NRGBA { return Rotate(img, 30, color.Black) }},
		{"CropCenter(2, 2)", func(img image.Image) *image.NRGBA { return CropCenter(img, 2, 2) }},
		{"Convolve3x3", func(img image.Image) *image.NRGBA {
			return Convolve3x3(img, [9]float64{0, -1, 0, -1, 5, -1, 0, -1, 0}, nil)
		}},
	}, geometricOps...), pixelOps...)
	rapid.Check(t, func(t *rapid.T) {
		img := genNRGBA(t, "img")
		op := rapid.SampledFrom(ops).Draw(t, "op")
		requireSame(t, op.name, op.fn(img), op.fn(atOrigin(img)), 0)
	})
}

// TestMetamorphic_ResultDoesNotDependOnTheImageType stores the same opaque
// pixels as *image.RGBA and *image.NRGBA. Without transparency the two hold
// identical bytes, so every operation has to give the same image for both.
func TestMetamorphic_ResultDoesNotDependOnTheImageType(t *testing.T) {
	t.Parallel()
	ops := append(append([]imageOp{
		{"Clone", Clone},
		{"Blur(1)", func(img image.Image) *image.NRGBA { return Blur(img, 1) }},
		{"Resize(6, 2, Lanczos)", func(img image.Image) *image.NRGBA { return Resize(img, 6, 2, Lanczos) }},
	}, geometricOps...), pixelOps...)
	rapid.Check(t, func(t *rapid.T) {
		img := genOpaque(t, "img")
		rgba := &image.RGBA{Pix: append([]byte(nil), img.Pix...), Stride: img.Stride, Rect: img.Rect}
		op := rapid.SampledFrom(ops).Draw(t, "op")
		requireSame(t, op.name, op.fn(rgba), op.fn(img), 0)
	})
}

// TestMetamorphic_GeometryKeepsTheHistogram checks that reorienting an image
// moves its pixels without changing any of them.
func TestMetamorphic_GeometryKeepsTheHistogram(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		img := genNRGBA(t, "img")
		g := rapid.SampledFrom(geometricOps).Draw(t, "geometricOp")
		if got, want := Histogram(g.fn(img)), Histogram(img); got != want {
			t.Fatalf("Histogram(%s(x)) = %v, want %v", g.name, got, want)
		}
	})
}

// TestMetamorphic_ResizeCommutesWithFlips checks that resampling treats both
// directions alike: every filter is symmetric, so flipping an image before
// resizing it gives the flip of the resized image, up to two levels of rounding.
//
// Two, not one, because Resize changes both sides in two passes with a byte
// image between them. The mirrored weights agree only to the last bit (0.3 and
// 0.29999999999999998 for 2 to 6 wide), so a value landing exactly on a half,
// 0.3*215 = 64.5, rounds to 65 one way and 64 the other in the first pass, and
// the second pass carries that level through a filter whose weights reach 1.1
// before rounding again.
//
// NearestNeighbor and Box are left out. Both jump at the edge of their support:
// nearest neighbor picks one pixel rather than weighing several, and a sample
// point falling exactly between two pixels (8 to 4 wide puts every one there)
// takes the right-hand one, which after a flip is the other one; Box takes a
// pixel lying exactly on its edge in or leaves it out depending on which side
// the floating-point error of the sample position falls. The other filters go
// to zero at their edge, so the same error moves the result by a rounding.
func TestMetamorphic_ResizeCommutesWithFlips(t *testing.T) {
	t.Parallel()
	filters := []struct {
		name   string
		filter ResampleFilter
	}{
		{"Linear", Linear}, {"CatmullRom", CatmullRom},
		{"Lanczos", Lanczos}, {"Gaussian", Gaussian}, {"MitchellNetravali", MitchellNetravali},
	}
	rapid.Check(t, func(t *rapid.T) {
		img := genNRGBA(t, "img")
		f := rapid.SampledFrom(filters).Draw(t, "filter")
		w := rapid.IntRange(1, 12).Draw(t, "width")
		h := rapid.IntRange(1, 12).Draw(t, "height")
		const tolerance = 2
		resize := func(x image.Image) *image.NRGBA { return Resize(x, w, h, f.filter) }
		requireSame(t, "Resize(FlipH(x)) vs FlipH(Resize(x)), "+f.name, resize(FlipH(img)), FlipH(resize(img)), tolerance)
		requireSame(t, "Resize(FlipV(x)) vs FlipV(Resize(x)), "+f.name, resize(FlipV(img)), FlipV(resize(img)), tolerance)
	})
}

// TestMetamorphic_UniformImagesStayUniform checks that resampling and blurring
// an image of one color gives an image of that color: the weights of every
// filter add up to one.
func TestMetamorphic_UniformImagesStayUniform(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		c := color.NRGBA{
			R: rapid.Byte().Draw(t, "r"), G: rapid.Byte().Draw(t, "g"),
			B: rapid.Byte().Draw(t, "b"), A: 0xff,
		}
		img := New(rapid.IntRange(1, 9).Draw(t, "w"), rapid.IntRange(1, 9).Draw(t, "h"), c)
		w := rapid.IntRange(1, 12).Draw(t, "width")
		h := rapid.IntRange(1, 12).Draw(t, "height")
		for _, f := range []ResampleFilter{NearestNeighbor, Box, Linear, CatmullRom, Lanczos, Gaussian, MitchellNetravali} {
			requireSame(t, "Resize of a uniform image", Resize(img, w, h, f), New(w, h, c), 1)
		}
		requireSame(t, "Blur of a uniform image", Blur(img, 1.5), New(img.Bounds().Dx(), img.Bounds().Dy(), c), 1)
	})
}

// TestMetamorphic_PasteThenCrop checks that pasting an image and cropping the
// pasted area out again gives back the image, and that the result keeps the
// background's size and pixels outside that area.
func TestMetamorphic_PasteThenCrop(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		bg := genNRGBA(t, "bg")
		img := genNRGBA(t, "img")
		b := bg.Bounds()
		pos := image.Pt(
			rapid.IntRange(b.Min.X-img.Bounds().Dx(), b.Max.X).Draw(t, "x"),
			rapid.IntRange(b.Min.Y-img.Bounds().Dy(), b.Max.Y).Draw(t, "y"),
		)
		got := Paste(bg, img, pos)
		if got.Bounds().Size() != b.Size() {
			t.Fatalf("Paste at %v gave size %v, want the background's %v", pos, got.Bounds().Size(), b.Size())
		}

		// The same paste done by image/draw, which is the reference here. It
		// goes through premultiplied color, which keeps an opaque pixel exact
		// and nothing else, so the reference is given an opaque image.
		opaque := genOpaque(t, "opaque")
		want := atOrigin(bg)
		draw.Draw(want, image.Rectangle{Min: pos, Max: pos.Add(opaque.Bounds().Size())}.Sub(b.Min), opaque, opaque.Bounds().Min, draw.Src)
		requireSame(t, fmt.Sprintf("Paste at %v vs draw.Draw with draw.Src", pos), Paste(bg, opaque, pos), want, 0)

		inside := image.Rectangle{Min: pos, Max: pos.Add(img.Bounds().Size())}
		if inside.In(b) {
			requireSame(t, "Crop(Paste(bg, x, p), area at p)", Crop(got, inside.Sub(b.Min)), img, 0)
		}
	})
}

// TestMetamorphic_OverlayAtTheEnds checks the two opacities whose answer is
// known: nothing at 0, and a plain paste at 1 for an opaque image.
func TestMetamorphic_OverlayAtTheEnds(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		bg := genNRGBA(t, "bg")
		img := genNRGBA(t, "img")
		pos := image.Pt(rapid.IntRange(-6, 6).Draw(t, "x"), rapid.IntRange(-6, 6).Draw(t, "y"))
		requireSame(t, "Overlay with opacity 0", Overlay(bg, img, pos, 0), Clone(bg), 0)

		opaque := genOpaque(t, "opaque")
		requireSame(t, "Overlay of an opaque image with opacity 1 vs Paste",
			Overlay(bg, opaque, pos, 1), Paste(bg, opaque, pos), 0)
	})
}

// TestMetamorphic_PNGRoundTrip checks that writing an image as PNG and reading
// it back gives the same pixels: PNG is lossless.
func TestMetamorphic_PNGRoundTrip(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		img := genNRGBA(t, "img")
		var buf bytes.Buffer
		if err := Encode(&buf, img, PNG); err != nil {
			t.Fatal(err)
		}
		got, err := Decode(&buf)
		if err != nil {
			t.Fatal(err)
		}
		requireSame(t, "Decode(Encode(x, PNG))", Clone(got), Clone(img), 0)
	})
}
