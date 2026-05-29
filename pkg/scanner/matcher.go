package scanner

import (
	"regexp"
	"strings"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/bits-and-blooms/bloom/v3"
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

// PreFilter uses a Bloom Filter to quickly discard lines that definitely don't match any pattern
type PreFilter struct {
	filter *bloom.BloomFilter
}

func NewPreFilter(patterns []models.Pattern) *PreFilter {
	// Estimate capacity: Number of patterns * average keywords
	capacity := uint(len(patterns) * 5)
	if capacity < 1000 {
		capacity = 1000
	}
	f := bloom.NewWithEstimates(capacity, 0.01)

	added := 0
	keywordRe := regexp.MustCompile(`[a-zA-Z0-9]{3,}`)
	for _, p := range patterns {
		// Extract potential keywords from regex
		keywords := keywordRe.FindAllString(p.Pattern, -1)
		for _, kw := range keywords {
			f.Add([]byte(strings.ToLower(kw)))
			added++
		}
	}

	if added == 0 {
		return nil
	}

	return &PreFilter{filter: f}
}

func (pf *PreFilter) MayMatch(content []byte) bool {
	if pf == nil || pf.filter == nil {
		return true
	}
	
	// Tokenize content loosely (by common delimiters)
	// We use a simpler regex for speed here
	keywordRe := regexp.MustCompile(`[a-zA-Z0-9]{3,}`)
	tokens := keywordRe.FindAll(content, -1)
	for _, token := range tokens {
		if pf.filter.Test(bytesToLower(token)) {
			return true // Found a potential match
		}
	}
	return false
}

func bytesToLower(b []byte) []byte {
	res := make([]byte, len(b))
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			res[i] = c + ('a' - 'A')
		} else {
			res[i] = c
		}
	}
	return res
}
