package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ishaf/cubit/internal/domain"
)

// -----------------------------------------------------------------------------
// State Data Import Tooling (KV, D1, R2) - PRD Feature 5.2
// -----------------------------------------------------------------------------

// KVBulkItem represents a single key-value record from wrangler kv bulk get or JSON dumps.
type KVBulkItem struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Value         string `json:"value"`
	Base64        bool   `json:"base64"`
	ExpirationTTL int    `json:"expiration_ttl"`
	Metadata      any    `json:"metadata"`
}

// KVImportResult summarizes the bulk KV import operation.
type KVImportResult struct {
	NamespaceID string `json:"namespace_id"`
	Total       int    `json:"total"`
	Imported    int    `json:"imported"`
}

// D1ImportResult summarizes SQL script execution against a D1 database.
type D1ImportResult struct {
	DatabaseID string  `json:"database_id"`
	Executed   int     `json:"executed"`
	DurationMS float64 `json:"duration_ms"`
}

// R2ImportObject represents a single object for batch R2 ingestion.
type R2ImportObject struct {
	Key         string `json:"key"`
	Content     string `json:"content"`
	Base64      bool   `json:"base64"`
	ContentType string `json:"content_type"`
}

// R2ImportRequest encapsulates a batch import payload for an R2 bucket.
type R2ImportRequest struct {
	Bucket  string           `json:"bucket"`
	Objects []R2ImportObject `json:"objects"`
}

// R2ImportResult summarizes the R2 object import operation.
type R2ImportResult struct {
	Bucket   string `json:"bucket"`
	Imported int    `json:"imported"`
}

// ParseKVBulk extracts KV items from raw bytes, supporting JSON arrays, wrapped objects, or map dumps.
func ParseKVBulk(data []byte) ([]KVBulkItem, error) {
	var items []KVBulkItem
	if err := json.Unmarshal(data, &items); err == nil {
		return items, nil
	}

	var wrapper struct {
		Items   []KVBulkItem `json:"items"`
		Entries []KVBulkItem `json:"entries"`
		Data    []KVBulkItem `json:"data"`
	}
	if errWrap := json.Unmarshal(data, &wrapper); errWrap == nil && (len(wrapper.Items) > 0 || len(wrapper.Entries) > 0 || len(wrapper.Data) > 0) {
		if len(wrapper.Items) > 0 {
			return wrapper.Items, nil
		}
		if len(wrapper.Entries) > 0 {
			return wrapper.Entries, nil
		}
		return wrapper.Data, nil
	}

	// Fallback: try parsing as key-value map
	var objMap map[string]interface{}
	if errMap := json.Unmarshal(data, &objMap); errMap == nil {
		for k, v := range objMap {
			items = append(items, KVBulkItem{
				Key:   k,
				Value: fmt.Sprintf("%v", v),
			})
		}
		return items, nil
	}

	return nil, domain.NewValidationError("invalid KV bulk import payload format")
}

// ImportKVBulk ingests key-value data exported from wrangler kv bulk get or JSON into a KV namespace.
func (u *ServicesService) ImportKVBulk(ctx context.Context, namespaceIDOrName string, data []byte) (*KVImportResult, error) {
	namespaceID := namespaceIDOrName
	if ns, err := u.repo.GetKVNamespace(ctx, namespaceIDOrName); err == nil {
		namespaceID = ns.ID
	} else {
		// If namespace does not exist, auto-create it
		if created, errCreate := u.CreateKVNamespace(ctx, namespaceIDOrName); errCreate == nil {
			namespaceID = created.ID
		}
	}

	items, err := ParseKVBulk(data)
	if err != nil {
		return nil, err
	}

	var entries []*domain.KVPair
	for _, item := range items {
		key := item.Key
		if key == "" {
			key = item.Name
		}
		if key == "" {
			continue
		}

		val := item.Value
		if item.Base64 {
			decoded, err := base64.StdEncoding.DecodeString(val)
			if err != nil {
				return nil, domain.NewValidationError(fmt.Sprintf("invalid base64 encoding for key %q: %v", key, err))
			}
			val = string(decoded)
		}

		metaStr := ""
		if item.Metadata != nil {
			if s, ok := item.Metadata.(string); ok {
				metaStr = s
			} else {
				if b, err := json.Marshal(item.Metadata); err == nil {
					metaStr = string(b)
				}
			}
		}

		pair, err := domain.NewKVPair(namespaceID, key, val, item.ExpirationTTL, metaStr)
		if err != nil {
			return nil, fmt.Errorf("failed creating kv pair for key %q: %w", key, err)
		}
		entries = append(entries, pair)
	}

	imported, err := u.repo.BulkInsertKVEntries(ctx, namespaceID, entries)
	if err != nil {
		return nil, fmt.Errorf("failed bulk inserting kv entries: %w", err)
	}

	return &KVImportResult{
		NamespaceID: namespaceID,
		Total:       len(items),
		Imported:    imported,
	}, nil
}

// ImportKVFromFile reads a wrangler kv bulk export JSON file from disk and imports it.
func (u *ServicesService) ImportKVFromFile(ctx context.Context, namespaceIDOrName, filePath string) (*KVImportResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed reading KV export file %q: %w", filePath, err)
	}
	return u.ImportKVBulk(ctx, namespaceIDOrName, data)
}

// ImportD1SQL imports and executes a wrangler d1 export SQL dump against a D1 database.
func (u *ServicesService) ImportD1SQL(ctx context.Context, databaseIDOrName, sqlScript string) (*D1ImportResult, error) {
	databaseID := databaseIDOrName
	if d1, err := u.repo.GetD1Database(ctx, databaseIDOrName); err == nil {
		databaseID = d1.ID
	} else {
		// Auto-create database if not found
		if created, errCreate := u.CreateD1Database(ctx, databaseIDOrName); errCreate == nil {
			databaseID = created.ID
		}
	}

	start := time.Now()
	executed, err := u.repo.ExecuteD1Script(ctx, databaseID, sqlScript)
	if err != nil {
		return nil, fmt.Errorf("failed executing d1 script: %w", err)
	}
	duration := time.Since(start).Seconds() * 1000

	return &D1ImportResult{
		DatabaseID: databaseID,
		Executed:   executed,
		DurationMS: duration,
	}, nil
}

// ImportD1FromFile reads a wrangler d1 export SQL dump file from disk and executes it.
func (u *ServicesService) ImportD1FromFile(ctx context.Context, databaseIDOrName, filePath string) (*D1ImportResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed reading D1 SQL export file %q: %w", filePath, err)
	}
	return u.ImportD1SQL(ctx, databaseIDOrName, string(data))
}

// ImportR2Objects imports a batch of objects into an R2 bucket.
func (u *ServicesService) ImportR2Objects(ctx context.Context, bucketName string, objects []R2ImportObject) (*R2ImportResult, error) {
	bucketName = strings.TrimSpace(bucketName)
	if bucketName == "" {
		return nil, domain.NewValidationError("bucket name must not be empty")
	}

	// Ensure bucket exists
	if _, err := u.CreateR2Bucket(ctx, bucketName); err != nil && !errors.Is(err, domain.ErrConflict) {
		return nil, fmt.Errorf("failed ensuring bucket %q exists: %w", bucketName, err)
	}

	imported := 0
	for _, obj := range objects {
		if obj.Key == "" {
			continue
		}

		var data []byte
		if obj.Base64 {
			decoded, err := base64.StdEncoding.DecodeString(obj.Content)
			if err != nil {
				return nil, domain.NewValidationError(fmt.Sprintf("invalid base64 encoding for object key %q: %v", obj.Key, err))
			}
			data = decoded
		} else {
			data = []byte(obj.Content)
		}

		if err := u.UploadR2Object(ctx, bucketName, obj.Key, data); err != nil {
			return nil, fmt.Errorf("failed importing object %q: %w", obj.Key, err)
		}
		imported++
	}

	return &R2ImportResult{
		Bucket:   bucketName,
		Imported: imported,
	}, nil
}

// ImportR2FromDirectory recursively reads files in dirPath and imports them into bucketName.
func (u *ServicesService) ImportR2FromDirectory(ctx context.Context, bucketName, dirPath string) (*R2ImportResult, error) {
	var objects []R2ImportObject
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relKey, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		objects = append(objects, R2ImportObject{
			Key:     filepath.ToSlash(relKey),
			Content: string(data),
			Base64:  false,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed walking directory %q: %w", dirPath, err)
	}
	return u.ImportR2Objects(ctx, bucketName, objects)
}
