# Testing

`make test` runs everything below except the long fuzz and property searches, which run on a schedule in CI.

A pull request runs the unit tests on Linux, macOS and Windows, each under the race detector, on the Go floor of `go.mod` and the newest release, and as `GOARCH=386`, where `int` is 32 bits; it builds for FreeBSD, OpenBSD and NetBSD. Coverage must stay at 99% or more and may not drop (`.octocov.yml`). Each fuzz target is searched for 15 seconds, the changed lines are mutated, and the benchmarks of the base and the head are compared on one runner (below).

## Examples and unit tests

`example_test.go` holds the package example, and every source file has a `_test.go` beside it whose cases spell out the pixels a function returns for a small image, including images whose bounds do not start at (0, 0).

## Golden images

`TestResizeGolden`, `TestFitGolden`, `TestFillGolden`, `TestBlurGolden`, `TestSharpenGolden` and the `Test*Golden` tests of the adjustments compare the result for an image in `testdata/` against a reviewed image in `testdata/out_*.png`: exactly on amd64, and within a level of rounding per channel elsewhere, where floating point can round the other way.

## Metamorphic tests

An image operation rarely has an expected output that is easy to write down, but the same operation on a changed input often has a known relation to the first answer. These tests draw small images with [rapid](https://pkg.go.dev/pgregory.net/rapid), which shrinks a failure to the smallest image that still fails. Their names start with `TestMetamorphic_` and they live in `metamorphic_test.go`.

- The eight orientations form a group: each flip and rotation is undone by its inverse, `Rotate180` is `FlipV` after `FlipH`, `Transpose` is `Rotate90` after `FlipH`, and `Rotate` by a right angle, a whole turn either way, is the matching quarter-turn function.
- An operation that reads one pixel at a time (`Grayscale`, `Invert`, the adjustments) gives the same image whether the image is reoriented before or after it, and reorienting keeps the histogram.
- Every operation gives the same result for an image whose bounds start away from (0, 0) as for the same pixels at the origin, and the same result for opaque pixels stored as `*image.RGBA` as for `*image.NRGBA`.
- Resizing with a filter that goes to zero at its edge commutes with flips, up to two levels of rounding: the two passes of `Resize` round to a byte in between, and a value exactly on a half can round the other way after a flip. `NearestNeighbor` and `Box` jump at their edge, so a sample point lying exactly on it can go either way after a flip; they are left out.
- Resizing or blurring an image of one color gives that color back.
- `Paste` keeps the background's size and gives back the pasted image when the pasted area is cropped out again; `Overlay` at opacity 0 returns the background, and at opacity 1 with an opaque image it is `Paste`.
- An image written as PNG reads back as the same pixels.

## Differential tests

`TestMetamorphic_PasteThenCrop` pastes an opaque image with `Paste` and with `image/draw`'s `draw.Draw` and `draw.Src`, which have to agree. `TestScannerMatchesImageAt` in `scanner_test.go` checks that the scanner's fast path for each concrete image type reads the color the image's own `At` reports.

`TestDifferential_OverlayMatchesDrawOver` composites with `Overlay` at opacity 1 and with `draw.Over`, which must agree within a level of rounding: draw works on 16-bit premultiplied color.

`differential_test.go` keeps the resize and blur passes as they were before they were rewritten for speed, and compares the new passes with them on images rapid draws (alpha often exactly 0 or 255). The vertical resize, which now adds whole rows instead of walking columns, and the nearest-neighbor downscale, which now copies from an NRGBA source directly, must agree bit for bit. The horizontal resize and both blur passes now premultiply a line by its alpha once rather than at every tap; that rounds the product of color, alpha and weight in another order, so they are allowed a level of rounding (with no tolerance, rapid finds a differing image within the first hundred).

## Running the generated tests longer

`go test` draws a hundred inputs per test. Draw more, or replay a failure, with rapid's flags:

```sh
go test . -run '^TestMetamorphic_' -rapid.checks=20000
go test . -run '^TestMetamorphic_PasteThenCrop$' -rapid.seed=<seed printed by the failure>
```

The `Property` workflow (`.github/workflows/property.yml`) draws 20,000 inputs per test every night with a new seed. A failure writes the shrunk input under `testdata/rapid/`, which is ignored by git and uploaded by the workflow.

## Fuzzing

`FuzzDecode` feeds arbitrary bytes to `Decode` with and without auto-orientation, `FuzzFormatFromFilename` arbitrary names to the format lookup, and `FuzzResizeDimensions` arbitrary sizes, rectangles and angles to the resizing and cropping functions, all in `fuzz_test.go`. `FuzzMalformedImage` in `malformed_test.go` builds each image type the scanner reads directly from bounds, strides and buffer lengths that need not agree, the kind of image of upstream issue disintegration/imaging#159, and checks that no operation panics. `go test` runs their seed corpus; the `Fuzz` workflow searches each of them every night.

```sh
go test . -run '^$' -fuzz '^FuzzDecode$' -fuzztime 1m
```

## Mutation testing

[gremlins](https://github.com/go-gremlins/gremlins) v0.6.0 changes one comparison, bound or arithmetic operator at a time and runs the package's tests; a mutant no test notices is a line the tests execute without checking. The `Mutation` workflow (`.github/workflows/mutation.yml`) mutates the lines a pull request changes, and the whole package once a week. Either run fails when the efficacy is 90% or less; the threshold is in `.gremlins.yaml`, because gremlins v0.6.0 ignores the value of `--threshold-efficacy`. A pull request that changes no mutable line has no mutants and passes.

```sh
export GOFLAGS=-count=1   # a cached test run sets every mutant's timeout to milliseconds
gremlins unleash --workers 4 --timeout-coefficient 10 .
git config diff.context 0; git config diff.interHunkContext 0   # one hunk per edit, so --diff sees every changed line
gremlins unleash --diff origin/main --workers 4 .
```

The four mutants that survive on the lines of the faster resize are in the test `resizeNearest` uses to choose between reading a non-NRGBA source pixel by pixel and converting it whole. Both branches give the same pixels, so no test can tell them apart: they are equivalent mutants. The other survivors there were removed by writing the loops so that a changed bound reads past a slice and panics, rather than by adding tests.

Efficacy (killed / (killed + lived)) with `--workers 8 --timeout-coefficient 10`, `GOFLAGS=-count=1` and a 4 GiB memory cap per process. A timed-out mutant is one whose tests no longer finish, counted apart from both:

| Measured | Killed | Lived | Not covered | Timed out | Efficacy |
|---|---|---|---|---|---|
| 2026-11-19, first full run (`--workers 4`) | 1157 | 119 | 8 | 4 | 90.67% |
| 2026-11-26, main at v1.0.13, re-measured with the settings above | 1108 | 168 | 8 | 4 | 86.83% |
| 2026-11-26, after this round of tests | 1240 | 109 | 8 | 8 | 91.92% |

The second row is the same code as the first, measured again: the first run's figure does not reproduce, so it is not a baseline to compare against. The tests that raised the third row: every resampling kernel against the textbook form of its formula (`kernel_test.go`), which killed 41 mutants in the filters no golden image uses; `scan` indexing the last byte of its destination, so a row slice of the wrong length panics instead of being written past its end within its capacity; `parallel` over a range that does not start at zero; and the margins of `PasteCenter` and `OverlayCenter`.

Most of the 109 survivors cannot be killed. A kernel that is continuous at the edge of its support gives 0 there whether the test is `x < 3` or `x <= 3`; a guard against an empty size returns an empty image either way; `exif.go` returning nil when an error is nil and carrying on gives the same orientation on any readable file; and some tests choose a speed rather than a result (the 100-pixel threshold in `Fill`, the source path in `resizeNearest`, the goroutine count in `parallel`). The rest are in `precomputeWeights` and the aspect-ratio arithmetic of `Fit` and `Fill`.

## Benchmark gate

`python3 scripts/ci_benchmark.py --base <commit> --head <commit>` builds the package's benchmarks at both commits from `git archive`, runs them in turn for 8 rounds at `GOMAXPROCS=1`, and fails when a benchmark's median is more than 1.25 times the base's and even its fastest sample is slower than the base's median. The `Benchmark` workflow runs it on every pull request that touches Go code and uploads the samples and the benchstat summary. It found two regressions in its first run: the panic recovery added to `parallel` cost two allocations per call until its state was merged into one struct, and the early-return `clamp` that speeds up resizing and blurring made `Convolve3x3` a third slower, which is why convolution uses `clampSaturating`.

## Benchmarks against disintegration/imaging

`scripts/upstreambench` is a module of its own that runs the same operations on the same 1920x1280 photo with this package and with the one it was forked from. `scripts/bench_upstream.sh [rounds] [regexp]` runs the two in turn, one process per round, and summarizes them with benchstat; the table in the README comes from it. On a shared machine, pin the runs (`CPUS=8-11`) and stop other heavy work: with a busy GPU sharing the power budget, the spread was ±45% and the comparison meant nothing.
