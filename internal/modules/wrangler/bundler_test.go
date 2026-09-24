package wrangler_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/ishaf/cubit/internal/modules/wrangler"
)

func TestGenerateEsbuildFlags_Rules(t *testing.T) {
	tests := []struct {
		name          string
		rules         []wrangler.RuleConfig
		expectedFlags []string
	}{
		{
			name: "Given Text rules with various glob patterns, When generating flags, Then maps to text loader",
			rules: []wrangler.RuleConfig{
				{
					Type:  "Text",
					Globs: []string{"**/*.txt", "*.md", "*.html"},
				},
			},
			expectedFlags: []string{
				"--loader:.html=text",
				"--loader:.md=text",
				"--loader:.txt=text",
			},
		},
		{
			name: "Given Data and CompiledWasm rules, When generating flags, Then maps to binary loader",
			rules: []wrangler.RuleConfig{
				{
					Type:  "Data",
					Globs: []string{"**/*.bin", "*.dat"},
				},
				{
					Type:  "CompiledWasm",
					Globs: []string{"**/*.wasm"},
				},
			},
			expectedFlags: []string{
				"--loader:.bin=binary",
				"--loader:.dat=binary",
				"--loader:.wasm=binary",
			},
		},
		{
			name: "Given duplicate extensions across rules, When generating flags, Then deduplicates deterministically",
			rules: []wrangler.RuleConfig{
				{
					Type:  "Text",
					Globs: []string{"**/*.txt"},
				},
				{
					Type:  "Text",
					Globs: []string{"subfolder/*.txt"},
				},
			},
			expectedFlags: []string{
				"--loader:.txt=text",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &wrangler.WranglerConfig{
				Rules: tc.rules,
			}

			flags := wrangler.GenerateEsbuildFlags(cfg)

			var loaderFlags []string
			for _, f := range flags {
				if strings.HasPrefix(f, "--loader:") {
					loaderFlags = append(loaderFlags, f)
				}
			}

			slices.Sort(loaderFlags)
			slices.Sort(tc.expectedFlags)

			if !reflect.DeepEqual(loaderFlags, tc.expectedFlags) {
				t.Errorf("expected loader flags %v, got %v", tc.expectedFlags, loaderFlags)
			}
		})
	}
}

func TestGenerateEsbuildFlags_Defines(t *testing.T) {
	tests := []struct {
		name            string
		cfg             *wrangler.WranglerConfig
		expectedDefines []string
	}{
		{
			name: "Given defines with string, numeric, and boolean types, When generating flags, Then correctly formats for direct CLI",
			cfg: &wrangler.WranglerConfig{
				Define: map[string]any{
					"process.env.NODE_ENV": "\"production\"",
					"API_HOST":             "api.example.com",
					"MAX_SIZE":             1048576,
					"DEBUG":                true,
				},
			},
			expectedDefines: []string{
				`--define:API_HOST="api.example.com"`,
				`--define:DEBUG=true`,
				`--define:MAX_SIZE=1048576`,
				`--define:process.env.NODE_ENV="production"`,
			},
		},
		{
			name: "Given vars and explicit defines with collision, When generating flags, Then explicit define overrides vars",
			cfg: &wrangler.WranglerConfig{
				Vars: map[string]any{
					"API_KEY":  "secret123",
					"ENDPOINT": "https://api.cubit.dev",
				},
				Define: map[string]any{
					"API_KEY": "\"explicit-override\"",
				},
			},
			expectedDefines: []string{
				`--define:API_KEY="explicit-override"`,
				`--define:ENDPOINT="https://api.cubit.dev"`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flags := wrangler.GenerateEsbuildFlags(tc.cfg)

			var defineFlags []string
			for _, f := range flags {
				if strings.HasPrefix(f, "--define:") {
					defineFlags = append(defineFlags, f)
				}
			}

			slices.Sort(defineFlags)
			slices.Sort(tc.expectedDefines)

			if !reflect.DeepEqual(defineFlags, tc.expectedDefines) {
				t.Errorf("expected define flags %v, got %v", tc.expectedDefines, defineFlags)
			}
		})
	}
}

func TestBuildEsbuildArgs_FullCommand(t *testing.T) {
	t.Run("Given wrangler config with rules and defines, When BuildEsbuildArgs is called, Then returns complete CLI arguments", func(t *testing.T) {
		cfg := &wrangler.WranglerConfig{
			Rules: []wrangler.RuleConfig{
				{Type: "CompiledWasm", Globs: []string{"**/*.wasm"}},
				{Type: "Text", Globs: []string{"**/*.txt"}},
			},
			Define: map[string]any{
				"VERSION": "\"1.2.3\"",
			},
		}

		args := wrangler.BuildEsbuildArgs(cfg, "src/index.ts", "dist/worker.js")

		// Verify base arguments
		if len(args) < 6 {
			t.Fatalf("expected at least 6 arguments, got %d: %v", len(args), args)
		}
		if args[0] != "src/index.ts" {
			t.Errorf("expected entrypoint first, got %s", args[0])
		}
		if args[1] != "--bundle" || args[2] != "--format=esm" || args[3] != "--target=es2022" {
			t.Errorf("expected standard bundle arguments, got %v", args[1:4])
		}
		if args[4] != "--outfile=dist/worker.js" {
			t.Errorf("expected outfile argument, got %s", args[4])
		}

		// Verify loader and define flags are present
		remaining := args[5:]
		var hasWasm, hasText, hasDefine bool
		for _, arg := range remaining {
			if arg == "--loader:.wasm=binary" {
				hasWasm = true
			}
			if arg == "--loader:.txt=text" {
				hasText = true
			}
			if arg == `--define:VERSION="1.2.3"` {
				hasDefine = true
			}
		}

		if !hasWasm {
			t.Errorf("missing --loader:.wasm=binary in %v", remaining)
		}
		if !hasText {
			t.Errorf("missing --loader:.txt=text in %v", remaining)
		}
		if !hasDefine {
			t.Errorf("missing --define:VERSION in %v", remaining)
		}
	})

	t.Run("Given nil wrangler config, When BuildEsbuildArgs is called, Then returns valid default arguments", func(t *testing.T) {
		args := wrangler.BuildEsbuildArgs(nil, "index.js", "out.js")
		expected := []string{"index.js", "--bundle", "--format=esm", "--target=es2022", "--outfile=out.js"}
		if !reflect.DeepEqual(args, expected) {
			t.Errorf("expected %v, got %v", expected, args)
		}
	})
}
