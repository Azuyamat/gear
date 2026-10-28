package command

import (
	"slices"
	"strings"
)

// suggest returns the candidates close enough to input to be what the user
// meant, closest first.
func suggest(input string, candidates []string) []string {
	type match struct {
		name string
		dist int
	}
	limit := 1
	if len(input) > 3 {
		limit = 2
	}
	var matches []match
	for _, c := range candidates {
		d := levenshtein(strings.ToLower(input), strings.ToLower(c))
		if d <= limit || len(input) >= 2 && strings.HasPrefix(c, input) {
			matches = append(matches, match{c, d})
		}
	}
	slices.SortStableFunc(matches, func(a, b match) int { return a.dist - b.dist })

	var names []string
	for _, m := range matches {
		if !slices.Contains(names, m.name) {
			names = append(names, m.name)
		}
	}
	return names
}

func levenshtein(a, b string) int {
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(br)]
}
