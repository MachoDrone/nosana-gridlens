package nosana

import (
	"path/filepath"
	"strings"
)

type Matcher struct {
	ExactNames []string
	Patterns   []string
}

func (m Matcher) Match(name string) (bool, string) {
	name = normalizeName(name)
	lowerName := strings.ToLower(name)

	for _, exact := range m.ExactNames {
		exact = normalizeName(exact)
		if strings.EqualFold(name, exact) {
			return true, "exact name: " + exact
		}
	}

	for _, pattern := range m.Patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		lowerPattern := strings.ToLower(pattern)
		if !strings.ContainsAny(lowerPattern, "*?[") {
			if lowerName == lowerPattern {
				return true, "pattern: " + pattern
			}
			continue
		}
		matched, err := filepath.Match(lowerPattern, lowerName)
		if err == nil && matched {
			return true, "pattern: " + pattern
		}
	}

	return false, ""
}

func normalizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "/")
	return name
}
