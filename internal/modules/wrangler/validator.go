package wrangler

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CompatibilityLevel indicates the overall deployment compatibility for celld.
type CompatibilityLevel string

const (
	LevelCompatible   CompatibilityLevel = "compatible"
	LevelWarning      CompatibilityLevel = "warning"
	LevelIncompatible CompatibilityLevel = "incompatible"
)

// FeatureStatus classifies an individual feature finding.
type FeatureStatus string

const (
	StatusSupported   FeatureStatus = "supported"
	StatusWarning     FeatureStatus = "warning"
	StatusUnsupported FeatureStatus = "unsupported"
)

// FeatureFinding details a detected feature, API, or configuration directive.
type FeatureFinding struct {
	Name        string        `json:"name"`
	Category    string        `json:"category"` // "config", "binding", "source_import", "api_usage"
	Status      FeatureStatus `json:"status"`   // "supported", "warning", "unsupported"
	File        string        `json:"file,omitempty"`
	Line        int           `json:"line,omitempty"`
	Details     string        `json:"details"`
	Remediation string        `json:"remediation,omitempty"`
}

// CompatibilityReport encapsulates the full pre-flight compatibility evaluation.
type CompatibilityReport struct {
	Level               CompatibilityLevel `json:"level"`
	CanDeploy           bool               `json:"can_deploy"`
	TotalIssues         int                `json:"total_issues"`
	UnsupportedCount    int                `json:"unsupported_count"`
	WarningCount        int                `json:"warning_count"`
	SupportedCount      int                `json:"supported_count"`
	SupportedFeatures   []FeatureFinding   `json:"supported_features"`
	UnsupportedFeatures []FeatureFinding   `json:"unsupported_features"`
	Warnings            []FeatureFinding   `json:"warnings"`
	Summary             string             `json:"summary"`
}

type patternRule struct {
	regex       *regexp.Regexp
	name        string
	category    string
	status      FeatureStatus
	details     string
	remediation string
}

var sourceRules = []patternRule{
	// Unsupported APIs
	{
		regex:       regexp.MustCompile(`(?:\bimport\s+.*?["']@cloudflare/ai["']|\benv\s*\.\s*AI\b|\bAi\s*\.\s*run\b)`),
		name:        "Cloudflare Workers AI (env.AI)",
		category:    "api_usage",
		status:      StatusUnsupported,
		details:     "Workers AI model inference (@cf/*) is not natively embedded in celld.",
		remediation: "Cubit runs V8 isolates with private SQLite cells. To run LLM inference, proxy calls to an external OpenAI/Ollama compatible endpoint or self-hosted LLM server via fetch().",
	},
	{
		regex:       regexp.MustCompile(`(?:\bimport\s+.*?["']@cloudflare/vectorize["']|\benv\s*\.\s*VECTORIZE\b)`),
		name:        "Cloudflare Vectorize (env.VECTORIZE)",
		category:    "api_usage",
		status:      StatusUnsupported,
		details:     "Vectorize vector database embeddings are not embedded in celld.",
		remediation: "Vectorize embeddings are not supported in celld isolates. Connect to PostgreSQL with pgvector, Qdrant, or Chroma via HTTP fetch().",
	},
	{
		regex:       regexp.MustCompile(`\benv\s*\.\s*HYPERDRIVE\b`),
		name:        "Cloudflare Hyperdrive (env.HYPERDRIVE)",
		category:    "api_usage",
		status:      StatusUnsupported,
		details:     "Hyperdrive connection pooling is a Cloudflare-managed feature.",
		remediation: "Hyperdrive connection pooling is a Cloudflare-managed feature. Connect directly to Postgres or MySQL using standard client connection pooling or a TCP proxy.",
	},
	{
		regex:       regexp.MustCompile(`(?:\bimport\s+.*?["']@cloudflare/puppeteer["']|\benv\s*\.\s*BROWSER\b)`),
		name:        "Cloudflare Browser Rendering (@cloudflare/puppeteer)",
		category:    "source_import",
		status:      StatusUnsupported,
		details:     "Cloudflare Browser Rendering uses external managed Chromium clusters.",
		remediation: "Headless browser instances require external container execution. Launch a dedicated Chromium container and control it via Chrome DevTools Protocol (CDP) over WebSockets.",
	},
	{
		regex:       regexp.MustCompile(`\benv\s*\.\s*ANALYTICS\b`),
		name:        "Cloudflare Analytics Engine (env.ANALYTICS)",
		category:    "api_usage",
		status:      StatusUnsupported,
		details:     "Analytics Engine datasets are proprietary to Cloudflare edge networks.",
		remediation: "Analytics Engine is proprietary to Cloudflare. Use SQLite inside a Durable Object, ClickHouse, or a Prometheus metrics endpoint.",
	},

	// Supported APIs
	{
		regex:    regexp.MustCompile(`\breq(?:uest)?\s*\.\s*cf\b`),
		name:     "request.cf Edge Context",
		category: "api_usage",
		status:   StatusSupported,
		details:  "Cloudflare edge metadata (country, city, clientIP, colo, rayID) synthesized by Cubit ingress.",
	},
	{
		regex:    regexp.MustCompile(`\bcaches\s*\.\s*default\b`),
		name:     "Cache API (caches.default)",
		category: "api_usage",
		status:   StatusSupported,
		details:  "Edge cache simulation supported in isolate runtime.",
	},
	{
		regex:    regexp.MustCompile(`\bnew\s+WebSocketPair\b`),
		name:     "WebSocketPair API",
		category: "api_usage",
		status:   StatusSupported,
		details:  "Bidirectional WebSockets fully supported in celld and isolate runtime.",
	},
	{
		regex:    regexp.MustCompile(`\bctx\s*\.\s*waitUntil\b`),
		name:     "ctx.waitUntil Lifecycle Hook",
		category: "api_usage",
		status:   StatusSupported,
		details:  "Asynchronous background task lifecycle fully supported.",
	},
}

// ValidateCompatibility scans a parsed Wrangler configuration and repository source files,
// generating an automated compatibility report with diagnostic remediation guidance.
func ValidateCompatibility(cfg *WranglerConfig, srcDir string) (*CompatibilityReport, error) {
	report := &CompatibilityReport{
		Level:               LevelCompatible,
		CanDeploy:           true,
		SupportedFeatures:   []FeatureFinding{},
		UnsupportedFeatures: []FeatureFinding{},
		Warnings:            []FeatureFinding{},
	}

	// 1. Inspect Wrangler configuration bindings
	if cfg != nil {
		for _, kv := range cfg.KVNamespaces {
			report.SupportedFeatures = append(report.SupportedFeatures, FeatureFinding{
				Name:     fmt.Sprintf("KV Namespace (%s)", kv.Binding),
				Category: "binding",
				Status:   StatusSupported,
				Details:  fmt.Sprintf("Backed by SQLite key-value store (id: %s)", kv.ID),
			})
		}

		for _, db := range cfg.D1Databases {
			report.SupportedFeatures = append(report.SupportedFeatures, FeatureFinding{
				Name:     fmt.Sprintf("D1 Database (%s)", db.Binding),
				Category: "binding",
				Status:   StatusSupported,
				Details:  fmt.Sprintf("Backed by private SQLite database (id: %s)", db.DatabaseID),
			})
		}

		for _, r2 := range cfg.R2Buckets {
			report.SupportedFeatures = append(report.SupportedFeatures, FeatureFinding{
				Name:     fmt.Sprintf("R2 Bucket (%s)", r2.Binding),
				Category: "binding",
				Status:   StatusSupported,
				Details:  fmt.Sprintf("Backed by Garage S3 distributed bucket (%s)", r2.BucketName),
			})
		}

		if cfg.DurableObjects != nil {
			for _, do := range cfg.DurableObjects.Bindings {
				report.SupportedFeatures = append(report.SupportedFeatures, FeatureFinding{
					Name:     fmt.Sprintf("Durable Object (%s)", do.Name),
					Category: "binding",
					Status:   StatusSupported,
					Details:  fmt.Sprintf("Backed by celld private SQLite cell for class %s", do.ClassName),
				})
			}
		}

		for _, wf := range cfg.Workflows {
			report.SupportedFeatures = append(report.SupportedFeatures, FeatureFinding{
				Name:     fmt.Sprintf("Workflow (%s)", wf.Binding),
				Category: "binding",
				Status:   StatusSupported,
				Details:  fmt.Sprintf("Durable execution engine workflow %s (class: %s)", wf.Name, wf.ClassName),
			})
		}

		for _, ct := range cfg.Containers {
			report.SupportedFeatures = append(report.SupportedFeatures, FeatureFinding{
				Name:     fmt.Sprintf("Container (%s)", ct.Name),
				Category: "binding",
				Status:   StatusSupported,
				Details:  fmt.Sprintf("Docker container workload %s (%s:%d)", ct.Name, ct.Image, ct.Port),
			})
		}

		for _, s := range cfg.Services {
			report.SupportedFeatures = append(report.SupportedFeatures, FeatureFinding{
				Name:     fmt.Sprintf("Service Binding (%s)", s.Binding),
				Category: "binding",
				Status:   StatusSupported,
				Details:  fmt.Sprintf("Worker-to-worker invocation target: %s", s.Service),
			})
		}

		if cfg.Assets != nil && cfg.Assets.Directory != "" {
			report.SupportedFeatures = append(report.SupportedFeatures, FeatureFinding{
				Name:     "Static Assets",
				Category: "config",
				Status:   StatusSupported,
				Details:  fmt.Sprintf("Directory %s served directly from Garage S3 / HTTP cache", cfg.Assets.Directory),
			})
		}

		if cfg.Triggers != nil && len(cfg.Triggers.Crons) > 0 {
			report.SupportedFeatures = append(report.SupportedFeatures, FeatureFinding{
				Name:     "Cron Triggers",
				Category: "config",
				Status:   StatusSupported,
				Details:  fmt.Sprintf("%d scheduled trigger expression(s)", len(cfg.Triggers.Crons)),
			})
		}

		if cfg.Queues != nil && len(cfg.Queues.Producers) > 0 {
			report.Warnings = append(report.Warnings, FeatureFinding{
				Name:        "Queue Producers",
				Category:    "config",
				Status:      StatusWarning,
				Details:     "Queues require external message broker or DO emulation.",
				Remediation: "Queue messages can be dispatched using SQLite Durable Objects or external Redis/RabbitMQ message queues.",
			})
		}
	}

	// 2. Scan source code files if directory is provided
	if srcDir != "" {
		if stat, err := os.Stat(srcDir); err == nil && stat.IsDir() {
			scannedFeatures := make(map[string]bool)

			_ = filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d == nil {
					return nil
				}

				if d.IsDir() {
					name := d.Name()
					if name == "node_modules" || name == ".git" || name == "dist" || name == ".wrangler" {
						return filepath.SkipDir
					}
					return nil
				}

				ext := strings.ToLower(filepath.Ext(path))
				if ext != ".js" && ext != ".ts" && ext != ".mjs" && ext != ".jsx" && ext != ".tsx" && ext != ".cjs" {
					return nil
				}

				content, err := os.ReadFile(path)
				if err != nil {
					return nil
				}

				relPath, _ := filepath.Rel(srcDir, path)
				if relPath == "" {
					relPath = filepath.Base(path)
				}

				scanner := bufio.NewScanner(bytes.NewReader(content))
				lineNum := 0
				for scanner.Scan() {
					lineNum++
					lineText := scanner.Text()

					for _, rule := range sourceRules {
						if rule.regex.MatchString(lineText) {
							key := fmt.Sprintf("%s:%s", rule.name, relPath)
							if scannedFeatures[key] {
								continue
							}
							scannedFeatures[key] = true

							finding := FeatureFinding{
								Name:        rule.name,
								Category:    rule.category,
								Status:      rule.status,
								File:        relPath,
								Line:        lineNum,
								Details:     rule.details,
								Remediation: rule.remediation,
							}

							switch rule.status {
							case StatusUnsupported:
								report.UnsupportedFeatures = append(report.UnsupportedFeatures, finding)
							case StatusWarning:
								report.Warnings = append(report.Warnings, finding)
							case StatusSupported:
								report.SupportedFeatures = append(report.SupportedFeatures, finding)
							}
						}
					}
				}
				return nil
			})
		}
	}

	// 3. Compute totals and overall compatibility level
	report.SupportedCount = len(report.SupportedFeatures)
	report.UnsupportedCount = len(report.UnsupportedFeatures)
	report.WarningCount = len(report.Warnings)
	report.TotalIssues = report.UnsupportedCount + report.WarningCount

	if report.UnsupportedCount > 0 {
		report.Level = LevelIncompatible
		report.CanDeploy = false
		report.Summary = fmt.Sprintf("Project has %d incompatible Cloudflare feature(s) that require remediation before deploying to celld.", report.UnsupportedCount)
	} else if report.WarningCount > 0 {
		report.Level = LevelWarning
		report.CanDeploy = true
		report.Summary = fmt.Sprintf("Project is deployable with %d warning(s). Review remediation steps for best performance.", report.WarningCount)
	} else {
		report.Level = LevelCompatible
		report.CanDeploy = true
		report.Summary = fmt.Sprintf("Project is fully compatible with celld (%d supported feature(s) detected).", report.SupportedCount)
	}

	return report, nil
}

// ScanProject locates wrangler config in the given directory and executes compatibility validation.
func ScanProject(projectPath string) (*CompatibilityReport, error) {
	stat, err := os.Stat(projectPath)
	if err != nil {
		return nil, fmt.Errorf("project path does not exist: %w", err)
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("project path is not a directory: %s", projectPath)
	}

	configCandidates := []string{
		"wrangler.jsonc",
		"wrangler.json",
		"wrangler.toml",
	}

	var parsedCfg *WranglerConfig
	for _, cand := range configCandidates {
		configPath := filepath.Join(projectPath, cand)
		if data, err := os.ReadFile(configPath); err == nil {
			var formatHint string
			if strings.HasSuffix(cand, ".toml") {
				formatHint = "toml"
			} else {
				formatHint = "json"
			}
			cfg, _, err := Parse(data, formatHint)
			if err == nil && cfg != nil {
				parsedCfg = cfg
				break
			}
		}
	}

	return ValidateCompatibility(parsedCfg, projectPath)
}
