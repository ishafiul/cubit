package domain_test

import (
	"testing"

	"github.com/ishaf/cubit/internal/domain"
)

func TestDomain(t *testing.T) {
	t.Run("Given valid domain routing attributes", func(t *testing.T) {
		id := "dom-1"
		appID := "app-1"
		host := "api.example.com"
		path := "/v1"

		t.Run("When creating a domain then it initializes with leading slash and inactive SSL", func(t *testing.T) {
			dom, err := domain.NewDomain(id, appID, host, path)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if dom.Hostname != "api.example.com" {
				t.Errorf("expected hostname api.example.com, got %s", dom.Hostname)
			}
			if dom.PathPrefix != "/v1" {
				t.Errorf("expected pathPrefix /v1, got %s", dom.PathPrefix)
			}
			if dom.SSLActive {
				t.Error("expected SSLActive to be false initially")
			}
		})

		t.Run("When creating a domain with a wildcard hostname then it initializes successfully", func(t *testing.T) {
			dom, err := domain.NewDomain("dom-wildcard", "app-1", "*.example.com", "/api")

			if err != nil {
				t.Fatalf("expected no error for wildcard hostname, got %v", err)
			}
			if dom.Hostname != "*.example.com" {
				t.Errorf("expected hostname *.example.com, got %s", dom.Hostname)
			}
			if dom.PathPrefix != "/api" {
				t.Errorf("expected pathPrefix /api, got %s", dom.PathPrefix)
			}
		})
	})

	t.Run("Given an invalid hostname", func(t *testing.T) {
		t.Run("When creating domain with malformed host then it returns validation error", func(t *testing.T) {
			_, err := domain.NewDomain("dom-1", "app-1", "invalid_host_no_tld", "/")

			if err == nil {
				t.Fatal("expected error for malformed hostname, got nil")
			}
		})
	})

	t.Run("Given an active domain", func(t *testing.T) {
		dom, err := domain.NewDomain("dom-1", "app-1", "api.example.com", "")
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		t.Run("When certificate is provisioned then SSL becomes active", func(t *testing.T) {
			dom.SetSSLActive(true)

			if !dom.SSLActive {
				t.Error("expected SSLActive to be true")
			}
		})
	})
}
