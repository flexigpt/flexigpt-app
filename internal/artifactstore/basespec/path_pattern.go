package basespec

import (
	"fmt"
	"path"
	"strings"
)

// ValidatePathPattern validates one portable slash-separated path pattern.
//
// A complete "**" segment has recursive semantics. Every other segment uses
// path.Match syntax, including "*", "?", and character classes.
func ValidatePathPattern(pattern string) error {
	if err := ValidateRequiredText(
		"path pattern",
		pattern,
		MaxLocatorBytes,
	); err != nil {
		return err
	}
	if strings.HasPrefix(pattern, "/") ||
		strings.ContainsAny(pattern, `\:`) {
		return fmt.Errorf(
			"%w: path pattern contains a disallowed path character",
			ErrInvalid,
		)
	}

	for segment := range strings.SplitSeq(pattern, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf(
				"%w: path pattern contains an invalid path segment",
				ErrInvalid,
			)
		}
		if segment == "**" {
			continue
		}
		if _, err := path.Match(segment, "candidate"); err != nil {
			return fmt.Errorf(
				"%w: invalid path pattern %q: %w",
				ErrInvalid,
				pattern,
				err,
			)
		}
	}
	return nil
}

// ValidatePathPatterns validates a bounded pattern list.
// Repeated patterns are harmless and preserve caller-owned configuration.
func ValidatePathPatterns(
	label string,
	patterns []string,
) error {
	if len(patterns) > MaxPathPatterns {
		return fmt.Errorf(
			"%w: %s exceeds %d entries",
			ErrInvalid,
			label,
			MaxPathPatterns,
		)
	}

	for index, pattern := range patterns {
		if err := ValidatePathPattern(pattern); err != nil {
			return fmt.Errorf("%s[%d]: %w", label, index, err)
		}
	}
	return nil
}

// MatchPathPattern matches one source-relative path.
//
// Unlike path.Match, a complete "**" segment consumes zero or more complete
// path segments. "*" and "?" remain confined to one segment.
func MatchPathPattern(
	pattern string,
	value string,
) (bool, error) {
	if err := ValidatePathPattern(pattern); err != nil {
		return false, err
	}
	if err := validatePatternValue(value); err != nil {
		return false, err
	}

	patternSegments := strings.Split(pattern, "/")
	valueSegments := strings.Split(value, "/")

	type state struct {
		patternIndex int
		valueIndex   int
	}

	known := make(map[state]bool)
	memo := make(map[state]bool)

	var match func(int, int) (bool, error)
	match = func(
		patternIndex int,
		valueIndex int,
	) (bool, error) {
		key := state{
			patternIndex: patternIndex,
			valueIndex:   valueIndex,
		}
		if known[key] {
			return memo[key], nil
		}

		var (
			matched bool
			err     error
		)

		switch {
		case patternIndex == len(patternSegments):
			matched = valueIndex == len(valueSegments)

		case patternSegments[patternIndex] == "**":
			// First try matching no path segment. If that fails, consume one
			// segment and keep the recursive pattern active.
			matched, err = match(patternIndex+1, valueIndex)
			if err == nil &&
				!matched &&
				valueIndex < len(valueSegments) {
				matched, err = match(
					patternIndex,
					valueIndex+1,
				)
			}

		case valueIndex < len(valueSegments):
			var segmentMatched bool
			segmentMatched, err = path.Match(
				patternSegments[patternIndex],
				valueSegments[valueIndex],
			)
			if err == nil && segmentMatched {
				matched, err = match(
					patternIndex+1,
					valueIndex+1,
				)
			}
		}
		if err != nil {
			return false, err
		}

		known[key] = true
		memo[key] = matched
		return matched, nil
	}

	return match(0, 0)
}

// MatchPathSelection evaluates inclusions before exclusions.
//
// An empty inclusion list selects all paths. Any matching exclusion removes
// the path regardless of which inclusion selected it.
func MatchPathSelection(
	value string,
	include []string,
	exclude []string,
) (bool, error) {
	included := len(include) == 0
	for _, pattern := range include {
		matched, err := MatchPathPattern(pattern, value)
		if err != nil {
			return false, err
		}
		if matched {
			included = true
			break
		}
	}
	if !included {
		return false, nil
	}

	for _, pattern := range exclude {
		matched, err := MatchPathPattern(pattern, value)
		if err != nil {
			return false, err
		}
		if matched {
			return false, nil
		}
	}
	return true, nil
}

func validatePatternValue(value string) error {
	if value == "." {
		return nil
	}
	if err := Locator(value).Validate(false); err != nil {
		return fmt.Errorf("path pattern candidate: %w", err)
	}
	return nil
}
