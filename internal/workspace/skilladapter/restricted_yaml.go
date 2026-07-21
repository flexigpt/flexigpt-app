package skilladapter

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
	"github.com/goccy/go-yaml"
)

var topLevelYAMLKey = regexp.MustCompile(
	restrictedYAMLTopLevelKeyPattern,
)

func decodeRestrictedYAML(
	raw []byte,
	target any,
) error {
	if len(raw) == 0 {
		return fmt.Errorf("%w: YAML document is empty", engine.ErrInvalidWorkspace)
	}
	if bytes.IndexByte(raw, 0) >= 0 {
		return fmt.Errorf("%w: YAML contains a NUL byte", engine.ErrInvalidWorkspace)
	}

	seenTopLevel := make(map[string]struct{})
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), artifactstore.MaxDefinitionBodyBytes)

	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == yamlDocumentStart || trimmed == yamlDocumentEnd {
			return fmt.Errorf(
				"%w: YAML frontmatter must contain one document",
				engine.ErrInvalidWorkspace,
			)
		}
		if strings.ContainsRune(line, '\t') {
			return fmt.Errorf(
				"%w: YAML tabs are not allowed at line %d",
				engine.ErrInvalidWorkspace,
				lineNumber,
			)
		}
		if hasRestrictedYAMLToken(line) {
			return fmt.Errorf(
				"%w: YAML aliases, anchors, merge keys, and tags are not allowed at line %d",
				engine.ErrInvalidWorkspace,
				lineNumber,
			)
		}
		if line == strings.TrimLeft(line, " ") {
			match := topLevelYAMLKey.FindStringSubmatch(line)
			if len(match) == 2 {
				key := match[1]
				if _, duplicate := seenTopLevel[key]; duplicate {
					return fmt.Errorf(
						"%w: duplicate YAML key %q",
						engine.ErrInvalidWorkspace,
						key,
					)
				}
				seenTopLevel[key] = struct{}{}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if err := yaml.Unmarshal(raw, target); err != nil {
		return fmt.Errorf(
			"%w: decode restricted YAML: %w",
			engine.ErrInvalidWorkspace,
			err,
		)
	}
	return nil
}

func hasRestrictedYAMLToken(line string) bool {
	var quote rune
	escaped := false

	for index, character := range line {
		if escaped {
			escaped = false
			continue
		}
		if quote == '"' && character == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if character == quote {
				quote = 0
			}
			continue
		}
		if character == '"' || character == '\'' {
			quote = character
			continue
		}
		if character == '#' {
			break
		}
		if character != '&' &&
			character != '*' &&
			character != '!' {
			continue
		}
		if index == 0 {
			return true
		}
		previous := line[index-1]
		if previous == ' ' ||
			previous == ':' ||
			previous == '[' ||
			previous == '{' ||
			previous == ',' {
			return true
		}
	}
	return strings.Contains(line, yamlMergeKey)
}
