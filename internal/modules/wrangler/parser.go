package wrangler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/ishaf/cubit/internal/domain"
)

// KVNamespaceBinding represents a Cloudflare KV namespace binding.
type KVNamespaceBinding struct {
	Binding   string `json:"binding" toml:"binding"`
	ID        string `json:"id" toml:"id"`
	PreviewID string `json:"preview_id,omitempty" toml:"preview_id,omitempty"`
}

// D1DatabaseBinding represents a Cloudflare D1 database binding.
type D1DatabaseBinding struct {
	Binding      string `json:"binding" toml:"binding"`
	DatabaseName string `json:"database_name" toml:"database_name"`
	DatabaseID   string `json:"database_id,omitempty" toml:"database_id,omitempty"`
}

// R2BucketBinding represents a Cloudflare R2 bucket binding.
type R2BucketBinding struct {
	Binding    string `json:"binding" toml:"binding"`
	BucketName string `json:"bucket_name" toml:"bucket_name"`
}

// ServiceBinding represents a Cloudflare Service RPC binding.
type ServiceBinding struct {
	Binding     string `json:"binding" toml:"binding"`
	Service     string `json:"service" toml:"service"`
	Environment string `json:"environment,omitempty" toml:"environment,omitempty"`
}

// QueueProducerBinding represents a Cloudflare Queue producer binding.
type QueueProducerBinding struct {
	Binding string `json:"binding" toml:"binding"`
	Queue   string `json:"queue" toml:"queue"`
}

// QueueConsumerBinding represents a Cloudflare Queue consumer binding.
type QueueConsumerBinding struct {
	Queue string `json:"queue" toml:"queue"`
}

// QueueConfig represents queue bindings in wrangler configuration.
type QueueConfig struct {
	Producers []QueueProducerBinding `json:"producers,omitempty" toml:"producers,omitempty"`
	Consumers []QueueConsumerBinding `json:"consumers,omitempty" toml:"consumers,omitempty"`
}

// TriggerConfig represents scheduled cron triggers.
type TriggerConfig struct {
	Crons []string `json:"crons,omitempty" toml:"crons,omitempty"`
}

// AssetsConfig represents static assets configuration in wrangler.
type AssetsConfig struct {
	Directory    string `json:"directory,omitempty" toml:"directory,omitempty"`
	Binding      string `json:"binding,omitempty" toml:"binding,omitempty"`
	HTMLHandling string `json:"html_handling,omitempty" toml:"html_handling,omitempty"`
}

// DurableObjectBinding represents a Durable Object binding in wrangler configuration.
type DurableObjectBinding struct {
	Name        string `json:"name" toml:"name"`
	ClassName   string `json:"class_name" toml:"class_name"`
	ScriptName  string `json:"script_name,omitempty" toml:"script_name,omitempty"`
	Environment string `json:"environment,omitempty" toml:"environment,omitempty"`
}

// DurableObjectsConfig represents the durable_objects block in wrangler configuration.
type DurableObjectsConfig struct {
	Bindings []DurableObjectBinding `json:"bindings" toml:"bindings"`
}

// MigrationRenamedClass represents a renamed class in a Durable Object migration.
type MigrationRenamedClass struct {
	From string `json:"from" toml:"from"`
	To   string `json:"to" toml:"to"`
}

// MigrationConfig represents a Durable Object migration step in wrangler configuration.
type MigrationConfig struct {
	Tag            string                  `json:"tag" toml:"tag"`
	NewClasses     []string                `json:"new_classes,omitempty" toml:"new_classes,omitempty"`
	RenamedClasses []MigrationRenamedClass `json:"renamed_classes,omitempty" toml:"renamed_classes,omitempty"`
	DeletedClasses []string                `json:"deleted_classes,omitempty" toml:"deleted_classes,omitempty"`
}

// WorkflowBinding represents a Cloudflare Workflow binding in wrangler configuration.
type WorkflowBinding struct {
	Name      string `json:"name" toml:"name"`
	Binding   string `json:"binding" toml:"binding"`
	ClassName string `json:"class_name" toml:"class_name"`
}

// ContainerBinding represents an experimental container sidecar binding in wrangler configuration.
type ContainerBinding struct {
	Name    string            `json:"name" toml:"name"`
	Image   string            `json:"image" toml:"image"`
	Port    int               `json:"port,omitempty" toml:"port,omitempty"`
	EnvVars map[string]string `json:"env_vars,omitempty" toml:"env_vars,omitempty"`
}

// RuleConfig represents a custom module bundling rule in wrangler configuration.
type RuleConfig struct {
	Type        string   `json:"type" toml:"type"`
	Globs       []string `json:"globs" toml:"globs"`
	Fallthrough bool     `json:"fallthrough,omitempty" toml:"fallthrough,omitempty"`
}

// WranglerConfig represents the schema of wrangler.json, wrangler.jsonc, and wrangler.toml.
type WranglerConfig struct {
	Name               string                    `json:"name,omitempty" toml:"name,omitempty"`
	Main               string                    `json:"main,omitempty" toml:"main,omitempty"`
	CompatibilityDate  string                    `json:"compatibility_date,omitempty" toml:"compatibility_date,omitempty"`
	CompatibilityFlags []string                  `json:"compatibility_flags,omitempty" toml:"compatibility_flags,omitempty"`
	Vars               map[string]any            `json:"vars,omitempty" toml:"vars,omitempty"`
	KVNamespaces       []KVNamespaceBinding      `json:"kv_namespaces,omitempty" toml:"kv_namespaces,omitempty"`
	D1Databases        []D1DatabaseBinding       `json:"d1_databases,omitempty" toml:"d1_databases,omitempty"`
	R2Buckets          []R2BucketBinding         `json:"r2_buckets,omitempty" toml:"r2_buckets,omitempty"`
	Services           []ServiceBinding          `json:"services,omitempty" toml:"services,omitempty"`
	Queues             *QueueConfig              `json:"queues,omitempty" toml:"queues,omitempty"`
	Triggers           *TriggerConfig            `json:"triggers,omitempty" toml:"triggers,omitempty"`
	Assets             *AssetsConfig             `json:"assets,omitempty" toml:"assets,omitempty"`
	DurableObjects     *DurableObjectsConfig     `json:"durable_objects,omitempty" toml:"durable_objects,omitempty"`
	Migrations         []MigrationConfig         `json:"migrations,omitempty" toml:"migrations,omitempty"`
	Workflows          []WorkflowBinding         `json:"workflows,omitempty" toml:"workflows,omitempty"`
	Containers         []ContainerBinding        `json:"containers,omitempty" toml:"containers,omitempty"`
	Rules              []RuleConfig              `json:"rules,omitempty" toml:"rules,omitempty"`
	Define             map[string]any            `json:"define,omitempty" toml:"define,omitempty"`
	Env                map[string]WranglerConfig `json:"env,omitempty" toml:"env,omitempty"`
}

// ImportSummary provides statistics about what was updated from the wrangler configuration.
type ImportSummary struct {
	Name                  string   `json:"name,omitempty"`
	Main                  string   `json:"main,omitempty"`
	AssetsDirectory       string   `json:"assetsDirectory,omitempty"`
	CompatibilityDate     string   `json:"compatibilityDate,omitempty"`
	CompatibilityFlags    []string `json:"compatibilityFlags,omitempty"`
	ImportedVarsCount     int      `json:"importedVarsCount"`
	PreservedSecretsCount int      `json:"preservedSecretsCount"`
	ImportedBindingsCount int      `json:"importedBindingsCount"`
	CronsCount            int      `json:"cronsCount"`
	ImportedRoutesCount   int      `json:"importedRoutesCount,omitempty"`
	ExtractedRoutes       []string `json:"extractedRoutes,omitempty"`
	DetectedFormat        string   `json:"detectedFormat"`
}

// StripComments removes single-line (//) and multi-line (/* */) comments from JSON/JSONC text,
// preserving characters inside quoted string literals.
func StripComments(src []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(src))

	inString := false
	inEscape := false
	n := len(src)

	for i := 0; i < n; i++ {
		b := src[i]

		if inString {
			buf.WriteByte(b)
			if inEscape {
				inEscape = false
			} else if b == '\\' {
				inEscape = true
			} else if b == '"' {
				inString = false
			}
			continue
		}

		// Check string literal start
		if b == '"' {
			inString = true
			buf.WriteByte(b)
			continue
		}

		// Check single-line comment //
		if b == '/' && i+1 < n && src[i+1] == '/' {
			// Skip until newline
			i += 2
			for i < n && src[i] != '\n' && src[i] != '\r' {
				i++
			}
			if i < n {
				buf.WriteByte(src[i])
			}
			continue
		}

		// Check multi-line comment /* */
		if b == '/' && i+1 < n && src[i+1] == '*' {
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i++ // skip '/'
			continue
		}

		buf.WriteByte(b)
	}

	return buf.Bytes()
}

// StripTrailingCommas removes trailing commas before closing braces/brackets in JSON.
func StripTrailingCommas(src []byte) []byte {
	cleaned := src
	inString := false
	inEscape := false

	var out bytes.Buffer
	out.Grow(len(src))

	for i := 0; i < len(cleaned); i++ {
		b := cleaned[i]

		if inString {
			out.WriteByte(b)
			if inEscape {
				inEscape = false
			} else if b == '\\' {
				inEscape = true
			} else if b == '"' {
				inString = false
			}
			continue
		}

		if b == '"' {
			inString = true
			out.WriteByte(b)
			continue
		}

		if b == ',' {
			// Look ahead for closing brace or bracket, skipping whitespace
			j := i + 1
			for j < len(cleaned) && (cleaned[j] == ' ' || cleaned[j] == '\t' || cleaned[j] == '\n' || cleaned[j] == '\r') {
				j++
			}
			if j < len(cleaned) && (cleaned[j] == '}' || cleaned[j] == ']') {
				// skip this comma
				continue
			}
		}

		out.WriteByte(b)
	}

	return out.Bytes()
}

// DetectFormat inspects data to determine whether it is JSON or TOML.
func DetectFormat(data []byte) string {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return "json"
	}
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return "json"
	}
	return "toml"
}

// Parse parses raw wrangler config bytes (JSON, JSONC, or TOML) into a WranglerConfig.
func Parse(data []byte, formatHint string) (*WranglerConfig, string, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, "", fmt.Errorf("wrangler configuration is empty")
	}

	format := strings.ToLower(strings.TrimSpace(formatHint))
	if format == "" || format == "auto" {
		format = DetectFormat(trimmed)
	}

	var cfg WranglerConfig

	if format == "toml" {
		if err := toml.Unmarshal(trimmed, &cfg); err != nil {
			// If toml unmarshal failed, try json as fallback
			cleaned := StripTrailingCommas(StripComments(trimmed))
			if jErr := json.Unmarshal(cleaned, &cfg); jErr == nil {
				return &cfg, "json", nil
			}
			return nil, "toml", fmt.Errorf("failed to parse wrangler.toml: %w", err)
		}
		return &cfg, "toml", nil
	}

	// JSON / JSONC
	cleaned := StripTrailingCommas(StripComments(trimmed))
	if err := json.Unmarshal(cleaned, &cfg); err != nil {
		// If json failed, try toml as fallback
		if tErr := toml.Unmarshal(trimmed, &cfg); tErr == nil {
			return &cfg, "toml", nil
		}
		return nil, "json", fmt.Errorf("failed to parse wrangler.json: %w", err)
	}

	return &cfg, "json", nil
}

// FormatValue converts any value (string, number, boolean, slice, map) to a string representation for env vars.
func FormatValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		if b, err := json.Marshal(val); err == nil {
			return string(b)
		}
		return fmt.Sprintf("%v", val)
	}
}

// MergeEnvironment merges an environment-specific sub-config over the base config.
func MergeEnvironment(base *WranglerConfig, envName string) *WranglerConfig {
	if base == nil {
		return nil
	}
	if envName == "" || base.Env == nil {
		return base
	}

	envCfg, exists := base.Env[envName]
	if !exists {
		return base
	}

	merged := *base

	if envCfg.Name != "" {
		merged.Name = envCfg.Name
	}
	if envCfg.Main != "" {
		merged.Main = envCfg.Main
	}
	if envCfg.CompatibilityDate != "" {
		merged.CompatibilityDate = envCfg.CompatibilityDate
	}
	if len(envCfg.CompatibilityFlags) > 0 {
		merged.CompatibilityFlags = envCfg.CompatibilityFlags
	}

	if len(envCfg.Vars) > 0 {
		if merged.Vars == nil {
			merged.Vars = make(map[string]any)
		}
		for k, v := range envCfg.Vars {
			merged.Vars[k] = v
		}
	}

	if len(envCfg.KVNamespaces) > 0 {
		merged.KVNamespaces = envCfg.KVNamespaces
	}
	if len(envCfg.D1Databases) > 0 {
		merged.D1Databases = envCfg.D1Databases
	}
	if len(envCfg.R2Buckets) > 0 {
		merged.R2Buckets = envCfg.R2Buckets
	}
	if len(envCfg.Services) > 0 {
		merged.Services = envCfg.Services
	}
	if envCfg.Queues != nil {
		merged.Queues = envCfg.Queues
	}
	if envCfg.Triggers != nil {
		merged.Triggers = envCfg.Triggers
	}
	if envCfg.Assets != nil {
		merged.Assets = envCfg.Assets
	}
	if envCfg.DurableObjects != nil {
		merged.DurableObjects = envCfg.DurableObjects
	}
	if len(envCfg.Migrations) > 0 {
		merged.Migrations = envCfg.Migrations
	}
	if len(envCfg.Workflows) > 0 {
		merged.Workflows = envCfg.Workflows
	}
	if len(envCfg.Containers) > 0 {
		merged.Containers = envCfg.Containers
	}
	if len(envCfg.Rules) > 0 {
		merged.Rules = envCfg.Rules
	}
	if len(envCfg.Define) > 0 {
		if merged.Define == nil {
			merged.Define = make(map[string]any)
		}
		for k, v := range envCfg.Define {
			merged.Define[k] = v
		}
	}

	return &merged
}

// ApplyToApplication synchronizes parsed Wrangler configuration into a domain.Application entity.
// It merges environment variables (preserving existing encrypted secrets), updates compatibility settings,
// and maps Cloudflare resource bindings.
func ApplyToApplication(app *domain.Application, cfg *WranglerConfig, envName string, detectedFormat string) (*ImportSummary, error) {
	if app == nil {
		return nil, fmt.Errorf("application cannot be nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("wrangler config cannot be nil")
	}

	effective := cfg
	if envName != "" {
		effective = MergeEnvironment(cfg, envName)
	}

	summary := &ImportSummary{
		Name:               effective.Name,
		Main:               effective.Main,
		CompatibilityDate:  effective.CompatibilityDate,
		CompatibilityFlags: effective.CompatibilityFlags,
		DetectedFormat:     detectedFormat,
	}

	// 1. Update compatibility settings
	if strings.TrimSpace(effective.CompatibilityDate) != "" {
		app.CompatibilityDate = strings.TrimSpace(effective.CompatibilityDate)
	}
	if len(effective.CompatibilityFlags) > 0 {
		flagSet := make(map[string]struct{})
		for _, f := range app.CompatibilityFlags {
			if strings.TrimSpace(f) != "" {
				flagSet[strings.TrimSpace(f)] = struct{}{}
			}
		}
		for _, f := range effective.CompatibilityFlags {
			if strings.TrimSpace(f) != "" {
				flagSet[strings.TrimSpace(f)] = struct{}{}
			}
		}
		mergedFlags := make([]string, 0, len(flagSet))
		for f := range flagSet {
			mergedFlags = append(mergedFlags, f)
		}
		sort.Strings(mergedFlags)
		app.CompatibilityFlags = mergedFlags
	}

	// 2. Merge Environment Variables & Preserve Secrets
	existingVarMap := make(map[string]domain.EnvironmentVariable)
	for _, ev := range app.EnvVars {
		existingVarMap[ev.Key] = ev
	}

	var importedCount int
	var preservedSecrets int

	// Collect keys from wrangler config in deterministic order
	var configKeys []string
	for k := range effective.Vars {
		configKeys = append(configKeys, k)
	}
	sort.Strings(configKeys)

	updatedVarMap := make(map[string]domain.EnvironmentVariable)
	for k, ev := range existingVarMap {
		updatedVarMap[k] = ev
	}

	for _, k := range configKeys {
		val := FormatValue(effective.Vars[k])
		if existing, exists := existingVarMap[k]; exists {
			// If existing is secret, maintain IsSecret = true
			isSecret := existing.IsSecret
			if isSecret {
				preservedSecrets++
			}
			updatedVarMap[k] = domain.EnvironmentVariable{
				Key:      k,
				Value:    val,
				IsSecret: isSecret,
			}
		} else {
			updatedVarMap[k] = domain.EnvironmentVariable{
				Key:      k,
				Value:    val,
				IsSecret: false,
			}
		}
		importedCount++
	}

	// Rebuild app.EnvVars sorted by Key
	var finalEnvVars []domain.EnvironmentVariable
	for _, ev := range updatedVarMap {
		finalEnvVars = append(finalEnvVars, ev)
	}
	sort.Slice(finalEnvVars, func(i, j int) bool {
		return finalEnvVars[i].Key < finalEnvVars[j].Key
	})
	app.EnvVars = finalEnvVars

	summary.ImportedVarsCount = importedCount
	summary.PreservedSecretsCount = preservedSecrets

	// 3. Resource Bindings
	bindingMap := make(map[string]domain.ResourceBinding)
	for _, b := range app.Bindings {
		bindingMap[string(b.Type)+":"+b.Name] = b
	}

	var importedBindings int

	// KV Namespaces
	for _, kv := range effective.KVNamespaces {
		resID := kv.ID
		if resID == "" {
			resID = kv.PreviewID
		}
		bKey := string(domain.BindingTypeKV) + ":" + kv.Binding
		bindingMap[bKey] = domain.ResourceBinding{
			Type:       domain.BindingTypeKV,
			Name:       kv.Binding,
			ResourceID: resID,
		}
		importedBindings++
	}

	// D1 Databases
	for _, d1 := range effective.D1Databases {
		resID := d1.DatabaseName
		if resID == "" {
			resID = d1.DatabaseID
		}
		bKey := string(domain.BindingTypeD1) + ":" + d1.Binding
		bindingMap[bKey] = domain.ResourceBinding{
			Type:       domain.BindingTypeD1,
			Name:       d1.Binding,
			ResourceID: resID,
		}
		importedBindings++
	}

	// R2 Buckets
	for _, r2 := range effective.R2Buckets {
		bKey := string(domain.BindingTypeR2) + ":" + r2.Binding
		bindingMap[bKey] = domain.ResourceBinding{
			Type:       domain.BindingTypeR2,
			Name:       r2.Binding,
			ResourceID: r2.BucketName,
		}
		importedBindings++
	}

	// Services
	for _, s := range effective.Services {
		bKey := string(domain.BindingTypeService) + ":" + s.Binding
		bindingMap[bKey] = domain.ResourceBinding{
			Type:       domain.BindingTypeService,
			Name:       s.Binding,
			ResourceID: s.Service,
		}
		importedBindings++
	}

	// Queue Producers
	if effective.Queues != nil {
		for _, q := range effective.Queues.Producers {
			bKey := string(domain.BindingTypeQueue) + ":" + q.Binding
			bindingMap[bKey] = domain.ResourceBinding{
				Type:       domain.BindingTypeQueue,
				Name:       q.Binding,
				ResourceID: q.Queue,
			}
			importedBindings++
		}
	}

	// Static Assets
	if effective.Assets != nil && effective.Assets.Directory != "" {
		summary.AssetsDirectory = effective.Assets.Directory
		bName := effective.Assets.Binding
		if bName == "" {
			bName = "ASSETS"
		}
		bKey := string(domain.BindingTypeAssets) + ":" + bName
		bindingMap[bKey] = domain.ResourceBinding{
			Type:       domain.BindingTypeAssets,
			Name:       bName,
			ResourceID: effective.Assets.Directory,
		}
		importedBindings++
	}

	var finalBindings []domain.ResourceBinding
	for _, b := range bindingMap {
		finalBindings = append(finalBindings, b)
	}
	sort.Slice(finalBindings, func(i, j int) bool {
		if finalBindings[i].Type != finalBindings[j].Type {
			return finalBindings[i].Type < finalBindings[j].Type
		}
		return finalBindings[i].Name < finalBindings[j].Name
	})
	app.Bindings = finalBindings
	summary.ImportedBindingsCount = importedBindings

	// 4. Scheduled Triggers
	if effective.Triggers != nil && len(effective.Triggers.Crons) > 0 {
		summary.CronsCount = len(effective.Triggers.Crons)
	}

	return summary, nil
}
