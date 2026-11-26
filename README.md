# Imaging

<!-- ALL-CONTRIBUTORS-BADGE:START - Do not remove or modify this section -->
[![All Contributors](https://img.shields.io/badge/all_contributors-2-orange.svg?style=flat-square)](#contributors-)
<!-- ALL-CONTRIBUTORS-BADGE:END -->
[![LinuxUnitTest](https://github.com/nao1215/imaging/actions/workflows/linux_test.yml/badge.svg)](https://github.com/nao1215/imaging/actions/workflows/linux_test.yml)
[![MacUnitTest](https://github.com/nao1215/imaging/actions/workflows/mac_test.yml/badge.svg)](https://github.com/nao1215/imaging/actions/workflows/mac_test.yml)
[![WindowsUnitTest](https://github.com/nao1215/imaging/actions/workflows/windows_test.yml/badge.svg)](https://github.com/nao1215/imaging/actions/workflows/windows_test.yml)
![Coverage](https://raw.githubusercontent.com/nao1215/octocovs-central-repo/main/badges/nao1215/imaging/coverage.svg)
[![Gitleaks](https://github.com/nao1215/imaging/actions/workflows/gitleaks.yml/badge.svg)](https://github.com/nao1215/imaging/actions/workflows/gitleaks.yml)
[![Trivy](https://github.com/nao1215/imaging/actions/workflows/trivy.yml/badge.svg)](https://github.com/nao1215/imaging/actions/workflows/trivy.yml)
[![govulncheck](https://github.com/nao1215/imaging/actions/workflows/govulncheck.yml/badge.svg)](https://github.com/nao1215/imaging/actions/workflows/govulncheck.yml)
[![Gosec](https://github.com/nao1215/imaging/actions/workflows/gosec.yml/badge.svg)](https://github.com/nao1215/imaging/actions/workflows/gosec.yml)
[![reviewdog](https://github.com/nao1215/imaging/actions/workflows/reviewdog.yml/badge.svg)](https://github.com/nao1215/imaging/actions/workflows/reviewdog.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/nao1215/imaging.svg)](https://pkg.go.dev/github.com/nao1215/imaging)
[![License](https://img.shields.io/github/license/nao1215/imaging)](./LICENSE)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/nao1215/imaging/badge)](https://scorecard.dev/viewer/?uri=github.com/nao1215/imaging)

Imaging is a Go package for basic image processing: resizing, rotation, cropping, blurring, sharpening, and brightness, contrast, gamma, saturation and hue adjustments. It is a fork of [disintegration/imaging](https://github.com/disintegration/imaging), which is updated slowly; this fork adds bug fixes and tests on Linux, macOS and Windows.

Every image processing function accepts any `image.Image` and returns a new `*image.NRGBA` (32-bit RGBA, non-premultiplied alpha). The full API is documented at https://pkg.go.dev/github.com/nao1215/imaging.

## Supported platforms

Unit tests run on Linux, macOS and Windows with Go 1.26 or later.

## Example

```go
package main

import (
	"image"
	"image/color"
	"log"

	"github.com/nao1215/imaging"
)

func main() {
	// Open a test image.
	src, err := imaging.Open("testdata/flowers.png")
	if err != nil {
		log.Fatalf("failed to open image: %v", err)
	}

	// Crop the original image to 300x300px size using the center anchor.
	src = imaging.CropAnchor(src, 300, 300, imaging.Center)

	// Resize the cropped image to width = 200px preserving the aspect ratio.
	src = imaging.Resize(src, 200, 0, imaging.Lanczos)

	// Create a blurred version of the image.
	img1 := imaging.Blur(src, 5)

	// Create a grayscale version of the image with higher contrast and sharpness.
	img2 := imaging.Grayscale(src)
	img2 = imaging.AdjustContrast(img2, 20)
	img2 = imaging.Sharpen(img2, 2)

	// Create an inverted version of the image.
	img3 := imaging.Invert(src)

	// Create an embossed version of the image using a convolution filter.
	img4 := imaging.Convolve3x3(
		src,
		[9]float64{
			-1, -1, 0,
			-1, 1, 1,
			0, 1, 1,
		},
		nil,
	)

	// Create a new image and paste the four produced images into it.
	dst := imaging.New(400, 400, color.NRGBA{0, 0, 0, 0})
	dst = imaging.Paste(dst, img1, image.Pt(0, 0))
	dst = imaging.Paste(dst, img2, image.Pt(0, 200))
	dst = imaging.Paste(dst, img3, image.Pt(200, 0))
	dst = imaging.Paste(dst, img4, image.Pt(200, 200))

	// Save the resulting image as JPEG.
	err = imaging.Save(dst, "testdata/out_example.jpg")
	if err != nil {
		log.Fatalf("failed to save image: %v", err)
	}
}
```

Output:

![dstImage](testdata/out_example.jpg)

## Resizing

```go
// Resize srcImage to size = 128x128px using the Lanczos filter.
dstImage128 := imaging.Resize(srcImage, 128, 128, imaging.Lanczos)

// Resize srcImage to width = 800px preserving the aspect ratio.
dstImage800 := imaging.Resize(srcImage, 800, 0, imaging.Lanczos)

// Scale down srcImage to fit the 800x600px bounding box.
dstImageFit := imaging.Fit(srcImage, 800, 600, imaging.Lanczos)

// Resize and crop the srcImage to fill the 100x100px area.
dstImageFill := imaging.Fill(srcImage, 100, 100, imaging.Center, imaging.Lanczos)
```

The most commonly used resampling filters:

- `Lanczos`: sharp results on photographs.
- `CatmullRom`: a sharp cubic filter, faster than Lanczos with similar results.
- `MitchellNetravali`: a cubic filter with smoother results and less ringing than CatmullRom.
- `Linear`: bilinear, smooth output, faster than the cubic filters.
- `Box`: a fast averaging filter for downscaling; when upscaling it is close to NearestNeighbor.
- `NearestNeighbor`: the fastest, with no antialiasing.

All supported filters: NearestNeighbor, Box, Linear, Hermite, MitchellNetravali, CatmullRom, BSpline, Gaussian, Lanczos, Hann, Hamming, Blackman, Bartlett, Welch, Cosine. You can define your own with the ResampleFilter struct.

The original 600x400px image:

![srcImage](testdata/branches.png)

Resized to 150x100px, from the fastest filter to the highest quality:

Filter                    | Resize result
--------------------------|---------------------------------------------
`imaging.NearestNeighbor` | ![dstImage](testdata/out_resize_nearest.png)
`imaging.Linear`          | ![dstImage](testdata/out_resize_linear.png)
`imaging.CatmullRom`      | ![dstImage](testdata/out_resize_catrom.png)
`imaging.Lanczos`         | ![dstImage](testdata/out_resize_lanczos.png)

## Blur and sharpen

```go
dstImage := imaging.Blur(srcImage, 0.5)
```

```go
dstImage := imaging.Sharpen(srcImage, 0.5)
```

Sigma sets the strength of both effects (`Sharpen` also uses a gaussian function).

Function | Original image | Sigma = 0.5 | Sigma = 1.5
---------|----------------|-------------|------------
`Blur` | ![srcImage](testdata/flowers_small.png) | ![dstImage](testdata/out_blur_0.5.png) | ![dstImage](testdata/out_blur_1.5.png)
`Sharpen` | ![srcImage](testdata/flowers_small.png) | ![dstImage](testdata/out_sharpen_0.5.png) | ![dstImage](testdata/out_sharpen_1.5.png)

## Color adjustments

```go
dstImage := imaging.AdjustGamma(srcImage, 0.75)
```

```go
dstImage := imaging.AdjustContrast(srcImage, 20)
```

```go
dstImage := imaging.AdjustBrightness(srcImage, 20)
```

```go
dstImage := imaging.AdjustSaturation(srcImage, 20)
```

```go
dstImage := imaging.AdjustHue(srcImage, 20)
```

Function | Original image | Result | Result
---------|----------------|--------|-------
`AdjustGamma` | ![srcImage](testdata/flowers_small.png) | Gamma = 0.75<br>![dstImage](testdata/out_gamma_0.75.png) | Gamma = 1.25<br>![dstImage](testdata/out_gamma_1.25.png)
`AdjustContrast` | ![srcImage](testdata/flowers_small.png) | Contrast = 15<br>![dstImage](testdata/out_contrast_p15.png) | Contrast = -15<br>![dstImage](testdata/out_contrast_m15.png)
`AdjustBrightness` | ![srcImage](testdata/flowers_small.png) | Brightness = 10<br>![dstImage](testdata/out_brightness_p10.png) | Brightness = -10<br>![dstImage](testdata/out_brightness_m10.png)
`AdjustSaturation` | ![srcImage](testdata/flowers_small.png) | Saturation = 30<br>![dstImage](testdata/out_saturation_p30.png) | Saturation = -30<br>![dstImage](testdata/out_saturation_m30.png)
`AdjustHue` | ![srcImage](testdata/flowers_small.png) | Hue = 60<br>![dstImage](testdata/out_hue_p60.png) | Hue = -60<br>![dstImage](testdata/out_hue_m60.png)

## FAQ

### The image is rotated after processing

The file probably has an EXIF orientation tag, which the standard `image/*` packages ignore. Open it with the `AutoOrientation` decode option, and the image is rotated according to the tag (if present) after decoding:

```go
img, err := imaging.Open("test.jpg", imaging.AutoOrientation(true))
```

## Testing

| Technique | Bug class caught here |
|---|---|
| Examples and unit tests ([example_test.go](example_test.go), [tools_test.go](tools_test.go)) | A function that no longer gives the pixels its tests spell out |
| Golden images ([resize_test.go](resize_test.go), [adjust_test.go](adjust_test.go), [effects_test.go](effects_test.go)) | A filter or adjustment whose output drifts from the reviewed image in `testdata/` |
| Metamorphic tests ([metamorphic_test.go](metamorphic_test.go)) | A result that changes with the image's origin or storage type, a flip or rotation that does not undo, or a resize that treats the two directions differently |
| Differential tests ([metamorphic_test.go](metamorphic_test.go), [scanner_test.go](scanner_test.go), [differential_test.go](differential_test.go)) | A paste that disagrees with `image/draw`, a fast path that reads another color than the image's own `At`, or a faster resize or blur pass that disagrees with the pass it replaced |
| Fuzzing ([fuzz_test.go](fuzz_test.go), [malformed_test.go](malformed_test.go)) | A crash or a wrong size on a malformed file, size, rectangle, angle, or image whose buffers do not match its bounds |
| Mutation testing ([.github/workflows/mutation.yml](.github/workflows/mutation.yml)) | A test that runs a changed comparison or bound without noticing the change |
| Paired benchmarks ([scripts/ci_benchmark.py](scripts/ci_benchmark.py)) | A pull request that makes an operation consistently slower than on main |

How to run each one is in [doc/testing.md](doc/testing.md).

### Performance compared with disintegration/imaging

A 1920x1280 JPEG (decoded as `*image.YCbCr`, or converted to `*image.NRGBA`), median of 10 interleaved runs, `GOMAXPROCS=4`, Go 1.26, AMD Ryzen AI MAX+ 395. "~" means no significant difference (p >= 0.05). Reproduce with `CPUS=8-11 scripts/bench_upstream.sh`.

| Operation | disintegration/imaging | nao1215/imaging | Change |
|---|---:|---:|---:|
| `Resize` to 640 px, `Lanczos`, NRGBA | 7.83 ms | 6.62 ms | -15.5% |
| `Resize` to 640 px, `Lanczos`, YCbCr | 9.19 ms | 7.99 ms | -13.1% |
| `Resize` to 640 px, `Linear`, NRGBA | 3.14 ms | 3.00 ms | -4.3% |
| `Resize` to 640 px, `NearestNeighbor`, NRGBA | 0.268 ms | 0.116 ms | -56.8% |
| `Fill` 300x300, `Lanczos` | 5.40 ms | 4.63 ms | -14.4% |
| `Thumbnail` 150x150, `CatmullRom`, YCbCr | 3.97 ms | 3.66 ms | -7.9% |
| `Blur` sigma 3 | 29.7 ms | 27.1 ms | -8.5% |
| `Sharpen` sigma 1 | 17.0 ms | 16.1 ms | -5.3% |
| `Convolve3x3` | 9.19 ms | 8.49 ms | -7.7% |
| `Grayscale`, YCbCr | 2.55 ms | 2.55 ms | ~ |
| `AdjustContrast` | 0.912 ms | 0.910 ms | ~ |
| `AdjustSaturation` | 13.6 ms | 13.6 ms | ~ |
| `Rotate90` | 0.999 ms | 1.01 ms | ~ |
| `Rotate` 30 degrees | 13.5 ms | 13.5 ms | ~ |
| `Histogram` | 1.03 ms | 1.05 ms | +1.1% |
| `Paste` 200 px image | 0.429 ms | 0.432 ms | ~ |
| `Overlay` 200 px image | 0.479 ms | 0.480 ms | ~ |
| `Clone`, YCbCr | 1.69 ms | 1.70 ms | ~ |

## Contributors ✨

Thanks goes to these wonderful people ([emoji key](https://allcontributors.org/en/reference/emoji-key/)):

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<table>
  <tbody>
    <tr>
      <td align="center" valign="top" width="14.28%"><a href="https://debimate.jp/"><img src="https://avatars.githubusercontent.com/u/22737008?v=4?s=75" width="75px;" alt="CHIKAMATSU Naohiro"/><br /><sub><b>CHIKAMATSU Naohiro</b></sub></a><br /><a href="https://github.com/nao1215/imaging/commits?author=nao1215" title="Code">💻</a> <a href="https://github.com/nao1215/imaging/commits?author=nao1215" title="Documentation">📖</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/disintegration"><img src="https://avatars.githubusercontent.com/u/2706839?v=4?s=75" width="75px;" alt="Grigory Dryapak"/><br /><sub><b>Grigory Dryapak</b></sub></a><br /><a href="https://github.com/nao1215/imaging/commits?author=disintegration" title="Code">💻</a></td>
    </tr>
  </tbody>
</table>

<!-- markdownlint-restore -->
<!-- prettier-ignore-end -->

<!-- ALL-CONTRIBUTORS-LIST:END -->

This project follows the [all-contributors](https://github.com/all-contributors/all-contributors) specification. Contributions of any kind welcome!

## Verifying releases

Each GitHub Release made from now on carries a source archive (`imaging-<tag>.tar.gz`), `checksums.txt`, a keyless cosign signature of it (`checksums.txt.sigstore.json`) and the build provenance of the archive (`imaging-<tag>.intoto.jsonl`). Download them from the release page (replace `<tag>` with the release tag), then verify:

```bash
gh attestation verify imaging-<tag>.tar.gz --repo nao1215/imaging \
  --bundle imaging-<tag>.intoto.jsonl

cosign verify-blob --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp '^https://github\.com/nao1215/imaging/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt

sha256sum --check checksums.txt
```

## License

MIT License. See [LICENSE](LICENSE). Original author: [Disintegration](https://github.com/disintegration).
