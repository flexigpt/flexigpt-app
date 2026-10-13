package spec

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/inference-go/capabilityoverride"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

const maximumProviderConfigurationTextBytes = 16 << 10

// RuntimeConfiguration contains only provider-ready execution configuration.
//
// It has no Artifact, Model selection, request-layering, Session, Agent,
// Workspace, Tool, Skill, MCP, conversation, or attachment semantics.
//
// Credentials and private provider configuration must never be serialized.
type RuntimeConfiguration struct {
	ProviderParam       inferenceSpec.ProviderParam                     `json:"-"`
	ModelParam          inferenceSpec.ModelParam                        `json:"-"`
	CapabilityOverrides []*capabilityoverride.ModelCapabilitiesOverride `json:"-"`
	Fingerprint         string                                          `json:"-"`
}

// Validate establishes the provider execution boundary once. Registration and
// capability-resolution helpers consume this checked configuration.
func (v RuntimeConfiguration) Validate() error {
	required := []struct {
		name  string
		value string
	}{
		{"provider name", string(v.ProviderParam.Name)},
		{"provider SDK type", string(v.ProviderParam.SDKType)},
		{"provider origin", v.ProviderParam.Origin},
		{"provider completion path", v.ProviderParam.ChatCompletionPathPrefix},
		{"provider API key header", v.ProviderParam.APIKeyHeaderKey},
		{"model name", string(v.ModelParam.Name)},
	}

	for _, field := range required {
		if strings.TrimSpace(field.value) == "" ||
			!utf8.ValidString(field.value) ||
			len(field.value) > maximumProviderConfigurationTextBytes ||
			strings.ContainsRune(field.value, 0) {
			return fmt.Errorf("invalid prepared %s", field.name)
		}
	}

	raw, found := strings.CutPrefix(v.Fingerprint, "sha256:")
	if !found || len(raw) != 64 {
		return errors.New("invalid prepared provider configuration fingerprint")
	}
	if _, err := hex.DecodeString(raw); err != nil {
		return fmt.Errorf("invalid prepared provider configuration fingerprint: %w", err)
	}

	for name, value := range v.ProviderParam.DefaultHeaders {
		if strings.TrimSpace(name) == "" ||
			strings.ContainsAny(name, "\r\n\x00") ||
			strings.ContainsAny(value, "\r\n\x00") {
			return errors.New("invalid prepared provider header")
		}
	}

	return nil
}
