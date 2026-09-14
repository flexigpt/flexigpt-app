package declaration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type LocatorKind string

const (
	LocatorKindPath    LocatorKind = "path"
	LocatorKindURL     LocatorKind = "url"
	LocatorKindGit     LocatorKind = "git"
	LocatorKindPackage LocatorKind = "package"
	LocatorKindCommand LocatorKind = "command"
)

type PackageManager string

const (
	PackageManagerNPM   PackageManager = "npm"
	PackageManagerPyPI  PackageManager = "pypi"
	PackageManagerCargo PackageManager = "cargo"
	PackageManagerGo    PackageManager = "go"
	PackageManagerMaven PackageManager = "maven"
	PackageManagerNuGet PackageManager = "nuget"
	PackageManagerOCI   PackageManager = "oci"
)

type Locator struct {
	Kind LocatorKind `json:"kind,omitempty"`

	Scalar string `json:"-"`

	Path      string `json:"path,omitempty"`
	URL       string `json:"url,omitempty"`
	Integrity string `json:"integrity,omitempty"`

	Repository string `json:"repository,omitempty"`
	Revision   string `json:"revision,omitempty"`

	Manager  PackageManager `json:"manager,omitempty"`
	Package  string         `json:"package,omitempty"`
	Version  string         `json:"version,omitempty"`
	Registry string         `json:"registry,omitempty"`

	Command string `json:"command,omitempty"`
}

type locatorObject struct {
	Kind LocatorKind `json:"kind"`

	Path      string `json:"path,omitempty"`
	URL       string `json:"url,omitempty"`
	Integrity string `json:"integrity,omitempty"`

	Repository string `json:"repository,omitempty"`
	Revision   string `json:"revision,omitempty"`

	Manager  PackageManager `json:"manager,omitempty"`
	Package  string         `json:"package,omitempty"`
	Version  string         `json:"version,omitempty"`
	Registry string         `json:"registry,omitempty"`

	Command string `json:"command,omitempty"`
}

func ScalarLocator(value string) Locator {
	return Locator{Scalar: value}
}

func PathLocator(value string) Locator {
	return Locator{
		Kind: LocatorKindPath,
		Path: value,
	}
}

func (l Locator) Clone() Locator {
	return l
}

func (l Locator) IsScalar() bool {
	return l.Kind == ""
}

func (l Locator) MarshalJSON() ([]byte, error) {
	if l.Kind == "" {
		if err := validateScalarLocator(l.Scalar); err != nil {
			return nil, err
		}
		raw, err := json.Marshal(l.Scalar)
		if err != nil {
			return nil, err
		}
		return jsonutil.Canonicalize(raw)
	}
	if err := l.Validate(); err != nil {
		return nil, err
	}
	return jsonutil.MarshalCanonicalObject(
		locatorObject{
			Kind:       l.Kind,
			Path:       l.Path,
			URL:        l.URL,
			Integrity:  l.Integrity,
			Repository: l.Repository,
			Revision:   l.Revision,
			Manager:    l.Manager,
			Package:    l.Package,
			Version:    l.Version,
			Registry:   l.Registry,
			Command:    l.Command,
		},
		basespec.MaxDefinitionBodyBytes,
	)
}

func (l *Locator) UnmarshalJSON(raw []byte) error {
	if l == nil {
		return fmt.Errorf(
			"%w: Locator target is nil",
			basespec.ErrInvalid,
		)
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return err
	}
	trimmed := bytes.TrimSpace(canonical)
	if len(trimmed) == 0 {
		return fmt.Errorf(
			"%w: Locator is empty",
			basespec.ErrInvalid,
		)
	}

	if trimmed[0] == '"' {
		var scalar string
		if err := json.Unmarshal(trimmed, &scalar); err != nil {
			return err
		}
		value := ScalarLocator(scalar)
		if err := value.Validate(); err != nil {
			return err
		}
		*l = value
		return nil
	}

	var object locatorObject
	if err := jsonutil.DecodeCanonicalObjectInto(
		trimmed,
		&object,
		basespec.MaxDefinitionBodyBytes,
	); err != nil {
		return err
	}
	value := Locator{
		Kind:       object.Kind,
		Path:       object.Path,
		URL:        object.URL,
		Integrity:  object.Integrity,
		Repository: object.Repository,
		Revision:   object.Revision,
		Manager:    object.Manager,
		Package:    object.Package,
		Version:    object.Version,
		Registry:   object.Registry,
		Command:    object.Command,
	}
	if err := value.Validate(); err != nil {
		return err
	}
	*l = value
	return nil
}

func (l Locator) Validate() error {
	if l.Kind == "" {
		if l.Path != "" ||
			l.URL != "" ||
			l.Integrity != "" ||
			l.Repository != "" ||
			l.Revision != "" ||
			l.Manager != "" ||
			l.Package != "" ||
			l.Version != "" ||
			l.Registry != "" ||
			l.Command != "" {
			return fmt.Errorf(
				"%w: scalar Locator cannot contain typed fields",
				basespec.ErrInvalid,
			)
		}
		return validateScalarLocator(l.Scalar)
	}
	if l.Scalar != "" {
		return fmt.Errorf(
			"%w: typed Locator cannot contain a scalar value",
			basespec.ErrInvalid,
		)
	}

	switch l.Kind {
	case LocatorKindPath:
		if err := ValidatePortableLocatorPath(
			"path Locator",
			l.Path,
			true,
		); err != nil {
			return err
		}
		return requireEmptyFields(
			"url", l.URL,
			"integrity", l.Integrity,
			"repository", l.Repository,
			"revision", l.Revision,
			"manager", string(l.Manager),
			"package", l.Package,
			"version", l.Version,
			"registry", l.Registry,
			"command", l.Command,
		)

	case LocatorKindURL:
		if err := ValidateAbsoluteURL(
			"URL Locator",
			l.URL,
		); err != nil {
			return err
		}
		if l.Integrity != "" {
			if err := cryptoutil.ValidateDigest(
				cryptoutil.Digest(l.Integrity),
			); err != nil {
				return fmt.Errorf("URL Locator integrity: %w", err)
			}
		}
		return requireEmptyFields(
			"path", l.Path,
			"repository", l.Repository,
			"revision", l.Revision,
			"manager", string(l.Manager),
			"package", l.Package,
			"version", l.Version,
			"registry", l.Registry,
			"command", l.Command,
		)

	case LocatorKindGit:
		if err := ValidateAbsoluteURL(
			"Git repository",
			l.Repository,
		); err != nil {
			return err
		}
		if err := validateOptionalText(
			"Git revision",
			l.Revision,
			basespec.MaxVersionBytes,
		); err != nil {
			return err
		}
		if l.Path != "" {
			if err := ValidatePortableLocatorPath(
				"Git path",
				l.Path,
				true,
			); err != nil {
				return err
			}
		}
		return requireEmptyFields(
			"url", l.URL,
			"integrity", l.Integrity,
			"manager", string(l.Manager),
			"package", l.Package,
			"version", l.Version,
			"registry", l.Registry,
			"command", l.Command,
		)

	case LocatorKindPackage:
		if err := l.Manager.Validate(); err != nil {
			return err
		}
		if err := validateRequiredText(
			"package Locator package",
			l.Package,
			basespec.MaxURIBytes,
		); err != nil {
			return err
		}
		if err := validateOptionalText(
			"package Locator version",
			l.Version,
			basespec.MaxVersionBytes,
		); err != nil {
			return err
		}
		if l.Registry != "" {
			if err := ValidateAbsoluteURL(
				"package Locator registry",
				l.Registry,
			); err != nil {
				return err
			}
		}
		if l.Path != "" {
			if err := ValidatePortableLocatorPath(
				"package Locator path",
				l.Path,
				true,
			); err != nil {
				return err
			}
		}
		return requireEmptyFields(
			"url", l.URL,
			"integrity", l.Integrity,
			"repository", l.Repository,
			"revision", l.Revision,
			"command", l.Command,
		)

	case LocatorKindCommand:
		if err := validateRequiredText(
			"command Locator command",
			l.Command,
			basespec.MaxURIBytes,
		); err != nil {
			return err
		}
		return requireEmptyFields(
			"path", l.Path,
			"url", l.URL,
			"integrity", l.Integrity,
			"repository", l.Repository,
			"revision", l.Revision,
			"manager", string(l.Manager),
			"package", l.Package,
			"version", l.Version,
			"registry", l.Registry,
		)

	default:
		return fmt.Errorf(
			"%w: unsupported Locator kind %q",
			basespec.ErrInvalid,
			l.Kind,
		)
	}
}

func (m PackageManager) Validate() error {
	switch m {
	case PackageManagerNPM,
		PackageManagerPyPI,
		PackageManagerCargo,
		PackageManagerGo,
		PackageManagerMaven,
		PackageManagerNuGet,
		PackageManagerOCI:
		return nil
	default:
		return fmt.Errorf(
			"%w: unsupported package manager %q",
			basespec.ErrInvalid,
			m,
		)
	}
}

func ValidatePortableLocatorPath(
	label string,
	value string,
	allowRoot bool,
) error {
	if after, ok := strings.CutPrefix(value, "./"); ok {
		value = after
	}
	if value == "" {
		return fmt.Errorf(
			"%w: %s is required",
			basespec.ErrInvalid,
			label,
		)
	}
	return basespec.Locator(value).ValidatePortable(allowRoot)
}

func ValidateAbsoluteURL(
	label string,
	value string,
) error {
	if err := validateRequiredText(
		label,
		value,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil ||
		!parsed.IsAbs() ||
		parsed.Scheme == "" ||
		strings.EqualFold(parsed.Scheme, "file") {
		return fmt.Errorf(
			"%w: %s must be an absolute non-file URL",
			basespec.ErrInvalid,
			label,
		)
	}
	return nil
}

func validateScalarLocator(value string) error {
	if err := validateRequiredText(
		"scalar Locator",
		value,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf(
			"%w: invalid scalar Locator: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if parsed.Scheme != "" {
		if strings.EqualFold(parsed.Scheme, "file") {
			return fmt.Errorf(
				"%w: scalar Locator cannot use file URI",
				basespec.ErrInvalid,
			)
		}
		return nil
	}

	pathValue, _, _ := strings.Cut(value, "#")
	return ValidatePortableLocatorPath(
		"scalar path Locator",
		pathValue,
		true,
	)
}

// ResolveSourceRelativePathLocator resolves a local portable declaration
// locator against the source entry that contains the declaration. It returns
// only strict basespec.Locator values suitable for an already opened Source.
//
// URL, Git, package, and command locators deliberately remain the
// responsibility of an external Locator Resolver.
func ResolveSourceRelativePathLocator(
	locator Locator,
	declarationLocator basespec.Locator,
) (basespec.Locator, error) {
	if err := locator.Validate(); err != nil {
		return "", err
	}
	if err := declarationLocator.Validate(false); err != nil {
		return "", err
	}

	var relative string
	switch locator.Kind {
	case "":
		parsed, err := url.Parse(locator.Scalar)
		if err != nil {
			return "", err
		}
		if parsed.Scheme != "" {
			return "", fmt.Errorf(
				"%w: external locator requires a Locator Resolver",
				basespec.ErrLocatorUnresolved,
			)
		}
		relative, _, _ = strings.Cut(locator.Scalar, "#")

	case LocatorKindPath:
		relative = locator.Path

	default:
		return "", fmt.Errorf(
			"%w: locator kind %q requires a Locator Resolver",
			basespec.ErrLocatorUnresolved,
			locator.Kind,
		)
	}

	relative = strings.TrimPrefix(relative, "./")
	if relative == "" {
		return "", fmt.Errorf(
			"%w: local declaration locator is empty",
			basespec.ErrInvalid,
		)
	}

	resolved := basespec.Locator(relative)
	if parent := path.Dir(string(declarationLocator)); parent != "." {
		resolved = basespec.Locator(path.Join(parent, relative))
	}
	if err := resolved.ValidatePortable(true); err != nil {
		return "", err
	}
	return resolved, nil
}

func validateRequiredText(
	label string,
	value string,
	maximum int,
) error {
	if value == "" ||
		len(value) > maximum ||
		!utf8.ValidString(value) ||
		strings.TrimSpace(value) != value {
		return fmt.Errorf(
			"%w: %s must be non-empty, bounded, valid UTF-8, and trimmed",
			basespec.ErrInvalid,
			label,
		)
	}
	for _, character := range value {
		if character == 0 || unicode.IsControl(character) {
			return fmt.Errorf(
				"%w: %s contains a control character",
				basespec.ErrInvalid,
				label,
			)
		}
	}
	return nil
}

func validateOptionalText(
	label string,
	value string,
	maximum int,
) error {
	if value == "" {
		return nil
	}
	return validateRequiredText(label, value, maximum)
}

func requireEmptyFields(values ...string) error {
	for index := 0; index+1 < len(values); index += 2 {
		if values[index+1] == "" {
			continue
		}
		return fmt.Errorf(
			"%w: Locator field %q is not valid for this Locator kind",
			basespec.ErrInvalid,
			values[index],
		)
	}
	return nil
}
