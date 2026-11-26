package imaging

import (
	"image"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
)

var maxProcs int64

// SetMaxProcs limits the number of concurrent processing goroutines to the given value.
// A value <= 0 clears the limit.
func SetMaxProcs(value int) {
	atomic.StoreInt64(&maxProcs, int64(value))
}

// parallel processes the data in separate goroutines.
//
// A panic in a worker would end the process: no caller can recover a panic
// raised in a goroutine it did not start. The first one is kept and raised
// again in the calling goroutine once every worker has returned.
func parallel(start, stop int, fn func(<-chan int)) {
	count := stop - start
	if count < 1 {
		return
	}

	procs := runtime.GOMAXPROCS(0)
	limit := int(atomic.LoadInt64(&maxProcs))
	if procs > limit && limit > 0 {
		procs = limit
	}
	if procs > count {
		procs = count
	}

	c := make(chan int, count)
	for i := start; i < stop; i++ {
		c <- i
	}
	close(c)

	// One struct, so that the state the workers share is one allocation.
	var run struct {
		wg       sync.WaitGroup
		mu       sync.Mutex
		panicked bool
		value    any
	}
	run.wg.Add(procs)
	for i := 0; i < procs; i++ {
		go func() {
			defer run.wg.Done()
			defer func() {
				if r := recover(); r != nil {
					run.mu.Lock()
					if !run.panicked {
						run.panicked, run.value = true, r
					}
					run.mu.Unlock()
				}
			}()
			fn(c)
		}()
	}
	run.wg.Wait()
	if run.panicked {
		panic(run.value)
	}
}

// absInt returns the absolute value of i.
func absInt(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

// clamp rounds and clamps float64 value to fit into uint8.
//
// It branches, and the early returns matter: resampling and blurring produce
// values that are almost always in range, the branches are then predicted
// right, and this form made Blur and Sharpen 30% faster than one that
// assigns the bound and falls through (which compiles to conditional moves).
// Where values often fall outside the range, use clampSaturating.
func clamp(x float64) uint8 {
	v := int64(x + 0.5)
	if v > 255 {
		return 255
	}
	if v < 1 {
		return 0
	}
	return uint8(v)
}

// clampSaturating is clamp for values that often fall outside 0..255, such as
// those of an edge-detecting convolution kernel. It compiles to conditional
// moves rather than branches, so an unpredictable value costs nothing extra:
// Convolve3x3 is a third faster with it than with clamp. Both give the same
// result for every input.
func clampSaturating(x float64) uint8 {
	return uint8(min(max(int64(x+0.5), 0), 255)) // #nosec G115 -- clamped to 0..255
}

// reverse reverses the order of pixels in the given slice.
func reverse(pix []uint8) {
	length := len(pix)
	if length <= 4 {
		return
	}
	i, j := 0, length-4
	for i < j {
		pi := pix[i : i+4]
		pj := pix[j : j+4]
		pi[0], pj[0] = pj[0], pi[0]
		pi[1], pj[1] = pj[1], pi[1]
		pi[2], pj[2] = pj[2], pi[2]
		pi[3], pj[3] = pj[3], pi[3]
		i += 4
		j -= 4
	}
}

// toNRGBA convert image.Image to *image.NRGBA.
func toNRGBA(img image.Image) *image.NRGBA {
	if img, ok := img.(*image.NRGBA); ok && bufferCovers(len(img.Pix), img.Stride, img.Rect.Dx(), img.Rect.Dy(), 4) {
		return &image.NRGBA{
			Pix:    img.Pix,
			Stride: img.Stride,
			Rect:   img.Rect.Sub(img.Rect.Min),
		}
	}
	return Clone(img)
}

// rgbToHSL converts a color from RGB to HSL.
func rgbToHSL(r, g, b uint8) (h float64, s float64, l float64) {
	rr := float64(r) / 255
	gg := float64(g) / 255
	bb := float64(b) / 255

	max := math.Max(rr, math.Max(gg, bb))
	min := math.Min(rr, math.Min(gg, bb))
	l = (max + min) / 2

	if max == min {
		return 0, 0, l
	}

	d := max - min
	if l > 0.5 {
		s = d / (2 - max - min)
	} else {
		s = d / (max + min)
	}

	switch max {
	case rr:
		h = (gg - bb) / d
		if gg < bb {
			h += 6
		}
	case gg:
		h = (bb-rr)/d + 2
	case bb:
		h = (rr-gg)/d + 4
	}
	h /= 6

	return h, s, l
}

// hslToRGB converts a color from HSL to RGB.
func hslToRGB(h, s, l float64) (uint8, uint8, uint8) {
	var r, g, b float64
	if s == 0 {
		v := clamp(l * 255)
		return v, v, v
	}

	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q

	r = hueToRGB(p, q, h+1.0/3.0)
	g = hueToRGB(p, q, h)
	b = hueToRGB(p, q, h-1.0/3.0)

	return clamp(r * 255), clamp(g * 255), clamp(b * 255)
}

// hueToRGB converts hue to RGB. Helper function for hslToRGB.
func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}
