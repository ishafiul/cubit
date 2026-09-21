package domain_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/ishaf/cubit/internal/domain"
)

func TestVerifyGitHubHMAC(t *testing.T) {
	t.Run("Given a payload and a webhook secret", func(t *testing.T) {
		secret := "super-secret-key-123"
		payload := []byte(`{"ref":"refs/heads/main","repository":{"full_name":"owner/repo"}}`)

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

		t.Run("When verifying a valid signature", func(t *testing.T) {
			t.Run("Then verification succeeds", func(t *testing.T) {
				if !domain.VerifyGitHubHMAC(payload, secret, validSig) {
					t.Fatalf("expected valid signature to verify successfully")
				}
			})
		})

		t.Run("When verifying an altered signature", func(t *testing.T) {
			invalidSig := "sha256=abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
			t.Run("Then verification fails", func(t *testing.T) {
				if domain.VerifyGitHubHMAC(payload, secret, invalidSig) {
					t.Fatalf("expected invalid signature to fail verification")
				}
			})
		})

		t.Run("When signature header has no sha256 prefix", func(t *testing.T) {
			malformedSig := "badprefix_12345"
			t.Run("Then verification fails", func(t *testing.T) {
				if domain.VerifyGitHubHMAC(payload, secret, malformedSig) {
					t.Fatalf("expected malformed signature to fail verification")
				}
			})
		})

		t.Run("When no secret is configured", func(t *testing.T) {
			t.Run("Then verification permits the payload", func(t *testing.T) {
				if !domain.VerifyGitHubHMAC(payload, "", "") {
					t.Fatalf("expected empty secret to permit payload")
				}
			})
		})
	})
}

func TestParseGitHubPushPayload(t *testing.T) {
	t.Run("Given a valid GitHub push payload JSON", func(t *testing.T) {
		raw := []byte(`{
			"ref": "refs/heads/release/v1",
			"repository": {
				"id": 12345,
				"name": "my-worker",
				"full_name": "ishaf/my-worker",
				"clone_url": "https://github.com/ishaf/my-worker.git"
			},
			"head_commit": {
				"id": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4",
				"message": "feat: update worker logic",
				"author": {
					"name": "Developer"
				}
			}
		}`)

		t.Run("When parsing the payload", func(t *testing.T) {
			event, err := domain.ParseGitHubPushPayload(raw)

			t.Run("Then it returns a valid GitHubPushEvent with branch name", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected parse error: %v", err)
				}
				if event.BranchName() != "release/v1" {
					t.Errorf("expected branch release/v1, got %s", event.BranchName())
				}
				if event.Repository.FullName != "ishaf/my-worker" {
					t.Errorf("expected full_name ishaf/my-worker, got %s", event.Repository.FullName)
				}
				if event.HeadCommit == nil || event.HeadCommit.ID != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4" {
					t.Errorf("expected commit id e3b0c..., got %+v", event.HeadCommit)
				}
				if event.HeadCommit.Message != "feat: update worker logic" {
					t.Errorf("expected commit message 'feat: update worker logic', got %s", event.HeadCommit.Message)
				}
			})
		})
	})
}
