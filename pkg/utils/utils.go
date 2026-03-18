package utils

import (
	"math"
)

// ShannonEntropy calculates the Shannon entropy of a string.
func ShannonEntropy(s string) float64 {
	if s == "" {
		return 0
	}
	counts := make(map[rune]int)
	for _, r := range s {
		counts[r]++
	}
	entropy := 0.0
	length := float64(len(s))
	for _, count := range counts {
		freq := float64(count) / length
		entropy -= freq * math.Log2(freq)
	}
	return entropy
}
