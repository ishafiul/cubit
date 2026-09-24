package wrangler

import (
	"bytes"
	"encoding/json"
	"fmt"

	toml "github.com/pelletier/go-toml/v2"
)

// TranspileTOMLToJSONC converts raw wrangler.toml bytes into valid wrangler.jsonc
// adhering to the celld configuration schema.
func TranspileTOMLToJSONC(tomlData []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(tomlData)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("wrangler.toml is empty")
	}

	var cfg WranglerConfig
	if err := toml.Unmarshal(trimmed, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse wrangler.toml: %w", err)
	}

	// Ensure no nested environment blocks leak under environments
	for envName, envCfg := range cfg.Env {
		if envCfg.Env != nil {
			envCfg.Env = nil
			cfg.Env[envName] = envCfg
		}
	}

	formatted, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize to jsonc: %w", err)
	}

	return formatted, nil
}
