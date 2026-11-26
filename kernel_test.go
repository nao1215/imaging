package imaging

import (
	"math"
	"testing"
)

// TestFilterKernelsMatchTheirFormulas compares every resampling kernel with the
// textbook form of its formula, written independently of resize.go: the
// windows as the cosines, polynomials and squares they are usually given as,
// and the cubic splines expanded for their B and C. Only Lanczos, CatmullRom,
// Linear and NearestNeighbor have golden images, so this is what checks the
// other filters at all; before it, mutation testing found most of the
// package's surviving mutants in their kernels.
func TestFilterKernelsMatchTheirFormulas(t *testing.T) {
	t.Parallel()

	sincRef := func(x float64) float64 {
		if x == 0 {
			return 1
		}
		return math.Sin(math.Pi*x) / (math.Pi * x)
	}
	cube := func(x float64) float64 { return x * x * x }

	testCases := []struct {
		name    string
		filter  ResampleFilter
		support float64
		want    func(x float64) float64 // for 0 <= x < support
	}{
		{"Box", Box, 0.5, func(float64) float64 { return 1 }},
		{"Linear", Linear, 1, func(x float64) float64 { return 1 - x }},
		{"Hermite", Hermite, 1, func(x float64) float64 { return 2*cube(x) - 3*x*x + 1 }},
		{"MitchellNetravali", MitchellNetravali, 2, func(x float64) float64 {
			if x < 1 {
				return (7*cube(x) - 12*x*x + 16.0/3) / 6
			}
			return (-7.0/3*cube(x) + 12*x*x - 20*x + 32.0/3) / 6
		}},
		{"CatmullRom", CatmullRom, 2, func(x float64) float64 {
			if x < 1 {
				return 1.5*cube(x) - 2.5*x*x + 1
			}
			return -0.5*cube(x) + 2.5*x*x - 4*x + 2
		}},
		{"BSpline", BSpline, 2, func(x float64) float64 {
			if x < 1 {
				return 2.0/3 - x*x + cube(x)/2
			}
			return cube(2-x) / 6
		}},
		{"Gaussian", Gaussian, 2, func(x float64) float64 { return math.Exp(-2 * x * x) }},
		{"Bartlett", Bartlett, 3, func(x float64) float64 { return sincRef(x) * (1 - x/3) }},
		{"Lanczos", Lanczos, 3, func(x float64) float64 { return sincRef(x) * sincRef(x/3) }},
		{"Hann", Hann, 3, func(x float64) float64 { c := math.Cos(math.Pi * x / 6); return sincRef(x) * c * c }},
		{"Hamming", Hamming, 3, func(x float64) float64 { return sincRef(x) * (0.54 + 0.46*math.Cos(math.Pi*x/3)) }},
		{"Blackman", Blackman, 3, func(x float64) float64 {
			return sincRef(x) * (0.42 + 0.5*math.Cos(math.Pi*x/3) + 0.08*math.Cos(2*math.Pi*x/3))
		}},
		{"Welch", Welch, 3, func(x float64) float64 { return sincRef(x) * (1 - (x/3)*(x/3)) }},
		{"Cosine", Cosine, 3, func(x float64) float64 { return sincRef(x) * math.Cos(math.Pi*x/6) }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.filter.Support != tc.support {
				t.Fatalf("Support = %v, want %v", tc.filter.Support, tc.support)
			}
			for i := 0; i <= 400; i++ {
				x := float64(i) / 100 // 0 to 4 in steps of 0.01, past every support
				want := 0.0
				if x < tc.support || (tc.name == "Box" && x == tc.support) {
					want = tc.want(x)
				}
				for _, sx := range []float64{x, -x} {
					if got := tc.filter.Kernel(sx); math.Abs(got-want) > 1e-12 {
						t.Fatalf("Kernel(%v) = %v, want %v", sx, got, want)
					}
				}
			}
		})
	}
}
