package scanner

import (
	"strings"
	"github.com/cloudflare/ahocorasick"
)

// LiteralMatcher uses Aho-Corasick for fast multi-pattern literal matching
type LiteralMatcher struct {
	matcher *ahocorasick.Matcher
	idMap   map[int]string
	litMap  map[int]string
}

func NewLiteralMatcher(literals map[string]string) *LiteralMatcher {
	if len(literals) == 0 {
		return nil
	}

	keys := make([][]byte, 0, len(literals))
	idMap := make(map[int]string)
	litMap := make(map[int]string)
	i := 0
	for lit, name := range literals {
		keys = append(keys, []byte(lit))
		idMap[i] = name
		litMap[i] = lit
		i++
	}

	return &LiteralMatcher{
		matcher: ahocorasick.NewMatcher(keys),
		idMap:   idMap,
		litMap:  litMap,
	}
}

type MatchResult struct {
	Name    string
	Literal string
}

func (m *LiteralMatcher) Match(content string) []MatchResult {
	if m == nil || m.matcher == nil {
		return nil
	}

	matches := m.matcher.Match([]byte(content))
	if len(matches) == 0 {
		return nil
	}

	var results []MatchResult
	seen := make(map[string]struct{})
	for _, id := range matches {
		name := m.idMap[id]
		lit := m.litMap[id]
		key := name + ":" + lit
		if _, ok := seen[key]; !ok {
			results = append(results, MatchResult{Name: name, Literal: lit})
			seen[key] = struct{}{}
		}
	}
	return results
}

// IsLiteral checks if a pattern string is a simple literal (no regex special chars)
func IsLiteral(p string) bool {
	special := `\.+*?()|[]{}^$`
	for _, char := range special {
		if strings.ContainsRune(p, char) {
			return false
		}
	}
	return true
}
