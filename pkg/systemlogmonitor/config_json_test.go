/*
Copyright 2026 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package systemlogmonitor

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// canonicalJSON reformats src into the canonical format for shipped configs: 2-space indent and a
// trailing newline. json.Indent only normalizes whitespace, so key order and values are preserved.
func canonicalJSON(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, bytes.TrimSpace(src), "", "  "); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// TestShippedConfigs verifies that every JSON file shipped in the config/ directory parses with
// encoding/json, the same way the monitors load them at startup, and that it is written in the
// canonical format. The parse check catches trailing commas, UTF-8 BOMs and other syntax errors
// that would make NPD fail to start with the config. The format check keeps diffs in PRs minimal.
// Run `make fmt-configs` to rewrite nonconforming files in the canonical format.
func TestShippedConfigs(t *testing.T) {
	configDir := filepath.Join("..", "..", "config")
	// Match shipped configs at the top level and one directory down (e.g. config/exporter,
	// config/guestosconfig); both are loaded via encoding/json and can crash NPD if malformed.
	var configs []string
	for _, pattern := range []string{"*.json", filepath.Join("*", "*.json")} {
		matches, err := filepath.Glob(filepath.Join(configDir, pattern))
		if err != nil {
			t.Fatalf("failed to glob config files: %v", err)
		}
		configs = append(configs, matches...)
	}
	if len(configs) == 0 {
		t.Fatal("no config files found in config/ directory")
	}
	for _, config := range configs {
		t.Run(filepath.Base(config), func(t *testing.T) {
			data, err := os.ReadFile(config)
			if err != nil {
				t.Fatalf("failed to read %q: %v", config, err)
			}
			var parsed interface{}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("failed to parse %q: %v", config, err)
			}
			canonical, err := canonicalJSON(data)
			if err != nil {
				t.Fatalf("failed to reformat %q: %v", config, err)
			}
			if bytes.Equal(data, canonical) {
				return
			}
			if os.Getenv("UPDATE_EXPECTED") != "" {
				if err := os.WriteFile(config, canonical, 0o644); err != nil {
					t.Fatalf("failed to rewrite %q: %v", config, err)
				}
				t.Logf("rewrote %q in the canonical format", config)
				return
			}
			assert.Equal(t, string(canonical), string(data),
				"%q is not in the canonical format (2-space indent, trailing newline); "+
					"run `make fmt-configs` to rewrite it",
				config)
		})
	}
}
