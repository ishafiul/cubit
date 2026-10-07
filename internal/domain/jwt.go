package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrTokenExpired    = errors.New("jwt token has expired")
	ErrInvalidSignature = errors.New("invalid jwt token signature")
	ErrMalformedToken   = errors.New("malformed jwt token")
	ErrEmptySecret      = errors.New("jwt secret cannot be empty")
)

// JWTHeader represents the standard JOSE header for HMAC-SHA256 JWTs.
type JWTHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// JWTClaims holds identity and permission claims for an authenticated caller.
type JWTClaims struct {
	UserID      string   `json:"sub"`
	Email       string   `json:"email"`
	RoleID      string   `json:"role_id"`
	Permissions []string `json:"permissions"`
	IssuedAt    int64    `json:"iat"`
	ExpiresAt   int64    `json:"exp"`
	Issuer      string   `json:"iss,omitempty"`
}

// GenerateJWT creates and signs a new compact HMAC-SHA256 JWT token string.
func GenerateJWT(claims JWTClaims, secret string, ttl time.Duration) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", ErrEmptySecret
	}

	now := time.Now().UTC()
	claims.IssuedAt = now.Unix()
	claims.ExpiresAt = now.Add(ttl).Unix()
	if claims.Issuer == "" {
		claims.Issuer = "cubit-control-plane"
	}

	header := JWTHeader{
		Alg: "HS256",
		Typ: "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal jwt header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal jwt claims: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := encodedHeader + "." + encodedClaims

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	signature := mac.Sum(nil)
	encodedSignature := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + encodedSignature, nil
}

// ValidateJWT verifies the signature and expiration of a compact JWT and returns the parsed claims.
func ValidateJWT(tokenStr, secret string) (*JWTClaims, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, ErrEmptySecret
	}

	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return nil, ErrMalformedToken
	}

	signingInput := parts[0] + "." + parts[1]
	providedSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("%w: invalid signature encoding", ErrMalformedToken)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expectedSignature := mac.Sum(nil)

	if !hmac.Equal(providedSignature, expectedSignature) {
		return nil, ErrInvalidSignature
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: invalid claims encoding", ErrMalformedToken)
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, fmt.Errorf("%w: failed to parse claims json", ErrMalformedToken)
	}

	now := time.Now().UTC().Unix()
	if claims.ExpiresAt < now {
		return nil, ErrTokenExpired
	}

	return &claims, nil
}
