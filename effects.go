package imaging

import (
	"image"
	"math"
)

// gaussianBlurKernel returns a Gaussian kernel with the given radius and sigma.
func gaussianBlurKernel(x, sigma float64) float64 {
	return math.Exp(-(x*x)/(2*sigma*sigma)) / (sigma * math.Sqrt(2*math.Pi))
}

// Blur produces a blurred version of the image using a Gaussian function.
// Sigma parameter must be positive and indicates how much the image will be blurred.
//
// Example:
//
//	dstImage := imaging.Blur(srcImage, 3.5)
func Blur(img image.Image, sigma float64) *image.NRGBA {
	if sigma <= 0 {
		return Clone(img)
	}

	radius := int(math.Ceil(sigma * 3.0))
	kernel := make([]float64, radius+1)

	for i := 0; i <= radius; i++ {
		kernel[i] = gaussianBlurKernel(float64(i), sigma)
	}

	return blurVertical(blurHorizontal(img, kernel), kernel)
}

func blurHorizontal(img image.Image, kernel []float64) *image.NRGBA {
	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, src.w, src.h))
	radius := len(kernel) - 1

	parallel(0, src.h, func(ys <-chan int) {
		scanLine := make([]uint8, src.w*4)
		scanLineF := make([]float64, len(scanLine))
		for y := range ys {
			src.scan(0, y, src.w, y+1, scanLine)
			premultiplyRow(scanLineF, scanLine)
			for x := 0; x < src.w; x++ {
				min := x - radius
				if min < 0 {
					min = 0
				}
				max := x + radius
				if max > src.w-1 {
					max = src.w - 1
				}
				var r, g, b, a, wsum float64
				for ix := min; ix <= max; ix++ {
					i := ix * 4
					weight := kernel[absInt(x-ix)]
					wsum += weight
					s := scanLineF[i : i+4 : i+4]
					r += s[0] * weight
					g += s[1] * weight
					b += s[2] * weight
					a += s[3] * weight
				}
				if a != 0 {
					aInv := 1 / a
					j := y*dst.Stride + x*4
					d := dst.Pix[j : j+4 : j+4]
					d[0] = clamp(r * aInv)
					d[1] = clamp(g * aInv)
					d[2] = clamp(b * aInv)
					d[3] = clamp(a / wsum)
				}
			}
		}
	})

	return dst
}

// blurVertical blurs the image along its columns. Each column is scanned once
// and premultiplied by its alpha into floats, as blurHorizontal does for a row,
// so each tap of the kernel costs four multiply-adds and no conversions.
func blurVertical(img image.Image, kernel []float64) *image.NRGBA {
	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, src.w, src.h))
	radius := len(kernel) - 1

	parallel(0, src.w, func(xs <-chan int) {
		scanLine := make([]uint8, src.h*4)
		scanLineF := make([]float64, len(scanLine))
		for x := range xs {
			src.scan(x, 0, x+1, src.h, scanLine)
			premultiplyRow(scanLineF, scanLine)
			for y := 0; y < src.h; y++ {
				lo := max(y-radius, 0)
				hi := min(y+radius, src.h-1)
				var r, g, b, a, wsum float64
				for iy := lo; iy <= hi; iy++ {
					weight := kernel[absInt(y-iy)]
					wsum += weight
					s := scanLineF[iy*4 : iy*4+4 : iy*4+4]
					r += s[0] * weight
					g += s[1] * weight
					b += s[2] * weight
					a += s[3] * weight
				}
				if a != 0 {
					aInv := 1 / a
					j := y*dst.Stride + x*4
					d := dst.Pix[j : j+4 : j+4]
					d[0] = clamp(r * aInv)
					d[1] = clamp(g * aInv)
					d[2] = clamp(b * aInv)
					d[3] = clamp(a / wsum)
				}
			}
		}
	})

	return dst
}

// premultiplyRow converts NRGBA bytes to floats with each colour channel
// multiplied by the pixel's alpha. The products are integers below 2^16, so
// they are exact, and a filter tap needs one multiplication per channel.
func premultiplyRow(dst []float64, src []uint8) {
	for i := 0; i+3 < len(src) && i+3 < len(dst); i += 4 {
		s := src[i : i+4 : i+4]
		p := dst[i : i+4 : i+4]
		a := float64(s[3])
		p[0] = float64(s[0]) * a
		p[1] = float64(s[1]) * a
		p[2] = float64(s[2]) * a
		p[3] = a
	}
}

// Sharpen produces a sharpened version of the image.
// Sigma parameter must be positive and indicates how much the image will be sharpened.
//
// Example:
//
//	dstImage := imaging.Sharpen(srcImage, 3.5)
func Sharpen(img image.Image, sigma float64) *image.NRGBA {
	if sigma <= 0 {
		return Clone(img)
	}

	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, src.w, src.h))
	blurred := Blur(img, sigma)

	parallel(0, src.h, func(ys <-chan int) {
		scanLine := make([]uint8, src.w*4)
		for y := range ys {
			src.scan(0, y, src.w, y+1, scanLine)
			j := y * dst.Stride
			for i := 0; i < src.w*4; i++ {
				val := int(scanLine[i])<<1 - int(blurred.Pix[j])
				if val < 0 {
					val = 0
				} else if val > 0xff {
					val = 0xff
				}
				dst.Pix[j] = uint8(val)
				j++
			}
		}
	})

	return dst
}
