package traefik

import (
	"context"
	"fmt"

	"github.com/ishaf/cubit/internal/domain"
)

// AppLister provides a list of registered applications.
type AppLister interface {
	List(ctx context.Context) ([]*domain.Application, error)
}

// DomainLister provides custom domains configured for an application.
type DomainLister interface {
	ListByAppID(ctx context.Context, appID string) ([]*domain.Domain, error)
}

// NodeLister provides active worker nodes.
type NodeLister interface {
	List(ctx context.Context) ([]*domain.Node, error)
}

// RouteSyncer coordinates Traefik dynamic configuration updates by reading active nodes and routes.
type RouteSyncer struct {
	provider     *FileProvider
	appLister    AppLister
	domainLister DomainLister
	nodeLister   NodeLister
}

// NewRouteSyncer constructs a new RouteSyncer.
func NewRouteSyncer(provider *FileProvider, appLister AppLister, domainLister DomainLister, nodeLister NodeLister) *RouteSyncer {
	return &RouteSyncer{
		provider:     provider,
		appLister:    appLister,
		domainLister: domainLister,
		nodeLister:   nodeLister,
	}
}

// SyncRoutes reads active nodes, running applications, and custom domains to rewrite Traefik routes.
func (s *RouteSyncer) SyncRoutes(ctx context.Context) error {
	nodes, err := s.nodeLister.List(ctx)
	if err != nil {
		return err
	}

	// Collect active node IPs and the shared fallback port
	var activeNodes []*domain.Node
	for _, n := range nodes {
		if n.Status == domain.NodeStatusActive {
			activeNodes = append(activeNodes, n)
		}
	}

	if len(activeNodes) == 0 {
		return s.provider.SyncRoutes(ctx, nil)
	}

	apps, err := s.appLister.List(ctx)
	if err != nil {
		return err
	}

	var rules []RouteRule
	for _, app := range apps {
		if app.Status != domain.AppStatusRunning {
			continue
		}

		// Build per-app target URLs using the app's assigned worker port.
		// Falls back to the node's shared worker port if no port is allocated.
		var targetURLs []string
		for _, n := range activeNodes {
			port := app.WorkerPort
			if port <= 0 {
				port = n.WorkerPort
			}
			targetURLs = append(targetURLs, fmt.Sprintf("http://%s:%d", n.IPAddress, port))
		}

		subdomain := app.Subdomain
		if subdomain == "" {
			subdomain = domain.SanitizeSubdomain(app.Name)
		}

		rules = append(rules, RouteRule{
			AppName:    app.Name,
			Hostname:   fmt.Sprintf("%s.localhost", subdomain),
			PathPrefix: "/",
			TargetURLs: targetURLs,
			EnableTLS:  false,
		})
		rules = append(rules, RouteRule{
			AppName:    app.Name,
			Hostname:   fmt.Sprintf("%s.cubit.local", subdomain),
			PathPrefix: "/",
			TargetURLs: targetURLs,
			EnableTLS:  false,
		})

		domains, err := s.domainLister.ListByAppID(ctx, app.ID)
		if err != nil {
			continue
		}

		for _, d := range domains {
			rules = append(rules, RouteRule{
				AppName:    app.Name,
				Hostname:   d.Hostname,
				PathPrefix: d.PathPrefix,
				TargetURLs: targetURLs,
				EnableTLS:  d.SSLActive,
			})
		}
	}

	return s.provider.SyncRoutes(ctx, rules)
}
