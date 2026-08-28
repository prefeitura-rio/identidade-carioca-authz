package mapping

import "testing"

func TestBuildMatchPattern_ExactPathIsAnchored(t *testing.T) {
	tests := []struct {
		name        string
		pathPattern string
		path        string
		wantMatch   bool
	}{
		{
			name:        "exact match",
			pathPattern: "/api/v1/healthz",
			path:        "/api/v1/healthz",
			wantMatch:   true,
		},
		{
			name:        "longer path is not over-matched as a substring",
			pathPattern: "/api/v1/healthz",
			path:        "/api/v1/healthzXYZ",
			wantMatch:   false,
		},
		{
			name:        "path nested under an unrelated prefix is not matched",
			pathPattern: "/api/v1/healthz",
			path:        "/other/api/v1/healthz",
			wantMatch:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re, err := compileMatchPattern(RedisMapping{PathPattern: tt.pathPattern})
			if err != nil {
				t.Fatalf("compileMatchPattern(%q) returned error: %v", tt.pathPattern, err)
			}
			if got := re.MatchString(tt.path); got != tt.wantMatch {
				t.Errorf("pattern %q matching %q = %v, want %v", tt.pathPattern, tt.path, got, tt.wantMatch)
			}
		})
	}
}

func TestBuildMatchPattern_SingleSegmentWildcard(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantMatch bool
	}{
		{name: "matches single segment", path: "/api/v1/users/123", wantMatch: true},
		{name: "does not cross a segment boundary", path: "/api/v1/users/123/extra", wantMatch: false},
		{name: "does not match missing segment", path: "/api/v1/users/", wantMatch: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re, err := compileMatchPattern(RedisMapping{PathPattern: "/api/v1/users/*"})
			if err != nil {
				t.Fatalf("compileMatchPattern returned error: %v", err)
			}
			if got := re.MatchString(tt.path); got != tt.wantMatch {
				t.Errorf("matching %q = %v, want %v", tt.path, got, tt.wantMatch)
			}
		})
	}
}

func TestBuildMatchPattern_MultiSegmentWildcard(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantMatch bool
	}{
		{name: "matches single segment", path: "/api/v1/users/123", wantMatch: true},
		{name: "matches multiple segments", path: "/api/v1/users/123/extra/more", wantMatch: true},
		{name: "does not match sibling prefix", path: "/api/v1/usersXYZ/123", wantMatch: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re, err := compileMatchPattern(RedisMapping{PathPattern: "/api/v1/users/**"})
			if err != nil {
				t.Fatalf("compileMatchPattern returned error: %v", err)
			}
			if got := re.MatchString(tt.path); got != tt.wantMatch {
				t.Errorf("matching %q = %v, want %v", tt.path, got, tt.wantMatch)
			}
		})
	}
}

func TestBuildMatchPattern_NamedParam(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantMatch bool
	}{
		{name: "matches a single segment value", path: "/api/v1/users/123", wantMatch: true},
		{name: "does not cross a segment boundary", path: "/api/v1/users/123/extra", wantMatch: false},
		{name: "does not match empty param", path: "/api/v1/users/", wantMatch: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re, err := compileMatchPattern(RedisMapping{PathPattern: "/api/v1/users/:id"})
			if err != nil {
				t.Fatalf("compileMatchPattern returned error: %v", err)
			}
			if got := re.MatchString(tt.path); got != tt.wantMatch {
				t.Errorf("matching %q = %v, want %v", tt.path, got, tt.wantMatch)
			}
		})
	}
}

func TestBuildMatchPattern_LegacyRawRegexPayloadIsAnchoredNotReescaped(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantMatch bool
	}{
		{name: "matches captured suffix", path: "/heimdall-admin/api/v1/users/123", wantMatch: true},
		{name: "matches nested suffix since group is greedy", path: "/heimdall-admin/api/v1/users/123/extra", wantMatch: true},
		{name: "does not match unrelated prefix", path: "/other/heimdall-admin/api/v1/users/123", wantMatch: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re, err := compileMatchPattern(RedisMapping{PathPattern: "/heimdall-admin/api/v1/users/(.*)"})
			if err != nil {
				t.Fatalf("compileMatchPattern returned error: %v", err)
			}
			if got := re.MatchString(tt.path); got != tt.wantMatch {
				t.Errorf("matching %q = %v, want %v", tt.path, got, tt.wantMatch)
			}
		})
	}
}

func TestBuildMatchPattern_InvalidRawRegexIsRejected(t *testing.T) {
	_, err := compileMatchPattern(RedisMapping{PathPattern: "/api/v1/(a|b))"})
	if err == nil {
		t.Fatal("expected an error for an invalid raw regex path_pattern, got nil")
	}
}

func TestBuildMatchPattern_MatchRegexTakesPrecedenceOverPathPattern(t *testing.T) {
	rm := RedisMapping{
		// A path_pattern that would match anything if it were used, proving
		// match_regex (not path_pattern) drove the decision below.
		PathPattern: "/api/v1/**",
		MatchRegex:  `^/api/v2/things/[0-9]+$`,
	}

	re, err := compileMatchPattern(rm)
	if err != nil {
		t.Fatalf("compileMatchPattern returned error: %v", err)
	}

	if !re.MatchString("/api/v2/things/42") {
		t.Error("expected match_regex to match its own pattern")
	}
	if re.MatchString("/api/v1/anything") {
		t.Error("expected match_regex to take precedence over path_pattern, but path_pattern was used")
	}
}

func TestBuildMatchPattern_MatchRegexIsAnchoredWhenAuthorMissedIt(t *testing.T) {
	re, err := compileMatchPattern(RedisMapping{MatchRegex: "/api/v3/foo"})
	if err != nil {
		t.Fatalf("compileMatchPattern returned error: %v", err)
	}

	if !re.MatchString("/api/v3/foo") {
		t.Error("expected exact match to succeed")
	}
	if re.MatchString("/api/v3/foobar") {
		t.Error("expected un-anchored match_regex to be defensively anchored, but suffix over-matched")
	}
}
