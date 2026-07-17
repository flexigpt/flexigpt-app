package workspace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

const (
	maxWorkspaceDiscoveryEntries = 256
	maxTrustReferenceBytes       = 512
)

var workspaceUUIDv7RE = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

type rootKindHook struct{}

func (rootKindHook) Kind() artifactstoreSpec.RootKind {
	return RootKind
}

func (rootKindHook) ValidateRootData(
	_ context.Context,
	root artifactstoreSpec.ArtifactRoot,
) []artifactstoreSpec.Diagnostic {
	if root.DataSchemaID != RootDataSchemaID {
		return workspaceDiagnostics(
			"workspace.root.schema",
			fmt.Sprintf("workspace root data schema must be %q", RootDataSchemaID),
		)
	}
	data, err := decodeRootData(root.Data)
	if err != nil {
		return workspaceDiagnostics("workspace.root.data", err.Error())
	}
	if err := validateRootData(data); err != nil {
		return workspaceDiagnostics("workspace.root.data", err.Error())
	}
	return nil
}

func (rootKindHook) ValidateSourceAttachment(
	_ context.Context,
	root artifactstoreSpec.ArtifactRoot,
	attachment artifactstoreSpec.RootSourceAttachment,
) []artifactstoreSpec.Diagnostic {
	data, err := decodeRootData(root.Data)
	if err != nil {
		return workspaceDiagnostics("workspace.attachment.root-data", err.Error())
	}
	if err := validateRootData(data); err != nil {
		return workspaceDiagnostics("workspace.attachment.root-data", err.Error())
	}
	switch attachment.Role {
	case RolePrimary, RoleAttachedPackage, RoleBuiltIn, RoleAppLibrary, RoleOverlay:
	default:
		return workspaceDiagnostics(
			"workspace.attachment.role",
			fmt.Sprintf("unsupported workspace attachment role %q", attachment.Role),
		)
	}
	if attachment.Role == RolePrimary {
		if !attachment.Enabled {
			return workspaceDiagnostics(
				"workspace.attachment.primary",
				"the primary workspace attachment must be enabled",
			)
		}
		if data.Mode != RootModeFilesystem || data.PrimarySourceID != attachment.SourceID {
			return workspaceDiagnostics(
				"workspace.attachment.primary",
				"primary attachment must match the filesystem workspace primarySourceID",
			)
		}
	} else if data.PrimarySourceID != "" && data.PrimarySourceID == attachment.SourceID {
		return workspaceDiagnostics(
			"workspace.attachment.primary",
			"the configured primary source must use the primary role",
		)
	}

	if len(bytes.TrimSpace(attachment.Data)) == 0 ||
		bytes.Equal(bytes.TrimSpace(attachment.Data), []byte("{}")) {
		if attachment.DataSchemaID != "" {
			return workspaceDiagnostics(
				"workspace.attachment.schema",
				"empty workspace attachment data must not declare a schema",
			)
		}
		return nil
	}
	if attachment.DataSchemaID != AttachmentDataSchemaID {
		return workspaceDiagnostics(
			"workspace.attachment.schema",
			fmt.Sprintf("workspace attachment data schema must be %q", AttachmentDataSchemaID),
		)
	}
	var typed AttachmentData
	if err := decodeStrictJSONObject(attachment.Data, &typed, true); err != nil {
		return workspaceDiagnostics("workspace.attachment.data", err.Error())
	}
	return nil
}

func (rootKindHook) ValidateSourceAttachmentSource(
	_ context.Context,
	_ artifactstoreSpec.ArtifactRoot,
	attachment artifactstoreSpec.RootSourceAttachment,
	source artifactstoreSpec.ArtifactSource,
) []artifactstoreSpec.Diagnostic {
	if attachment.Role != RolePrimary {
		return nil
	}
	if source.Kind != artifactstoreSpec.SourceKindFSDirectory {
		return workspaceDiagnostics(
			"workspace.attachment.primary-source",
			fmt.Sprintf(
				"primary workspace attachment source kind must be %q, got %q",
				artifactstoreSpec.SourceKindFSDirectory,
				source.Kind,
			),
		)
	}
	return nil
}

func (rootKindHook) ValidateSourceAttachments(
	_ context.Context,
	root artifactstoreSpec.ArtifactRoot,
	attachments []artifactstoreSpec.RootSourceAttachment,
) []artifactstoreSpec.Diagnostic {
	data, err := decodeRootData(root.Data)
	if err != nil {
		return workspaceDiagnostics("workspace.attachment-set.root-data", err.Error())
	}
	if err := validateRootData(data); err != nil {
		return workspaceDiagnostics("workspace.attachment-set.root-data", err.Error())
	}
	if err := validateWorkspaceAttachmentSet(data, attachments); err != nil {
		return workspaceDiagnostics("workspace.attachment-set", err.Error())
	}
	return nil
}

func validateWorkspaceAttachmentSet(
	data RootData,
	attachments []artifactstoreSpec.RootSourceAttachment,
) error {
	primaryCount := 0
	for _, attachment := range attachments {
		if attachment.Role == RolePrimary {
			primaryCount++
			if data.Mode != RootModeFilesystem {
				return errors.New("only filesystem workspaces may have a primary attachment")
			}
			if attachment.SourceID != data.PrimarySourceID {
				return errors.New("primary attachment does not match primarySourceID")
			}
			if !attachment.Enabled {
				return errors.New("primary attachment must be enabled")
			}
			continue
		}
		if attachment.SourceID == data.PrimarySourceID {
			return errors.New("primarySourceID must use the primary attachment role")
		}
	}
	switch data.Mode {
	case RootModeFilesystem:
		if primaryCount != 1 {
			return fmt.Errorf(
				"filesystem workspace requires exactly one primary attachment, found %d",
				primaryCount,
			)
		}
	case RootModeEmpty:
		if primaryCount != 0 {
			return errors.New("empty workspace must not have a primary attachment")
		}
	}
	return nil
}

type collectionKindHook struct{}

func (collectionKindHook) Kind() artifactstoreSpec.CollectionKind {
	return CollectionKind
}

func (collectionKindHook) ValidateCollectionData(
	_ context.Context,
	collection artifactstoreSpec.ArtifactCollection,
) []artifactstoreSpec.Diagnostic {
	if collection.DataSchemaID != CollectionDataSchemaID {
		return workspaceDiagnostics(
			"workspace.collection.schema",
			fmt.Sprintf("workspace collection data schema must be %q", CollectionDataSchemaID),
		)
	}
	var data CollectionData
	if err := decodeStrictJSONObject(collection.Data, &data, true); err != nil {
		return workspaceDiagnostics("workspace.collection.data", err.Error())
	}
	if strings.TrimSpace(string(data.ArtifactKind)) == "" {
		return workspaceDiagnostics(
			"workspace.collection.kind",
			"workspace collection artifactKind is required",
		)
	}
	return nil
}

func (collectionKindHook) ValidateRecordPlacement(
	_ context.Context,
	collection artifactstoreSpec.ArtifactCollection,
	record artifactstoreSpec.ArtifactRecord,
) []artifactstoreSpec.Diagnostic {
	var data CollectionData
	if err := decodeStrictJSONObject(collection.Data, &data, true); err != nil {
		return workspaceDiagnostics("workspace.collection.data", err.Error())
	}
	if data.ArtifactKind != record.Kind {
		return workspaceDiagnostics(
			"workspace.collection.placement",
			fmt.Sprintf(
				"record kind %q cannot be placed in collection for %q",
				record.Kind,
				data.ArtifactKind,
			),
		)
	}
	return nil
}

func decodeRootData(raw json.RawMessage) (RootData, error) {
	var data RootData
	if err := decodeStrictJSONObject(raw, &data, true); err != nil {
		return RootData{}, err
	}
	return data, nil
}

func validateRootData(data RootData) error {
	switch data.Mode {
	case RootModeEmpty:
		if data.PrimarySourceID != "" {
			return errors.New("empty workspace must not declare primarySourceID")
		}
	case RootModeFilesystem:
		if !workspaceUUIDv7RE.MatchString(string(data.PrimarySourceID)) {
			return errors.New("filesystem workspace primarySourceID must be a canonical UUIDv7")
		}
	default:
		return fmt.Errorf("unsupported workspace mode %q", data.Mode)
	}
	if data.CapabilityProfileVersion != CapabilityProfileVersion {
		return fmt.Errorf(
			"capabilityProfileVersion must be %q",
			CapabilityProfileVersion,
		)
	}
	if err := validateBoundedText(
		"rootTrustReference",
		data.RootTrustReference,
		maxTrustReferenceBytes,
		true,
	); err != nil {
		return err
	}
	if err := validateDiscoveryPreferences(data.DiscoveryPreferences); err != nil {
		return err
	}
	if err := validateBoundedText(
		"displayPreferences.defaultCategory",
		data.DisplayPreferences.DefaultCategory,
		128,
		true,
	); err != nil {
		return err
	}
	return nil
}

func validateDiscoveryPreferences(value DiscoveryPreferences) error {
	if len(value.AdditionalLocators) > maxWorkspaceDiscoveryEntries ||
		len(value.AdditionalRoots) > maxWorkspaceDiscoveryEntries {
		return errors.New("workspace discovery preferences contain too many entries")
	}
	seenLocators := make(map[artifactstoreSpec.SourceLocator]struct{}, len(value.AdditionalLocators))
	for _, locator := range value.AdditionalLocators {
		if err := validateWorkspaceLocator(locator, false); err != nil {
			return fmt.Errorf("additional locator %q: %w", locator, err)
		}
		if _, duplicate := seenLocators[locator]; duplicate {
			return fmt.Errorf("duplicate additional locator %q", locator)
		}
		seenLocators[locator] = struct{}{}
	}
	seenRoots := make(map[artifactstoreSpec.SourceLocator]struct{}, len(value.AdditionalRoots))
	for _, root := range value.AdditionalRoots {
		if err := validateWorkspaceLocator(root.Root, true); err != nil {
			return fmt.Errorf("additional root %q: %w", root.Root, err)
		}
		if _, duplicate := seenRoots[root.Root]; duplicate {
			return fmt.Errorf("duplicate additional root %q", root.Root)
		}
		seenRoots[root.Root] = struct{}{}
		for _, pattern := range root.IncludePatterns {
			if strings.TrimSpace(pattern) != pattern || pattern == "" {
				return errors.New("include pattern must be non-empty and trimmed")
			}
			if _, err := path.Match(pattern, "candidate"); err != nil {
				return fmt.Errorf("invalid include pattern %q: %w", pattern, err)
			}
		}
	}
	return nil
}

func validateWorkspaceLocator(locator artifactstoreSpec.SourceLocator, allowRoot bool) error {
	value := string(locator)
	if allowRoot && value == "." {
		return nil
	}
	if value == "" || !utf8.ValidString(value) ||
		strings.Contains(value, "\\") ||
		strings.Contains(value, ":") ||
		strings.ContainsRune(value, 0) ||
		path.IsAbs(value) ||
		path.Clean(value) != value ||
		value == "." ||
		value == ".." ||
		strings.HasPrefix(value, "../") {
		return errors.New("locator must be a normalized relative source path")
	}
	return nil
}

func validateBoundedText(label, value string, maximum int, optional bool) error {
	if value == "" && optional {
		return nil
	}
	if value == "" || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return fmt.Errorf("%s must be non-empty, valid UTF-8, and trimmed", label)
	}
	if len(value) > maximum {
		return fmt.Errorf("%s exceeds %d bytes", label, maximum)
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return fmt.Errorf("%s contains a control character", label)
		}
	}
	return nil
}

func decodeStrictJSONObject(raw []byte, target any, disallowUnknown bool) error {
	if err := validateJSONDocument(raw); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if disallowUnknown {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("JSON contains trailing values")
		}
		return err
	}
	return nil
}

func validateJSONDocument(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	first, err := decoder.Token()
	if err != nil {
		return err
	}
	if err := consumeJSONToken(decoder, first); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return errors.New("JSON contains trailing values")
		}
		return err
	}
	return nil
}

func consumeJSONToken(decoder *json.Decoder, token json.Token) error {
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]struct{}{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[key] = struct{}{}
			value, err := decoder.Token()
			if err != nil {
				return err
			}
			if err := consumeJSONToken(decoder, value); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return errors.New("invalid JSON object terminator")
		}
	case '[':
		for decoder.More() {
			value, err := decoder.Token()
			if err != nil {
				return err
			}
			if err := consumeJSONToken(decoder, value); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return errors.New("invalid JSON array terminator")
		}
	default:
		return errors.New("invalid JSON delimiter")
	}
	return nil
}

func workspaceDiagnostics(code, message string) []artifactstoreSpec.Diagnostic {
	message = strings.Map(func(character rune) rune {
		if unicode.IsControl(character) {
			return ' '
		}
		return character
	}, message)
	if len(message) > artifactstoreSpec.MaxDiagnosticMessageBytes {
		message = truncateUTF8(message, artifactstoreSpec.MaxDiagnosticMessageBytes)
	}
	return []artifactstoreSpec.Diagnostic{{
		Severity: artifactstoreSpec.DiagnosticSeverityError,
		Code:     code,
		Message:  message,
	}}
}

func truncateUTF8(value string, maximum int) string {
	if maximum <= 0 || len(value) <= maximum {
		return value
	}
	end := maximum
	for end > 0 && !utf8.ValidString(value[:end]) {
		end--
	}
	return value[:end]
}

var (
	_ artifactstoreSpec.RootKindHook             = rootKindHook{}
	_ artifactstoreSpec.CollectionKindHook       = collectionKindHook{}
	_ artifactstoreSpec.RootAttachmentSetHook    = rootKindHook{}
	_ artifactstoreSpec.RootAttachmentSourceHook = rootKindHook{}
)
