# Changelog

All notable changes to this project are documented in this file. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Changes in earlier versions are listed in the [GitHub Releases](https://github.com/nao1215/imaging/releases).

## [Unreleased]

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
