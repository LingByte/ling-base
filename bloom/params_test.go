package bloom_test

import (
	"math"
	"testing"

	"github.com/LingByte/ling-base/bloom"
)

func TestEstimate(t *testing.T) {
	cases := []struct {
		name    string
		n       uint64
		p       float64
		wantErr error
	}{
		{"1k @ 1%", 1000, 0.01, nil},
		{"1M @ 0.1%", 1_000_000, 0.001, nil},
		{"10 @ 5%", 10, 0.05, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ps, err := bloom.Estimate(tc.n, tc.p)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ps.M == 0 || ps.K == 0 {
				t.Fatalf("got zero params: %+v", ps)
			}
			// k should be reasonable (>=1, < ~30 for typical p).
			if ps.K < 1 || ps.K > 30 {
				t.Fatalf("k out of expected range: %d", ps.K)
			}
			// m should be >= n.
			if ps.M < tc.n {
				t.Fatalf("m < n: %d < %d", ps.M, tc.n)
			}
		})
	}
}

func TestEstimateKnownValues(t *testing.T) {
	// n=1000, p=0.01 -> m ~= 9585, k ~= 7 (well-known reference values).
	ps, err := bloom.Estimate(1000, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(float64(ps.M)-9585) > 5 {
		t.Fatalf("m = %d, want ~9585", ps.M)
	}
	if ps.K != 7 {
		t.Fatalf("k = %d, want 7", ps.K)
	}
}

func TestEstimateErrors(t *testing.T) {
	if _, err := bloom.Estimate(0, 0.01); err != bloom.ErrInvalidCapacity {
		t.Fatalf("n=0 = %v, want ErrInvalidCapacity", err)
	}
	if _, err := bloom.Estimate(100, 0); err != bloom.ErrInvalidFalsePositiveRate {
		t.Fatalf("p=0 = %v, want ErrInvalidFalsePositiveRate", err)
	}
	if _, err := bloom.Estimate(100, 1); err != bloom.ErrInvalidFalsePositiveRate {
		t.Fatalf("p=1 = %v, want ErrInvalidFalsePositiveRate", err)
	}
	if _, err := bloom.Estimate(100, -0.1); err != bloom.ErrInvalidFalsePositiveRate {
		t.Fatalf("p<0 = %v, want ErrInvalidFalsePositiveRate", err)
	}
}

func TestBitsToBytes(t *testing.T) {
	if bloom.BitsToBytes(0) != 0 {
		t.Fatal("0 bits")
	}
	if bloom.BitsToBytes(1) != 1 {
		t.Fatal("1 bit -> 1 byte")
	}
	if bloom.BitsToBytes(8) != 1 {
		t.Fatal("8 bits -> 1 byte")
	}
	if bloom.BitsToBytes(9) != 2 {
		t.Fatal("9 bits -> 2 bytes")
	}
}
