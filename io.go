package imaging

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"golang.org/x/sync/errgroup"
)

// Open loads an image from file. After opening the image file, decoding is
// performed and Open() returns the image.Image interface.
//
// Examples:
//
//	// Load an image from file.
//	img, err := imaging.Open("test.jpg")
//
//	// Load an image and transform it depending on the EXIF orientation tag (if present).
//	img, err := imaging.Open("test.jpg", imaging.AutoOrientation(true))
func Open(filename string, opts ...DecodeOption) (img image.Image, err error) {
	file, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			} else {
				err = fmt.Errorf("original error: %s, defer close error: %w", err.Error(), closeErr)
			}
		}
	}()
	return Decode(file, opts...)
}

// decodeConfig holds the optional parameters for the Decode().
type decodeConfig struct {
	// autoOrientation enables or disables the auto-orientation mode.
	autoOrientation bool
}

// defaultDecodeConfig is the default decode config.
var defaultDecodeConfig = decodeConfig{
	autoOrientation: false,
}

// DecodeOption sets an optional parameter for the Decode and Open functions.
type DecodeOption func(*decodeConfig)

// AutoOrientation returns a DecodeOption that sets the auto-orientation mode.
// If auto-orientation is enabled, the image will be transformed after decoding
// according to the EXIF orientation tag (if present). By default it's disabled.
func AutoOrientation(enabled bool) DecodeOption {
	return func(c *decodeConfig) {
		c.autoOrientation = enabled
	}
}

// Decode reads an image from io.Reader.
func Decode(r io.Reader, opts ...DecodeOption) (image.Image, error) {
	cfg := defaultDecodeConfig
	for _, option := range opts {
		option(&cfg)
	}

	if !cfg.autoOrientation {
		img, _, err := image.Decode(r)
		return img, err
	}
	return decodeWithAutoOrientation(r)
}

// decodeWithAutoOrientation reads an image from io.Reader and automatically orientates it.
func decodeWithAutoOrientation(r io.Reader) (image.Image, error) {
	var orient Orientation

	pr, pw := io.Pipe()
	r = io.TeeReader(r, pw)

	eg := errgroup.Group{}
	eg.Go(func() error {
		orient = ReadOrientation(pr)
		if _, err := io.Copy(io.Discard, pr); err != nil {
			return err
		}
		return nil
	})

	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}

	// If the pipe writer is not closed at this point, a deadlock will occur.
	if err := pw.Close(); err != nil {
		return nil, err
	}
	if err = eg.Wait(); err != nil {
		return nil, err
	}
	return FixOrientation(img, orient), nil
}

// Format is an image file format.
type Format int

// Image file formats.
const (
	// JPEG (Joint Photographic Experts Group): A widely used image format
	// known for its lossy compression algorithm, making it suitable for photographs
	// and complex images.
	JPEG Format = iota
	// PNG (Portable Network Graphics): An image format that supports lossless compression,
	// providing high-quality graphics with transparency support. It is commonly used for
	// web graphics and images with sharp edges.
	PNG
	// GIF (Graphics Interchange Format): An image format that supports animations and a
	// limited color palette of 256 colors. It is commonly used for simple animations and
	// graphics with areas of uniform color.
	GIF
	// TIFF (Tagged Image File Format): A flexible image format that supports lossless
	// compression and can store high-quality images with a wide range of color depths.
	// It is often used in professional settings and for archiving images.
	TIFF
	// BMP (Bitmap): A basic image format that stores pixel data without compression.
	// It is widely supported but results in larger file sizes compared to compressed formats.
	BMP
)

// formatExts maps image format extensions to Format.
var formatExts = map[string]Format{
	"jpg":  JPEG,
	"jpeg": JPEG,
	"png":  PNG,
	"gif":  GIF,
	"tif":  TIFF,
	"tiff": TIFF,
	"bmp":  BMP,
}

// formatNames maps image formats to their names.
var formatNames = map[Format]string{
	JPEG: "JPEG",
	PNG:  "PNG",
	GIF:  "GIF",
	TIFF: "TIFF",
	BMP:  "BMP",
}

// String returns the name of the image format.
func (f Format) String() string {
	return formatNames[f]
}

// ErrUnsupportedFormat means the given image format is not supported.
var ErrUnsupportedFormat = errors.New("imaging: unsupported image format")

// FormatFromExtension parses image format from filename extension:
// "jpg" (or "jpeg"), "png", "gif", "tif" (or "tiff") and "bmp" are supported.
func FormatFromExtension(ext string) (Format, error) {
	if f, ok := formatExts[strings.ToLower(strings.TrimPrefix(ext, "."))]; ok {
		return f, nil
	}
	return -1, ErrUnsupportedFormat
}

// FormatFromFilename parses image format from filename:
// "jpg" (or "jpeg"), "png", "gif", "tif" (or "tiff") and "bmp" are supported.
func FormatFromFilename(filename string) (Format, error) {
	ext := filepath.Ext(filename)
	return FormatFromExtension(ext)
}

// encodeConfig holds the optional parameters for the Encode() and Save() functions.
type encodeConfig struct {
	// jpegQuality JPEG quality (1-100). Default is 95.
	jpegQuality int
	// gifNumColors GIF encoder number of colors (1-256). Default is 256.
	gifNumColors int
	// gifQuantizer GIF encoder quantizer. Default is nil (use the default quantizer).
	gifQuantizer draw.Quantizer
	// gifDrawer GIF encoder drawer. Default is nil (use the default drawer).
	gifDrawer draw.Drawer
	// pngCompressionLevel PNG compression level (1-9). Default is DefaultCompression.
	pngCompressionLevel png.CompressionLevel
}

// defaultEncodeConfig is the default encoding configuration.
var defaultEncodeConfig = encodeConfig{
	jpegQuality:         95,
	gifNumColors:        256,
	gifQuantizer:        nil,
	gifDrawer:           nil,
	pngCompressionLevel: png.DefaultCompression,
}

// EncodeOption sets an optional parameter for the Encode and Save functions.
type EncodeOption func(*encodeConfig)

// JPEGQuality returns an EncodeOption that sets the output JPEG quality.
// Quality ranges from 1 to 100 inclusive, higher is better. Default is 95.
func JPEGQuality(quality int) EncodeOption {
	return func(c *encodeConfig) {
		c.jpegQuality = quality
	}
}

// GIFNumColors returns an EncodeOption that sets the maximum number of colors
// used in the GIF-encoded image. It ranges from 1 to 256.  Default is 256.
func GIFNumColors(numColors int) EncodeOption {
	return func(c *encodeConfig) {
		c.gifNumColors = numColors
	}
}

// GIFQuantizer returns an EncodeOption that sets the quantizer that is used to produce
// a palette of the GIF-encoded image.
func GIFQuantizer(quantizer draw.Quantizer) EncodeOption {
	return func(c *encodeConfig) {
		c.gifQuantizer = quantizer
	}
}

// GIFDrawer returns an EncodeOption that sets the drawer that is used to convert
// the source image to the desired palette of the GIF-encoded image.
func GIFDrawer(drawer draw.Drawer) EncodeOption {
	return func(c *encodeConfig) {
		c.gifDrawer = drawer
	}
}

// PNGCompressionLevel returns an EncodeOption that sets the compression level
// of the PNG-encoded image. Default is png.DefaultCompression.
func PNGCompressionLevel(level png.CompressionLevel) EncodeOption {
	return func(c *encodeConfig) {
		c.pngCompressionLevel = level
	}
}

// Encode writes the image img to w in the specified format (JPEG, PNG, GIF, TIFF or BMP).
func Encode(w io.Writer, img image.Image, format Format, opts ...EncodeOption) error {
	cfg := defaultEncodeConfig
	for _, option := range opts {
		option(&cfg)
	}

	switch format {
	case JPEG:
		if nrgba, ok := img.(*image.NRGBA); ok && nrgba.Opaque() {
			rgba := &image.RGBA{
				Pix:    nrgba.Pix,
				Stride: nrgba.Stride,
				Rect:   nrgba.Rect,
			}
			return jpeg.Encode(w, rgba, &jpeg.Options{Quality: cfg.jpegQuality})
		}
		return jpeg.Encode(w, img, &jpeg.Options{Quality: cfg.jpegQuality})

	case PNG:
		encoder := png.Encoder{CompressionLevel: cfg.pngCompressionLevel}
		return encoder.Encode(w, img)

	case GIF:
		return gif.Encode(w, img, &gif.Options{
			NumColors: cfg.gifNumColors,
			Quantizer: cfg.gifQuantizer,
			Drawer:    cfg.gifDrawer,
		})

	case TIFF:
		return tiff.Encode(w, img, &tiff.Options{Compression: tiff.Deflate, Predictor: true})

	case BMP:
		return bmp.Encode(w, img)
	}

	return ErrUnsupportedFormat
}

// Save saves the image to file with the specified filename.
// The format is determined from the filename extension:
// "jpg" (or "jpeg"), "png", "gif", "tif" (or "tiff") and "bmp" are supported.
//
// Examples:
//
//	// Save the image as PNG.
//	err := imaging.Save(img, "out.png")
//
//	// Save the image as JPEG with optional quality parameter set to 80.
//	err := imaging.Save(img, "out.jpg", imaging.JPEGQuality(80))
func Save(img image.Image, filename string, opts ...EncodeOption) (err error) {
	f, err := FormatFromFilename(filename)
	if err != nil {
		return err
	}
	file, err := fs.Create(filename)
	if err != nil {
		return err
	}

	err = Encode(file, img, f, opts...)
	errClose := file.Close()
	if err == nil {
		err = errClose
	}
	return err
}

// Orientation is an EXIF flag that specifies the transformation
// that should be applied to image to display it correctly.
type Orientation int

const (
	// OrientationUnspecified is the default value. It means that the orientation is not specified.
	OrientationUnspecified Orientation = 0
	// OrientationNormal means that the image should be displayed as it is.
	OrientationNormal Orientation = 1
	// OrientationFlipH means that the image should be flipped horizontally.
	OrientationFlipH Orientation = 2
	// OrientationRotate180 means that the image should be rotated 180 degrees clockwise.
	OrientationRotate180 Orientation = 3
	// OrientationFlipV means that the image should be flipped vertically.
	OrientationFlipV Orientation = 4
	// OrientationTranspose means that the image should be transposed (flip vertically and rotate 270 degrees clockwise).
	OrientationTranspose Orientation = 5
	// OrientationRotate270 means that the image should be rotated 270 degrees clockwise.
	OrientationRotate270 Orientation = 6
	// OrientationTransverse means that the image should be transversed (flip horizontally and rotate 270 degrees clockwise).
	OrientationTransverse Orientation = 7
	// OrientationRotate90 means that the image should be rotated 90 degrees clockwise.
	OrientationRotate90 Orientation = 8
)

// FixOrientation applies a transform to img corresponding to the given orientation flag.
func FixOrientation(img image.Image, o Orientation) image.Image {
	switch o {
	case OrientationNormal:
	case OrientationFlipH:
		img = FlipH(img)
	case OrientationFlipV:
		img = FlipV(img)
	case OrientationRotate90:
		img = Rotate90(img)
	case OrientationRotate180:
		img = Rotate180(img)
	case OrientationRotate270:
		img = Rotate270(img)
	case OrientationTranspose:
		img = Transpose(img)
	case OrientationTransverse:
		img = Transverse(img)
	}
	return img
}
