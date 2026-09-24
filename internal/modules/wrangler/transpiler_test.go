package wrangler_test

import (
	"strings"
	"testing"

	"github.com/ishaf/cubit/internal/modules/wrangler"
)

func TestTranspileTOMLToJSONC_EmptyAndInvalidInput(t *testing.T) {
	t.Run("Given empty TOML input, When TranspileTOMLToJSONC is called, Then return an empty error", func(t *testing.T) {
		// Given
		emptyTOML := []byte("   \n\t  ")

		// When
		_, err := wrangler.TranspileTOMLToJSONC(emptyTOML)

		// Then
		if err == nil {
			t.Fatal("expected error for empty TOML, got nil")
		}
		if !strings.Contains(err.Error(), "empty") {
			t.Errorf("expected error message to mention 'empty', got: %v", err)
		}
	})

	t.Run("Given malformed TOML syntax, When TranspileTOMLToJSONC is called, Then return a syntax error", func(t *testing.T) {
		// Given
		invalidTOML := []byte("name = [unclosed array")

		// When
		_, err := wrangler.TranspileTOMLToJSONC(invalidTOML)

		// Then
		if err == nil {
			t.Fatal("expected error for malformed TOML, got nil")
		}
		if !strings.Contains(err.Error(), "failed to parse") {
			t.Errorf("expected error message to contain 'failed to parse', got: %v", err)
		}
	})
}

func TestTranspileTOMLToJSONC_BaseConfiguration(t *testing.T) {
	t.Run("Given wrangler.toml with base properties and typed vars, When TranspileTOMLToJSONC is called, Then return matching formatted JSON", func(t *testing.T) {
		// Given
		rawTOML := []byte(`
name = "edge-api"
main = "src/index.ts"
compatibility_date = "2024-09-23"
compatibility_flags = ["nodejs_compat", "streams_enable_constructors"]

[vars]
API_URL = "https://api.example.com"
MAX_RETRIES = 5
DEBUG = true
RATE_LIMIT = 250.5
`)

		// When
		out, err := wrangler.TranspileTOMLToJSONC(rawTOML)

		// Then
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, format, err := wrangler.Parse(out, "json")
		if err != nil {
			t.Fatalf("failed to parse transpiled output: %v\nOutput was:\n%s", err, string(out))
		}
		if format != "json" {
			t.Errorf("expected format 'json', got '%s'", format)
		}
		if cfg.Name != "edge-api" {
			t.Errorf("expected name 'edge-api', got '%s'", cfg.Name)
		}
		if cfg.Main != "src/index.ts" {
			t.Errorf("expected main 'src/index.ts', got '%s'", cfg.Main)
		}
		if cfg.CompatibilityDate != "2024-09-23" {
			t.Errorf("expected compat date '2024-09-23', got '%s'", cfg.CompatibilityDate)
		}
		if len(cfg.CompatibilityFlags) != 2 || cfg.CompatibilityFlags[0] != "nodejs_compat" || cfg.CompatibilityFlags[1] != "streams_enable_constructors" {
			t.Errorf("unexpected compat flags: %v", cfg.CompatibilityFlags)
		}
		if cfg.Vars["API_URL"] != "https://api.example.com" {
			t.Errorf("expected API_URL 'https://api.example.com', got %v", cfg.Vars["API_URL"])
		}
		if cfg.Vars["MAX_RETRIES"] != int64(5) && cfg.Vars["MAX_RETRIES"] != float64(5) {
			t.Errorf("expected MAX_RETRIES 5, got %v", cfg.Vars["MAX_RETRIES"])
		}
		if cfg.Vars["DEBUG"] != true {
			t.Errorf("expected DEBUG true, got %v", cfg.Vars["DEBUG"])
		}
		if cfg.Vars["RATE_LIMIT"] != 250.5 {
			t.Errorf("expected RATE_LIMIT 250.5, got %v", cfg.Vars["RATE_LIMIT"])
		}
	})
}

func TestTranspileTOMLToJSONC_ResourceBindings(t *testing.T) {
	t.Run("Given wrangler.toml with standard resource bindings, When TranspileTOMLToJSONC is called, Then all bindings serialize properly into JSONC", func(t *testing.T) {
		// Given
		rawTOML := []byte(`
name = "bindings-worker"
main = "src/index.js"

[[kv_namespaces]]
binding = "CACHE"
id = "kv-123"

[[d1_databases]]
binding = "DB"
database_name = "prod-db"
database_id = "d1-uuid-456"

[[r2_buckets]]
binding = "BUCKET"
bucket_name = "prod-bucket"

[[services]]
binding = "AUTH"
service = "auth-worker"
environment = "production"

[queues]
[[queues.producers]]
binding = "OUTBOX"
queue = "events"

[[queues.consumers]]
queue = "incoming"

[triggers]
crons = ["0 * * * *", "*/15 * * * *"]

[assets]
directory = "./public"
binding = "STATIC"
html_handling = "auto-trailing-slash"
`)

		// When
		out, err := wrangler.TranspileTOMLToJSONC(rawTOML)

		// Then
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, format, err := wrangler.Parse(out, "json")
		if err != nil {
			t.Fatalf("failed to parse transpiled JSON: %v\nJSON:\n%s", err, string(out))
		}
		if format != "json" {
			t.Errorf("expected format 'json', got '%s'", format)
		}

		if len(cfg.KVNamespaces) != 1 || cfg.KVNamespaces[0].Binding != "CACHE" || cfg.KVNamespaces[0].ID != "kv-123" {
			t.Errorf("unexpected KV: %+v", cfg.KVNamespaces)
		}
		if len(cfg.D1Databases) != 1 || cfg.D1Databases[0].Binding != "DB" || cfg.D1Databases[0].DatabaseName != "prod-db" || cfg.D1Databases[0].DatabaseID != "d1-uuid-456" {
			t.Errorf("unexpected D1: %+v", cfg.D1Databases)
		}
		if len(cfg.R2Buckets) != 1 || cfg.R2Buckets[0].Binding != "BUCKET" || cfg.R2Buckets[0].BucketName != "prod-bucket" {
			t.Errorf("unexpected R2: %+v", cfg.R2Buckets)
		}
		if len(cfg.Services) != 1 || cfg.Services[0].Binding != "AUTH" || cfg.Services[0].Service != "auth-worker" || cfg.Services[0].Environment != "production" {
			t.Errorf("unexpected Service: %+v", cfg.Services)
		}
		if cfg.Queues == nil || len(cfg.Queues.Producers) != 1 || cfg.Queues.Producers[0].Binding != "OUTBOX" || cfg.Queues.Producers[0].Queue != "events" {
			t.Errorf("unexpected Queue producers: %+v", cfg.Queues)
		}
		if cfg.Queues == nil || len(cfg.Queues.Consumers) != 1 || cfg.Queues.Consumers[0].Queue != "incoming" {
			t.Errorf("unexpected Queue consumers: %+v", cfg.Queues)
		}
		if cfg.Triggers == nil || len(cfg.Triggers.Crons) != 2 || cfg.Triggers.Crons[0] != "0 * * * *" {
			t.Errorf("unexpected Triggers: %+v", cfg.Triggers)
		}
		if cfg.Assets == nil || cfg.Assets.Directory != "./public" || cfg.Assets.Binding != "STATIC" || cfg.Assets.HTMLHandling != "auto-trailing-slash" {
			t.Errorf("unexpected Assets: %+v", cfg.Assets)
		}
	})
}

func TestTranspileTOMLToJSONC_ExtendedBindings(t *testing.T) {
	t.Run("Given wrangler.toml with extended celld bindings (DO, migrations, workflows, containers, rules, define), When TranspileTOMLToJSONC is called, Then they are preserved in JSONC", func(t *testing.T) {
		// Given
		rawTOML := []byte(`
name = "extended-worker"
main = "src/index.ts"

[durable_objects]
bindings = [
  { name = "COUNTER_DO", class_name = "CounterDO", script_name = "shared-worker", environment = "production" }
]

[[migrations]]
tag = "v1"
new_classes = ["CounterDO"]

[[migrations]]
tag = "v2"
renamed_classes = [{ from = "OldDO", to = "NewDO" }]
deleted_classes = ["LegacyDO"]

[[workflows]]
name = "order-processing"
binding = "ORDER_WORKFLOW"
class_name = "OrderWorkflow"

[[containers]]
name = "redis-cache"
image = "redis:7-alpine"
port = 6379

[[rules]]
type = "Text"
globs = ["**/*.txt", "**/*.md"]
fallthrough = true

[define]
"process.env.DEBUG" = "\"true\""
"VERSION" = "\"1.2.3\""
`)

		// When
		out, err := wrangler.TranspileTOMLToJSONC(rawTOML)

		// Then
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, format, err := wrangler.Parse(out, "json")
		if err != nil {
			t.Fatalf("failed to parse transpiled JSON: %v\nJSON:\n%s", err, string(out))
		}
		if format != "json" {
			t.Errorf("expected format 'json', got '%s'", format)
		}

		// Durable Objects
		if cfg.DurableObjects == nil || len(cfg.DurableObjects.Bindings) != 1 {
			t.Fatalf("expected 1 DO binding, got: %+v", cfg.DurableObjects)
		}
		doBinding := cfg.DurableObjects.Bindings[0]
		if doBinding.Name != "COUNTER_DO" || doBinding.ClassName != "CounterDO" || doBinding.ScriptName != "shared-worker" || doBinding.Environment != "production" {
			t.Errorf("unexpected DO binding: %+v", doBinding)
		}

		// Migrations
		if len(cfg.Migrations) != 2 {
			t.Fatalf("expected 2 migrations, got: %d", len(cfg.Migrations))
		}
		if cfg.Migrations[0].Tag != "v1" || len(cfg.Migrations[0].NewClasses) != 1 || cfg.Migrations[0].NewClasses[0] != "CounterDO" {
			t.Errorf("unexpected migration v1: %+v", cfg.Migrations[0])
		}
		if cfg.Migrations[1].Tag != "v2" || len(cfg.Migrations[1].RenamedClasses) != 1 || cfg.Migrations[1].RenamedClasses[0].From != "OldDO" || cfg.Migrations[1].RenamedClasses[0].To != "NewDO" || len(cfg.Migrations[1].DeletedClasses) != 1 || cfg.Migrations[1].DeletedClasses[0] != "LegacyDO" {
			t.Errorf("unexpected migration v2: %+v", cfg.Migrations[1])
		}

		// Workflows
		if len(cfg.Workflows) != 1 {
			t.Fatalf("expected 1 workflow, got: %d", len(cfg.Workflows))
		}
		if cfg.Workflows[0].Name != "order-processing" || cfg.Workflows[0].Binding != "ORDER_WORKFLOW" || cfg.Workflows[0].ClassName != "OrderWorkflow" {
			t.Errorf("unexpected workflow: %+v", cfg.Workflows[0])
		}

		// Containers
		if len(cfg.Containers) != 1 {
			t.Fatalf("expected 1 container, got: %d", len(cfg.Containers))
		}
		if cfg.Containers[0].Name != "redis-cache" || cfg.Containers[0].Image != "redis:7-alpine" || cfg.Containers[0].Port != 6379 {
			t.Errorf("unexpected container: %+v", cfg.Containers[0])
		}

		// Rules
		if len(cfg.Rules) != 1 {
			t.Fatalf("expected 1 rule, got: %d", len(cfg.Rules))
		}
		if cfg.Rules[0].Type != "Text" || len(cfg.Rules[0].Globs) != 2 || !cfg.Rules[0].Fallthrough {
			t.Errorf("unexpected rule: %+v", cfg.Rules[0])
		}

		// Define
		if cfg.Define == nil || cfg.Define["VERSION"] != "\"1.2.3\"" {
			t.Errorf("unexpected define: %+v", cfg.Define)
		}
	})
}

func TestTranspileTOMLToJSONC_EnvironmentBlocks(t *testing.T) {
	t.Run("Given wrangler.toml with [env.production] and [env.staging] blocks, When TranspileTOMLToJSONC is called, Then environment blocks are translated into the env object in JSONC", func(t *testing.T) {
		// Given
		rawTOML := []byte(`
name = "base-worker"
main = "src/index.ts"
compatibility_date = "2024-01-01"

[vars]
API_URL = "https://default.example.com"
DEBUG = false

[[kv_namespaces]]
binding = "BASE_KV"
id = "kv-base-id"

[env.production]
name = "prod-worker"
compatibility_date = "2024-06-01"
compatibility_flags = ["nodejs_compat"]

[env.production.vars]
API_URL = "https://prod.example.com"
DEBUG = true

[[env.production.kv_namespaces]]
binding = "PROD_KV"
id = "kv-prod-id"

[env.production.durable_objects]
bindings = [
  { name = "PROD_DO", class_name = "ProdDO" }
]

[env.staging]
name = "staging-worker"

[env.staging.vars]
API_URL = "https://staging.example.com"
`)

		// When
		out, err := wrangler.TranspileTOMLToJSONC(rawTOML)

		// Then
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, format, err := wrangler.Parse(out, "json")
		if err != nil {
			t.Fatalf("failed to parse transpiled JSON: %v\nJSON:\n%s", err, string(out))
		}
		if format != "json" {
			t.Errorf("expected format 'json', got '%s'", format)
		}

		// Top-level verification
		if cfg.Name != "base-worker" {
			t.Errorf("expected top-level name 'base-worker', got '%s'", cfg.Name)
		}
		if cfg.CompatibilityDate != "2024-01-01" {
			t.Errorf("expected top-level compat date '2024-01-01', got '%s'", cfg.CompatibilityDate)
		}
		if cfg.Vars["API_URL"] != "https://default.example.com" || cfg.Vars["DEBUG"] != false {
			t.Errorf("unexpected top-level vars: %+v", cfg.Vars)
		}
		if len(cfg.KVNamespaces) != 1 || cfg.KVNamespaces[0].Binding != "BASE_KV" {
			t.Errorf("unexpected top-level KV: %+v", cfg.KVNamespaces)
		}

		// Env dictionary verification
		if len(cfg.Env) != 2 {
			t.Fatalf("expected 2 envs (production, staging), got: %d", len(cfg.Env))
		}

		prodEnv, ok := cfg.Env["production"]
		if !ok {
			t.Fatal("missing 'production' env block")
		}
		if prodEnv.Name != "prod-worker" {
			t.Errorf("expected prod name 'prod-worker', got '%s'", prodEnv.Name)
		}
		if prodEnv.CompatibilityDate != "2024-06-01" {
			t.Errorf("expected prod compat date '2024-06-01', got '%s'", prodEnv.CompatibilityDate)
		}
		if len(prodEnv.CompatibilityFlags) != 1 || prodEnv.CompatibilityFlags[0] != "nodejs_compat" {
			t.Errorf("unexpected prod compat flags: %+v", prodEnv.CompatibilityFlags)
		}
		if prodEnv.Vars["API_URL"] != "https://prod.example.com" || prodEnv.Vars["DEBUG"] != true {
			t.Errorf("unexpected prod vars: %+v", prodEnv.Vars)
		}
		if len(prodEnv.KVNamespaces) != 1 || prodEnv.KVNamespaces[0].Binding != "PROD_KV" {
			t.Errorf("unexpected prod KV: %+v", prodEnv.KVNamespaces)
		}
		if prodEnv.DurableObjects == nil || len(prodEnv.DurableObjects.Bindings) != 1 || prodEnv.DurableObjects.Bindings[0].Name != "PROD_DO" {
			t.Errorf("unexpected prod DO: %+v", prodEnv.DurableObjects)
		}
		if prodEnv.Env != nil {
			t.Errorf("prodEnv should not have nested env: %+v", prodEnv.Env)
		}

		stagingEnv, ok := cfg.Env["staging"]
		if !ok {
			t.Fatal("missing 'staging' env block")
		}
		if stagingEnv.Name != "staging-worker" {
			t.Errorf("expected staging name 'staging-worker', got '%s'", stagingEnv.Name)
		}
		if stagingEnv.Vars["API_URL"] != "https://staging.example.com" {
			t.Errorf("unexpected staging vars: %+v", stagingEnv.Vars)
		}
	})
}

func TestTranspileTOMLToJSONC_FiltersProhibitedKeys(t *testing.T) {
	t.Run("Given wrangler.toml with routes, account_id, and proprietary keys, When TranspileTOMLToJSONC is called, Then only celld-allowed keys remain in JSONC", func(t *testing.T) {
		// Given
		rawTOML := []byte(`
name = "migrated-worker"
main = "src/index.ts"
compatibility_date = "2024-09-23"

# Prohibited top-level Cloudflare keys
account_id = "cloudflare-account-xyz-123"
workers_dev = true
route = "example.com/*"
routes = ["api.example.com/*", "auth.example.com/*"]
send_email = [{ name = "SEND_MAIL" }]
ai = { binding = "AI" }
vectorize = [{ binding = "VEC", index_name = "my-index" }]
hyperdrive = [{ binding = "HYPER", id = "hd-123" }]
browser = { binding = "BROWSER" }

# Allowed celld binding
[durable_objects]
bindings = [
  { name = "STORAGE", class_name = "StorageDO" }
]

# Prohibited keys inside an environment block
[env.production]
name = "migrated-worker-prod"
account_id = "prod-account-999"
route = "prod.example.com/*"
routes = ["prod.example.com/*"]
`)

		// When
		out, err := wrangler.TranspileTOMLToJSONC(rawTOML)

		// Then
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		outStr := string(out)

		// Prohibited keys must be stripped
		prohibitedKeys := []string{
			"account_id",
			"workers_dev",
			"\"route\"",
			"\"routes\"",
			"send_email",
			"\"ai\"",
			"vectorize",
			"hyperdrive",
			"\"browser\"",
		}
		for _, key := range prohibitedKeys {
			if strings.Contains(outStr, key) {
				t.Errorf("expected transpiled output NOT to contain prohibited key %s, but found it in:\n%s", key, outStr)
			}
		}

		// Allowed keys must be present
		cfg, format, err := wrangler.Parse(out, "json")
		if err != nil {
			t.Fatalf("failed to parse sanitized output: %v\nJSON:\n%s", err, outStr)
		}
		if format != "json" {
			t.Errorf("expected format 'json', got '%s'", format)
		}
		if cfg.Name != "migrated-worker" {
			t.Errorf("expected name 'migrated-worker', got '%s'", cfg.Name)
		}
		if cfg.DurableObjects == nil || len(cfg.DurableObjects.Bindings) != 1 || cfg.DurableObjects.Bindings[0].Name != "STORAGE" {
			t.Errorf("expected durable_objects preserved, got: %+v", cfg.DurableObjects)
		}
		if cfg.Env == nil || cfg.Env["production"].Name != "migrated-worker-prod" {
			t.Errorf("expected env.production.name 'migrated-worker-prod', got: %+v", cfg.Env)
		}
	})
}
