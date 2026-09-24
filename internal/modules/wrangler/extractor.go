package wrangler

import (
	"fmt"
	"strings"

	"github.com/ishaf/cubit/internal/domain"
)

// ExtractedRoute represents a normalized ingress route extracted from Wrangler configuration.
type ExtractedRoute struct {
	Hostname   string
	PathPrefix string
}

// ParseRoutePattern parses a single Cloudflare route pattern (e.g. "api.example.com/*", "*sub.domain.com/path*")
// into one or more ExtractedRoute rules with normalized Hostname and PathPrefix.
func ParseRoutePattern(pattern string) ([]ExtractedRoute, error) {
	raw := strings.TrimSpace(pattern)
	if raw == "" {
		return nil, fmt.Errorf("route pattern cannot be empty")
	}

	// Strip URL scheme if present (e.g. https:// or http://)
	if idx := strings.Index(raw, "://"); idx != -1 {
		raw = raw[idx+3:]
	}

	// Separate host part from path part
	var hostPart, pathPart string
	slashIdx := strings.Index(raw, "/")
	if slashIdx == -1 {
		hostPart = raw
		pathPart = ""
	} else {
		hostPart = raw[:slashIdx]
		pathPart = raw[slashIdx:]
	}

	// Strip port if present in host
	if colonIdx := strings.Index(hostPart, ":"); colonIdx != -1 {
		hostPart = hostPart[:colonIdx]
	}

	hostPart = strings.ToLower(strings.TrimSpace(hostPart))
	if hostPart == "" || hostPart == "*" || hostPart == "*." {
		return nil, fmt.Errorf("invalid route pattern: missing or invalid hostname in %q", pattern)
	}

	var hostnames []string
	if strings.HasPrefix(hostPart, "*.") {
		// Explicit wildcard prefix: *.example.com
		apex := strings.TrimPrefix(hostPart, "*.")
		if !domain.IsValidHostname(apex) {
			return nil, fmt.Errorf("invalid wildcard hostname in %q: %s", pattern, hostPart)
		}
		hostnames = []string{hostPart}
	} else if strings.HasPrefix(hostPart, "*") {
		// Wildcard without dot: *example.com or *sub.domain.com
		apex := strings.TrimPrefix(hostPart, "*")
		if !domain.IsValidHostname(apex) {
			return nil, fmt.Errorf("invalid wildcard hostname in %q: %s", pattern, hostPart)
		}
		// Expand to both apex domain and wildcard subdomain
		hostnames = []string{apex, "*." + apex}
	} else {
		// Regular hostname: api.example.com
		if !domain.IsValidHostname(hostPart) {
			return nil, fmt.Errorf("invalid hostname format in %q: %s", pattern, hostPart)
		}
		hostnames = []string{hostPart}
	}

	// Normalize path prefix
	pathPrefix := "/" + strings.TrimPrefix(pathPart, "/")
	pathPrefix = strings.TrimSuffix(pathPrefix, "*")
	if len(pathPrefix) > 1 {
		pathPrefix = strings.TrimSuffix(pathPrefix, "/")
	}
	if pathPrefix == "" {
		pathPrefix = "/"
	}

	routes := make([]ExtractedRoute, 0, len(hostnames))
	for _, h := range hostnames {
		routes = append(routes, ExtractedRoute{
			Hostname:   h,
			PathPrefix: pathPrefix,
		})
	}

	return routes, nil
}

// ExtractRouteRules parses multiple raw route pattern strings, deduplicating the resulting ExtractedRoute rules.
func ExtractRouteRules(patterns []string) ([]ExtractedRoute, error) {
	var allRoutes []ExtractedRoute
	seen := make(map[string]struct{})

	for _, p := range patterns {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}

		routes, err := ParseRoutePattern(trimmed)
		if err != nil {
			return nil, err
		}

		for _, r := range routes {
			key := r.Hostname + ":" + r.PathPrefix
			if _, exists := seen[key]; !exists {
				seen[key] = struct{}{}
				allRoutes = append(allRoutes, r)
			}
		}
	}

	return allRoutes, nil
}
