package providerapi

import (
	"path"
	"strings"
	"unicode"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func logicalNameForLocator(
	prefix string,
	locator basespec.Locator,
) basespec.LogicalName {
	segments := strings.Split(string(locator), "/")
	parts := make([]string, 0, len(segments)+1)
	parts = append(parts, prefix)

	for _, segment := range segments {
		segment = strings.TrimSuffix(segment, path.Ext(segment))
		segment = strings.ToLower(segment)
		segment = strings.Map(func(value rune) rune {
			switch {
			case value >= 'a' && value <= 'z':
				return value
			case value >= '0' && value <= '9':
				return value
			default:
				return '-'
			}
		}, segment)
		segment = strings.Trim(segment, ".-_")
		segment = collapseSeparators(segment)
		if segment != "" {
			parts = append(parts, segment)
		}
	}

	value := strings.Join(parts, "-")
	value = strings.Trim(value, ".-_")
	if value == "" {
		value = prefix
	}
	if len(value) > basespec.MaxLogicalNameBytes {
		digest := strings.TrimPrefix(
			string(cryptoutil.DigestBytes([]byte(string(locator)))),
			cryptoutil.DigestSHA256Prefix,
		)
		value = prefix + "-" + digest[:16]
	}
	if err := basespec.LogicalName(value).Validate(); err == nil {
		return basespec.LogicalName(value)
	}

	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes([]byte(string(locator)))),
		cryptoutil.DigestSHA256Prefix,
	)
	return basespec.LogicalName(prefix + "-" + digest[:16])
}

func collapseSeparators(value string) string {
	var output strings.Builder
	separator := false
	for _, character := range value {
		if character == '-' {
			if separator {
				continue
			}
			separator = true
			output.WriteRune(character)
			continue
		}
		separator = false
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			output.WriteRune(character)
		}
	}
	return strings.Trim(output.String(), "-")
}
