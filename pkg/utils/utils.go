package utils

import (
	"math"
	"os"
	"path/filepath"
	"strings"
)

func ShannonEntropy(data string) float64 {
	if len(data) == 0 {
		return 0
	}
	frequencies := make(map[rune]float64)
	for _, char := range data {
		frequencies[char]++
	}
	var entropy float64
	lenData := float64(len(data))
	for _, freq := range frequencies {
		p := freq / lenData
		entropy -= p * math.Log2(p)
	}
	return entropy
}

// ExpandPath handles tilde (~) expansion to the user's home directory
func ExpandPath(path string) string {
	if path == "" || !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}
