package imaging

import (
	"encoding/binary"
	"errors"
	"io"
)

// ReadOrientation tries to read the orientation EXIF flag from image data in r.
// If the EXIF data block is not found or the orientation flag is not found
// or any other error occures while reading the data, it returns the
// orientationUnspecified (0) value.
func ReadOrientation(r io.Reader) Orientation {
	if err := findJPEGSOIMarker(r); err != nil {
		return OrientationUnspecified
	}

	if err := findJPEGAPP1Marker(r); err != nil {
		return OrientationUnspecified
	}

	if err := findEXIFHeader(r); err != nil {
		return OrientationUnspecified
	}

	byteOrder, err := readByteOrder(r)
	if err != nil {
		return OrientationUnspecified
	}

	if err := skipEXIFOffset(r, byteOrder); err != nil {
		return OrientationUnspecified
	}

	numTags, err := readNumTags(r, byteOrder)
	if err != nil {
		return OrientationUnspecified
	}

	orientation, err := findOrientationTag(r, byteOrder, numTags)
	if err != nil {
		return OrientationUnspecified
	}
	return orientation
}

// findJPEGSOIMarker tries to find the JPEG SOI marker in r.
// This function assumes that the reader is positioned at the beginning of the file.
func findJPEGSOIMarker(r io.Reader) error {
	const (
		markerSOI = 0xffd8
	)

	var soi uint16
	if err := binary.Read(r, binary.BigEndian, &soi); err != nil {
		return err
	}
	if soi != markerSOI {
		return errors.New("Missing JPEG SOI marker")
	}
	return nil
}

// findJPEGAPP1Marker tries to find the JPEG APP1 marker in r.
// This function assumes that the reader is positioned after the JPEG SOI marker.
func findJPEGAPP1Marker(r io.Reader) error {
	const (
		markerAPP1 = 0xffe1
	)

	for {
		var marker, size uint16
		if err := binary.Read(r, binary.BigEndian, &marker); err != nil {
			return err
		}
		if err := binary.Read(r, binary.BigEndian, &size); err != nil {
			return err
		}
		if marker>>8 != 0xff {
			return errors.New("Invalid JPEG marker")
		}
		if marker == markerAPP1 {
			break
		}
		if size < 2 {
			return errors.New("Invalid block size")
		}
		if _, err := io.CopyN(io.Discard, r, int64(size-2)); err != nil {
			return err
		}
	}
	return nil
}

// findEXIFHeader tries to find the EXIF header in r.
// This function assumes that the reader is positioned after the JPEG APP1 marker.
func findEXIFHeader(r io.Reader) error {
	const (
		exifHeader = 0x45786966
	)

	var header uint32
	if err := binary.Read(r, binary.BigEndian, &header); err != nil {
		return err
	}
	if header != exifHeader {
		return errors.New("EXIF header not found")
	}
	if _, err := io.CopyN(io.Discard, r, 2); err != nil {
		return err
	}
	return nil
}

// readByteOrder reads the byte order from r.
// This function assumes that the reader is positioned after the EXIF header.
func readByteOrder(r io.Reader) (binary.ByteOrder, error) {
	const (
		byteOrderBE = 0x4d4d
		byteOrderLE = 0x4949
	)

	var byteOrderTag uint16
	if err := binary.Read(r, binary.BigEndian, &byteOrderTag); err != nil {
		return nil, err
	}

	var byteOrder binary.ByteOrder
	switch byteOrderTag {
	case byteOrderBE:
		byteOrder = binary.BigEndian
	case byteOrderLE:
		byteOrder = binary.LittleEndian
	default:
		return nil, errors.New("Invalid byte order flag")
	}

	if _, err := io.CopyN(io.Discard, r, 2); err != nil {
		return nil, err
	}
	return byteOrder, nil
}

// skipEXIFOffset skips the EXIF offset in r.
// This function assumes that the reader is positioned after the byte order tag.
func skipEXIFOffset(r io.Reader, byteOrder binary.ByteOrder) error {
	var offset uint32
	if err := binary.Read(r, byteOrder, &offset); err != nil {
		return err
	}
	if offset < 8 {
		return errors.New("Invalid offset value")
	}
	if _, err := io.CopyN(io.Discard, r, int64(offset-8)); err != nil {
		return err
	}
	return nil
}

// readNumTags reads the number of tags from r.
// This function assumes that the reader is positioned after the EXIF offset.
func readNumTags(r io.Reader, byteOrder binary.ByteOrder) (uint16, error) {
	var numTags uint16
	if err := binary.Read(r, byteOrder, &numTags); err != nil {
		return 0, err
	}
	return numTags, nil
}

// findOrientationTag tries to find the orientation tag in r.
// This function assumes that the reader is positioned after the number of tags.
func findOrientationTag(r io.Reader, byteOrder binary.ByteOrder, numTags uint16) (Orientation, error) {
	const (
		orientationTag = 0x0112
	)

	for i := 0; i < int(numTags); i++ {
		var tag uint16
		if err := binary.Read(r, byteOrder, &tag); err != nil {
			return OrientationUnspecified, err
		}
		if tag != orientationTag {
			if _, err := io.CopyN(io.Discard, r, 10); err != nil {
				return OrientationUnspecified, err
			}
			continue
		}

		if _, err := io.CopyN(io.Discard, r, 6); err != nil {
			return OrientationUnspecified, err
		}

		var val uint16
		if err := binary.Read(r, byteOrder, &val); err != nil {
			return OrientationUnspecified, err
		}
		if val < 1 || val > 8 {
			return OrientationUnspecified, errors.New("Invalid tag value")
		}

		return Orientation(val), nil
	}
	return OrientationUnspecified, nil // Missing orientation tag.
}
