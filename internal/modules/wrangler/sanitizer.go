package wrangler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// SanitizeConfig sanitizes a parsed WranglerConfig structure by optionally merging
// an environment block, stripping unallowed celld keys (such as containers),
// and removing top-level env blocks so celld deploy does not error on unknown keys.
func SanitizeConfig(cfg *WranglerConfig, envName string) *WranglerConfig {
	if cfg == nil {
		return nil
	}

	effective := cfg
	if envName != "" {
		effective = MergeEnvironment(cfg, envName)
	} else {
		// Clone shallowly to avoid mutating caller's config
		clone := *cfg
		effective = &clone
	}

	// celld strictly rejects top-level "env" and "containers"
	effective.Env = nil
	effective.Containers = nil

	return effective
}

// SanitizeForCelld inspects raw wrangler configuration bytes (JSON, JSONC, or TOML),
// strips all keys strictly prohibited by celld (such as routes, route, account_id,
// workers_dev, proprietary cloud service keys, containers, and env blocks),
// extracts any route definitions for ingress registration, and serializes
// the clean configuration into valid celld JSON.
func SanitizeForCelld(raw []byte) ([]byte, []string, error) {
	return SanitizeForCelldWithEnv(raw, "")
}

// SanitizeForCelldWithEnv performs sanitization for celld, optionally applying
// an environment override before stripping prohibited keys.
func SanitizeForCelldWithEnv(raw []byte, envName string) ([]byte, []string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil, fmt.Errorf("configuration is empty")
	}

	format := DetectFormat(trimmed)
	var rawMap map[string]any
	var cleanedBytes []byte

	if format == "toml" {
		if err := toml.Unmarshal(trimmed, &rawMap); err != nil {
			// Fallback: try parsing as JSONC
			cleanedJSON := StripTrailingCommas(StripComments(trimmed))
			if jErr := json.Unmarshal(cleanedJSON, &rawMap); jErr == nil {
				format = "json"
				cleanedBytes = cleanedJSON
			} else {
				return nil, nil, fmt.Errorf("failed to parse TOML configuration: %w (JSON fallback error: %v)", err, jErr)
			}
		} else {
			cleanedBytes = trimmed
		}
	} else {
		cleanedJSON := StripTrailingCommas(StripComments(trimmed))
		if err := json.Unmarshal(cleanedJSON, &rawMap); err != nil {
			// Fallback: try parsing as TOML
			if tErr := toml.Unmarshal(trimmed, &rawMap); tErr == nil {
				format = "toml"
				cleanedBytes = trimmed
			} else {
				return nil, nil, fmt.Errorf("failed to parse JSON configuration: %w (TOML fallback error: %v)", err, tErr)
			}
		} else {
			cleanedBytes = cleanedJSON
		}
	}

	var cfg WranglerConfig
	if format == "toml" {
		if err := toml.Unmarshal(cleanedBytes, &cfg); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal TOML into wrangler config: %w", err)
		}
	} else {
		if err := json.Unmarshal(cleanedBytes, &cfg); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal JSON into wrangler config: %w", err)
		}
	}

	sanitizedCfg := SanitizeConfig(&cfg, envName)

	sanitizedJSON, err := json.MarshalIndent(sanitizedCfg, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize sanitized celld configuration: %w", err)
	}

	extractedRoutes := extractRoutes(rawMap, envName)

	return sanitizedJSON, extractedRoutes, nil
}

// extractRoutes pulls all route and routes declarations from the raw config map.
// If targetEnv is specified, it extracts root routes and the requested environment's routes;
// if targetEnv is empty, it extracts root routes and all environment routes, deduplicating them.
func extractRoutes(rawMap map[string]any, targetEnv string) []string {
	if rawMap == nil {
		return []string{}
	}

	var routes []string

	processRouteValue := func(val any) {
		if val == nil {
			return
		}
		switch v := val.(type) {
		case string:
			trimmed := strings.TrimSpace(v)
			if trimmed != "" {
				routes = append(routes, trimmed)
			}
		case map[string]any:
			if patternVal, ok := v["pattern"].(string); ok && strings.TrimSpace(patternVal) != "" {
				routes = append(routes, strings.TrimSpace(patternVal))
			} else if customDomain, ok := v["custom_domain"].(string); ok && strings.TrimSpace(customDomain) != "" {
				routes = append(routes, strings.TrimSpace(customDomain))
			}
		}
	}

	processRoutes := func(m map[string]any) {
		if m == nil {
			return
		}
		if routeVal, ok := m["route"]; ok {
			processRouteValue(routeVal)
		}
		if routesVal, ok := m["routes"]; ok {
			if sliceVal, ok := routesVal.([]any); ok {
				for _, item := range sliceVal {
					processRouteValue(item)
				}
			} else {
				processRouteValue(routesVal)
			}
		}
	}

	// 1. Root level routes
	processRoutes(rawMap)

	// 2. Environment level routes
	if envVal, ok := rawMap["env"]; ok {
		if envMap, ok := envVal.(map[string]any); ok {
			if targetEnv != "" {
				if subMap, ok := envMap[targetEnv].(map[string]any); ok {
					processRoutes(subMap)
				}
			} else {
				var envNames []string
				for name := range envMap {
					envNames = append(envNames, name)
				}
				sort.Strings(envNames)

				for _, name := range envNames {
					if subMap, ok := envMap[name].(map[string]any); ok {
						processRoutes(subMap)
					}
				}
			}
		}
	}

	// Deduplicate preserving order
	unique := make([]string, 0, len(routes))
	seen := make(map[string]struct{}, len(routes))
	for _, r := range routes {
		if _, exists := seen[r]; !exists {
			seen[r] = struct{}{}
			unique = append(unique, r)
		}
	}

	return unique
}
