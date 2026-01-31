package storehelper

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/tool/httprunner"
	"github.com/flexigpt/flexigpt-app/internal/tool/spec"
)

// ValidateTool performs structural validation of a Tool object.
func ValidateTool(t *spec.Tool) error {
	if t == nil {
		return errors.New("tool is nil")
	}
	if t.SchemaVersion != spec.SchemaVersion {
		return fmt.Errorf(
			"schemaVersion %q does not match expected %q",
			t.SchemaVersion,
			spec.SchemaVersion,
		)
	}
	if err := bundleitemutils.ValidateItemSlug(t.Slug); err != nil {
		return fmt.Errorf("invalid slug %s: %w", t.Slug, err)
	}
	if err := bundleitemutils.ValidateItemVersion(t.Version); err != nil {
		return fmt.Errorf("invalid version %s: %w", t.Version, err)
	}
	if strings.TrimSpace(t.DisplayName) == "" {
		return errors.New("displayName is empty")
	}
	if strings.TrimSpace(string(t.ID)) == "" {
		return errors.New("id is empty")
	}
	if t.ArgSchema == nil || !json.Valid(t.ArgSchema) {
		return errors.New("argSchema is missing or invalid")
	}

	if t.UserArgSchema != nil && !json.Valid(t.UserArgSchema) {
		return errors.New("userArgSchema is invalid JSON")
	}

	// LLMToolType sanity.
	switch t.LLMToolType {
	case spec.ToolStoreChoiceTypeFunction, spec.ToolStoreChoiceTypeCustom, spec.ToolStoreChoiceTypeWebSearch:
		// Ok.
	default:
		return fmt.Errorf("invalid llmToolType %q", t.LLMToolType)
	}

	if t.CreatedAt.IsZero() {
		return errors.New("createdAt is zero")
	}
	if t.ModifiedAt.IsZero() {
		return errors.New("modifiedAt is zero")
	}

	// Type / implementation sanity.
	switch t.Type {
	case spec.ToolTypeGo:
		if t.GoImpl == nil || strings.TrimSpace(t.GoImpl.Func) == "" {
			return errors.New("goImpl.func is required for type 'go'")
		}
		if t.HTTPImpl != nil {
			return errors.New("httpImpl must be unset for type 'go'")
		}
		if t.SDKImpl != nil {
			return errors.New("sdkImpl must be unset for type 'go'")
		}
	case spec.ToolTypeHTTP:
		if t.HTTPImpl == nil {
			return errors.New("httpImpl is required for type 'http'")
		}
		if t.GoImpl != nil {
			return errors.New("goImpl must be unset for type 'http'")
		}
		if t.SDKImpl != nil {
			return errors.New("sdkImpl must be unset for type 'http'")
		}
		if err := httprunner.ValidateHTTPImpl(t.HTTPImpl); err != nil {
			return errors.New("invalid implementation for type 'http'")
		}
	case spec.ToolTypeSDK:
		// SDK-backed tools are surfaced to the provider SDK as
		// server tools; they are not invoked via ToolStore.
		if t.GoImpl != nil {
			return errors.New("goImpl must be unset for type 'sdk'")
		}
		if t.HTTPImpl != nil {
			return errors.New("httpImpl must be unset for type 'sdk'")
		}
		if t.SDKImpl == nil {
			return errors.New("sdk metadata is required for type 'sdk'")
		}
		if strings.TrimSpace(t.SDKImpl.SDKType) == "" {
			return errors.New("sdk.sdkType is required for type 'sdk'")
		}
	default:
		return fmt.Errorf("invalid type %q", t.Type)
	}

	if err := bundleitemutils.ValidateTags(t.Tags); err != nil {
		return err
	}
	return nil
}
