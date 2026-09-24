package deployment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/modules/wrangler"
)

// HTTPClient abstracts HTTP client interactions for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// CelldModule represents an individual code or asset module in a celld deployment.
type CelldModule struct {
	Name   string `json:"name"`
	Type   string `json:"type"` // "esm", "commonjs", "wasm", "text", "data"
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

// CelldAssetConfig represents static assets routing configuration for celld.
type CelldAssetConfig struct {
	Directory        string `json:"directory,omitempty"`
	Binding          string `json:"binding,omitempty"`
	HTMLHandling     string `json:"html_handling,omitempty"`
	NotFoundHandling string `json:"not_found_handling,omitempty"`
	RunWorkerFirst   bool   `json:"run_worker_first,omitempty"`
}

// CelldManifest encapsulates the deployment specification executed by celld nodes.
type CelldManifest struct {
	SchemaVersion    int                    `json:"schema_version"`
	Version          string                 `json:"version"`
	ScriptName       string                 `json:"script_name"`
	MainModule       string                 `json:"main_module"`
	Modules          []CelldModule          `json:"modules"`
	DOClasses        []string               `json:"do_classes"`
	SQLiteClasses    []string               `json:"sqlite_classes"`
	Assets           *CelldAssetConfig      `json:"assets,omitempty"`
	Crons            []string               `json:"crons,omitempty"`
	QueueConsumers   []string               `json:"queue_consumers,omitempty"`
	Containers       []string               `json:"containers,omitempty"`
	RequiredFeatures []string               `json:"required_features,omitempty"`
	RawMetadata      map[string]interface{} `json:"raw_metadata,omitempty"`
}

// CelldCurrentPointer specifies the active deployment pointer stored in deploy/current.json.
type CelldCurrentPointer struct {
	Version      string         `json:"version"`
	ScriptName   string         `json:"script_name"`
	ManifestPath string         `json:"manifest_path"`
	BundlePath   string         `json:"bundle_path,omitempty"`
	Digest       string         `json:"digest,omitempty"`
	UpdatedAt    string         `json:"updated_at,omitempty"`
	Manifest     *CelldManifest `json:"manifest,omitempty"`
}

// ComputeSHA256 returns the lowercase hex-encoded sha256 hash prefixed with sha256:
func ComputeSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

// BuildCelldManifest constructs a normalized celld deployment manifest for an application and deployment.
func BuildCelldManifest(
	app *domain.Application,
	dep *domain.Deployment,
	bundle []byte,
	sanitizedConfig *wrangler.WranglerConfig,
) (*CelldManifest, error) {
	if app == nil {
		return nil, domain.NewValidationError("application must not be nil")
	}
	if dep == nil {
		return nil, domain.NewValidationError("deployment must not be nil")
	}

	digest := ComputeSHA256(bundle)
	scriptName := app.Name
	if scriptName == "" && sanitizedConfig != nil {
		scriptName = sanitizedConfig.Name
	}

	mainModule := "worker.js"
	modules := []CelldModule{
		{
			Name:   mainModule,
			Type:   "esm",
			Digest: digest,
			Size:   int64(len(bundle)),
		},
	}

	// Gather DO classes and SQLite-backed classes
	doClassesSet := make(map[string]struct{})
	var doClasses []string
	addDOClass := func(cls string) {
		cls = strings.TrimSpace(cls)
		if cls == "" {
			return
		}
		if _, exists := doClassesSet[cls]; !exists {
			doClassesSet[cls] = struct{}{}
			doClasses = append(doClasses, cls)
		}
	}

	for _, b := range app.Bindings {
		if b.Type == domain.BindingTypeDurableObject && b.ClassName != "" {
			addDOClass(b.ClassName)
		}
	}

	if sanitizedConfig != nil && sanitizedConfig.DurableObjects != nil {
		for _, b := range sanitizedConfig.DurableObjects.Bindings {
			addDOClass(b.ClassName)
		}
	}

	for _, step := range app.Migrations {
		for _, cls := range step.NewClasses {
			addDOClass(cls)
		}
		for _, renamed := range step.RenamedClasses {
			addDOClass(renamed.To)
		}
	}

	sort.Strings(doClasses)
	if doClasses == nil {
		doClasses = []string{}
	}

	// In celld, Durable Objects default to private SQLite-backed cells
	sqliteClasses := make([]string, len(doClasses))
	copy(sqliteClasses, doClasses)

	// Gather crons
	var crons []string
	if sanitizedConfig != nil && sanitizedConfig.Triggers != nil && len(sanitizedConfig.Triggers.Crons) > 0 {
		crons = make([]string, len(sanitizedConfig.Triggers.Crons))
		copy(crons, sanitizedConfig.Triggers.Crons)
	}

	// Gather assets
	var assets *CelldAssetConfig
	if sanitizedConfig != nil && sanitizedConfig.Assets != nil {
		assets = &CelldAssetConfig{
			Directory:    sanitizedConfig.Assets.Directory,
			Binding:      sanitizedConfig.Assets.Binding,
			HTMLHandling: sanitizedConfig.Assets.HTMLHandling,
		}
	} else {
		for _, b := range app.Bindings {
			if b.Type == domain.BindingTypeAssets {
				assets = &CelldAssetConfig{
					Binding: b.Name,
				}
				break
			}
		}
	}

	// Collect required features
	var requiredFeatures []string
	featureSet := make(map[string]struct{})
	addFeature := func(feat string) {
		if _, exists := featureSet[feat]; !exists {
			featureSet[feat] = struct{}{}
			requiredFeatures = append(requiredFeatures, feat)
		}
	}

	for _, b := range app.Bindings {
		switch b.Type {
		case domain.BindingTypeKV:
			addFeature("kv-v1")
		case domain.BindingTypeD1:
			addFeature("d1-v1")
		case domain.BindingTypeR2:
			addFeature("r2-v1")
		case domain.BindingTypeDurableObject:
			addFeature("durable-objects")
		case domain.BindingTypeWorkflow:
			addFeature("workflows")
		case domain.BindingTypeContainer:
			addFeature("containers")
		}
	}
	if len(doClasses) > 0 {
		addFeature("durable-objects")
	}
	sort.Strings(requiredFeatures)

	// Assemble raw metadata
	rawMetadata := make(map[string]interface{})
	if app.CompatibilityDate != "" {
		rawMetadata["compatibility_date"] = app.CompatibilityDate
	}
	if len(app.CompatibilityFlags) > 0 {
		rawMetadata["compatibility_flags"] = app.CompatibilityFlags
	}
	rawMetadata["generated_at"] = time.Now().UTC().Format(time.RFC3339)

	manifest := &CelldManifest{
		SchemaVersion:    1,
		Version:          dep.ID,
		ScriptName:       scriptName,
		MainModule:       mainModule,
		Modules:          modules,
		DOClasses:        doClasses,
		SQLiteClasses:    sqliteClasses,
		Assets:           assets,
		Crons:            crons,
		RequiredFeatures: requiredFeatures,
		RawMetadata:      rawMetadata,
	}

	return manifest, nil
}

// BuildCelldDeployment generates the celld deployment JSON manifest specified in PRD Feature 3.2.
func BuildCelldDeployment(ctx context.Context, app *domain.Application, dep *domain.Deployment) ([]byte, error) {
	if app == nil {
		return nil, domain.NewValidationError("application must not be nil")
	}
	if dep == nil {
		return nil, domain.NewValidationError("deployment must not be nil")
	}
	bundle := []byte(app.InlineCode)
	if len(bundle) == 0 {
		bundle = []byte(domain.DefaultHelloWorldWorker)
	}
	manifest, err := BuildCelldManifest(app, dep, bundle, nil)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal celld deployment manifest: %w", err)
	}
	return data, nil
}

// BuildCurrentPointer creates the active pointer structure for deploy/current.json.
func BuildCurrentPointer(manifest *CelldManifest, manifestPath, bundlePath string) *CelldCurrentPointer {
	var digest string
	if manifest != nil && len(manifest.Modules) > 0 {
		digest = manifest.Modules[0].Digest
	}
	version := ""
	scriptName := ""
	if manifest != nil {
		version = manifest.Version
		scriptName = manifest.ScriptName
	}

	return &CelldCurrentPointer{
		Version:      version,
		ScriptName:   scriptName,
		ManifestPath: manifestPath,
		BundlePath:   bundlePath,
		Digest:       digest,
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
		Manifest:     manifest,
	}
}

// FormatCurrentPointer marshals the deployment pointer to indented JSON.
func FormatCurrentPointer(manifest *CelldManifest, manifestPath, bundlePath string) ([]byte, error) {
	ptr := BuildCurrentPointer(manifest, manifestPath, bundlePath)
	data, err := json.MarshalIndent(ptr, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format current pointer: %w", err)
	}
	return data, nil
}

// TriggerCelldReload sends a zero-downtime hot reload request to celld's internal listener (:8081 or specified port).
func TriggerCelldReload(ctx context.Context, internalPort int) error {
	if internalPort <= 0 {
		internalPort = 8081
	}
	targetURL := fmt.Sprintf("http://127.0.0.1:%d/reload", internalPort)
	return TriggerCelldReloadURL(ctx, nil, targetURL)
}

// TriggerCelldReloadURL sends a reload trigger to an arbitrary celld internal URL with a custom HTTPClient.
func TriggerCelldReloadURL(ctx context.Context, client HTTPClient, targetURL string) error {
	targetURL = strings.TrimSpace(targetURL)
	if targetURL == "" {
		return domain.NewValidationError("target URL cannot be empty")
	}

	u, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid reload URL %q: %w", targetURL, err)
	}

	if !strings.HasSuffix(u.Path, "/reload") {
		u.Path = strings.TrimSuffix(u.Path, "/") + "/reload"
	}
	fullURL := u.String()

	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create reload request: %w", err)
	}
	req.Header.Set("User-Agent", "Cubit-Control-Plane")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("celld reload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("celld reload failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	return nil
}

// TriggerFleetReload triggers zero-downtime hot reloads across all specified celld listener URLs concurrently.
func TriggerFleetReload(ctx context.Context, client HTTPClient, targetURLs []string) []error {
	if len(targetURLs) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for _, targetURL := range targetURLs {
		if strings.TrimSpace(targetURL) == "" {
			continue
		}
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			if err := TriggerCelldReloadURL(ctx, client, url); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(targetURL)
	}

	wg.Wait()
	return errs
}
