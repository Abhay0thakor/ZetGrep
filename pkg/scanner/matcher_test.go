package scanner

import (
	"testing"
)

func TestLiteralMatcher(t *testing.T) {
	literals := map[string]string{
		"password": "pass-pattern",
		"apikey":   "key-pattern",
		"secret":   "secret-pattern",
	}

	m := NewLiteralMatcher(literals)

	tests := []struct {
		content  string
		expected []string
	}{
		{"this is a password", []string{"pass-pattern"}},
		{"my apikey is 123", []string{"key-pattern"}},
		{"no matches here", nil},
		{"password and apikey", []string{"pass-pattern", "key-pattern"}},
	}

	for _, tt := range tests {
		matches := m.Match(tt.content)
		if len(matches) != len(tt.expected) {
			t.Errorf("Match(%q) expected %v, got %v", tt.content, tt.expected, matches)
			continue
		}

		for _, exp := range tt.expected {
			found := false
			for _, got := range matches {
				if got.Name == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Match(%q) expected to find %s", tt.content, exp)
			}
		}
	}
}

func TestIsLiteral(t *testing.T) {
	tests := []struct {
		p   string
		exp bool
	}{
		{"password", true},
		{"pass.*word", false},
		{"[0-9]+", false},
		{"API_KEY", true},
		{"(a|b)", false},
	}

	for _, tt := range tests {
		if got := IsLiteral(tt.p); got != tt.exp {
			t.Errorf("IsLiteral(%q) = %v, want %v", tt.p, got, tt.exp)
		}
	}
}
