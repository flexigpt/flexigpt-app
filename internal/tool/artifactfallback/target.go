package artifactfallback

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
)

const (
	// MappedTargetProviderV1 identifies ToolStore-backed mapped targets.
	MappedTargetProviderV1 = "flexigpt.tool.artifactfallback.v1"

	targetIdentifierVersion = "v1"
)

// TargetV1 is the stable identity of one ToolStore Tool version.
type TargetV1 struct {
	BundleID    bundleitemutils.BundleID    `json:"bundleID"`
	ToolID      bundleitemutils.ItemID      `json:"toolID"`
	ToolSlug    bundleitemutils.ItemSlug    `json:"toolSlug"`
	ToolVersion bundleitemutils.ItemVersion `json:"toolVersion"`
}

func (t TargetV1) Validate() error {
	if err := basespec.ValidateRequiredText(
		"Tool fallback bundle ID",
		string(t.BundleID),
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"Tool fallback Tool ID",
		string(t.ToolID),
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := bundleitemutils.ValidateItemSlug(t.ToolSlug); err != nil {
		return err
	}
	if err := bundleitemutils.ValidateItemVersion(t.ToolVersion); err != nil {
		return err
	}
	if err := basespec.LogicalName(t.ToolSlug).Validate(); err != nil {
		return err
	}
	return nil
}

// NewMappedTarget creates a Tool mapped target for one exact ToolStore Tool.
func NewMappedTarget(
	name basespec.LogicalName,
	value TargetV1,
) (resolve.MappedTarget, error) {
	if err := name.Validate(); err != nil {
		return resolve.MappedTarget{}, err
	}
	if err := value.Validate(); err != nil {
		return resolve.MappedTarget{}, err
	}
	if string(name) != string(value.ToolSlug) {
		return resolve.MappedTarget{}, fmt.Errorf(
			"%w: Tool mapped target name %q does not match Tool slug %q",
			basespec.ErrInvalid,
			name,
			value.ToolSlug,
		)
	}

	identifier, err := resolve.EncodeMappedIdentifier(
		targetIdentifierVersion,
		value,
	)
	if err != nil {
		return resolve.MappedTarget{}, err
	}

	return resolve.MappedTarget{
		Provider:   MappedTargetProviderV1,
		Identifier: identifier,
		Type:       declaration.TypeTool,
		Name:       name,
		Builtin:    true,
	}, nil
}

// DecodeTarget decodes and validates one Tool fallback target without reading
// ToolStore. Store-backed callers should use Service.ResolveTarget.
func DecodeTarget(
	target resolve.MappedTarget,
) (TargetV1, error) {
	if err := target.Validate(); err != nil {
		return TargetV1{}, err
	}
	if target.Provider != MappedTargetProviderV1 {
		return TargetV1{}, fmt.Errorf(
			"%w: unsupported Tool mapped target provider %q",
			basespec.ErrUnsupported,
			target.Provider,
		)
	}
	if target.Type != declaration.TypeTool {
		return TargetV1{}, fmt.Errorf(
			"%w: mapped target type is %q, expected Tool",
			basespec.ErrInvalid,
			target.Type,
		)
	}
	if !target.Builtin {
		return TargetV1{}, fmt.Errorf(
			"%w: Tool mapped target is not built-in",
			basespec.ErrInvalid,
		)
	}

	value, err := resolve.DecodeMappedIdentifier[TargetV1](
		target.Identifier,
		targetIdentifierVersion,
	)
	if err != nil {
		return TargetV1{}, err
	}
	if err := value.Validate(); err != nil {
		return TargetV1{}, err
	}
	if string(target.Name) != string(value.ToolSlug) {
		return TargetV1{}, fmt.Errorf(
			"%w: Tool mapped target name %q does not match payload slug %q",
			basespec.ErrInvalid,
			target.Name,
			value.ToolSlug,
		)
	}
	return value, nil
}
