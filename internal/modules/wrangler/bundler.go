package wrangler

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// resolveLoader maps Cloudflare custom module rule types (Text, Data, CompiledWasm)
// to esbuild loader types. Returns false if the rule type is unsupported.
func resolveLoader(ruleType string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(ruleType)) {
	case "text":
		return "text", true
	case "data":
		return "binary", true
	case "compiledwasm", "wasm":
		return "binary", true
	default:
		return "", false
	}
}

// extractExtensionFromGlob parses a glob or file pattern and extracts the file extension with leading dot.
func extractExtensionFromGlob(pattern string) string {
	p := strings.TrimSpace(pattern)
	if p == "" {
		return ""
	}
	lastDot := strings.LastIndex(p, ".")
	if lastDot == -1 {
		return ""
	}
	ext := p[lastDot:]
	if len(ext) <= 1 {
		return ""
	}
	if strings.ContainsAny(ext, "/*?{}[]") {
		return ""
	}
	return strings.ToLower(ext)
}

// formatDefineValue converts any value (string, number, boolean) into a valid esbuild define argument.
// For direct CLI execution without a shell, strings are wrapped in double quotes (e.g. "value")
// so esbuild replaces the identifier with a string literal in the bundle.
func formatDefineValue(v any) string {
	switch val := v.(type) {
	case string:
		unquoted := strings.TrimSpace(val)
		// Strip any outer quotes if the user already provided them
		if strings.HasPrefix(unquoted, "\\\"") && strings.HasSuffix(unquoted, "\\\"") && len(unquoted) >= 4 {
			unquoted = unquoted[2 : len(unquoted)-2]
		} else if strings.HasPrefix(unquoted, "\"") && strings.HasSuffix(unquoted, "\"") && len(unquoted) >= 2 {
			unquoted = unquoted[1 : len(unquoted)-1]
		} else if strings.HasPrefix(unquoted, "'") && strings.HasSuffix(unquoted, "'") && len(unquoted) >= 2 {
			unquoted = unquoted[1 : len(unquoted)-1]
		}
		return fmt.Sprintf("\"%s\"", unquoted)
	case bool:
		return strconv.FormatBool(val)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// GenerateEsbuildFlags generates custom esbuild flags (--loader:* and --define:*)
// from a WranglerConfig's rules, define, and vars mappings.
func GenerateEsbuildFlags(cfg *WranglerConfig) []string {
	if cfg == nil {
		return nil
	}

	var flags []string

	// 1. Process custom loader rules
	loaderMap := make(map[string]string)
	for _, rule := range cfg.Rules {
		loader, ok := resolveLoader(rule.Type)
		if !ok {
			continue
		}

		for _, glob := range rule.Globs {
			ext := extractExtensionFromGlob(glob)
			if ext != "" {
				loaderMap[ext] = loader
			}
		}
	}

	var sortedExts []string
	for ext := range loaderMap {
		sortedExts = append(sortedExts, ext)
	}
	slices.Sort(sortedExts)

	for _, ext := range sortedExts {
		flags = append(flags, fmt.Sprintf("--loader:%s=%s", ext, loaderMap[ext]))
	}

	// 2. Process defines (vars and explicit defines)
	defineMap := make(map[string]string)

	// First, include vars
	for k, v := range cfg.Vars {
		kTrim := strings.TrimSpace(k)
		if kTrim == "" {
			continue
		}
		defineMap[kTrim] = formatDefineValue(v)
	}

	// Then, explicit defines take precedence and overwrite matching keys
	for k, v := range cfg.Define {
		kTrim := strings.TrimSpace(k)
		if kTrim == "" {
			continue
		}
		defineMap[kTrim] = formatDefineValue(v)
	}

	var sortedDefines []string
	for k := range defineMap {
		sortedDefines = append(sortedDefines, k)
	}
	slices.Sort(sortedDefines)

	for _, k := range sortedDefines {
		flags = append(flags, fmt.Sprintf("--define:%s=%s", k, defineMap[k]))
	}

	return flags
}

// BuildEsbuildArgs returns the complete CLI argument slice for bundling a worker entrypoint with esbuild.
func BuildEsbuildArgs(cfg *WranglerConfig, entrypoint, outPath string) []string {
	args := []string{
		entrypoint,
		"--bundle",
		"--format=esm",
		"--target=es2022",
		"--outfile=" + outPath,
	}

	if cfg != nil {
		customFlags := GenerateEsbuildFlags(cfg)
		args = append(args, customFlags...)
	}

	return args
}
