# Changelog

All notable changes to this project are documented in this file. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Changes in earlier versions are listed in the [GitHub Releases](https://github.com/nao1215/imaging/releases).

## [Unreleased]

## [1.0.12] - 2026-09-28

### Fixed

- `CropAnchor` and `CropCenter` return an empty image for a negative width or height, as `Resize`, `Fit` and `Fill` do, instead of cropping a region of the absolute size.

### Removed

- The gina command-line tool (`cmd/gina`) is removed; imaging is now a library only.
