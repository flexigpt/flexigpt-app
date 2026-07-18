package workspace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"unicode"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/baseutils"
	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

const (
	formatJSON     = "json"
	formatYAML     = "yaml"
	formatMarkdown = "markdown"
)

type nativeDocumentClass struct {
	Kind                 artifactstoreSpec.ArtifactKind
	Format               string
	MCPCollection        bool
	RequireMCPCollection bool
}

type nativeFrontend struct {
	descriptors map[artifactstoreSpec.ArtifactKind]KindDescriptor
	yamlDecoder YAMLDecoder
}

func (f *nativeFrontend) ID() artifactstoreSpec.FrontendID {
	return NativeFrontendID
}

func (f *nativeFrontend) Recognizes(
	_ context.Context,
	candidate artifactstoreSpec.ArtifactCandidate,
) artifactstoreSpec.Recognition {
	class, ok := classifyNativeDocument(candidate.Locator, candidate.Content)
	if !ok {
		return artifactstoreSpec.RecognitionNone
	}
	if f.yamlDecoder == nil &&
		(class.Format == formatYAML ||
			(class.Format == formatMarkdown && class.Kind == KindSkillDefinition)) {
		return artifactstoreSpec.RecognitionNone
	}
	if class.Format == formatJSON {
		var header struct {
			Format string `json:"format"`
		}
		if json.Unmarshal(candidate.Content, &header) == nil &&
			header.Format == artifactstoreSpec.ArtifactDefinitionFileFormatV1 {
			return artifactstoreSpec.RecognitionNone
		}
	}
	return artifactstoreSpec.RecognitionPreferred
}

func (f *nativeFrontend) Decode(
	ctx context.Context,
	candidate artifactstoreSpec.ArtifactCandidate,
) ([]artifactstoreSpec.DecodedArtifact, []artifactstoreSpec.Diagnostic) {
	class, ok := classifyNativeDocument(candidate.Locator, candidate.Content)
	if !ok {
		return nil, nil
	}
	var (
		decoded []artifactstoreSpec.DecodedArtifact
		err     error
	)
	switch class.Format {
	case formatMarkdown:
		decoded, err = f.decodeMarkdown(ctx, class.Kind, candidate.Locator, candidate.Content)
	case formatJSON, formatYAML:
		decoded, err = f.decodeStructured(ctx, class, candidate.Locator, candidate.Content)
	default:
		err = fmt.Errorf("unsupported workspace source format %q", class.Format)
	}
	if err == nil {
		return decoded, nil
	}
	diagnostics := workspaceDiagnostics("workspace.frontend.decode", err.Error())
	diagnostics[0].Location = &artifactstoreSpec.DiagnosticLocation{Locator: candidate.Locator}
	return nil, diagnostics
}

func (f *nativeFrontend) ValidateStructure(
	_ context.Context,
	definition artifactstoreSpec.CanonicalDefinition,
) []artifactstoreSpec.Diagnostic {
	descriptor, ok := f.descriptors[definition.Kind]
	if !ok {
		return workspaceDiagnostics(
			"workspace.frontend.kind",
			fmt.Sprintf("workspace artifact kind %q is not registered", definition.Kind),
		)
	}
	if definition.SchemaID != descriptor.DefinitionSchemaID {
		return workspaceDiagnostics(
			"workspace.frontend.schema",
			fmt.Sprintf(
				"definition schema %q does not match expected schema %q",
				definition.SchemaID,
				descriptor.DefinitionSchemaID,
			),
		)
	}
	if err := validateJSONDocument(definition.DefinitionJSON); err != nil {
		return workspaceDiagnostics("workspace.frontend.structure", err.Error())
	}
	return nil
}

func (f *nativeFrontend) ValidateSemantic(
	_ context.Context,
	definition artifactstoreSpec.CanonicalDefinition,
) []artifactstoreSpec.Diagnostic {
	if err := validateWorkspaceCanonicalDefinition(definition); err != nil {
		return workspaceDiagnostics(
			"workspace.frontend.semantic",
			err.Error(),
		)
	}
	return nil
}

func (f *nativeFrontend) ExtractDependencies(
	_ context.Context,
	definition artifactstoreSpec.CanonicalDefinition,
) ([]artifactstoreSpec.ArtifactSelector, []artifactstoreSpec.Diagnostic) {
	var document struct {
		Dependencies []artifactstoreSpec.ArtifactSelector `json:"dependencies,omitempty"`
	}
	if err := json.Unmarshal(definition.DefinitionJSON, &document); err != nil {
		return nil, workspaceDiagnostics("workspace.dependencies.decode", err.Error())
	}
	selectors := append([]artifactstoreSpec.ArtifactSelector(nil), document.Dependencies...)
	if definition.Kind == KindAgentDefinition {
		var agent AgentDefinitionDocument
		if err := json.Unmarshal(definition.DefinitionJSON, &agent); err != nil {
			return nil, workspaceDiagnostics("workspace.dependencies.decode", err.Error())
		}
		if agent.StartingModel != nil {
			selectors = append(selectors, *agent.StartingModel)
		}
		for _, selection := range agent.StartingTools {
			selectors = append(selectors, selection.Selector)
		}
		for _, selection := range agent.StartingSkills {
			selectors = append(selectors, selection.Selector)
		}
		for _, selection := range agent.StartingMCPServers {
			selectors = append(selectors, selection.Selector)
		}
	}
	return selectors, nil
}

func (f *nativeFrontend) ValidateRecordData(
	_ context.Context,
	_ artifactstoreSpec.CanonicalDefinition,
	_ artifactstoreSpec.ArtifactRecordDraft,
) []artifactstoreSpec.Diagnostic {
	return nil
}

func (f *nativeFrontend) DescribeExportClosure(
	_ context.Context,
	definition artifactstoreSpec.CanonicalDefinition,
) (artifactstoreSpec.ExportClosure, []artifactstoreSpec.Diagnostic) {
	return artifactstoreSpec.ExportClosure{
		DefinitionDigests: []artifactstoreSpec.Digest{definition.Digest},
		Assets:            append([]artifactstoreSpec.AssetManifestEntry(nil), definition.AssetManifest...),
	}, nil
}

func (f *nativeFrontend) decodeStructured(
	ctx context.Context,
	class nativeDocumentClass,
	locator artifactstoreSpec.SourceLocator,
	content []byte,
) ([]artifactstoreSpec.DecodedArtifact, error) {
	raw := json.RawMessage(content)
	if class.Format == formatYAML {
		if f.yamlDecoder == nil {
			return nil, errors.New("YAML decoder is not configured")
		}
		decoded, err := f.yamlDecoder(ctx, append([]byte(nil), content...))
		if err != nil {
			return nil, err
		}
		if len(decoded) > artifactstoreSpec.MaxDefinitionJSONBytes {
			return nil, fmt.Errorf(
				"decoded YAML exceeds %d bytes",
				artifactstoreSpec.MaxDefinitionJSONBytes,
			)
		}
		raw = decoded
	}
	if err := validateJSONDocument(raw); err != nil {
		return nil, err
	}
	canonical, err := baseutils.CanonicalizeJSON(raw)
	if err != nil {
		return nil, err
	}
	if len(canonical) == 0 || canonical[0] != '{' {
		return nil, errors.New("workspace structured document must be a JSON object")
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(canonical, &object); err != nil {
		return nil, err
	}
	if class.MCPCollection {
		return f.decodeMCPCollection(class, locator, object)
	}
	name := fixedLogicalName(class.Kind)
	if name == "" {
		name = nativeLogicalName(class.Kind, object, locator)
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(name) != name {
		return nil, errors.New("workspace artifact name is required and must be trimmed")
	}
	normalizedRaw, normalizedObject, err := normalizeNativeStructuredDocument(
		class.Kind,
		object,
	)
	if err != nil {
		return nil, err
	}
	definition, err := f.definitionFor(
		class.Kind,
		artifactstoreSpec.LogicalName(name),
		class.Format,
		normalizedRaw,
		normalizedObject,
	)
	if err != nil {
		return nil, err
	}
	return []artifactstoreSpec.DecodedArtifact{{Definition: definition}}, nil
}

func (f *nativeFrontend) decodeMCPCollection(
	class nativeDocumentClass,
	locator artifactstoreSpec.SourceLocator,
	object map[string]json.RawMessage,
) ([]artifactstoreSpec.DecodedArtifact, error) {
	var servers map[string]json.RawMessage
	rawServers, foundServers := object["mcpServers"]
	if foundServers {
		if err := json.Unmarshal(rawServers, &servers); err != nil {
			return nil, fmt.Errorf("mcpServers must be an object: %w", err)
		}
		if servers == nil {
			return nil, errors.New("mcpServers must be an object")
		}
	}
	if !foundServers {
		if class.RequireMCPCollection {
			return nil, errors.New("MCP collection document must contain mcpServers")
		}
		name := nativeLogicalName(KindMCPServerDefinition, object, locator)
		if strings.TrimSpace(name) == "" {
			return nil, errors.New("MCP server name is required")
		}
		normalizedRaw, normalizedObject, err := normalizeNativeStructuredDocument(
			KindMCPServerDefinition,
			object,
		)
		if err != nil {
			return nil, err
		}
		definition, err := f.definitionFor(
			KindMCPServerDefinition,
			artifactstoreSpec.LogicalName(name),
			class.Format,
			normalizedRaw,
			normalizedObject,
		)
		if err != nil {
			return nil, err
		}
		return []artifactstoreSpec.DecodedArtifact{{Definition: definition}}, nil
	}

	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	slices.Sort(names)
	out := make([]artifactstoreSpec.DecodedArtifact, 0, len(names))
	for _, name := range names {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(name) != name {
			return nil, fmt.Errorf("MCP server name %q is invalid", name)
		}
		raw, err := baseutils.CanonicalizeJSON(servers[name])
		if err != nil {
			return nil, fmt.Errorf("MCP server %q: %w", name, err)
		}
		if len(raw) == 0 || raw[0] != '{' {
			return nil, fmt.Errorf("MCP server %q must be an object", name)
		}
		var serverObject map[string]json.RawMessage
		if err := json.Unmarshal(raw, &serverObject); err != nil {
			return nil, err
		}
		normalizedRaw, normalizedObject, err := normalizeNativeStructuredDocument(
			KindMCPServerDefinition,
			serverObject,
		)
		if err != nil {
			return nil, fmt.Errorf("MCP server %q: %w", name, err)
		}
		digest := strings.TrimPrefix(string(baseutils.DigestBytes([]byte(name))), "sha256:")
		subresource := artifactstoreSpec.SubresourceLocator(
			"servers/" + portableSegment(name) + "-" + digest[:12],
		)
		definition, err := f.definitionFor(
			KindMCPServerDefinition,
			artifactstoreSpec.LogicalName(name),
			class.Format,
			normalizedRaw,
			normalizedObject,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, artifactstoreSpec.DecodedArtifact{
			SubresourceLocator: subresource,
			Definition:         definition,
		})
	}
	return out, nil
}

func (f *nativeFrontend) decodeMarkdown(
	ctx context.Context,
	kind artifactstoreSpec.ArtifactKind,
	locator artifactstoreSpec.SourceLocator,
	content []byte,
) ([]artifactstoreSpec.DecodedArtifact, error) {
	if !bytes.Equal(bytes.ToValidUTF8(content, nil), content) {
		return nil, errors.New("workspace Markdown must be valid UTF-8")
	}
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("workspace Markdown must not be empty")
	}
	name := fixedLogicalName(kind)
	object := map[string]json.RawMessage{}
	var frontmatter json.RawMessage
	if kind == KindSkillDefinition {
		if f.yamlDecoder == nil {
			return nil, errors.New("YAML decoder is required for SKILL.md")
		}
		yamlFrontmatter, err := splitMarkdownFrontmatter(text)
		if err != nil {
			return nil, err
		}
		decoded, err := f.yamlDecoder(ctx, []byte(yamlFrontmatter))
		if err != nil {
			return nil, fmt.Errorf("decode SKILL.md frontmatter: %w", err)
		}
		if len(decoded) > artifactstoreSpec.MaxExtensionsJSONBytes {
			return nil, fmt.Errorf(
				"SKILL.md frontmatter exceeds %d bytes",
				artifactstoreSpec.MaxExtensionsJSONBytes,
			)
		}
		if err := validateJSONDocument(decoded); err != nil {
			return nil, fmt.Errorf("validate SKILL.md frontmatter: %w", err)
		}
		canonical, err := baseutils.CanonicalizeJSON(decoded)
		if err != nil {
			return nil, err
		}
		if len(canonical) == 0 || canonical[0] != '{' {
			return nil, errors.New("SKILL.md frontmatter must be a YAML object")
		}
		frontmatter = json.RawMessage(canonical)
		if err := json.Unmarshal(frontmatter, &object); err != nil {
			return nil, err
		}
		name = stringField(object, "name")
		if strings.TrimSpace(name) == "" {
			return nil, errors.New("SKILL.md requires a top-level frontmatter name")
		}
	}
	raw, err := json.Marshal(struct {
		Markdown    string          `json:"markdown"`
		Frontmatter json.RawMessage `json:"frontmatter,omitempty"`
	}{
		Markdown:    text,
		Frontmatter: frontmatter,
	})
	if err != nil {
		return nil, err
	}
	definition, err := f.definitionFor(
		kind,
		artifactstoreSpec.LogicalName(name),
		formatMarkdown,
		raw,
		object,
	)
	if err != nil {
		return nil, err
	}
	decoded := artifactstoreSpec.DecodedArtifact{Definition: definition}
	if kind == KindSkillDefinition {
		decoded.AssetRoots = []artifactstoreSpec.SourceAssetRoot{{
			Root:      artifactstoreSpec.SourceLocator(path.Dir(string(locator))),
			Recursive: true,
		}}
	}
	return []artifactstoreSpec.DecodedArtifact{decoded}, nil
}

func (f *nativeFrontend) definitionFor(
	kind artifactstoreSpec.ArtifactKind,
	name artifactstoreSpec.LogicalName,
	format string,
	definitionJSON json.RawMessage,
	object map[string]json.RawMessage,
) (artifactstoreSpec.CanonicalDefinition, error) {
	descriptor, ok := f.descriptors[kind]
	if !ok {
		return artifactstoreSpec.CanonicalDefinition{}, fmt.Errorf(
			"workspace artifact kind %q is not registered",
			kind,
		)
	}
	extensions, err := json.Marshal(map[string]string{"sourceFormat": format})
	if err != nil {
		return artifactstoreSpec.CanonicalDefinition{}, err
	}
	extensions, err = baseutils.CanonicalizeJSON(extensions)
	if err != nil {
		return artifactstoreSpec.CanonicalDefinition{}, err
	}
	labels := map[string]string{}
	if rawLabels, ok := object["labels"]; ok {
		if err := json.Unmarshal(rawLabels, &labels); err != nil {
			return artifactstoreSpec.CanonicalDefinition{}, errors.New("labels must be a string map")
		}
	}
	if len(labels) == 0 {
		labels = nil
	}
	if kind == KindModelDefinition {
		if providerName := nativeModelProviderName(object); providerName != "" {
			if labels == nil {
				labels = map[string]string{}
			}
			labels["provider"] = providerName
		}
	}
	return artifactstoreSpec.CanonicalDefinition{
		Kind:           kind,
		SchemaID:       descriptor.DefinitionSchemaID,
		SchemaVersion:  workspaceDefinitionSchemaV1,
		LogicalName:    name,
		LogicalVersion: artifactstoreSpec.LogicalVersion(stringField(object, "version")),
		DisplayName:    stringField(object, "displayName"),
		Description:    stringField(object, "description"),
		Labels:         labels,
		Extensions:     extensions,
		DefinitionJSON: append(json.RawMessage(nil), definitionJSON...),
	}, nil
}

func classifyNativeDocument(
	locator artifactstoreSpec.SourceLocator,
	content []byte,
) (nativeDocumentClass, bool) {
	if class, ok := classifyNativePath(locator); ok {
		return class, true
	}
	if strings.EqualFold(path.Ext(string(locator)), ".json") {
		return classifyCurrentDomainDocument(content)
	}
	return nativeDocumentClass{}, false
}

func classifyNativePath(locator artifactstoreSpec.SourceLocator) (nativeDocumentClass, bool) {
	value := strings.ToLower(string(locator))
	base := strings.ToLower(path.Base(value))
	switch value {
	case workspaceDefinitionJSONLocator:
		return nativeDocumentClass{Kind: KindWorkspaceDefinition, Format: formatJSON}, true
	case workspaceDefinitionYAMLLocator, workspaceDefinitionYMLLocator:
		return nativeDocumentClass{Kind: KindWorkspaceDefinition, Format: formatYAML}, true
	case strings.ToLower(workspaceAgentsLocator):
		return nativeDocumentClass{Kind: KindInstructionDocument, Format: formatMarkdown}, true
	case strings.ToLower(workspaceReadmeLocator):
		return nativeDocumentClass{Kind: KindContextDocument, Format: formatMarkdown}, true
	case workspaceMCPDotJSONLocator, workspaceMCPDotsJSONLocator, workspaceMCPJSONLocator, workspaceMCPsJSONLocator:
		return nativeDocumentClass{
			Kind:                 KindMCPServerDefinition,
			Format:               formatJSON,
			MCPCollection:        true,
			RequireMCPCollection: true,
		}, true
	}
	if base == workspaceSkillMarkdownFileName {
		return nativeDocumentClass{Kind: KindSkillDefinition, Format: formatMarkdown}, true
	}
	var format string
	switch path.Ext(value) {
	case ".json":
		format = formatJSON
	case ".yaml", ".yml":
		format = formatYAML
	default:
		return nativeDocumentClass{}, false
	}
	switch {
	case hasPathPrefix(value, workspaceAgentsDirectory):
		return nativeDocumentClass{Kind: KindAgentDefinition, Format: format}, true
	case hasPathPrefix(value, workspaceModelsDirectory):
		return nativeDocumentClass{Kind: KindModelDefinition, Format: format}, true
	case hasPathPrefix(value, workspaceMCPDirectory):
		return nativeDocumentClass{
			Kind:          KindMCPServerDefinition,
			Format:        format,
			MCPCollection: true,
		}, true
	case hasPathPrefix(value, workspaceToolsDirectory):
		return nativeDocumentClass{Kind: KindToolDefinition, Format: format}, true
	default:
		return nativeDocumentClass{}, false
	}
}

func hasPathPrefix(value, prefix string) bool {
	return strings.HasPrefix(value, prefix)
}

func fixedLogicalName(kind artifactstoreSpec.ArtifactKind) string {
	switch kind {
	case KindWorkspaceDefinition:
		return "workspace"
	case KindInstructionDocument:
		return "workspace-instructions"
	case KindContextDocument:
		return "workspace-context"
	default:
		return ""
	}
}

func stringField(object map[string]json.RawMessage, fields ...string) string {
	for _, field := range fields {
		var value string
		if raw, ok := object[field]; ok && json.Unmarshal(raw, &value) == nil {
			return value
		}
	}
	return ""
}

func splitMarkdownFrontmatter(value string) (string, error) {
	lines := strings.Split(value, "\n")
	if len(lines) == 0 || lines[0] != yamlSeparator {
		return "", errors.New("SKILL.md must begin with YAML frontmatter")
	}
	for index := 1; index < len(lines); index++ {
		if lines[index] == yamlSeparator {
			frontmatter := strings.Join(lines[1:index], "\n")
			if strings.TrimSpace(frontmatter) == "" {
				return "", errors.New("SKILL.md frontmatter must not be empty")
			}
			return frontmatter, nil
		}
	}
	return "", errors.New("SKILL.md YAML frontmatter is not terminated")
}

func sourceLogicalName(locator artifactstoreSpec.SourceLocator) string {
	base := path.Base(string(locator))
	extension := path.Ext(base)
	name := strings.TrimSuffix(base, extension)
	name = strings.TrimPrefix(name, ".")
	return strings.TrimSpace(name)
}

func portableSegment(value string) string {
	var builder strings.Builder
	lastHyphen := false
	for _, character := range strings.ToLower(value) {
		switch {
		case unicode.IsLetter(character), unicode.IsDigit(character):
			builder.WriteRune(character)
			lastHyphen = false
		default:
			if builder.Len() > 0 && !lastHyphen {
				builder.WriteByte('-')
				lastHyphen = true
			}
		}
		if builder.Len() >= 48 {
			break
		}
	}
	out := strings.Trim(builder.String(), "-")
	if out != "" {
		return out
	}
	digest := strings.TrimPrefix(string(baseutils.DigestBytes([]byte(value))), "sha256:")
	return "artifact-" + digest[:12]
}

var _ artifactstoreSpec.ArtifactFrontend = (*nativeFrontend)(nil)
