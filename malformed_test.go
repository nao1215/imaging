package imaging

import (
	"fmt"
	"image"
	"image/color"
	"testing"
)

// shortBufferImages returns, for every concrete type the scanner reads
// directly, an image whose pixel buffers end before its bounds do. A decoder or
// a caller filling the struct by hand can hand over such an image; the one in
// upstream issue disintegration/imaging#159 is a 4:2:0 YCbCr whose chroma
// planes were sized as CStride*height/2, a row short for an odd height.
func shortBufferImages(rect image.Rectangle) map[string]image.Image {
	images := scannerTestImages(rect)
	out := make(map[string]image.Image, len(images))
	for name, img := range images {
		switch m := img.(type) {
		case *image.NRGBA:
			m.Pix = m.Pix[:len(m.Pix)-3]
		case *image.NRGBA64:
			m.Pix = m.Pix[:len(m.Pix)-7]
		case *image.RGBA:
			m.Pix = m.Pix[:len(m.Pix)-3]
		case *image.RGBA64:
			m.Pix = m.Pix[:len(m.Pix)-7]
		case *image.Gray:
			m.Pix = m.Pix[:len(m.Pix)-1]
		case *image.Gray16:
			m.Pix = m.Pix[:len(m.Pix)-1]
		case *image.Paletted:
			m.Pix = m.Pix[:len(m.Pix)-1]
		case *image.YCbCr:
			m.Cb = m.Cb[:len(m.Cb)-1]
			m.Cr = m.Cr[:len(m.Cr)-1]
		default:
			continue
		}
		out[name] = img
	}
	return out
}

// safeAt is the colour the image reports at (x, y), or transparent where the
// image's own At panics because its buffers do not hold that pixel.
func safeAt(img image.Image, x, y int) (c color.NRGBA) {
	defer func() {
		if recover() != nil {
			c = color.NRGBA{}
		}
	}()
	return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
}

// TestShortBuffers checks that a pixel the buffers hold keeps its colour, that
// one they do not hold reads as transparent (the rule the scanner already
// applies to a palette index the palette does not define), and that no
// operation panics. Before the fix every one of these panicked inside a
// goroutine started by parallel, where no caller can recover.
func TestShortBuffers(t *testing.T) {
	t.Parallel()

	for _, rect := range []image.Rectangle{
		image.Rect(0, 0, 7, 5),
		image.Rect(3, 3, 10, 8),
		image.Rect(0, 0, 1, 1),
	} {
		for name, img := range shortBufferImages(rect) {
			t.Run(fmt.Sprintf("%s%v", name, rect), func(t *testing.T) {
				t.Parallel()

				got := Clone(img)
				b := img.Bounds()
				for y := 0; y < b.Dy(); y++ {
					for x := 0; x < b.Dx(); x++ {
						want := safeAt(img, b.Min.X+x, b.Min.Y+y)
						g := got.NRGBAAt(x, y)
						if channelDelta(premultiply(g), premultiply(want)) > 1 {
							t.Fatalf("at (%d,%d): got %v, want %v", x, y, g, want)
						}
					}
				}

				_ = Grayscale(img)
				_ = Invert(img)
				_ = Rotate90(img)
				_ = Rotate(img, 30, color.Black)
				_ = Blur(img, 1)
				_ = Sharpen(img, 1)
				_ = Resize(img, 3, 3, Lanczos)
				_ = Convolve3x3(img, [9]float64{1, 1, 1, 1, 1, 1, 1, 1, 1}, nil)
				_ = Histogram(img)
				_ = Paste(New(4, 4, color.White), img, image.Pt(1, 1))
				_ = Overlay(New(4, 4, color.White), img, image.Pt(1, 1), 0.5)
			})
		}
	}
}

// TestIssue159 is the image from upstream issue disintegration/imaging#159.
func TestIssue159(t *testing.T) {
	t.Parallel()

	const (
		yStride = 288
		cStride = 144
		width   = 288
		height  = 147
	)
	img := &image.YCbCr{
		Y:              make([]uint8, yStride*height),
		Cb:             make([]uint8, cStride*height/2),
		Cr:             make([]uint8, cStride*height/2),
		YStride:        yStride,
		CStride:        cStride,
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		Rect:           image.Rect(0, 0, width, height),
	}
	dst := Rotate90(img)
	if got, want := dst.Bounds().Size(), image.Pt(height, width); got != want {
		t.Fatalf("size: got %v, want %v", got, want)
	}
	// The first row of chroma is held, so the top left pixel keeps its colour.
	if got, want := dst.NRGBAAt(0, width-1), safeAt(img, 0, 0); got != want {
		t.Fatalf("held pixel: got %v, want %v", got, want)
	}
}

// panicImage is an image whose At panics, as a caller's own type can.
type panicImage struct{ image.Rectangle }

func (panicImage) ColorModel() color.Model   { return color.NRGBAModel }
func (p panicImage) Bounds() image.Rectangle { return p.Rectangle }
func (panicImage) At(int, int) color.Color   { panic("boom") }

// TestPanicReachesCaller checks that a panic inside the worker goroutines is
// raised again in the goroutine that called the function, so the caller can
// recover from it instead of the process dying.
func TestPanicReachesCaller(t *testing.T) {
	t.Parallel()

	var got any
	func() {
		defer func() { got = recover() }()
		Clone(panicImage{image.Rect(0, 0, 64, 64)})
	}()
	if got != "boom" {
		t.Fatalf("recovered %v, want boom", got)
	}
}

// FuzzMalformedImage builds each concrete image type the scanner reads
// directly from arbitrary fields (bounds, strides and buffer lengths that need
// not agree) and runs the operations over it. The property is that nothing
// panics and that every result has the size the bounds give it.
func FuzzMalformedImage(f *testing.F) {
	f.Add(uint8(0), int8(0), int8(0), uint8(7), uint8(5), 28, 100, 0, 0)
	f.Add(uint8(7), int8(0), int8(0), uint8(255), uint8(147), 288, 288*147, 144, 144*147/2)
	f.Add(uint8(7), int8(3), int8(3), uint8(5), uint8(5), 5, 25, 3, 5)
	f.Add(uint8(6), int8(1), int8(1), uint8(4), uint8(4), -4, 16, 0, 0)
	f.Add(uint8(2), int8(-2), int8(5), uint8(3), uint8(3), 1<<40, 3, 0, 0)

	f.Fuzz(func(t *testing.T, kind uint8, minX, minY int8, w, h uint8, stride, n, cstride, cn int) {
		rect := image.Rect(int(minX), int(minY), int(minX)+int(w%64), int(minY)+int(h%64))
		n = int(uint(n) % (1 << 16))
		cn = int(uint(cn) % (1 << 16))
		var img image.Image
		switch kind % 8 {
		case 0:
			img = &image.NRGBA{Pix: make([]uint8, n), Stride: stride, Rect: rect}
		case 1:
			img = &image.NRGBA64{Pix: make([]uint8, n), Stride: stride, Rect: rect}
		case 2:
			img = &image.RGBA{Pix: make([]uint8, n), Stride: stride, Rect: rect}
		case 3:
			img = &image.RGBA64{Pix: make([]uint8, n), Stride: stride, Rect: rect}
		case 4:
			img = &image.Gray{Pix: make([]uint8, n), Stride: stride, Rect: rect}
		case 5:
			img = &image.Gray16{Pix: make([]uint8, n), Stride: stride, Rect: rect}
		case 6:
			img = &image.Paletted{Pix: make([]uint8, n), Stride: stride, Rect: rect, Palette: color.Palette{color.White}}
		default:
			ratios := []image.YCbCrSubsampleRatio{
				image.YCbCrSubsampleRatio444, image.YCbCrSubsampleRatio422, image.YCbCrSubsampleRatio420,
				image.YCbCrSubsampleRatio440, image.YCbCrSubsampleRatio411, image.YCbCrSubsampleRatio410,
			}
			img = &image.YCbCr{
				Y: make([]uint8, n), Cb: make([]uint8, cn), Cr: make([]uint8, cn),
				YStride: stride, CStride: cstride, SubsampleRatio: ratios[int(uint(cstride)%uint(len(ratios)))],
				Rect: rect,
			}
		}

		size := rect.Size()
		for name, got := range map[string]*image.NRGBA{
			"Clone":       Clone(img),
			"Grayscale":   Grayscale(img),
			"Blur":        Blur(img, 1),
			"Convolve3x3": Convolve3x3(img, [9]float64{0, 1, 0, 1, 1, 1, 0, 1, 0}, nil),
			"FlipH":       FlipH(img),
		} {
			if got.Bounds().Size() != size {
				t.Fatalf("%s: size %v, want %v", name, got.Bounds().Size(), size)
			}
		}
		if got := Rotate90(img).Bounds().Size(); got != image.Pt(size.Y, size.X) {
			t.Fatalf("Rotate90: size %v", got)
		}
		_ = Resize(img, 3, 2, Linear)
		_ = Rotate(img, 17, color.Black)
		_ = Histogram(img)
		_ = Paste(New(8, 8, color.White), img, image.Pt(-1, 2))
	})
}
