package declaration

import (
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// DeriveLogicalName creates a deterministic portable name from one
// source-relative locator. Physical format adapters use this only when their
// native format has no explicit portable declaration name.
func DeriveLogicalName(
	prefix string,
	locator model.Locator,
) (model.LogicalName, error) {
	if err := model.ValidatePortableName(
		"derived logical name prefix",
		prefix,
	); err != nil {
		return "", err
	}
	if err := locator.Validate(false); err != nil {
		return "", err
	}

	parts := make([]string, 0, strings.Count(string(locator), "/")+2)
	parts = append(parts, prefix)
	for segment := range strings.SplitSeq(string(locator), "/") {
		segment = normalizeLocatorNameSegment(segment)
		if segment != "" {
			parts = append(parts, segment)
		}
	}

	candidate := strings.Trim(
		strings.Join(parts, "-"),
		".-_",
	)
	if value := model.LogicalName(candidate); value.Validate() == nil {
		return value, nil
	}

	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes([]byte(locator))),
		cryptoutil.DigestSHA256Prefix,
	)
	fallback := prefix + "-" + digest[:16]
	value := model.LogicalName(fallback)
	if err := value.Validate(); err != nil {
		return "", err
	}
	return value, nil
}

const nestedNameDigestLength = 16

// DeriveNestedLogicalName derives a deterministic named declaration identity
// from a parent declaration name and a relationship label.
//
// The normal result is:
//
//	<parent>-<relationship>
//
// If that value would exceed the portable name limit, the parent is shortened
// and a digest suffix preserves deterministic uniqueness.
func DeriveNestedLogicalName(
	parent model.LogicalName,
	relationship string,
) (model.LogicalName, error) {
	if err := parent.Validate(); err != nil {
		return "", err
	}
	if err := model.ValidateIdentifier(
		"derived nested declaration relationship",
		relationship,
		model.MaxKindBytes,
	); err != nil {
		return "", err
	}

	candidate := string(parent) + "-" + relationship
	if value := model.LogicalName(candidate); value.Validate() == nil {
		return value, nil
	}

	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes(
			[]byte(string(parent)+"\x00"+relationship),
		)),
		cryptoutil.DigestSHA256Prefix,
	)
	suffix := "-" + relationship + "-" + digest[:nestedNameDigestLength]
	maximumParentBytes := model.MaxLogicalNameBytes - len(suffix)
	shortenedParent := strings.TrimRight(
		string(parent)[:maximumParentBytes],
		".-_",
	)

	value := model.LogicalName(shortenedParent + suffix)
	if err := value.Validate(); err != nil {
		return "", err
	}
	return value, nil
}

func normalizeLocatorNameSegment(
	value string,
) string {
	value = strings.TrimSuffix(value, path.Ext(value))
	value = strings.ToLower(value)

	var output strings.Builder
	separator := false
	for _, character := range value {
		asciiLetter := character >= 'a' && character <= 'z'
		asciiDigit := character >= '0' && character <= '9'
		if asciiLetter || asciiDigit {
			output.WriteRune(character)
			separator = false
			continue
		}
		if !separator {
			output.WriteByte('-')
			separator = true
		}
	}
	return strings.Trim(output.String(), ".-_")
}
