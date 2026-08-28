package mapping

import (
	"fmt"
	"regexp"
	"strings"
)

// paramSegment matches a ":name" path parameter token inside an
// already-escaped path pattern. Colons are not regexp metacharacters in Go
// (regexp.QuoteMeta leaves them untouched), so this can run directly against
// the escaped pattern.
var paramSegment = regexp.MustCompile(`:([A-Za-z_][A-Za-z0-9_]*)`)

// buildMatchPattern returns a fully anchored regular expression string used
// to test whether a request path satisfies a Redis mapping entry.
//
// It prefers the canonical MatchRegex field when present: that field is
// expected to already be a deterministic, anchored regex written by the
// mapping service. When MatchRegex is absent (backward compatibility with
// mappings written before it existed), a safe anchored pattern is derived
// from the legacy PathPattern field instead.
func buildMatchPattern(rm RedisMapping) (string, error) {
	if rm.MatchRegex != "" {
		return anchorPattern(rm.MatchRegex), nil
	}
	return pathPatternToRegex(rm.PathPattern)
}

// compileMatchPattern builds and compiles the regex for a Redis mapping
// entry. Compilation failures are returned as errors so callers can skip the
// offending mapping instead of matching against a broken pattern.
func compileMatchPattern(rm RedisMapping) (*regexp.Regexp, error) {
	pattern, err := buildMatchPattern(rm)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid mapping pattern %q: %w", pattern, err)
	}

	return re, nil
}

// pathPatternToRegex derives a safe, fully-anchored regex from a legacy
// path_pattern value. Supported syntax mirrors the mapping service's
// documented intent:
//
//   - ":name" matches exactly one path segment (no "/"), exposed as a named
//     capture group.
//   - "*"     matches within a single path segment.
//   - "**"    matches across multiple path segments.
//
// A pattern containing literal parentheses is treated as an already
// hand-authored regex fragment (the legacy raw-regex payload shape) and is
// only anchored, never re-escaped, so existing raw patterns keep working.
func pathPatternToRegex(pathPattern string) (string, error) {
	if strings.Contains(pathPattern, "(") && strings.Contains(pathPattern, ")") {
		pattern := anchorPattern(pathPattern)
		if _, err := regexp.Compile(pattern); err != nil {
			return "", fmt.Errorf("invalid raw regex path_pattern %q: %w", pathPattern, err)
		}
		return pattern, nil
	}

	escaped := regexp.QuoteMeta(pathPattern)
	escaped = paramSegment.ReplaceAllString(escaped, `(?P<$1>[^/]+)`)
	escaped = strings.ReplaceAll(escaped, `\*\*`, `.*`)
	escaped = strings.ReplaceAll(escaped, `\*`, `[^/]*`)

	return anchorPattern(escaped), nil
}

// anchorPattern ensures a regex pattern is anchored at both ends, so
// matching is exact rather than a substring/prefix search. Patterns that
// already carry an explicit anchor are left as-is.
func anchorPattern(pattern string) string {
	if !strings.HasPrefix(pattern, "^") {
		pattern = "^" + pattern
	}
	if !strings.HasSuffix(pattern, "$") {
		pattern += "$"
	}
	return pattern
}
