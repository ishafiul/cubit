package wrangler_test

import (
	"testing"

	"github.com/ishaf/cubit/internal/modules/wrangler"
)

func TestParseRoutePattern(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		expectError bool
		expected    []wrangler.ExtractedRoute
	}{
		{
			name:    "a standard hostname with wildcard path",
			pattern: "api.example.com/*",
			expected: []wrangler.ExtractedRoute{
				{Hostname: "api.example.com", PathPrefix: "/"},
			},
		},
		{
			name:    "a hostname with subpath pattern",
			pattern: "example.com/api/*",
			expected: []wrangler.ExtractedRoute{
				{Hostname: "example.com", PathPrefix: "/api"},
			},
		},
		{
			name:    "a wildcard subdomain pattern (*sub.domain.com/path*)",
			pattern: "*sub.domain.com/path*",
			expected: []wrangler.ExtractedRoute{
				{Hostname: "sub.domain.com", PathPrefix: "/path"},
				{Hostname: "*.sub.domain.com", PathPrefix: "/path"},
			},
		},
		{
			name:    "an explicit wildcard subdomain pattern (*.example.com/*)",
			pattern: "*.example.com/*",
			expected: []wrangler.ExtractedRoute{
				{Hostname: "*.example.com", PathPrefix: "/"},
			},
		},
		{
			name:    "a route with HTTP scheme and port",
			pattern: "https://api.example.com:8443/v1/*",
			expected: []wrangler.ExtractedRoute{
				{Hostname: "api.example.com", PathPrefix: "/v1"},
			},
		},
		{
			name:    "a bare custom domain with no path",
			pattern: "custom.domain.org",
			expected: []wrangler.ExtractedRoute{
				{Hostname: "custom.domain.org", PathPrefix: "/"},
			},
		},
		{
			name:        "an empty pattern",
			pattern:     "",
			expectError: true,
		},
		{
			name:        "a whitespace pattern",
			pattern:     "   ",
			expectError: true,
		},
		{
			name:        "a path only pattern without hostname",
			pattern:     "/only-a-path/*",
			expectError: true,
		},
		{
			name:        "an invalid host without TLD",
			pattern:     "invalid_host_no_tld",
			expectError: true,
		},
		{
			name:        "a single asterisk",
			pattern:     "*",
			expectError: true,
		},
		{
			name:        "an asterisk dot only pattern",
			pattern:     "*.",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run("Given "+tt.name, func(t *testing.T) {
			t.Run("When parsed", func(t *testing.T) {
				routes, err := wrangler.ParseRoutePattern(tt.pattern)

				t.Run("Then outcome matches expectation", func(t *testing.T) {
					if tt.expectError {
						if err == nil {
							t.Fatalf("expected error for pattern %q, got nil", tt.pattern)
						}
						return
					}

					if err != nil {
						t.Fatalf("unexpected error for pattern %q: %v", tt.pattern, err)
					}
					if len(routes) != len(tt.expected) {
						t.Fatalf("expected %d routes, got %d", len(tt.expected), len(routes))
					}
					for i, exp := range tt.expected {
						if routes[i].Hostname != exp.Hostname || routes[i].PathPrefix != exp.PathPrefix {
							t.Errorf("route[%d] mismatch: expected %+v, got %+v", i, exp, routes[i])
						}
					}
				})
			})
		})
	}
}

func TestExtractRouteRules(t *testing.T) {
	t.Run("Given multiple route patterns with duplicates and wildcards", func(t *testing.T) {
		patterns := []string{
			"api.example.com/*",
			"*sub.domain.com/path*",
			"api.example.com/*", // duplicate
			"custom.domain.org",
		}

		t.Run("When route rules are extracted", func(t *testing.T) {
			rules, err := wrangler.ExtractRouteRules(patterns)

			t.Run("Then all rules are parsed and deduplicated", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}

				if len(rules) != 4 {
					t.Fatalf("expected 4 deduplicated rules, got %d", len(rules))
				}

				expected := []wrangler.ExtractedRoute{
					{Hostname: "api.example.com", PathPrefix: "/"},
					{Hostname: "sub.domain.com", PathPrefix: "/path"},
					{Hostname: "*.sub.domain.com", PathPrefix: "/path"},
					{Hostname: "custom.domain.org", PathPrefix: "/"},
				}

				for i, exp := range expected {
					if rules[i].Hostname != exp.Hostname || rules[i].PathPrefix != exp.PathPrefix {
						t.Errorf("rule[%d] mismatch: expected %+v, got %+v", i, exp, rules[i])
					}
				}
			})
		})
	})
}
