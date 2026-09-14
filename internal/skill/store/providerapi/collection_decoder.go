package providerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/collectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

// CollectionDecoder adapts a physical Skill Collection manifest into one
// canonical collection Artifact and source-backed Skill Artifacts.
//
// The source manifest accepts an ordered skills array of path locators, or a
// map whose values are path locators or objects containing locator. Each path
// identifies either SKILL.md or its containing Skill directory.
type CollectionDecoder struct{}

type skillCollectionManifest struct {
	Kind          string          `json:"kind"`
	SchemaID      string          `json:"schemaID,omitempty"`
	SchemaVersion string          `json:"schemaVersion,omitempty"`
	Name          string          `json:"name,omitempty"`
	Description   string          `json:"description,omitempty"`
	Version       string          `json:"version,omitempty"`
	Skills        json.RawMessage `json:"skills,omitempty"`
}

func NewCollectionDecoder() *CollectionDecoder {
	return &CollectionDecoder{}
}

func (*CollectionDecoder) ID() basespec.DecoderID {
	return skillDomain.SkillCollectionDecoderID
}

func (*CollectionDecoder) Revision() string {
	return "agent.skill-collection/v1"
}

func (*CollectionDecoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	_, recognized, err := parseSkillCollectionManifest(candidate.Content)
	if err != nil || !recognized {
		return providerapi.RecognitionNone
	}
	return providerapi.RecognitionPreferred
}

func (d *CollectionDecoder) Decode(
	ctx context.Context,
	candidate providerapi.Candidate,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
	return d.decode(ctx, candidate, nil)
}

func (d *CollectionDecoder) DecodeWithSource(
	ctx context.Context,
	candidate providerapi.Candidate,
	reader providerapi.SourceEntryReader,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
	return d.decode(ctx, candidate, reader)
}

func (*CollectionDecoder) decode(
	ctx context.Context,
	candidate providerapi.Candidate,
	reader providerapi.SourceEntryReader,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
	manifest, recognized, err := parseSkillCollectionManifest(candidate.Content)
	if err != nil {
		return nil, skillCollectionDiagnostics(candidate.Locator, "", err)
	}
	if !recognized {
		return nil, nil
	}
	if reader == nil {
		return nil, skillCollectionDiagnostics(
			candidate.Locator,
			"",
			fmt.Errorf(
				"%w: Skill Collection decoder requires bounded source reads",
				basespec.ErrInvalid,
			),
		)
	}

	rawMembers, err := skillCollectionMembers(manifest.Skills)
	if err != nil {
		return nil, skillCollectionDiagnostics(candidate.Locator, "", err)
	}

	members := make([]declaration.Entry, 0, len(rawMembers))
	skillOutputs := make([]providerapi.Decoded, 0, len(rawMembers))
	for index, rawMember := range rawMembers {
		subresource := basespec.SubresourceLocator(
			"skills/" + strconv.Itoa(index),
		)
		locator, err := decodeSkillCollectionMember(rawMember)
		if err != nil {
			skillOutputs = append(skillOutputs, providerapi.Decoded{
				SubresourceLocator: subresource,
				Diagnostics: skillCollectionDiagnostics(
					candidate.Locator,
					subresource,
					err,
				),
			})
			continue
		}

		documentLocator, err := skillCollectionDocumentLocator(
			locator,
			candidate.Locator,
		)
		if err != nil {
			skillOutputs = append(skillOutputs, providerapi.Decoded{
				SubresourceLocator: subresource,
				Diagnostics: skillCollectionDiagnostics(
					candidate.Locator,
					subresource,
					err,
				),
			})
			continue
		}

		sourceContent, err := reader.ReadSourceEntry(
			ctx,
			documentLocator,
		)
		if err != nil {
			skillOutputs = append(skillOutputs, providerapi.Decoded{
				OriginLocator: documentLocator,
				Diagnostics: skillCollectionDiagnostics(
					documentLocator,
					"",
					err,
				),
			})
			continue
		}

		definitionValue, warnings, err := skillDomain.DecodeSkillDocument(
			sourceContent.Content,
			expectedSkillName(sourceContent.Locator),
		)
		if err != nil {
			digest := sourceContent.Digest
			skillOutputs = append(skillOutputs, providerapi.Decoded{
				OriginLocator:       sourceContent.Locator,
				OriginContentDigest: &digest,
				Diagnostics: skillCollectionDiagnostics(
					sourceContent.Locator,
					"",
					err,
				),
			})
			continue
		}
		for warningIndex := range warnings {
			warnings[warningIndex].Location = &diagnostic.Location{
				Locator: sourceContent.Locator,
			}
		}

		member, err := declaration.NewSymbolicEntry(
			declaration.TypeSkill,
			definitionValue.LogicalName,
		)
		if err != nil {
			return nil, skillCollectionDiagnostics(
				candidate.Locator,
				"",
				err,
			)
		}
		members = append(members, member)

		digest := sourceContent.Digest
		skillOutputs = append(skillOutputs, providerapi.Decoded{
			OriginLocator:       sourceContent.Locator,
			OriginContentDigest: &digest,
			Definition:          definitionValue,
			Diagnostics:         warnings,
		})
	}

	collectionName := basespec.LogicalName(manifest.Name)
	if collectionName == "" {
		collectionName, err = declaration.DeriveLogicalName(
			"skill-collection",
			candidate.Locator,
		)
		if err != nil {
			return nil, skillCollectionDiagnostics(
				candidate.Locator,
				"",
				err,
			)
		}
	}

	collection := collectionv1.CollectionDocument{
		APIVersion:  collectionv1.CollectionSchemaVersion,
		Type:        collectionv1.CollectionType,
		Name:        string(collectionName),
		Description: manifest.Description,
		Version:     manifest.Version,
		Members:     members,
	}
	entry, err := declaration.NewEntry(collection)
	if err != nil {
		return nil, skillCollectionDiagnostics(candidate.Locator, "", err)
	}
	collectionDefinition, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return nil, skillCollectionDiagnostics(candidate.Locator, "", err)
	}

	output := make([]providerapi.Decoded, 0, 1+len(skillOutputs))
	output = append(output, providerapi.Decoded{
		SubresourceLocator: "collection",
		Definition:         collectionDefinition,
	})
	output = append(output, skillOutputs...)
	return output, nil
}

func parseSkillCollectionManifest(
	content []byte,
) (skillCollectionManifest, bool, error) {
	raw, err := canonicalSkillCollectionJSON(content)
	if err != nil {
		return skillCollectionManifest{}, false, err
	}
	var header struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return skillCollectionManifest{}, false, err
	}
	if header.Kind != "skill.collection" && header.Kind != "skill.bundle" {
		return skillCollectionManifest{}, false, nil
	}

	var value skillCollectionManifest
	if err := json.Unmarshal(raw, &value); err != nil {
		return skillCollectionManifest{}, false, err
	}
	return value, true, nil
}

func canonicalSkillCollectionJSON(content []byte) ([]byte, error) {
	raw, err := jsonutil.CanonicalizeObject(
		content,
		basespec.MaxDefinitionBytes,
	)
	if err == nil {
		return raw, nil
	}
	return yamlutil.CanonicalObjectJSON(
		content,
		basespec.MaxDefinitionBytes,
	)
}

func skillCollectionMembers(
	raw json.RawMessage,
) ([]json.RawMessage, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err == nil {
		return values, nil
	}

	var keyed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keyed); err != nil {
		return nil, fmt.Errorf(
			"%w: Skill Collection skills must be an array or object",
			basespec.ErrInvalid,
		)
	}
	if _, found := keyed["locator"]; found {
		return []json.RawMessage{append(json.RawMessage(nil), raw...)}, nil
	}
	if _, found := keyed["kind"]; found {
		return []json.RawMessage{append(json.RawMessage(nil), raw...)}, nil
	}

	keys := make([]string, 0, len(keyed))
	for key := range keyed {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	output := make([]json.RawMessage, 0, len(keys))
	for _, key := range keys {
		output = append(
			output,
			append(json.RawMessage(nil), keyed[key]...),
		)
	}
	return output, nil
}

func decodeSkillCollectionMember(
	raw json.RawMessage,
) (declaration.Locator, error) {
	var direct declaration.Locator
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct, nil
	}

	var descriptor struct {
		Locator json.RawMessage `json:"locator"`
		Path    string          `json:"path"`
	}
	if err := json.Unmarshal(raw, &descriptor); err != nil {
		return declaration.Locator{}, err
	}
	if len(descriptor.Locator) != 0 {
		if err := json.Unmarshal(descriptor.Locator, &direct); err != nil {
			return declaration.Locator{}, err
		}
		return direct, nil
	}
	if descriptor.Path != "" {
		direct = declaration.ScalarLocator(descriptor.Path)
		if err := direct.Validate(); err != nil {
			return declaration.Locator{}, err
		}
		return direct, nil
	}
	return declaration.Locator{}, fmt.Errorf(
		"%w: Skill Collection member requires locator",
		basespec.ErrInvalid,
	)
}

func skillCollectionDocumentLocator(
	locator declaration.Locator,
	declarationLocator basespec.Locator,
) (basespec.Locator, error) {
	target, err := declaration.ResolveSourceRelativePathLocator(
		locator,
		declarationLocator,
	)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(
		path.Base(string(target)),
		string(skillDomain.SkillDefinitionFileName),
	) {
		target = basespec.Locator(path.Join(
			string(target),
			string(skillDomain.SkillDefinitionFileName),
		))
	}
	if err := target.Validate(false); err != nil {
		return "", err
	}
	return target, nil
}

func expectedSkillName(locator basespec.Locator) string {
	parent := path.Dir(string(locator))
	if parent == "." {
		return ""
	}
	return path.Base(parent)
}

func skillCollectionDiagnostics(
	locator basespec.Locator,
	subresource basespec.SubresourceLocator,
	err error,
) []diagnostic.Diagnostic {
	location := &diagnostic.Location{Locator: locator}
	if subresource != "" {
		location.SubresourceLocator = subresource
	}
	return []diagnostic.Diagnostic{{
		Severity: diagnostic.SeverityError,
		Code:     "agent.skill-collection.invalid",
		Message:  diagnostic.BoundedMessage(err.Error()),
		Location: location,
	}}
}
