package artifactbuiltin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/builtin"
)

const embeddedRegistryFile = "mcp_artifact_registry.json"

// LoadEmbeddedRegistry loads only the source-controlled converted MCP
// registry. It intentionally does not inspect or convert legacy runtime data,
// overlays, secrets, or user state.
func LoadEmbeddedRegistry() (Registry, fs.FS, error) {
	packages, err := builtin.EmbeddedMCPArtifactPackages()
	if err != nil {
		return Registry{}, nil, err
	}

	raw, err := fs.ReadFile(packages, embeddedRegistryFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Registry{}, nil, fmt.Errorf(
				"converted built-in MCP registry %q is required",
				embeddedRegistryFile,
			)
		}
		return Registry{}, nil, err
	}

	// The converted registry is intentionally mandatory. Normal startup never
	// falls back to the legacy embedded MCP document tree.
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	var registry Registry
	if err := decoder.Decode(&registry); err != nil {
		return Registry{}, nil, fmt.Errorf(
			"decode converted built-in MCP registry: %w",
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("converted built-in MCP registry has trailing JSON")
		}
		return Registry{}, nil, err
	}
	if err := registry.Validate(); err != nil {
		return Registry{}, nil, err
	}

	return registry, packages, nil
}
