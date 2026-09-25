package traefik

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
)

// Standard Cloudflare edge header keys
const (
	HeaderCFConnectingIP = "CF-Connecting-IP"
	HeaderCFIPCountry    = "CF-IPCountry"
	HeaderCFIPCity       = "CF-IPCity"
	HeaderCFRay          = "CF-Ray"
	HeaderCFVisitor      = "CF-Visitor"
)

// DefaultColo is the default datacenter/colo identifier for Cubit edge instances.
const DefaultColo = "CUBIT"

// EdgeHeaderOptions configures request header derivation and injection.
type EdgeHeaderOptions struct {
	Colo           string
	DefaultCountry string
	DefaultCity    string
	GeoIPResolver  *GeoIPResolver
}

// GeoIPOptions defines configuration and fallbacks for GeoIP resolution.
type GeoIPOptions struct {
	DefaultCountry string
	DefaultCity    string
	GeoIPDBPath    string
}

// GeoIPResolver resolves country and city metadata from client IPs.
type GeoIPResolver struct {
	opts GeoIPOptions
	mu   sync.RWMutex
}

// NewGeoIPResolver constructs a new GeoIPResolver.
func NewGeoIPResolver(opts GeoIPOptions) *GeoIPResolver {
	if opts.DefaultCountry == "" {
		opts.DefaultCountry = "XX"
	}
	if opts.DefaultCity == "" {
		opts.DefaultCity = "Default"
	}
	return &GeoIPResolver{opts: opts}
}

// Lookup determines country and city based on IP address and incoming headers.
func (r *GeoIPResolver) Lookup(ipStr string, headers http.Header) (country, city string) {
	if headers != nil {
		if c := headers.Get("X-GeoIP-Country"); c != "" {
			country = c
		} else if c := headers.Get("X-Country-Code"); c != "" {
			country = c
		} else if c := headers.Get(HeaderCFIPCountry); c != "" {
			country = c
		}

		if ct := headers.Get("X-GeoIP-City"); ct != "" {
			city = ct
		} else if ct := headers.Get("X-City-Name"); ct != "" {
			city = ct
		} else if ct := headers.Get(HeaderCFIPCity); ct != "" {
			city = ct
		}

		if country != "" && city != "" {
			return country, city
		}
	}

	parsedIP := net.ParseIP(ipStr)
	if parsedIP != nil {
		if parsedIP.IsLoopback() || parsedIP.IsPrivate() || parsedIP.IsUnspecified() {
			if country == "" {
				country = "XX"
			}
			if city == "" {
				city = "Localhost"
			}
			return country, city
		}
	}

	if country == "" {
		country = r.opts.DefaultCountry
	}
	if city == "" {
		city = r.opts.DefaultCity
	}
	return country, city
}

var defaultGeoIPResolver = NewGeoIPResolver(GeoIPOptions{
	DefaultCountry: "XX",
	DefaultCity:    "Default",
})

// ResolveGeoLocation resolves country and city using the default resolver.
func ResolveGeoLocation(ipStr string, headers http.Header) (country, city string) {
	return defaultGeoIPResolver.Lookup(ipStr, headers)
}

// GenerateRayID produces a 16-hex Cloudflare-compliant ray ID suffix-tagged with the given datacenter/colo.
func GenerateRayID(colo string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	hexStr := hex.EncodeToString(b)

	colo = strings.TrimSpace(colo)
	if colo == "" {
		colo = DefaultColo
	}
	return fmt.Sprintf("%s-%s", hexStr, strings.ToUpper(colo))
}

// BuildCFVisitor returns JSON serialization for the request scheme.
func BuildCFVisitor(isTLS bool) string {
	if isTLS {
		return `{"scheme":"https"}`
	}
	return `{"scheme":"http"}`
}

// DeriveClientIP extracts the real client IP from Cloudflare, reverse proxy headers, or RemoteAddr.
func DeriveClientIP(req *http.Request) string {
	if ip := req.Header.Get(HeaderCFConnectingIP); ip != "" {
		return strings.TrimSpace(ip)
	}

	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			first := strings.TrimSpace(parts[0])
			if first != "" {
				return first
			}
		}
	}

	if realIP := req.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	remote := strings.TrimSpace(req.RemoteAddr)
	if remote == "" {
		return "127.0.0.1"
	}

	host, _, err := net.SplitHostPort(remote)
	if err == nil && host != "" {
		return host
	}

	return strings.Trim(remote, "[]")
}

// InjectEdgeHeaders inspects an incoming HTTP request and attaches all standard Cloudflare edge headers.
func InjectEdgeHeaders(req *http.Request, opts EdgeHeaderOptions) {
	clientIP := req.Header.Get(HeaderCFConnectingIP)
	if clientIP == "" {
		clientIP = DeriveClientIP(req)
		req.Header.Set(HeaderCFConnectingIP, clientIP)
	}

	resolver := opts.GeoIPResolver
	if resolver == nil {
		if opts.DefaultCountry != "" || opts.DefaultCity != "" {
			resolver = NewGeoIPResolver(GeoIPOptions{
				DefaultCountry: opts.DefaultCountry,
				DefaultCity:    opts.DefaultCity,
			})
		} else {
			resolver = defaultGeoIPResolver
		}
	}

	if req.Header.Get(HeaderCFIPCountry) == "" || req.Header.Get(HeaderCFIPCity) == "" {
		country, city := resolver.Lookup(clientIP, req.Header)
		if req.Header.Get(HeaderCFIPCountry) == "" {
			req.Header.Set(HeaderCFIPCountry, country)
		}
		if req.Header.Get(HeaderCFIPCity) == "" {
			req.Header.Set(HeaderCFIPCity, city)
		}
	}

	if req.Header.Get(HeaderCFRay) == "" {
		req.Header.Set(HeaderCFRay, GenerateRayID(opts.Colo))
	}

	if req.Header.Get(HeaderCFVisitor) == "" {
		isTLS := req.TLS != nil ||
			strings.EqualFold(req.Header.Get("X-Forwarded-Proto"), "https") ||
			req.URL.Scheme == "https"
		req.Header.Set(HeaderCFVisitor, BuildCFVisitor(isTLS))
	}
}
