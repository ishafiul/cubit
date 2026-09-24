package wrangler_test

import (
	"strings"
	"testing"

	"github.com/ishaf/cubit/internal/modules/wrangler"
)

func TestSanitizeForCelld_EmptyAndInvalidInput(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		errContains string
	}{
		{
			name:        "Given empty configuration input, When SanitizeForCelld is called, Then it returns an error",
			input:       []byte("   \n\t  "),
			errContains: "empty",
		},
		{
			name:        "Given malformed syntax input, When SanitizeForCelld is called, Then it returns an error",
			input:       []byte(`{ name: "missing-quotes-invalid-json" `),
			errContains: "failed to parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized, routes, err := wrangler.SanitizeForCelld(tt.input)
			if err == nil {
				t.Fatalf("expected error containing '%s', got nil", tt.errContains)
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("expected error message to contain '%s', got: %v", tt.errContains, err)
			}
			if sanitized != nil {
				t.Errorf("expected sanitized to be nil, got: %s", string(sanitized))
			}
			if routes != nil {
				t.Errorf("expected routes to be nil, got: %+v", routes)
			}
		})
	}
}

func TestSanitizeForCelld_StripsProhibitedKeys(t *testing.T) {
	tests := []struct {
		name                 string
		rawInput             []byte
		targetEnv            string
		expectedName         string
		expectedCompat       string
		expectedRoutes       []string
		prohibitedSubstrings []string
	}{
		{
			name: "Given JSONC with prohibited keys and env block without targetEnv, When SanitizeForCelld is called, Then prohibited keys and env are stripped",
			rawInput: []byte(`{
				// Metadata
				"name": "migrated-app",
				"main": "src/worker.ts",
				"compatibility_date": "2024-09-24",

				/* Prohibited Cloudflare keys */
				"account_id": "cf-acc-12345",
				"workers_dev": true,
				"route": "example.com/*",
				"routes": [
					"api.example.com/*",
					{ "pattern": "auth.example.com/*", "zone_id": "z-123" }
				],
				"send_email": [{ "name": "EMAIL_SERVICE" }],
				"ai": { "binding": "AI" },
				"vectorize": [{ "binding": "VEC", "index_name": "idx" }],
				"hyperdrive": [{ "binding": "HYPER", "id": "hd-1" }],
				"browser": { "binding": "BROWSER" },
				"containers": [{ "name": "sidecar", "image": "redis:alpine" }],

				// Allowed celld bindings
				"vars": {
					"ENVIRONMENT": "base",
					"PORT": 8080
				},
				"durable_objects": {
					"bindings": [
						{ "name": "ROOM", "class_name": "ChatRoom" }
					]
				},

				// Env block
				"env": {
					"production": {
						"name": "migrated-app-prod",
						"account_id": "cf-prod-acc-999",
						"route": "prod.example.com/*",
						"vars": {
							"ENVIRONMENT": "production"
						}
					}
				}
			}`),
			targetEnv:      "",
			expectedName:   "migrated-app",
			expectedCompat: "2024-09-24",
			expectedRoutes: []string{
				"example.com/*",
				"api.example.com/*",
				"auth.example.com/*",
				"prod.example.com/*",
			},
			prohibitedSubstrings: []string{
				"account_id",
				"workers_dev",
				"\"route\"",
				"\"routes\"",
				"send_email",
				"\"ai\"",
				"vectorize",
				"hyperdrive",
				"\"browser\"",
				"\"containers\"",
				"\"env\"",
				"cf-acc-12345",
				"cf-prod-acc-999",
			},
		},
		{
			name: "Given JSONC with production env override, When SanitizeForCelldWithEnv is called with 'production', Then env overrides are merged and env block is stripped",
			rawInput: []byte(`{
				"name": "migrated-app",
				"main": "src/worker.ts",
				"compatibility_date": "2024-09-24",
				"account_id": "cf-acc-12345",
				"route": "example.com/*",
				"env": {
					"production": {
						"name": "migrated-app-prod",
						"account_id": "cf-prod-acc-999",
						"route": "prod.example.com/*",
						"vars": {
							"ENVIRONMENT": "production"
						}
					}
				}
			}`),
			targetEnv:      "production",
			expectedName:   "migrated-app-prod",
			expectedCompat: "2024-09-24",
			expectedRoutes: []string{
				"example.com/*",
				"prod.example.com/*",
			},
			prohibitedSubstrings: []string{
				"account_id",
				"\"route\"",
				"\"routes\"",
				"\"env\"",
				"cf-acc-12345",
				"cf-prod-acc-999",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sanitized []byte
			var routes []string
			var err error

			if tt.targetEnv != "" {
				sanitized, routes, err = wrangler.SanitizeForCelldWithEnv(tt.rawInput, tt.targetEnv)
			} else {
				sanitized, routes, err = wrangler.SanitizeForCelld(tt.rawInput)
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sanitized == nil {
				t.Fatal("expected sanitized JSON, got nil")
			}

			sanitizedStr := string(sanitized)

			// 1. Check prohibited substrings are stripped
			for _, p := range tt.prohibitedSubstrings {
				if strings.Contains(sanitizedStr, p) {
					t.Errorf("expected sanitized output NOT to contain %s, but found it in:\n%s", p, sanitizedStr)
				}
			}

			// 2. Parsed verification
			cfg, format, parseErr := wrangler.Parse(sanitized, "json")
			if parseErr != nil {
				t.Fatalf("failed to parse sanitized output: %v\nJSON:\n%s", parseErr, sanitizedStr)
			}
			if format != "json" {
				t.Errorf("expected format 'json', got '%s'", format)
			}
			if cfg.Name != tt.expectedName {
				t.Errorf("expected name '%s', got '%s'", tt.expectedName, cfg.Name)
			}
			if cfg.CompatibilityDate != tt.expectedCompat {
				t.Errorf("expected compat date '%s', got '%s'", tt.expectedCompat, cfg.CompatibilityDate)
			}
			if cfg.Env != nil {
				t.Errorf("expected cfg.Env to be nil, got: %+v", cfg.Env)
			}
			if cfg.Containers != nil {
				t.Errorf("expected cfg.Containers to be nil, got: %+v", cfg.Containers)
			}

			// 3. Extracted routes verification
			if len(routes) != len(tt.expectedRoutes) {
				t.Fatalf("expected %d routes, got %d: %+v", len(tt.expectedRoutes), len(routes), routes)
			}
			for i, exp := range tt.expectedRoutes {
				if routes[i] != exp {
					t.Errorf("expected route[%d] = '%s', got '%s'", i, exp, routes[i])
				}
			}
		})
	}
}

func TestSanitizeForCelld_TOMLInput(t *testing.T) {
	tests := []struct {
		name                 string
		rawTOML              []byte
		expectedName         string
		expectedRoutes       []string
		prohibitedSubstrings []string
	}{
		{
			name: "Given wrangler.toml with routes and prohibited keys, When SanitizeForCelld is called, Then it returns sanitized JSON and extracts routes",
			rawTOML: []byte(`
name = "toml-worker"
main = "dist/index.js"
compatibility_date = "2024-08-01"

# Prohibited top-level keys
account_id = "cf-acc-toml-789"
workers_dev = false
route = "toml.example.com/*"
routes = ["sub1.example.com/*", "sub2.example.com/*"]
send_email = [{ name = "SEND" }]
ai = { binding = "AI_MODEL" }

[vars]
GREETING = "Hello from TOML"

[[kv_namespaces]]
binding = "CONFIG_KV"
id = "kv-12345"

[env.production]
name = "toml-worker-prod"
account_id = "cf-acc-prod"
route = "prod-toml.example.com/*"

[[env.production.routes]]
pattern = "pattern-toml.example.com/*"
zone_id = "zid-99"
`),
			expectedName: "toml-worker",
			expectedRoutes: []string{
				"toml.example.com/*",
				"sub1.example.com/*",
				"sub2.example.com/*",
				"prod-toml.example.com/*",
				"pattern-toml.example.com/*",
			},
			prohibitedSubstrings: []string{
				"account_id",
				"workers_dev",
				"\"route\"",
				"\"routes\"",
				"send_email",
				"\"ai\"",
				"\"env\"",
				"cf-acc-toml-789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized, routes, err := wrangler.SanitizeForCelld(tt.rawTOML)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			sanitizedStr := string(sanitized)

			for _, p := range tt.prohibitedSubstrings {
				if strings.Contains(sanitizedStr, p) {
					t.Errorf("expected output NOT to contain %s, but found it in:\n%s", p, sanitizedStr)
				}
			}

			cfg, format, parseErr := wrangler.Parse(sanitized, "json")
			if parseErr != nil {
				t.Fatalf("failed to parse sanitized output: %v\nJSON:\n%s", parseErr, sanitizedStr)
			}
			if format != "json" {
				t.Errorf("expected json output, got: %s", format)
			}
			if cfg.Name != tt.expectedName {
				t.Errorf("expected name '%s', got '%s'", tt.expectedName, cfg.Name)
			}
			if len(cfg.KVNamespaces) != 1 || cfg.KVNamespaces[0].Binding != "CONFIG_KV" {
				t.Errorf("expected KV namespace preserved, got: %+v", cfg.KVNamespaces)
			}
			if cfg.Env != nil {
				t.Errorf("expected cfg.Env to be nil, got: %+v", cfg.Env)
			}

			if len(routes) != len(tt.expectedRoutes) {
				t.Fatalf("expected %d routes, got %d: %+v", len(tt.expectedRoutes), len(routes), routes)
			}
			for i, exp := range tt.expectedRoutes {
				if routes[i] != exp {
					t.Errorf("expected route[%d] = '%s', got '%s'", i, exp, routes[i])
				}
			}
		})
	}
}

func TestSanitizeForCelld_ExtractsAndDeduplicatesRoutes(t *testing.T) {
	tests := []struct {
		name           string
		rawJSON        []byte
		expectedRoutes []string
	}{
		{
			name: "Given configuration with duplicates, custom_domain objects, and whitespace, When SanitizeForCelld is called, Then routes are deduplicated and trimmed",
			rawJSON: []byte(`{
				"name": "dedup-worker",
				"route": "  dup.example.com/*  ",
				"routes": [
					"dup.example.com/*",
					{ "pattern": "api.example.com/*", "custom_domain": true },
					{ "custom_domain": "custom.domain.org" },
					"api.example.com/*"
				],
				"env": {
					"production": {
						"routes": [
							"dup.example.com/*",
							"prod-only.example.com/*"
						]
					}
				}
			}`),
			expectedRoutes: []string{
				"dup.example.com/*",
				"api.example.com/*",
				"custom.domain.org",
				"prod-only.example.com/*",
			},
		},
		{
			name: "Given configuration without any routes, When SanitizeForCelld is called, Then an empty non-nil route slice is returned",
			rawJSON: []byte(`{
				"name": "no-routes-worker",
				"main": "src/index.ts"
			}`),
			expectedRoutes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, routes, err := wrangler.SanitizeForCelld(tt.rawJSON)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if routes == nil {
				t.Fatal("expected non-nil routes slice, got nil")
			}
			if len(routes) != len(tt.expectedRoutes) {
				t.Fatalf("expected %d routes, got %d: %+v", len(tt.expectedRoutes), len(routes), routes)
			}
			for i, exp := range tt.expectedRoutes {
				if routes[i] != exp {
					t.Errorf("expected route[%d] = '%s', got '%s'", i, exp, routes[i])
				}
			}
		})
	}
}

func TestSanitizeForCelld_PreservesAllCelldAllowedBindings(t *testing.T) {
	tests := []struct {
		name                 string
		rawJSON              []byte
		prohibitedSubstrings []string
	}{
		{
			name: "Given configuration with all celld-supported bindings, When SanitizeForCelld is called, Then all allowed bindings are preserved and disallowed stripped",
			rawJSON: []byte(`{
				"name": "full-worker",
				"main": "src/index.ts",
				"compatibility_date": "2024-09-01",
				"compatibility_flags": ["nodejs_compat"],
				"vars": {
					"VAR1": "val1"
				},
				"kv_namespaces": [
					{ "binding": "KV1", "id": "id-kv" }
				],
				"d1_databases": [
					{ "binding": "DB", "database_name": "my-db" }
				],
				"r2_buckets": [
					{ "binding": "BUCKET", "bucket_name": "my-bucket" }
				],
				"services": [
					{ "binding": "SVC", "service": "auth-service" }
				],
				"queues": {
					"producers": [{ "binding": "QUEUE_PROD", "queue": "my-queue" }],
					"consumers": [{ "queue": "my-queue" }]
				},
				"triggers": {
					"crons": ["0 0 * * *"]
				},
				"assets": {
					"directory": "./dist",
					"binding": "ASSETS"
				},
				"durable_objects": {
					"bindings": [{ "name": "DO1", "class_name": "ClassDO" }]
				},
				"migrations": [
					{ "tag": "v1", "new_classes": ["ClassDO"] }
				],
				"workflows": [
					{ "name": "wf", "binding": "WF", "class_name": "WfClass" }
				],
				"containers": [
					{ "name": "c1", "image": "img:latest", "port": 8080 }
				],
				"rules": [
					{ "type": "Text", "globs": ["*.txt"] }
				],
				"define": {
					"BUILD_TIME": "\"2024\""
				},
				"account_id": "should-be-stripped",
				"route": "full.example.com/*"
			}`),
			prohibitedSubstrings: []string{
				"account_id",
				"should-be-stripped",
				"\"route\"",
				"\"routes\"",
				"\"containers\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized, routes, err := wrangler.SanitizeForCelld(tt.rawJSON)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(routes) != 1 || routes[0] != "full.example.com/*" {
				t.Errorf("unexpected extracted routes: %+v", routes)
			}

			cfg, format, err := wrangler.Parse(sanitized, "json")
			if err != nil {
				t.Fatalf("failed to parse sanitized output: %v", err)
			}
			if format != "json" {
				t.Errorf("expected json format, got: %s", format)
			}

			// Verify celld-allowed bindings are preserved
			if cfg.Name != "full-worker" || cfg.Main != "src/index.ts" {
				t.Errorf("name/main mismatch: %+v", cfg)
			}
			if len(cfg.CompatibilityFlags) != 1 || cfg.CompatibilityFlags[0] != "nodejs_compat" {
				t.Errorf("compatibility_flags mismatch: %+v", cfg.CompatibilityFlags)
			}
			if cfg.Vars["VAR1"] != "val1" {
				t.Errorf("vars mismatch: %+v", cfg.Vars)
			}
			if len(cfg.KVNamespaces) != 1 || cfg.KVNamespaces[0].Binding != "KV1" {
				t.Errorf("kv mismatch: %+v", cfg.KVNamespaces)
			}
			if len(cfg.D1Databases) != 1 || cfg.D1Databases[0].Binding != "DB" {
				t.Errorf("d1 mismatch: %+v", cfg.D1Databases)
			}
			if len(cfg.R2Buckets) != 1 || cfg.R2Buckets[0].Binding != "BUCKET" {
				t.Errorf("r2 mismatch: %+v", cfg.R2Buckets)
			}
			if len(cfg.Services) != 1 || cfg.Services[0].Binding != "SVC" {
				t.Errorf("services mismatch: %+v", cfg.Services)
			}
			if cfg.Queues == nil || len(cfg.Queues.Producers) != 1 || cfg.Queues.Producers[0].Binding != "QUEUE_PROD" {
				t.Errorf("queues mismatch: %+v", cfg.Queues)
			}
			if cfg.Triggers == nil || len(cfg.Triggers.Crons) != 1 || cfg.Triggers.Crons[0] != "0 0 * * *" {
				t.Errorf("triggers mismatch: %+v", cfg.Triggers)
			}
			if cfg.Assets == nil || cfg.Assets.Directory != "./dist" {
				t.Errorf("assets mismatch: %+v", cfg.Assets)
			}
			if cfg.DurableObjects == nil || len(cfg.DurableObjects.Bindings) != 1 || cfg.DurableObjects.Bindings[0].Name != "DO1" {
				t.Errorf("durable_objects mismatch: %+v", cfg.DurableObjects)
			}
			if len(cfg.Migrations) != 1 || cfg.Migrations[0].Tag != "v1" {
				t.Errorf("migrations mismatch: %+v", cfg.Migrations)
			}
			if len(cfg.Workflows) != 1 || cfg.Workflows[0].Name != "wf" {
				t.Errorf("workflows mismatch: %+v", cfg.Workflows)
			}
			if len(cfg.Rules) != 1 || cfg.Rules[0].Type != "Text" {
				t.Errorf("rules mismatch: %+v", cfg.Rules)
			}
			if cfg.Define == nil || cfg.Define["BUILD_TIME"] != "\"2024\"" {
				t.Errorf("define mismatch: %+v", cfg.Define)
			}

			// Ensure unallowed / prohibited keys are absent from JSON
			outStr := string(sanitized)
			for _, p := range tt.prohibitedSubstrings {
				if strings.Contains(outStr, p) {
					t.Errorf("expected output NOT to contain %s, but found it in:\n%s", p, outStr)
				}
			}
		})
	}
}

func TestSanitizeConfig(t *testing.T) {
	tests := []struct {
		name           string
		inputCfg       *wrangler.WranglerConfig
		envName        string
		expectedName   string
		expectedVarVal string
	}{
		{
			name: "Given nil WranglerConfig, When SanitizeConfig is called, Then return nil",
			inputCfg: nil,
			envName: "",
			expectedName: "",
		},
		{
			name: "Given WranglerConfig with containers and env block without envName, When SanitizeConfig is called, Then containers and env are stripped",
			inputCfg: &wrangler.WranglerConfig{
				Name: "base-app",
				Vars: map[string]any{"ENV": "base"},
				Containers: []wrangler.ContainerBinding{
					{Name: "sidecar", Image: "redis:alpine"},
				},
				Env: map[string]wrangler.WranglerConfig{
					"staging": {
						Name: "staging-app",
						Vars: map[string]any{"ENV": "staging"},
					},
				},
			},
			envName:        "",
			expectedName:   "base-app",
			expectedVarVal: "base",
		},
		{
			name: "Given WranglerConfig with envName 'staging', When SanitizeConfig is called, Then staging env overrides are merged and env is stripped",
			inputCfg: &wrangler.WranglerConfig{
				Name: "base-app",
				Vars: map[string]any{"ENV": "base"},
				Env: map[string]wrangler.WranglerConfig{
					"staging": {
						Name: "staging-app",
						Vars: map[string]any{"ENV": "staging"},
					},
				},
			},
			envName:        "staging",
			expectedName:   "staging-app",
			expectedVarVal: "staging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrangler.SanitizeConfig(tt.inputCfg, tt.envName)
			if tt.inputCfg == nil {
				if result != nil {
					t.Fatalf("expected nil result, got: %+v", result)
				}
				return
			}
			if result.Name != tt.expectedName {
				t.Errorf("expected name '%s', got '%s'", tt.expectedName, result.Name)
			}
			if result.Vars["ENV"] != tt.expectedVarVal {
				t.Errorf("expected var ENV '%s', got '%v'", tt.expectedVarVal, result.Vars["ENV"])
			}
			if result.Env != nil {
				t.Errorf("expected result.Env to be nil, got: %+v", result.Env)
			}
			if result.Containers != nil {
				t.Errorf("expected result.Containers to be nil, got: %+v", result.Containers)
			}
		})
	}
}
