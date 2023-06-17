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
}

// newScanner creates a new scanner for the given image.
// It also converts the palette to color.NRGBA slice.
func newScanner(img image.Image) *scanner {
	s := &scanner{
		image: img,
		w:     img.Bounds().Dx(),
		h:     img.Bounds().Dy(),
	}
	if palImg, ok := img.(*image.Paletted); ok {
		s.palette = make([]color.NRGBA, len(palImg.Palette))
		for i, c := range palImg.Palette {
			if rgba, ok := color.NRGBAModel.Convert(c).(color.NRGBA); ok {
				s.palette[i] = rgba
			}
		}
	}
	return s
}

// scan scans the given rectangular region of the image into dst.
func (s *scanner) scan(x1, y1, x2, y2 int, dst []uint8) {
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
				d[0] = uint8(r16 * 0xff / a16)
				d[1] = uint8(g16 * 0xff / a16)
				d[2] = uint8(b16 * 0xff / a16)
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
				d[0] = uint8((r32 * 0xffff / a32) >> 8)
				d[1] = uint8((g32 * 0xffff / a32) >> 8)
				d[2] = uint8((b32 * 0xffff / a32) >> 8)
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
			if uint32(r)&0xff000000 == 0 {
				r >>= 16
			} else {
				r = ^(r >> 31)
			}

			g := yy1 - 22554*cb1 - 46802*cr1
			if uint32(g)&0xff000000 == 0 {
				g >>= 16
			} else {
				g = ^(g >> 31)
			}

			b := yy1 + 116130*cb1
			if uint32(b)&0xff000000 == 0 {
				b >>= 16
			} else {
				b = ^(b >> 31)
			}

			d := dst[j : j+4 : j+4]
			d[0] = uint8(r)
			d[1] = uint8(g)
			d[2] = uint8(b)
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
				d[0] = uint8(r16 >> 8)
				d[1] = uint8(g16 >> 8)
				d[2] = uint8(b16 >> 8)
				d[3] = 0xff
			case 0:
				d[0] = 0
				d[1] = 0
				d[2] = 0
				d[3] = 0
			default:
				d[0] = uint8(((r16 * 0xffff) / a16) >> 8)
				d[1] = uint8(((g16 * 0xffff) / a16) >> 8)
				d[2] = uint8(((b16 * 0xffff) / a16) >> 8)
				d[3] = uint8(a16 >> 8)
			}
			j += 4
		}
	}
}
