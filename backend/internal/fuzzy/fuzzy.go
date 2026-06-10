package fuzzy

import (
	"sort"
	"strings"
)

// Levenshtein computes the edit distance between two strings.
func Levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// Match is a scored suggestion.
type Match struct {
	Value    string `json:"value"`
	Distance int    `json:"distance"`
}

// Suggest returns candidates close to query, sorted by relevance.
// Substring matches are boosted; results beyond maxDistance are dropped.
func Suggest(query string, candidates []string, limit, maxDistance int) []Match {
	q := normalize(query)
	if q == "" {
		return nil
	}
	out := make([]Match, 0, len(candidates))
	for _, c := range candidates {
		nc := normalize(c)
		d := Levenshtein(q, nc)
		if strings.Contains(nc, q) || strings.Contains(q, nc) {
			d = 0
		}
		if d <= maxDistance {
			out = append(out, Match{Value: c, Distance: d})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Distance != out[j].Distance {
			return out[i].Distance < out[j].Distance
		}
		return out[i].Value < out[j].Value
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
