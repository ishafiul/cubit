package traefik_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/ishaf/cubit/internal/adapters/out/traefik"
)

func TestGenerateRayID(t *testing.T) {
	t.Run("Given a colo identifier When generating Ray ID", func(t *testing.T) {
		rayID := traefik.GenerateRayID("DFW")

		t.Run("Then format matches 16-character hex followed by hyphen and colo", func(t *testing.T) {
			re := regexp.MustCompile(`^[0-9a-f]{16}-DFW$`)
			if !re.MatchString(rayID) {
				t.Errorf("expected format [0-9a-f]{16}-DFW, got %q", rayID)
			}
		})

		t.Run("Then consecutive calls produce unique IDs", func(t *testing.T) {
			rayID2 := traefik.GenerateRayID("DFW")
			if rayID == rayID2 {
				t.Errorf("expected unique ray IDs, got identical %q", rayID)
			}
		})
	})

	t.Run("Given empty colo When generating Ray ID Then defaults to CUBIT", func(t *testing.T) {
		rayID := traefik.GenerateRayID("")
		if !strings.HasSuffix(rayID, "-CUBIT") {
			t.Errorf("expected suffix -CUBIT for empty colo, got %q", rayID)
		}
	})
}

func TestBuildCFVisitor(t *testing.T) {
	tests := []struct {
		name     string
		isTLS    bool
		expected string
	}{
		{
			name:     "Given HTTPS/TLS request Then returns https scheme JSON",
			isTLS:    true,
			expected: `{"scheme":"https"}`,
		},
		{
			name:     "Given plain HTTP request Then returns http scheme JSON",
			isTLS:    false,
			expected: `{"scheme":"http"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := traefik.BuildCFVisitor(tc.isTLS)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestDeriveClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name: "Given existing CF-Connecting-IP header Then returns it directly",
			headers: map[string]string{
				"CF-Connecting-IP": "203.0.113.195",
			},
			remoteAddr: "10.0.0.1:54321",
			expectedIP: "203.0.113.195",
		},
		{
			name: "Given X-Forwarded-For with multiple IPs Then returns first client IP",
			headers: map[string]string{
				"X-Forwarded-For": "198.51.100.42, 10.0.0.1",
			},
			remoteAddr: "10.0.0.1:54321",
			expectedIP: "198.51.100.42",
		},
		{
			name: "Given X-Real-IP header Then returns X-Real-IP",
			headers: map[string]string{
				"X-Real-IP": "198.51.100.99",
			},
			remoteAddr: "10.0.0.1:54321",
			expectedIP: "198.51.100.99",
		},
		{
			name:       "Given only RemoteAddr with port Then extracts IP without port",
			headers:    map[string]string{},
			remoteAddr: "192.0.2.1:12345",
			expectedIP: "192.0.2.1",
		},
		{
			name:       "Given RemoteAddr with IPv6 and port Then extracts clean IPv6",
			headers:    map[string]string{},
			remoteAddr: "[2001:db8::1]:8080",
			expectedIP: "2001:db8::1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
			req.RemoteAddr = tc.remoteAddr
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			ip := traefik.DeriveClientIP(req)
			if ip != tc.expectedIP {
				t.Errorf("expected %q, got %q", tc.expectedIP, ip)
			}
		})
	}
}

func TestResolveGeoLocation(t *testing.T) {
	t.Run("Given private or loopback IP When resolving location", func(t *testing.T) {
		country, city := traefik.ResolveGeoLocation("127.0.0.1", nil)
		if country != "XX" {
			t.Errorf("expected country XX for loopback, got %q", country)
		}
		if city != "Localhost" {
			t.Errorf("expected city Localhost for loopback, got %q", city)
		}
	})

	t.Run("Given upstream geo headers When resolving location", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-GeoIP-Country", "GB")
		headers.Set("X-GeoIP-City", "London")

		country, city := traefik.ResolveGeoLocation("82.165.197.1", headers)
		if country != "GB" {
			t.Errorf("expected country GB, got %q", country)
		}
		if city != "London" {
			t.Errorf("expected city London, got %q", city)
		}
	})

	t.Run("Given public IP without headers When resolving location with custom options", func(t *testing.T) {
		resolver := traefik.NewGeoIPResolver(traefik.GeoIPOptions{
			DefaultCountry: "US",
			DefaultCity:    "San Francisco",
		})
		country, city := resolver.Lookup("8.8.8.8", nil)
		if country != "US" {
			t.Errorf("expected default country US, got %q", country)
		}
		if city != "San Francisco" {
			t.Errorf("expected default city San Francisco, got %q", city)
		}
	})
}

func TestInjectEdgeHeaders(t *testing.T) {
	t.Run("Given an incoming HTTP request without Cloudflare headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/test", nil)
		req.RemoteAddr = "198.51.100.5:4321"

		opts := traefik.EdgeHeaderOptions{
			Colo:           "SJC",
			DefaultCountry: "US",
			DefaultCity:    "San Jose",
		}

		traefik.InjectEdgeHeaders(req, opts)

		t.Run("Then CF-Connecting-IP is populated with client IP", func(t *testing.T) {
			if got := req.Header.Get("CF-Connecting-IP"); got != "198.51.100.5" {
				t.Errorf("expected CF-Connecting-IP 198.51.100.5, got %q", got)
			}
		})

		t.Run("Then CF-IPCountry is populated", func(t *testing.T) {
			if got := req.Header.Get("CF-IPCountry"); got != "US" {
				t.Errorf("expected CF-IPCountry US, got %q", got)
			}
		})

		t.Run("Then CF-IPCity is populated", func(t *testing.T) {
			if got := req.Header.Get("CF-IPCity"); got != "San Jose" {
				t.Errorf("expected CF-IPCity San Jose, got %q", got)
			}
		})

		t.Run("Then CF-Ray is populated with valid ray ID containing colo", func(t *testing.T) {
			ray := req.Header.Get("CF-Ray")
			if !strings.HasSuffix(ray, "-SJC") {
				t.Errorf("expected CF-Ray ending with -SJC, got %q", ray)
			}
		})

		t.Run("Then CF-Visitor is populated with JSON scheme", func(t *testing.T) {
			visitor := req.Header.Get("CF-Visitor")
			if visitor != `{"scheme":"https"}` {
				t.Errorf("expected CF-Visitor {\"scheme\":\"https\"}, got %q", visitor)
			}
		})
	})

	t.Run("Given an incoming request that already contains CF-Connecting-IP and CF-Ray", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.example.com/test", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		req.Header.Set("CF-Connecting-IP", "203.0.113.88")
		req.Header.Set("CF-Ray", "existing-ray-id-1234")

		opts := traefik.EdgeHeaderOptions{
			Colo: "ORD",
		}

		traefik.InjectEdgeHeaders(req, opts)

		t.Run("Then existing CF-Connecting-IP is preserved", func(t *testing.T) {
			if got := req.Header.Get("CF-Connecting-IP"); got != "203.0.113.88" {
				t.Errorf("expected preserved CF-Connecting-IP 203.0.113.88, got %q", got)
			}
		})

		t.Run("Then existing CF-Ray is preserved", func(t *testing.T) {
			if got := req.Header.Get("CF-Ray"); got != "existing-ray-id-1234" {
				t.Errorf("expected preserved CF-Ray existing-ray-id-1234, got %q", got)
			}
		})
	})
}
