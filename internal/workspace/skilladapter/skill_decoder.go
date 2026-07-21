package skilladapter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
)

var workspaceSkillNamePattern = regexp.MustCompile(
	workspaceSkillNamePatternText,
)

type SkillDecoder struct{}

func NewSkillDecoder() *SkillDecoder {
	return &SkillDecoder{}
}

func DiscoveryProfile() engine.DiscoveryProfile {
	return discoveryProfile
}

func ArtifactSupport() engine.ArtifactSupport {
	return artifactSupport
}

func (*SkillDecoder) ID() artifactstore.DecoderID {
	return skillDecoderID
}

func (*SkillDecoder) Recognize(
	_ context.Context,
	candidate discovery.Candidate,
) discovery.Recognition {
	if path.Base(string(candidate.Locator)) != skillDefinitionFileName {
		return discovery.RecognitionNone
	}
	return discovery.RecognitionPreferred
}

func (*SkillDecoder) Decode(
	_ context.Context,
	candidate discovery.Candidate,
) ([]discovery.Decoded, []artifactstore.Diagnostic) {
	document, err := decodeSkillMarkdown(candidate.Locator, candidate.Content)
	if err != nil {
		return nil, engine.WorkspaceArtifactDiagnostics(
			candidate.Locator,
			engine.DiagnosticCodeSkillInvalid,
			err.Error(),
		)
	}

	raw, err := json.Marshal(document)
	if err != nil {
		return nil, engine.WorkspaceArtifactErrorDiagnostics(candidate.Locator, err)
	}
	raw, err = jsoncanon.CanonicalizeObject(
		raw,
		artifactstore.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, engine.WorkspaceArtifactErrorDiagnostics(candidate.Locator, err)
	}

	labels := map[string]string{
		skillInsertLabelKey: document.Insert,
	}
	value := definition.Definition{
		Kind:          skillKind,
		SchemaID:      skillSchemaID,
		SchemaVersion: workspaceSkillsSchemaVersionV1,
		LogicalName:   artifactstore.LogicalName(document.Name),
		DisplayName:   document.DisplayName,
		Description:   document.Description,
		Labels:        labels,
		Body:          raw,
	}
	return []discovery.Decoded{{Definition: value}}, nil
}

func decodeSkillMarkdown(
	locator artifactstore.Locator,
	content []byte,
) (skillDefinition, error) {
	if len(content) > maxWorkspaceSkillBytes {
		return skillDefinition{}, fmt.Errorf(
			"SKILL.md exceeds %d bytes",
			maxWorkspaceSkillBytes,
		)
	}
	if !utf8.Valid(content) {
		return skillDefinition{}, errors.New("SKILL.md must contain valid UTF-8")
	}

	frontmatter, body, err := splitSkillFrontmatter(content)
	if err != nil {
		return skillDefinition{}, err
	}
	rawProperties := map[string]any{}
	if err := decodeRestrictedYAML(frontmatter, &rawProperties); err != nil {
		return skillDefinition{}, err
	}

	name, ok := rawProperties[skillFrontmatterNameKey].(string)
	if !ok {
		return skillDefinition{}, errors.New("frontmatter.name must be a string")
	}
	if !workspaceSkillNamePattern.MatchString(name) ||
		strings.Contains(name, skillNameRepeatedHyphen) {
		return skillDefinition{}, errors.New(
			"frontmatter.name must be a lowercase hyphenated name of at most 64 characters",
		)
	}
	parentName := path.Base(path.Dir(string(locator)))
	if name != parentName {
		return skillDefinition{}, fmt.Errorf(
			"frontmatter.name %q must match containing directory %q",
			name,
			parentName,
		)
	}

	description, ok := rawProperties[skillFrontmatterDescriptionKey].(string)
	if !ok ||
		strings.TrimSpace(description) != description ||
		description == "" {
		return skillDefinition{}, errors.New(
			"frontmatter.description must be a non-empty trimmed string",
		)
	}
	if len(description) > maxSkillDescriptionBytes {
		return skillDefinition{}, errors.New(
			"frontmatter.description exceeds 1024 bytes",
		)
	}

	insert := string(agentskillsSpec.SkillInsertInstructions)
	if rawInsert, exists := rawProperties[skillFrontmatterInsertKey]; exists {
		value, ok := rawInsert.(string)
		if !ok {
			return skillDefinition{}, errors.New(
				"frontmatter.insert must be a string",
			)
		}
		switch agentskillsSpec.SkillInsert(value) {
		case agentskillsSpec.SkillInsertInstructions,
			agentskillsSpec.SkillInsertUserMessage:
			insert = value
		default:
			return skillDefinition{}, fmt.Errorf(
				"unsupported frontmatter.insert value %q",
				value,
			)
		}
	}

	arguments, err := decodeSkillArguments(rawProperties[skillFrontmatterArgumentsKey])
	if err != nil {
		return skillDefinition{}, err
	}
	tags, err := decodeSkillTags(rawProperties[skillFrontmatterTagsKey])
	if err != nil {
		return skillDefinition{}, err
	}

	normalizedBody := strings.TrimLeft(
		strings.ReplaceAll(body, "\r\n", "\n"),
		"\n",
	)
	if strings.TrimSpace(normalizedBody) == "" {
		return skillDefinition{}, errors.New("SKILL.md body is empty")
	}
	displayName := firstSkillHeading(normalizedBody)
	if displayName == "" {
		displayName = name
	}

	return skillDefinition{
		Name:           name,
		DisplayName:    displayName,
		Description:    description,
		Insert:         insert,
		Arguments:      arguments,
		Tags:           tags,
		MarkdownBody:   normalizedBody,
		RawFrontmatter: rawProperties,
	}, nil
}

func splitSkillFrontmatter(
	content []byte,
) (frontmatterBytes []byte, bodyStr string, err error) {
	reader := bufio.NewReader(bytes.NewReader(content))
	first, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, "", err
	}
	if strings.TrimSpace(strings.TrimRight(first, "\r\n")) != yamlDocumentStart {
		return nil, "", errors.New("SKILL.md requires YAML frontmatter")
	}

	var frontmatter strings.Builder
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return nil, "", readErr
		}
		if strings.TrimSpace(strings.TrimRight(line, "\r\n")) == yamlDocumentStart {
			break
		}
		frontmatter.WriteString(line)
		if errors.Is(readErr, io.EOF) {
			return nil, "", errors.New(
				"SKILL.md has unterminated YAML frontmatter",
			)
		}
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", err
	}
	return []byte(frontmatter.String()), string(body), nil
}

func decodeSkillArguments(
	raw any,
) ([]skillArgumentDefinition, error) {
	if raw == nil {
		return nil, nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, errors.New("frontmatter.arguments must be a list")
	}

	output := make([]skillArgumentDefinition, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		properties, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf(
				"frontmatter.arguments[%d] must be an object",
				index,
			)
		}
		name, ok := properties[skillArgumentNameKey].(string)
		if !ok || !validSkillArgumentName(name) {
			return nil, fmt.Errorf(
				"frontmatter.arguments[%d].name is invalid",
				index,
			)
		}
		if _, duplicate := seen[name]; duplicate {
			return nil, fmt.Errorf("duplicate Skill argument %q", name)
		}
		seen[name] = struct{}{}

		description, err := optionalString(properties, skillArgumentDescriptionKey)
		if err != nil {
			return nil, err
		}
		defaultValue, err := optionalString(properties, skillArgumentDefaultKey)
		if err != nil {
			return nil, err
		}
		output = append(output, skillArgumentDefinition{
			Name:        name,
			Description: description,
			Default:     defaultValue,
		})
	}
	return output, nil
}

func decodeSkillTags(raw any) ([]string, error) {
	if raw == nil {
		return nil, nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, errors.New("frontmatter.tags must be a list")
	}
	output := make([]string, 0, len(values))
	for index, value := range values {
		tag, ok := value.(string)
		if !ok || tag == "" || strings.TrimSpace(tag) != tag {
			return nil, fmt.Errorf("frontmatter.tags[%d] is invalid", index)
		}
		output = append(output, tag)
	}
	slices.Sort(output)
	output = slices.Compact(output)
	return output, nil
}

func optionalString(properties map[string]any, key string) (string, error) {
	value, exists := properties[key]
	if !exists || value == nil {
		return "", nil
	}
	output, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}
	return output, nil
}

func validSkillArgumentName(value string) bool {
	for index, character := range value {
		switch {
		case character == '_':
		case character >= 'a' && character <= 'z':
		case character >= 'A' && character <= 'Z':
		case index > 0 && character >= '0' && character <= '9':
		default:
			return false
		}
	}
	return value != ""
}

func firstSkillHeading(body string) string {
	for line := range strings.SplitSeq(body, "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, skillMarkdownHeadingPrefix); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}
