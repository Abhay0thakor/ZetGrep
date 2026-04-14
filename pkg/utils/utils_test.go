package utils

import (
	"math"
	"testing"
)

func TestShannonEntropy(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"", 0},
		{"aaaaa", 0},
		{"abcde", math.Log2(5)}, // Each char appears once, p = 1/5. Entropy = -5 * (1/5 * log2(1/5)) = -log2(1/5) = log2(5)
		{"aabb", 1.0},           // p(a)=0.5, p(b)=0.5. Entropy = -(0.5*-1 + 0.5*-1) = 1.0
	}

	for _, tt := range tests {
		got := ShannonEntropy(tt.input)
		if math.Abs(got-tt.expected) > 1e-9 {
			t.Errorf("ShannonEntropy(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestExpandPath(t *testing.T) {
	// This is hard to test deterministically without mocking home dir,
	// but we can test non-tilde paths.
	if ExpandPath("/tmp/test") != "/tmp/test" {
		t.Errorf("expected /tmp/test, got %s", ExpandPath("/tmp/test"))
	}
	if ExpandPath("") != "" {
		t.Errorf("expected empty string, got %s", ExpandPath(""))
	}
}
