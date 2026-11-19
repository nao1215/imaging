# Testing

`make test` runs everything below except the long fuzz and property searches, which run on a schedule in CI.

## Examples and unit tests

`example_test.go` holds the package example, and every source file has a `_test.go` beside it whose cases spell out the pixels a function returns for a small image, including images whose bounds do not start at (0, 0).

## Golden images

`TestResizeGolden`, `TestFitGolden`, `TestFillGolden`, `TestBlurGolden`, `TestSharpenGolden` and the `Test*Golden` tests of the adjustments compare the result for an image in `testdata/` against a reviewed image in `testdata/out_*.png`: exactly on amd64, and within a level of rounding per channel elsewhere, where floating point can round the other way.

## Metamorphic tests

An image operation rarely has an expected output that is easy to write down, but the same operation on a changed input often has a known relation to the first answer. These tests draw small images with [rapid](https://pkg.go.dev/pgregory.net/rapid), which shrinks a failure to the smallest image that still fails. Their names start with `TestMetamorphic_` and they live in `metamorphic_test.go`.

- The eight orientations form a group: each flip and rotation is undone by its inverse, `Rotate180` is `FlipV` after `FlipH`, `Transpose` is `Rotate90` after `FlipH`, and `Rotate` by a right angle, a whole turn either way, is the matching quarter-turn function.
- An operation that reads one pixel at a time (`Grayscale`, `Invert`, the adjustments) gives the same image whether the image is reoriented before or after it, and reorienting keeps the histogram.
- Every operation gives the same result for an image whose bounds start away from (0, 0) as for the same pixels at the origin, and the same result for opaque pixels stored as `*image.RGBA` as for `*image.NRGBA`.
- Resizing with a filter that goes to zero at its edge commutes with flips, up to a level of rounding. `NearestNeighbor` and `Box` jump at their edge, so a sample point lying exactly on it can go either way after a flip; they are left out.
- Resizing or blurring an image of one color gives that color back.
- `Paste` keeps the background's size and gives back the pasted image when the pasted area is cropped out again; `Overlay` at opacity 0 returns the background, and at opacity 1 with an opaque image it is `Paste`.
- An image written as PNG reads back as the same pixels.

## Differential tests

`TestMetamorphic_PasteThenCrop` pastes an opaque image with `Paste` and with `image/draw`'s `draw.Draw` and `draw.Src`, which have to agree. `TestScannerMatchesImageAt` in `scanner_test.go` checks that the scanner's fast path for each concrete image type reads the color the image's own `At` reports.

## Running the generated tests longer

`go test` draws a hundred inputs per test. Draw more, or replay a failure, with rapid's flags:

```sh
go test . -run '^TestMetamorphic_' -rapid.checks=20000
go test . -run '^TestMetamorphic_PasteThenCrop$' -rapid.seed=<seed printed by the failure>
```

The `Property` workflow (`.github/workflows/property.yml`) draws 20,000 inputs per test every night with a new seed. A failure writes the shrunk input under `testdata/rapid/`, which is ignored by git and uploaded by the workflow.

## Fuzzing

`FuzzDecode` feeds arbitrary bytes to `Decode` with and without auto-orientation, `FuzzFormatFromFilename` arbitrary names to the format lookup, and `FuzzResizeDimensions` arbitrary sizes, rectangles and angles to the resizing and cropping functions, all in `fuzz_test.go`. `go test` runs their seed corpus; the `Fuzz` workflow searches each of them every night.

```sh
go test . -run '^$' -fuzz '^FuzzDecode$' -fuzztime 1m
```

## Mutation testing

[gremlins](https://github.com/go-gremlins/gremlins) v0.6.0 changes one comparison, bound or arithmetic operator at a time and runs the package's tests; a mutant no test notices is a line the tests execute without checking. The `Mutation` workflow (`.github/workflows/mutation.yml`) mutates the lines a pull request changes, and the whole package once a week.

```sh
gremlins unleash --workers 4 --timeout-coefficient 10 .
git config diff.context 0; git config diff.interHunkContext 0   # one hunk per edit, so --diff sees every changed line
gremlins unleash --diff origin/main --workers 4 .
```

Efficacy (killed / (killed + lived)) measured on 2026-11-19, the first full run, with `--workers 4 --timeout-coefficient 10` and a 4 GiB memory cap per process. A timed-out mutant is one whose tests no longer finish, counted apart from both:

| Package | Killed | Lived | Not covered | Timed out | Efficacy |
|---|---|---|---|---|---|
| `.` (the package) | 1157 | 119 | 8 | 4 | 90.67% |

Half of the survivors (59) are in `resize.go`, most of them in the kernels of the filters no golden image uses (`Hermite`, `BSpline`, `Bartlett`, `Hann`, `Hamming`, `Blackman`, `Welch`, `Cosine`) and in the bounds `precomputeWeights`, `Fit` and `Fill` compare against, which is where a test is worth adding next. Some cannot be killed: the length of the slice a row is scanned into can grow without changing what the scanner writes. A pull request should not lower the efficacy below this table.
