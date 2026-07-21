package workspace

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
)

const maxWorkspaceSkillBytes = 2 << 20

var workspaceSkillNamePattern = regexp.MustCompile(
	`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`,
)

type SkillDecoder struct{}

func NewSkillDecoder() *SkillDecoder {
	return &SkillDecoder{}
}

func (*SkillDecoder) ID() artifactstore.DecoderID {
	return SkillDecoderID
}

func (*SkillDecoder) Recognize(
	_ context.Context,
	candidate discovery.Candidate,
) discovery.Recognition {
	if path.Base(string(candidate.Locator)) != "SKILL.md" {
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
		return nil, workspaceArtifactDiagnostics(
			candidate.Locator,
			"workspace.skill.invalid",
			err.Error(),
		)
	}

	raw, err := json.Marshal(document)
	if err != nil {
		return nil, workspaceArtifactErrorDiagnostics(candidate.Locator, err)
	}
	raw, err = jsoncanon.CanonicalizeObject(
		raw,
		artifactstore.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, workspaceArtifactErrorDiagnostics(candidate.Locator, err)
	}

	labels := map[string]string{
		"skill.insert": document.Insert,
	}
	value := definition.Definition{
		Kind:          SkillKind,
		SchemaID:      SkillSchemaID,
		SchemaVersion: "1",
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
) (SkillDefinition, error) {
	if len(content) > maxWorkspaceSkillBytes {
		return SkillDefinition{}, fmt.Errorf(
			"SKILL.md exceeds %d bytes",
			maxWorkspaceSkillBytes,
		)
	}
	if !utf8.Valid(content) {
		return SkillDefinition{}, errors.New("SKILL.md must contain valid UTF-8")
	}

	frontmatter, body, err := splitSkillFrontmatter(content)
	if err != nil {
		return SkillDefinition{}, err
	}
	rawProperties := map[string]any{}
	if err := decodeRestrictedYAML(frontmatter, &rawProperties); err != nil {
		return SkillDefinition{}, err
	}

	name, ok := rawProperties["name"].(string)
	if !ok {
		return SkillDefinition{}, errors.New("frontmatter.name must be a string")
	}
	if !workspaceSkillNamePattern.MatchString(name) ||
		strings.Contains(name, "--") {
		return SkillDefinition{}, errors.New(
			"frontmatter.name must be a lowercase hyphenated name of at most 64 characters",
		)
	}
	parentName := path.Base(path.Dir(string(locator)))
	if name != parentName {
		return SkillDefinition{}, fmt.Errorf(
			"frontmatter.name %q must match containing directory %q",
			name,
			parentName,
		)
	}

	description, ok := rawProperties["description"].(string)
	if !ok ||
		strings.TrimSpace(description) != description ||
		description == "" {
		return SkillDefinition{}, errors.New(
			"frontmatter.description must be a non-empty trimmed string",
		)
	}
	if len(description) > 1024 {
		return SkillDefinition{}, errors.New(
			"frontmatter.description exceeds 1024 bytes",
		)
	}

	insert := string(agentskillsSpec.SkillInsertInstructions)
	if rawInsert, exists := rawProperties["insert"]; exists {
		value, ok := rawInsert.(string)
		if !ok {
			return SkillDefinition{}, errors.New(
				"frontmatter.insert must be a string",
			)
		}
		switch agentskillsSpec.SkillInsert(value) {
		case agentskillsSpec.SkillInsertInstructions,
			agentskillsSpec.SkillInsertUserMessage:
			insert = value
		default:
			return SkillDefinition{}, fmt.Errorf(
				"unsupported frontmatter.insert value %q",
				value,
			)
		}
	}

	arguments, err := decodeSkillArguments(rawProperties["arguments"])
	if err != nil {
		return SkillDefinition{}, err
	}
	tags, err := decodeSkillTags(rawProperties["tags"])
	if err != nil {
		return SkillDefinition{}, err
	}

	normalizedBody := strings.TrimLeft(
		strings.ReplaceAll(body, "\r\n", "\n"),
		"\n",
	)
	if strings.TrimSpace(normalizedBody) == "" {
		return SkillDefinition{}, errors.New("SKILL.md body is empty")
	}
	displayName := firstSkillHeading(normalizedBody)
	if displayName == "" {
		displayName = name
	}

	return SkillDefinition{
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
	if strings.TrimSpace(strings.TrimRight(first, "\r\n")) != "---" {
		return nil, "", errors.New("SKILL.md requires YAML frontmatter")
	}

	var frontmatter strings.Builder
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return nil, "", readErr
		}
		if strings.TrimSpace(strings.TrimRight(line, "\r\n")) == "---" {
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
) ([]SkillArgumentDefinition, error) {
	if raw == nil {
		return nil, nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, errors.New("frontmatter.arguments must be a list")
	}

	output := make([]SkillArgumentDefinition, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		properties, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf(
				"frontmatter.arguments[%d] must be an object",
				index,
			)
		}
		name, ok := properties["name"].(string)
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

		description, err := optionalString(properties, "description")
		if err != nil {
			return nil, err
		}
		defaultValue, err := optionalString(properties, "default")
		if err != nil {
			return nil, err
		}
		output = append(output, SkillArgumentDefinition{
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
		if after, ok := strings.CutPrefix(line, "# "); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

var _ discovery.Decoder = (*SkillDecoder)(nil)
