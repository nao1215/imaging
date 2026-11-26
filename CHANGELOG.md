# Changelog

All notable changes to this project are documented in this file. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Changes in earlier versions are listed in the [GitHub Releases](https://github.com/nao1215/imaging/releases).

## [Unreleased]

### Fixed

- Every function accepts an `*image.NRGBA`, `*image.NRGBA64`, `*image.RGBA`, `*image.RGBA64`, `*image.Gray`, `*image.Gray16`, `*image.Paletted` or `*image.YCbCr` whose buffers end before its bounds do, such as the 4:2:0 YCbCr of upstream issue disintegration/imaging#159 whose chroma planes are a row short for an odd height. A pixel the buffers do not hold reads as transparent, as a palette index the palette does not define already did, also when the buffer has capacity past its length (the image types' own `At` methods read into that capacity) and when the chroma stride is so large that the offset of a later row overflows. Such an image panicked with an index out of range inside a worker goroutine and ended the process.
- An image with no columns but a non-zero stride no longer panics: the scanner sliced its empty buffer at the offset of the second row.
- A panic raised while processing an image, for example by the `At` method of a caller's own image type, is raised again in the goroutine that called the function, where it can be recovered. It was raised in a worker goroutine and ended the process.

### Changed

- The documentation of `Paste` says that it replaces the pixels it covers, transparent ones included, and points to `Overlay` with an opacity of 1 for drawing an image over the background (upstream issue disintegration/imaging#164). A new differential test checks that `Overlay` at opacity 1 agrees with `image/draw`'s `draw.Over`.
- Against disintegration/imaging on a 1920x1280 photo: `Resize` and `Fill` with `Lanczos` are 13 to 16% faster, `Thumbnail` with `CatmullRom` 8%, `NearestNeighbor` downscaling of an `*image.NRGBA` 57%, `Blur`, `Sharpen` and `Convolve3x3` 5 to 9%; nothing else moves by more than 1%. The vertical resize adds whole source rows rather than walking columns, and gives the same result bit for bit; the horizontal resize and both blur passes premultiply each line by its alpha once rather than at every tap of the filter, which can round a channel one level the other way. `clamp` takes its early returns again, because the version that assigned and fell through made every filtered resize about 12% slower than upstream; convolution, whose values are often out of range, uses a branchless `clampSaturating` instead, a third faster there. The README lists the comparison, and `scripts/bench_upstream.sh` reproduces it.
- CI is stricter. Pull requests compare the benchmarks of the base and the head on one runner and fail on a consistent slowdown of more than 25% (`scripts/ci_benchmark.py`); fuzz every target for 15 seconds; run the tests as `GOARCH=386` and, on Linux, under the race detector; and need 99% coverage with no drop (it was 90%). Mutation testing fails at an efficacy of 90% or less, set in `.gremlins.yaml` because gremlins v0.6.0 ignores `--threshold-efficacy`, and runs with `GOFLAGS=-count=1`, without which a cached test run made every mutant time out and the run report 0%. Every test job has a timeout.
- Mutation efficacy is 91.92% (86.83% for v1.0.13 measured the same way). Every resampling kernel is checked against the textbook form of its formula, `scan` panics on a destination shorter than the region instead of writing past its length within its capacity, and `PasteCenter` and `OverlayCenter` are checked by the margins they leave.
- `FuzzMalformedImage` builds every concrete image type from arbitrary bounds, strides and buffer lengths and runs the operations over it.

## [1.0.13] - 2026-11-19

### Fixed

- `Paste` and `PasteCenter` return an image of the background's size when the pasted image covers all of the background. They returned the pasted image whole, at its own larger size.
- `Overlay` and `OverlayCenter` leave a background pixel as it is where neither layer has any alpha: a transparent background under a transparent pixel, or under any pixel at opacity 0. The blend weights were 0/0 there, and the NaN converted to a byte wiped the color the background pixel carried (to 0 on amd64; the conversion is implementation-defined).

### Changed

- CI scans the repository with gosec and Trivy on every pull request (Trivy covers dependency vulnerabilities, committed secrets and misconfigurations, and also runs daily). Workflow actions are recorded in `.github/workflows/actions.lock`, which CI verifies against the pinned commits.
- Releases now attach a source archive, `checksums.txt`, a keyless cosign signature of the checksums and the build provenance of the archive to the GitHub Release. See "Verifying releases" in the README.
- Dependencies updated: `golang.org/x/image` 0.48.0 and `golang.org/x/sync` 0.25.0. The Go minimum stays at 1.26.
- CI scans the commits of pull requests and of pushes to main for committed secrets with gitleaks (Dependabot pull requests are skipped: they only change version numbers, and their read-only token cannot list the commits) and moves govulncheck to v1.10.0. A new push to a pull request cancels the run it supersedes.
- Metamorphic and differential tests (with rapid) check the orientations, the independence of every operation from the image's origin and storage type, resizing against flips, `Paste` against `image/draw` and `Overlay` at its ends, and draw 20,000 inputs per test every night. Mutation testing with gremlins runs on the lines a pull request changes and on the whole package weekly; `doc/testing.md` records the first score and how to run each technique.

## [1.0.12] - 2026-09-28

### Fixed

- `CropAnchor` and `CropCenter` return an empty image for a negative width or height, as `Resize`, `Fit` and `Fill` do, instead of cropping a region of the absolute size.

### Removed

- The gina command-line tool (`cmd/gina`) is removed; imaging is now a library only.
