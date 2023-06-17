package imaging

import (
	"os"
	"strings"
	"testing"
)

func TestReadOrientation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		path   string
		orient Orientation
	}{
		{"testdata/orientation_0.jpg", 0},
		{"testdata/orientation_1.jpg", 1},
		{"testdata/orientation_2.jpg", 2},
		{"testdata/orientation_3.jpg", 3},
		{"testdata/orientation_4.jpg", 4},
		{"testdata/orientation_5.jpg", 5},
		{"testdata/orientation_6.jpg", 6},
		{"testdata/orientation_7.jpg", 7},
		{"testdata/orientation_8.jpg", 8},
	}
	for _, tc := range testCases {
		tc := tc
		f, err := os.Open(tc.path)
		if err != nil {
			t.Fatalf("%q: failed to open: %v", tc.path, err)
		}
		orient := ReadOrientation(f)
		if orient != tc.orient {
			t.Fatalf("%q: got orientation %d want %d", tc.path, orient, tc.orient)
		}
	}
}

func TestReadOrientationFails(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		data string
	}{
		{
			"empty",
			"",
		},
		{
			"missing SOI marker",
			"\xff\xe1",
		},
		{
			"missing APP1 marker",
			"\xff\xd8",
		},
		{
			"short read marker",
			"\xff\xd8\xff",
		},
		{
			"short read block size",
			"\xff\xd8\xff\xe1\x00",
		},
		{
			"invalid marker",
			"\xff\xd8\x00\xe1\x00\x00",
		},
		{
			"block size too small",
			"\xff\xd8\xff\xe0\x00\x01",
		},
		{
			"short read block",
			"\xff\xd8\xff\xe0\x00\x08\x00",
		},
		{
			"missing EXIF header",
			"\xff\xd8\xff\xe1\x00\xff",
		},
		{
			"invalid EXIF header",
			"\xff\xd8\xff\xe1\x00\xff\x00\x00\x00\x00",
		},
		{
			"missing EXIF header tail",
			"\xff\xd8\xff\xe1\x00\xff\x45\x78\x69\x66",
		},
		{
			"missing byte order tag",
			"\xff\xd8\xff\xe1\x00\xff\x45\x78\x69\x66\x00\x00",
		},
		{
			"invalid byte order tag",
			"\xff\xd8\xff\xe1\x00\xff\x45\x78\x69\x66\x00\x00\x00\x00",
		},
		{
			"missing byte order tail",
			"\xff\xd8\xff\xe1\x00\xff\x45\x78\x69\x66\x00\x00\x49\x49",
		},
		{
			"missing exif offset",
			"\xff\xd8\xff\xe1\x00\xff\x45\x78\x69\x66\x00\x00\x49\x49\x00\x2a",
		},
		{
			"invalid exif offset",
			"\xff\xd8\xff\xe1\x00\xff\x45\x78\x69\x66\x00\x00\x4d\x4d\x00\x2a\x00\x00\x00\x07",
		},
		{
			"read exif offset error",
			"\xff\xd8\xff\xe1\x00\xff\x45\x78\x69\x66\x00\x00\x4d\x4d\x00\x2a\x00\x00\x00\x09",
		},
		{
			"missing number of tags",
			"\xff\xd8\xff\xe1\x00\xff\x45\x78\x69\x66\x00\x00\x4d\x4d\x00\x2a\x00\x00\x00\x08",
		},
		{
			"zero number of tags",
			"\xff\xd8\xff\xe1\x00\xff\x45\x78\x69\x66\x00\x00\x4d\x4d\x00\x2a\x00\x00\x00\x08\x00\x00",
		},
		{
			"missing tag",
			"\xff\xd8\xff\xe1\x00\xff\x45\x78\x69\x66\x00\x00\x4d\x4d\x00\x2a\x00\x00\x00\x08\x00\x01",
		},
		{
			"missing tag offset",
			"\xff\xd8\xff\xe1\x00\xff\x45\x78\x69\x66\x00\x00\x4d\x4d\x00\x2a\x00\x00\x00\x08\x00\x01\x00\x00",
		},
		{
			"missing orientation tag",
			"\xff\xd8\xff\xe1\x00\xff\x45\x78\x69\x66\x00\x00\x4d\x4d\x00\x2a\x00\x00\x00\x08\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00",
		},
		{
			"missing orientation tag value offset",
			"\xff\xd8\xff\xe1\x00\xff\x45\x78\x69\x66\x00\x00\x4d\x4d\x00\x2a\x00\x00\x00\x08\x00\x01\x01\x12",
		},
		{
			"missing orientation value",
			"\xff\xd8\xff\xe1\x00\xff\x45\x78\x69\x66\x00\x00\x4d\x4d\x00\x2a\x00\x00\x00\x08\x00\x01\x01\x12\x00\x03\x00\x00\x00\x01",
		},
		{
			"invalid orientation value",
			"\xff\xd8\xff\xe1\x00\xff\x45\x78\x69\x66\x00\x00\x4d\x4d\x00\x2a\x00\x00\x00\x08\x00\x01\x01\x12\x00\x03\x00\x00\x00\x01\x00\x09",
		},
	}
	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if o := ReadOrientation(strings.NewReader(tc.data)); o != OrientationUnspecified {
				t.Fatalf("got orientation %d want %d", o, OrientationUnspecified)
			}
		})
	}
}
