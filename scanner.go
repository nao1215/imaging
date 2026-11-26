package imaging

import (
	"image"
	"image/color"
)

// scanner is a helper struct that allows us to read pixels from the image
type scanner struct {
	// image is the image to read pixels from
	image image.Image
	// w, h are the width and height of the image
	w, h int
	// palette is the palette of the image, if any
	palette []color.NRGBA
	// short reports that the image is one of the concrete types scan reads
	// directly but its buffers end before its bounds do, so scan has to check
	// every read.
	short bool
}

// bufferCovers reports whether a buffer of length n, read row by row with the
// given stride, holds h rows of w pixels of bpp bytes each, the first pixel at
// offset 0. It is written with divisions so that no product can overflow.
func bufferCovers(n, stride, w, h, bpp int) bool {
	if w <= 0 || h <= 0 {
		return true
	}
	if stride < 0 || w > n/bpp {
		return false
	}
	last := n - w*bpp // the offset the last row may start at
	return stride == 0 || h-1 <= last/stride
}

// isShort reports whether img is a concrete type scan reads directly whose
// buffers do not hold every pixel of its bounds.
func isShort(img image.Image) bool {
	switch m := img.(type) {
	case *image.NRGBA:
		return !bufferCovers(len(m.Pix), m.Stride, m.Rect.Dx(), m.Rect.Dy(), 4)
	case *image.NRGBA64:
		return !bufferCovers(len(m.Pix), m.Stride, m.Rect.Dx(), m.Rect.Dy(), 8)
	case *image.RGBA:
		return !bufferCovers(len(m.Pix), m.Stride, m.Rect.Dx(), m.Rect.Dy(), 4)
	case *image.RGBA64:
		return !bufferCovers(len(m.Pix), m.Stride, m.Rect.Dx(), m.Rect.Dy(), 8)
	case *image.Gray:
		return !bufferCovers(len(m.Pix), m.Stride, m.Rect.Dx(), m.Rect.Dy(), 1)
	case *image.Gray16:
		return !bufferCovers(len(m.Pix), m.Stride, m.Rect.Dx(), m.Rect.Dy(), 2)
	case *image.Paletted:
		return !bufferCovers(len(m.Pix), m.Stride, m.Rect.Dx(), m.Rect.Dy(), 1)
	case *image.YCbCr:
		return ycbcrShort(m)
	}
	return false
}

// ycbcrShort reports whether the luma or chroma planes of img end before its
// bounds do.
func ycbcrShort(m *image.YCbCr) bool {
	r := m.Rect
	if r.Empty() {
		return false
	}
	if !bufferCovers(len(m.Y), m.YStride, r.Dx(), r.Dy(), 1) {
		return true
	}
	if m.CStride < 0 {
		return true
	}
	// The chroma offset grows with x and with y, so the planes hold every
	// sample when they hold the first and the last.
	first, last := m.COffset(r.Min.X, r.Min.Y), m.COffset(r.Max.X-1, r.Max.Y-1)
	return first < 0 || last >= len(m.Cb) || last >= len(m.Cr)
}

// paletteSize is the number of entries scanner.palette always holds. A pixel of
// a paletted image is one byte, so every value it can take indexes this table.
const paletteSize = 256

// newScanner creates a new scanner for the given image.
// It also converts the palette to color.NRGBA slice.
//
// The converted palette is always paletteSize entries long, however short the
// image's own palette is, and entries the image does not define are left as the
// zero color.NRGBA, which is transparent. An index a palette does not define has
// no correct color, and answering with a transparent one keeps the undefined
// pixel visibly absent rather than painting it with an arbitrary neighbour.
// Sizing the table to the pixel type rather than to the palette is also what
// lets scanPaletted index it without a per-pixel bounds check.
func newScanner(img image.Image) *scanner {
	s := &scanner{
		image: img,
		w:     img.Bounds().Dx(),
		h:     img.Bounds().Dy(),
		short: isShort(img),
	}
	if palImg, ok := img.(*image.Paletted); ok {
		s.palette = make([]color.NRGBA, paletteSize)
		for i, c := range palImg.Palette {
			if i >= paletteSize {
				break
			}
			// A palette may hold a nil color, which the model cannot convert.
			if c == nil {
				continue
			}
			if rgba, ok := color.NRGBAModel.Convert(c).(color.NRGBA); ok {
				s.palette[i] = rgba
			}
		}
	}
	return s
}

// scan scans the given rectangular region of the image into dst.
func (s *scanner) scan(x1, y1, x2, y2 int, dst []uint8) {
	// An empty region reads nothing. The fast paths below would still slice
	// the buffer at the row's offset, which lies past the end of the empty
	// buffer of an image with no columns but a stride.
	if x1 >= x2 || y1 >= y2 {
		return
	}
	if s.short {
		s.scanShort(x1, y1, x2, y2, dst)
		return
	}
	switch img := s.image.(type) {
	case *image.NRGBA:
		s.scanNRGBA(img, x1, y1, x2, y2, dst)
	case *image.NRGBA64:
		s.scanNRGBA64(img, x1, y1, x2, y2, dst)
	case *image.RGBA:
		s.scanRGBA(img, x1, y1, x2, y2, dst)
	case *image.RGBA64:
		s.scanRGBA64(img, x1, y1, x2, y2, dst)
	case *image.Gray:
		s.scanGray(img, x1, y1, x2, y2, dst)
	case *image.Gray16:
		s.scanGray16(img, x1, y1, x2, y2, dst)
	case *image.YCbCr:
		s.scanYCbCr(img, x1, y1, x2, y2, dst)
	case *image.Paletted:
		s.scanPaletted(img, x1, y1, x2, y2, dst)
	default:
		s.scanDefault(x1, y1, x2, y2, dst)
	}
}

// scanNRGBA scans the given rectangular region of the NRGBA image into dst.
func (s *scanner) scanNRGBA(img *image.NRGBA, x1, y1, x2, y2 int, dst []uint8) {
	size := (x2 - x1) * 4
	j := 0
	i := y1*img.Stride + x1*4
	if size == 4 {
		for y := y1; y < y2; y++ {
			d := dst[j : j+4 : j+4]
			p := img.Pix[i : i+4 : i+4]
			d[0] = p[0]
			d[1] = p[1]
			d[2] = p[2]
			d[3] = p[3]
			j += size
			i += img.Stride
		}
	} else {
		for y := y1; y < y2; y++ {
			copy(dst[j:j+size], img.Pix[i:i+size])
			j += size
			i += img.Stride
		}
	}
}

// scanNRGBA64 scans the given rectangular region of the NRGBA64 image into dst.
func (s *scanner) scanNRGBA64(img *image.NRGBA64, x1, y1, x2, y2 int, dst []uint8) {
	j := 0
	for y := y1; y < y2; y++ {
		i := y*img.Stride + x1*8
		for x := x1; x < x2; x++ {
			p := img.Pix[i : i+8 : i+8]
			d := dst[j : j+4 : j+4]
			d[0] = p[0]
			d[1] = p[2]
			d[2] = p[4]
			d[3] = p[6]
			j += 4
			i += 8
		}
	}
}

// scanRGBA scans the given rectangular region of the RGBA image into dst.
func (s *scanner) scanRGBA(img *image.RGBA, x1, y1, x2, y2 int, dst []uint8) {
	j := 0
	for y := y1; y < y2; y++ {
		i := y*img.Stride + x1*4
		for x := x1; x < x2; x++ {
			d := dst[j : j+4 : j+4]
			a := img.Pix[i+3]
			switch a {
			case 0:
				d[0] = 0
				d[1] = 0
				d[2] = 0
				d[3] = a
			case 0xff:
				s := img.Pix[i : i+4 : i+4]
				d[0] = s[0]
				d[1] = s[1]
				d[2] = s[2]
				d[3] = a
			default:
				s := img.Pix[i : i+4 : i+4]
				r16 := uint16(s[0])
				g16 := uint16(s[1])
				b16 := uint16(s[2])
				a16 := uint16(a)
				d[0] = uint8(r16 * 0xff / a16) // #nosec G115 -- premultiplied color is at most alpha, so the quotient fits a byte
				d[1] = uint8(g16 * 0xff / a16) // #nosec G115 -- premultiplied color is at most alpha, so the quotient fits a byte
				d[2] = uint8(b16 * 0xff / a16) // #nosec G115 -- premultiplied color is at most alpha, so the quotient fits a byte
				d[3] = a
			}
			j += 4
			i += 4
		}
	}
}

// scanRGBA64 scans the given rectangular region of the RGBA64 image into dst.
func (s *scanner) scanRGBA64(img *image.RGBA64, x1, y1, x2, y2 int, dst []uint8) {
	j := 0
	for y := y1; y < y2; y++ {
		i := y*img.Stride + x1*8
		for x := x1; x < x2; x++ {
			s := img.Pix[i : i+8 : i+8]
			d := dst[j : j+4 : j+4]
			a := s[6]
			switch a {
			case 0:
				d[0] = 0
				d[1] = 0
				d[2] = 0
			case 0xff:
				d[0] = s[0]
				d[1] = s[2]
				d[2] = s[4]
			default:
				r32 := uint32(s[0])<<8 | uint32(s[1])
				g32 := uint32(s[2])<<8 | uint32(s[3])
				b32 := uint32(s[4])<<8 | uint32(s[5])
				a32 := uint32(s[6])<<8 | uint32(s[7])
				d[0] = uint8((r32 * 0xffff / a32) >> 8) // #nosec G115 -- premultiplied color is at most alpha, so the quotient fits a byte
				d[1] = uint8((g32 * 0xffff / a32) >> 8) // #nosec G115 -- premultiplied color is at most alpha, so the quotient fits a byte
				d[2] = uint8((b32 * 0xffff / a32) >> 8) // #nosec G115 -- premultiplied color is at most alpha, so the quotient fits a byte
			}
			d[3] = a
			j += 4
			i += 8
		}
	}
}

// scanGray scans the given rectangular region of the Gray image into dst.
func (s *scanner) scanGray(img *image.Gray, x1, y1, x2, y2 int, dst []uint8) {
	j := 0
	for y := y1; y < y2; y++ {
		i := y*img.Stride + x1
		for x := x1; x < x2; x++ {
			c := img.Pix[i]
			d := dst[j : j+4 : j+4]
			d[0] = c
			d[1] = c
			d[2] = c
			d[3] = 0xff
			j += 4
			i++
		}
	}
}

// scanGray16 scans the given rectangular region of the Gray16 image into dst.
func (s *scanner) scanGray16(img *image.Gray16, x1, y1, x2, y2 int, dst []uint8) {
	j := 0
	for y := y1; y < y2; y++ {
		i := y*img.Stride + x1*2
		for x := x1; x < x2; x++ {
			c := img.Pix[i]
			d := dst[j : j+4 : j+4]
			d[0] = c
			d[1] = c
			d[2] = c
			d[3] = 0xff
			j += 4
			i += 2
		}
	}
}

// scanYCbCr scans the given rectangular region of the YCbCr image into dst.
func (s *scanner) scanYCbCr(img *image.YCbCr, x1, y1, x2, y2 int, dst []uint8) {
	j := 0
	x1 += img.Rect.Min.X
	x2 += img.Rect.Min.X
	y1 += img.Rect.Min.Y
	y2 += img.Rect.Min.Y

	hy := img.Rect.Min.Y / 2
	hx := img.Rect.Min.X / 2
	for y := y1; y < y2; y++ {
		iy := (y-img.Rect.Min.Y)*img.YStride + (x1 - img.Rect.Min.X)

		var yBase int
		switch img.SubsampleRatio {
		case image.YCbCrSubsampleRatio444, image.YCbCrSubsampleRatio422:
			yBase = (y - img.Rect.Min.Y) * img.CStride
		case image.YCbCrSubsampleRatio420, image.YCbCrSubsampleRatio440:
			yBase = (y/2 - hy) * img.CStride
		default:
		}

		for x := x1; x < x2; x++ {
			var ic int
			switch img.SubsampleRatio {
			case image.YCbCrSubsampleRatio444, image.YCbCrSubsampleRatio440:
				ic = yBase + (x - img.Rect.Min.X)
			case image.YCbCrSubsampleRatio422, image.YCbCrSubsampleRatio420:
				ic = yBase + (x/2 - hx)
			default:
				ic = img.COffset(x, y)
			}

			yy1 := int32(img.Y[iy]) * 0x10101
			cb1 := int32(img.Cb[ic]) - 128
			cr1 := int32(img.Cr[ic]) - 128

			r := yy1 + 91881*cr1
			if uint32(r)&0xff000000 == 0 { // #nosec G115 -- the sign-bit test of color.YCbCrToRGB
				r >>= 16
			} else {
				r = ^(r >> 31)
			}

			g := yy1 - 22554*cb1 - 46802*cr1
			if uint32(g)&0xff000000 == 0 { // #nosec G115 -- the sign-bit test of color.YCbCrToRGB
				g >>= 16
			} else {
				g = ^(g >> 31)
			}

			b := yy1 + 116130*cb1
			if uint32(b)&0xff000000 == 0 { // #nosec G115 -- the sign-bit test of color.YCbCrToRGB
				b >>= 16
			} else {
				b = ^(b >> 31)
			}

			d := dst[j : j+4 : j+4]
			d[0] = uint8(r) // #nosec G115 -- r was clamped to 0..255 above
			d[1] = uint8(g) // #nosec G115 -- g was clamped to 0..255 above
			d[2] = uint8(b) // #nosec G115 -- b was clamped to 0..255 above
			d[3] = 0xff

			iy++
			j += 4
		}
	}
}

// scanPaletted scans the given rectangular region of the Paletted image into dst.
func (s *scanner) scanPaletted(img *image.Paletted, x1, y1, x2, y2 int, dst []uint8) {
	j := 0
	for y := y1; y < y2; y++ {
		i := y*img.Stride + x1
		for x := x1; x < x2; x++ {
			c := s.palette[img.Pix[i]]
			d := dst[j : j+4 : j+4]
			d[0] = c.R
			d[1] = c.G
			d[2] = c.B
			d[3] = c.A
			j += 4
			i++
		}
	}
}

// scanDefault scans the given rectangular region of the image using the default case.
func (s *scanner) scanDefault(x1, y1, x2, y2 int, dst []uint8) {
	j := 0
	b := s.image.Bounds()
	x1 += b.Min.X
	x2 += b.Min.X
	y1 += b.Min.Y
	y2 += b.Min.Y
	for y := y1; y < y2; y++ {
		for x := x1; x < x2; x++ {
			r16, g16, b16, a16 := s.image.At(x, y).RGBA()
			d := dst[j : j+4 : j+4]
			switch a16 {
			case 0xffff:
				d[0] = uint8(r16 >> 8) // #nosec G115 -- a 16-bit channel shifted right by 8 fits a byte
				d[1] = uint8(g16 >> 8) // #nosec G115 -- a 16-bit channel shifted right by 8 fits a byte
				d[2] = uint8(b16 >> 8) // #nosec G115 -- a 16-bit channel shifted right by 8 fits a byte
				d[3] = 0xff
			case 0:
				d[0] = 0
				d[1] = 0
				d[2] = 0
				d[3] = 0
			default:
				d[0] = uint8(((r16 * 0xffff) / a16) >> 8) // #nosec G115 -- premultiplied color is at most alpha, so the quotient is at most 0xffff
				d[1] = uint8(((g16 * 0xffff) / a16) >> 8) // #nosec G115 -- premultiplied color is at most alpha, so the quotient is at most 0xffff
				d[2] = uint8(((b16 * 0xffff) / a16) >> 8) // #nosec G115 -- premultiplied color is at most alpha, so the quotient is at most 0xffff
				d[3] = uint8(a16 >> 8)                    // #nosec G115 -- a 16-bit channel shifted right by 8 fits a byte
			}
			j += 4
		}
	}
}

// scanShort scans an image whose buffers end before its bounds do. It asks the
// image for each pixel, as scanDefault does, and reads a pixel the buffers do
// not hold as transparent: the same answer newScanner gives for a palette
// index the palette does not define. Such a pixel has no correct colour, and
// the alternative, the index out of range the image's own At raises, would end
// the process from inside a worker goroutine.
func (s *scanner) scanShort(x1, y1, x2, y2 int, dst []uint8) {
	j := 0
	b := s.image.Bounds()
	for y := y1 + b.Min.Y; y < y2+b.Min.Y; y++ {
		for x := x1 + b.Min.X; x < x2+b.Min.X; x++ {
			c := s.shortAt(x, y)
			d := dst[j : j+4 : j+4]
			d[0], d[1], d[2], d[3] = c.R, c.G, c.B, c.A
			j += 4
		}
	}
}

// shortAt returns the colour at (x, y), or transparent where the image cannot
// produce one.
func (s *scanner) shortAt(x, y int) (c color.NRGBA) {
	if p, ok := s.image.(*image.Paletted); ok {
		i := p.PixOffset(x, y)
		if i < 0 || i >= len(p.Pix) {
			return color.NRGBA{}
		}
		return s.palette[p.Pix[i]]
	}
	defer func() {
		if recover() != nil {
			c = color.NRGBA{}
		}
	}()
	return color.NRGBAModel.Convert(s.image.At(x, y)).(color.NRGBA)
}

// rowReader reads whole rows of an image as NRGBA bytes. For a well-formed
// *image.NRGBA it hands out the image's own rows without copying them; any
// other image is scanned into the caller's buffer.
type rowReader struct {
	w, h  int
	nrgba *image.NRGBA
	scan  *scanner
}

func newRowReader(img image.Image) rowReader {
	s := newScanner(img)
	r := rowReader{w: s.w, h: s.h, scan: s}
	if m, ok := img.(*image.NRGBA); ok && !s.short {
		r.nrgba = m
	}
	return r
}

// row returns row y (counted from the top of the bounds) of w*4 bytes. The
// returned slice may be buf or the image's own memory and must not be written.
func (r rowReader) row(y int, buf []uint8) []uint8 {
	if r.nrgba != nil {
		i := y * r.nrgba.Stride
		return r.nrgba.Pix[i : i+r.w*4 : i+r.w*4]
	}
	r.scan.scan(0, y, r.w, y+1, buf)
	return buf
}
