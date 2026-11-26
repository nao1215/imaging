// Package upstreambench measures this fork against the module it was forked
// from, github.com/disintegration/imaging, on the same input. It is a module of
// its own so that the root module does not depend on the upstream one; see
// scripts/bench_upstream.sh.
package upstreambench

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"testing"

	up "github.com/disintegration/imaging"
	me "github.com/nao1215/imaging"
)

// source is testdata/branches.jpg enlarged to 1920x1280 and encoded as a JPEG
// again, so that it decodes to an *image.YCbCr of a common photo size.
func source() *image.YCbCr {
	img, err := me.Open("../../testdata/branches.jpg")
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, me.Resize(img, 1920, 0, me.Lanczos), &jpeg.Options{Quality: 90}); err != nil {
		panic(err)
	}
	dec, err := jpeg.Decode(&buf)
	if err != nil {
		panic(err)
	}
	return dec.(*image.YCbCr)
}

var (
	jpg   = source()
	nrgba = me.Clone(jpg)
	small = me.Resize(nrgba, 200, 0, me.Box)
)

type impl struct {
	name string
	run  map[string]func()
}

func ops() []impl {
	return []impl{
		{"upstream", map[string]func(){
			"Resize/Lanczos/NRGBA":   func() { up.Resize(nrgba, 640, 0, up.Lanczos) },
			"Resize/Lanczos/YCbCr":   func() { up.Resize(jpg, 640, 0, up.Lanczos) },
			"Resize/Linear/NRGBA":    func() { up.Resize(nrgba, 640, 0, up.Linear) },
			"Resize/Nearest/NRGBA":   func() { up.Resize(nrgba, 640, 0, up.NearestNeighbor) },
			"Fill/Lanczos":           func() { up.Fill(nrgba, 300, 300, up.Center, up.Lanczos) },
			"Thumbnail/CatmullRom":   func() { up.Thumbnail(jpg, 150, 150, up.CatmullRom) },
			"Blur/3":                 func() { up.Blur(nrgba, 3) },
			"Sharpen/1":              func() { up.Sharpen(nrgba, 1) },
			"Grayscale/YCbCr":        func() { up.Grayscale(jpg) },
			"AdjustContrast/NRGBA":   func() { up.AdjustContrast(nrgba, 20) },
			"AdjustSaturation/NRGBA": func() { up.AdjustSaturation(nrgba, 20) },
			"Rotate90/NRGBA":         func() { up.Rotate90(nrgba) },
			"Rotate/30":              func() { up.Rotate(nrgba, 30, color.Black) },
			"Convolve3x3":            func() { up.Convolve3x3(nrgba, [9]float64{-1, -1, 0, -1, 1, 1, 0, 1, 1}, nil) },
			"Histogram":              func() { up.Histogram(nrgba) },
			"Paste":                  func() { up.Paste(nrgba, small, image.Pt(10, 10)) },
			"Overlay":                func() { up.Overlay(nrgba, small, image.Pt(10, 10), 0.5) },
			"Clone/YCbCr":            func() { up.Clone(jpg) },
		}},
		{"fork", map[string]func(){
			"Resize/Lanczos/NRGBA":   func() { me.Resize(nrgba, 640, 0, me.Lanczos) },
			"Resize/Lanczos/YCbCr":   func() { me.Resize(jpg, 640, 0, me.Lanczos) },
			"Resize/Linear/NRGBA":    func() { me.Resize(nrgba, 640, 0, me.Linear) },
			"Resize/Nearest/NRGBA":   func() { me.Resize(nrgba, 640, 0, me.NearestNeighbor) },
			"Fill/Lanczos":           func() { me.Fill(nrgba, 300, 300, me.Center, me.Lanczos) },
			"Thumbnail/CatmullRom":   func() { me.Thumbnail(jpg, 150, 150, me.CatmullRom) },
			"Blur/3":                 func() { me.Blur(nrgba, 3) },
			"Sharpen/1":              func() { me.Sharpen(nrgba, 1) },
			"Grayscale/YCbCr":        func() { me.Grayscale(jpg) },
			"AdjustContrast/NRGBA":   func() { me.AdjustContrast(nrgba, 20) },
			"AdjustSaturation/NRGBA": func() { me.AdjustSaturation(nrgba, 20) },
			"Rotate90/NRGBA":         func() { me.Rotate90(nrgba) },
			"Rotate/30":              func() { me.Rotate(nrgba, 30, color.Black) },
			"Convolve3x3":            func() { me.Convolve3x3(nrgba, [9]float64{-1, -1, 0, -1, 1, 1, 0, 1, 1}, nil) },
			"Histogram":              func() { me.Histogram(nrgba) },
			"Paste":                  func() { me.Paste(nrgba, small, image.Pt(10, 10)) },
			"Overlay":                func() { me.Overlay(nrgba, small, image.Pt(10, 10), 0.5) },
			"Clone/YCbCr":            func() { me.Clone(jpg) },
		}},
	}
}

// BenchmarkCompare runs the operations of the implementation IMPL names
// ("upstream" or "fork"), so that the two can be run in turn as separate
// processes and compared with benchstat.
func BenchmarkCompare(b *testing.B) {
	for _, im := range ops() {
		if im.name != os.Getenv("IMPL") {
			continue
		}
		for name, f := range im.run {
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					f()
				}
			})
		}
	}
}
