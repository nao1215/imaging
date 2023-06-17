package imaging

import (
	"encoding/binary"
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

type decodeConfig struct {
	autoOrientation bool
}

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
func Open(filename string, opts ...DecodeOption) (image.Image, error) {
	file, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			} else {
				err = fmt.Errorf("original error: %v, defer close error: %v", err, closeErr)
			}
		}
	}()
	return Decode(file, opts...)
}

// Format is an image file format.
type Format int

// Image file formats.
const (
	JPEG Format = iota
	PNG
	GIF
	TIFF
	BMP
)

var formatExts = map[string]Format{
	"jpg":  JPEG,
	"jpeg": JPEG,
	"png":  PNG,
	"gif":  GIF,
	"tif":  TIFF,
	"tiff": TIFF,
	"bmp":  BMP,
}

var formatNames = map[Format]string{
	JPEG: "JPEG",
	PNG:  "PNG",
	GIF:  "GIF",
	TIFF: "TIFF",
	BMP:  "BMP",
}

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

type encodeConfig struct {
	jpegQuality         int
	gifNumColors        int
	gifQuantizer        draw.Quantizer
	gifDrawer           draw.Drawer
	pngCompressionLevel png.CompressionLevel
}

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
	errc := file.Close()
	if err == nil {
		err = errc
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

// ReadOrientation tries to read the orientation EXIF flag from image data in r.
// If the EXIF data block is not found or the orientation flag is not found
// or any other error occures while reading the data, it returns the
// orientationUnspecified (0) value.
func ReadOrientation(r io.Reader) Orientation {
	const (
		markerSOI      = 0xffd8
		markerAPP1     = 0xffe1
		exifHeader     = 0x45786966
		byteOrderBE    = 0x4d4d
		byteOrderLE    = 0x4949
		orientationTag = 0x0112
	)

	// Check if JPEG SOI marker is present.
	var soi uint16
	if err := binary.Read(r, binary.BigEndian, &soi); err != nil {
		return OrientationUnspecified
	}
	if soi != markerSOI {
		return OrientationUnspecified // Missing JPEG SOI marker.
	}

	// Find JPEG APP1 marker.
	for {
		var marker, size uint16
		if err := binary.Read(r, binary.BigEndian, &marker); err != nil {
			return OrientationUnspecified
		}
		if err := binary.Read(r, binary.BigEndian, &size); err != nil {
			return OrientationUnspecified
		}
		if marker>>8 != 0xff {
			return OrientationUnspecified // Invalid JPEG marker.
		}
		if marker == markerAPP1 {
			break
		}
		if size < 2 {
			return OrientationUnspecified // Invalid block size.
		}
		if _, err := io.CopyN(io.Discard, r, int64(size-2)); err != nil {
			return OrientationUnspecified
		}
	}

	// Check if EXIF header is present.
	var header uint32
	if err := binary.Read(r, binary.BigEndian, &header); err != nil {
		return OrientationUnspecified
	}
	if header != exifHeader {
		return OrientationUnspecified
	}
	if _, err := io.CopyN(io.Discard, r, 2); err != nil {
		return OrientationUnspecified
	}

	// Read byte order information.
	var (
		byteOrderTag uint16
		byteOrder    binary.ByteOrder
	)
	if err := binary.Read(r, binary.BigEndian, &byteOrderTag); err != nil {
		return OrientationUnspecified
	}
	switch byteOrderTag {
	case byteOrderBE:
		byteOrder = binary.BigEndian
	case byteOrderLE:
		byteOrder = binary.LittleEndian
	default:
		return OrientationUnspecified // Invalid byte order flag.
	}
	if _, err := io.CopyN(io.Discard, r, 2); err != nil {
		return OrientationUnspecified
	}

	// Skip the EXIF offset.
	var offset uint32
	if err := binary.Read(r, byteOrder, &offset); err != nil {
		return OrientationUnspecified
	}
	if offset < 8 {
		return OrientationUnspecified // Invalid offset value.
	}
	if _, err := io.CopyN(io.Discard, r, int64(offset-8)); err != nil {
		return OrientationUnspecified
	}

	// Read the number of tags.
	var numTags uint16
	if err := binary.Read(r, byteOrder, &numTags); err != nil {
		return OrientationUnspecified
	}

	// Find the orientation tag.
	for i := 0; i < int(numTags); i++ {
		var tag uint16
		if err := binary.Read(r, byteOrder, &tag); err != nil {
			return OrientationUnspecified
		}
		if tag != orientationTag {
			if _, err := io.CopyN(io.Discard, r, 10); err != nil {
				return OrientationUnspecified
			}
			continue
		}
		if _, err := io.CopyN(io.Discard, r, 6); err != nil {
			return OrientationUnspecified
		}
		var val uint16
		if err := binary.Read(r, byteOrder, &val); err != nil {
			return OrientationUnspecified
		}
		if val < 1 || val > 8 {
			return OrientationUnspecified // Invalid tag value.
		}
		return Orientation(val)
	}
	return OrientationUnspecified // Missing orientation tag.
}

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
