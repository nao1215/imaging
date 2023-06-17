package imaging

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

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
			for i, v := range scanLine {
				scanLineF[i] = float64(v)
			}
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
					weight := kernel[absint(x-ix)]
					wsum += weight
					s := scanLineF[i : i+4 : i+4]
					wa := s[3] * weight
					r += s[0] * wa
					g += s[1] * wa
					b += s[2] * wa
					a += wa
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

func blurVertical(img image.Image, kernel []float64) *image.NRGBA {
	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, src.w, src.h))
	radius := len(kernel) - 1

	parallel(0, src.w, func(xs <-chan int) {
		scanLine := make([]uint8, src.h*4)
		scanLineF := make([]float64, len(scanLine))
		for x := range xs {
			src.scan(x, 0, x+1, src.h, scanLine)
			for i, v := range scanLine {
				scanLineF[i] = float64(v)
			}
			for y := 0; y < src.h; y++ {
				min := y - radius
				if min < 0 {
					min = 0
				}
				max := y + radius
				if max > src.h-1 {
					max = src.h - 1
				}
				var r, g, b, a, wsum float64
				for iy := min; iy <= max; iy++ {
					i := iy * 4
					weight := kernel[absint(y-iy)]
					wsum += weight
					s := scanLineF[i : i+4 : i+4]
					wa := s[3] * weight
					r += s[0] * wa
					g += s[1] * wa
					b += s[2] * wa
					a += wa
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

// Mosaic produces a mosaic version of the image.
// rect is the rectangle that apply the mosaic effect.
// The rect must be within the image bounds otherwise the function returns a copy of the src image.
//
// Example:
//
//	rect := image.Rect(50, 50, 250, 250)
//	dstImage := imaging.Mosaic(srcImage, rect)
func Mosaic(img image.Image, rect image.Rectangle) *image.NRGBA {
	const (
		blockSize  = 10
		mosaicSize = 4
	)

	rect = rect.Intersect(img.Bounds())
	if !isValidRectangle(rect) {
		// Return a copy of the input image if the rectangle is invalid
		copyOutput := image.NewNRGBA(img.Bounds())
		draw.Draw(copyOutput, img.Bounds(), img, image.Point{}, draw.Src)
		return copyOutput
	}
	output := image.NewNRGBA(rect)
	draw.Draw(output, rect, img, rect.Min, draw.Src) // copy image to output rectangle area

	for row := rect.Min.Y; row < rect.Max.Y; row += blockSize {
		for col := rect.Min.X; col < rect.Max.X; col += blockSize {
			avgColor := getAverageColor(img, col, row, blockSize, mosaicSize)
			blockRect := image.Rect(col, row, col+blockSize, row+blockSize)
			draw.Draw(output, blockRect, &image.Uniform{avgColor}, image.Point{}, draw.Src)
		}
	}
	return output
}

// getAverageColor get average color of the block in the image.
// x, y is the top left corner of the block. blockSize is the size of the block. mosaicSize is the size of the mosaic.
func getAverageColor(img image.Image, x, y, blockSize, mosaicSize int) color.Color {
	var totalR, totalG, totalB, totalA uint32

	for row := y; row < y+blockSize; row += mosaicSize {
		for col := x; col < x+blockSize; col += mosaicSize {
			for subRow := row; subRow < row+mosaicSize && subRow < y+blockSize; subRow++ {
				for subCol := col; subCol < col+mosaicSize && subCol < x+blockSize; subCol++ {
					r, g, b, a := img.At(subCol, subRow).RGBA()
					totalR += r
					totalG += g
					totalB += b
					totalA += a
				}
			}
		}
	}

	totalPixels := uint32((blockSize / mosaicSize) * (blockSize / mosaicSize))
	avgR := totalR / totalPixels
	avgG := totalG / totalPixels
	avgB := totalB / totalPixels
	avgA := totalA / totalPixels

	return color.NRGBA{uint8(avgR >> 8), uint8(avgG >> 8), uint8(avgB >> 8), uint8(avgA >> 8)}
}

// isValidRectangle returns true if the rectangle is valid.
// A rectangle is valid if it has positive width and height.
func isValidRectangle(rect image.Rectangle) bool {
	return rect.Dx() > 0 && rect.Dy() > 0
}
