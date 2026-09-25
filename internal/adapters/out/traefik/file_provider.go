package traefik

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// RouteRule encapsulates an ingress route rule for Traefik.
type RouteRule struct {
	AppName     string
	Hostname    string
	PathPrefix  string
	TargetURLs  []string
	EnableTLS   bool
	Middlewares []string
}

// EdgeHeaderConfig configures Traefik dynamic middleware for Cloudflare edge headers.
type EdgeHeaderConfig struct {
	EnableEdgeHeaders bool
	DefaultCountry    string
	DefaultCity       string
	DefaultColo       string
	GeoIPDBPath       string
	CustomHeaders     map[string]string
}

// TraefikConfig represents the YAML dynamic configuration structure for Traefik v3.
type TraefikConfig struct {
	HTTP HTTPConfig `yaml:"http"`
}

type HTTPConfig struct {
	Routers     map[string]RouterConfig     `yaml:"routers"`
	Services    map[string]ServiceConfig    `yaml:"services"`
	Middlewares map[string]MiddlewareConfig `yaml:"middlewares,omitempty"`
}

type RouterConfig struct {
	Rule        string   `yaml:"rule"`
	EntryPoints []string `yaml:"entryPoints"`
	Service     string   `yaml:"service"`
	Middlewares []string `yaml:"middlewares,omitempty"`
	TLS         *TLSConf `yaml:"tls,omitempty"`
}

type TLSConf struct {
	CertResolver string `yaml:"certResolver"`
}

type MiddlewareConfig struct {
	Headers *HeadersMiddlewareDef     `yaml:"headers,omitempty"`
	Plugin  map[string]map[string]any `yaml:"plugin,omitempty"`
}

type HeadersMiddlewareDef struct {
	CustomRequestHeaders map[string]string `yaml:"customRequestHeaders,omitempty"`
}

type ServiceConfig struct {
	LoadBalancer LoadBalancerConfig `yaml:"loadBalancer"`
}

type LoadBalancerConfig struct {
	Servers     []ServerEntry    `yaml:"servers"`
	HealthCheck *HealthCheckConf `yaml:"healthCheck,omitempty"`
}

type ServerEntry struct {
	URL string `yaml:"url"`
}

type HealthCheckConf struct {
	Path     string `yaml:"path"`
	Interval string `yaml:"interval"`
}

// FileProvider implements usecase.ProxyPort by writing dynamic YAML files for Traefik.
type FileProvider struct {
	outputPath   string
	certResolver string
	edgeCfg      EdgeHeaderConfig
	mu           sync.Mutex
}

// NewFileProvider creates a new FileProvider with default edge headers enabled.
func NewFileProvider(outputPath, certResolver string) *FileProvider {
	if certResolver == "" {
		certResolver = "letsencrypt"
	}
	return &FileProvider{
		outputPath:   outputPath,
		certResolver: certResolver,
		edgeCfg: EdgeHeaderConfig{
			EnableEdgeHeaders: true,
			DefaultCountry:    "XX",
			DefaultCity:       "Default",
			DefaultColo:       DefaultColo,
		},
	}
}

// NewFileProviderWithEdgeHeaders creates a FileProvider with custom edge header and GeoIP configuration.
func NewFileProviderWithEdgeHeaders(outputPath, certResolver string, edgeCfg EdgeHeaderConfig) *FileProvider {
	if certResolver == "" {
		certResolver = "letsencrypt"
	}
	return &FileProvider{
		outputPath:   outputPath,
		certResolver: certResolver,
		edgeCfg:      edgeCfg,
	}
}

// WithEdgeHeaders updates the edge header configuration on the FileProvider.
func (p *FileProvider) WithEdgeHeaders(edgeCfg EdgeHeaderConfig) *FileProvider {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.edgeCfg = edgeCfg
	return p
}

// SyncRoutes generates and writes the dynamic configuration for Traefik.
func (p *FileProvider) SyncRoutes(ctx context.Context, routes []RouteRule) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cfg := TraefikConfig{
		HTTP: HTTPConfig{
			Routers:     make(map[string]RouterConfig),
			Services:    make(map[string]ServiceConfig),
			Middlewares: make(map[string]MiddlewareConfig),
		},
	}

	var activeMiddlewares []string
	if p.edgeCfg.EnableEdgeHeaders {
		edgeMwName := "cubit-edge-headers"
		headers := map[string]string{
			HeaderCFVisitor:   `{"scheme":"https"}`,
			HeaderCFRay:       p.edgeCfg.DefaultColo,
			HeaderCFIPCountry: p.edgeCfg.DefaultCountry,
			HeaderCFIPCity:    p.edgeCfg.DefaultCity,
		}
		if headers[HeaderCFRay] == "" {
			headers[HeaderCFRay] = DefaultColo
		}
		if headers[HeaderCFIPCountry] == "" {
			headers[HeaderCFIPCountry] = "XX"
		}
		if headers[HeaderCFIPCity] == "" {
			headers[HeaderCFIPCity] = "Default"
		}
		for k, v := range p.edgeCfg.CustomHeaders {
			headers[k] = v
		}

		cfg.HTTP.Middlewares[edgeMwName] = MiddlewareConfig{
			Headers: &HeadersMiddlewareDef{
				CustomRequestHeaders: headers,
			},
		}
		activeMiddlewares = append(activeMiddlewares, edgeMwName)

		if p.edgeCfg.GeoIPDBPath != "" {
			geoMwName := "cubit-geoip"
			cfg.HTTP.Middlewares[geoMwName] = MiddlewareConfig{
				Plugin: map[string]map[string]any{
					"geoip": {
						"dbPath": p.edgeCfg.GeoIPDBPath,
					},
				},
			}
			activeMiddlewares = append(activeMiddlewares, geoMwName)
		}
	}

	for _, r := range routes {
		serviceName := "svc-" + sanitize(r.AppName)
		routerName := fmt.Sprintf("rt-%s-%s", sanitize(r.AppName), sanitize(r.Hostname))
		if r.PathPrefix != "" && r.PathPrefix != "/" {
			routerName = fmt.Sprintf("%s-%s", routerName, sanitize(strings.Trim(r.PathPrefix, "/")))
		}

		// Build Traefik routing rule
		var ruleParts []string
		if r.Hostname != "" {
			ruleParts = append(ruleParts, fmt.Sprintf("Host(`%s`)", r.Hostname))
		}
		if r.PathPrefix != "" && r.PathPrefix != "/" {
			ruleParts = append(ruleParts, fmt.Sprintf("PathPrefix(`%s`)", r.PathPrefix))
		}

		rule := strings.Join(ruleParts, " && ")
		if rule == "" {
			rule = "PathPrefix(`/`)"
		}

		routerMiddlewares := append([]string(nil), activeMiddlewares...)
		routerMiddlewares = append(routerMiddlewares, r.Middlewares...)

		router := RouterConfig{
			Rule:        rule,
			EntryPoints: []string{"web", "websecure"},
			Service:     serviceName,
			Middlewares: routerMiddlewares,
		}

		if r.EnableTLS {
			router.TLS = &TLSConf{CertResolver: p.certResolver}
		}

		cfg.HTTP.Routers[routerName] = router

		// Build LoadBalancer service
		if _, exists := cfg.HTTP.Services[serviceName]; !exists {
			var servers []ServerEntry
			for _, u := range r.TargetURLs {
				servers = append(servers, ServerEntry{URL: u})
			}
			cfg.HTTP.Services[serviceName] = ServiceConfig{
				LoadBalancer: LoadBalancerConfig{
					Servers: servers,
					HealthCheck: &HealthCheckConf{
						Path:     "/health",
						Interval: "5s",
					},
				},
			}
		}
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed marshaling traefik yaml: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(p.outputPath), 0755); err != nil {
		return fmt.Errorf("failed creating traefik dynamic config directory: %w", err)
	}

	// Atomic write via temp file
	tmpFile := p.outputPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed writing traefik temp config: %w", err)
	}

	if err := os.Rename(tmpFile, p.outputPath); err != nil {
		return fmt.Errorf("failed renaming traefik config file: %w", err)
	}

	return nil
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "*", "wildcard")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, "/", "-")
	return strings.ToLower(s)
}
