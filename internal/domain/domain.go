package domain

import (
	"regexp"
	"strings"
	"time"
)

var hostnameRegex = regexp.MustCompile(`^(\*\.)?([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

// IsValidHostname reports whether the given hostname adheres to standard domain or wildcard domain format.
func IsValidHostname(hostname string) bool {
	return hostnameRegex.MatchString(strings.ToLower(strings.TrimSpace(hostname)))
}

// Domain represents a custom domain routing rule bound to an application.
type Domain struct {
	ID            string
	ApplicationID string
	Hostname      string
	PathPrefix    string
	SSLActive     bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewDomain constructs and validates a Domain routing entity.
func NewDomain(id, applicationID, hostname, pathPrefix string) (*Domain, error) {
	if strings.TrimSpace(id) == "" {
		return nil, NewValidationError("domain ID cannot be empty")
	}
	if strings.TrimSpace(applicationID) == "" {
		return nil, NewValidationError("application ID cannot be empty")
	}

	hostname = strings.ToLower(strings.TrimSpace(hostname))
	if !IsValidHostname(hostname) {
		return nil, NewValidationError("invalid hostname format: " + hostname)
	}

	pathPrefix = strings.TrimSpace(pathPrefix)
	if pathPrefix == "" {
		pathPrefix = "/"
	} else if !strings.HasPrefix(pathPrefix, "/") {
		pathPrefix = "/" + pathPrefix
	}

	now := time.Now().UTC()
	return &Domain{
		ID:            id,
		ApplicationID: applicationID,
		Hostname:      hostname,
		PathPrefix:    pathPrefix,
		SSLActive:     false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// SetSSLActive updates whether the Let's Encrypt certificate is verified active.
func (d *Domain) SetSSLActive(active bool) {
	d.SSLActive = active
	d.UpdatedAt = time.Now().UTC()
}
