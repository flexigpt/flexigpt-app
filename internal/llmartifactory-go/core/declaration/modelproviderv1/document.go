package modelproviderv1

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

const (
	ModelProviderType          = declaration.TypeModelProvider
	ModelProviderSchemaID      = "artifact.model-provider.v1"
	ModelProviderSchemaVersion = declaration.SchemaVersionV1
)

//go:embed model-provider-v1.schema.json
var schemaJSON []byte

var compiledModelProviderSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var ModelProviderSchemaKey = schemaModel.ArtifactKey(
	artifactModel.ArtifactKind(ModelProviderType),
	schemaModel.SchemaID(ModelProviderSchemaID),
	ModelProviderSchemaVersion,
)

// ProviderDocument is a source-backed model provider declaration.
//
// Defaults, capabilities, and adapterParameters remain canonical portable JSON
// objects. Their typed runtime projection belongs to the store and the external inference adapter, not to this contract
// package.
type ProviderDocument struct {
	declaration.Header

	Adapter           string                             `json:"adapter"`
	Connection        json.RawMessage                    `json:"connection,omitempty"`
	Authentication    json.RawMessage                    `json:"authentication,omitempty"`
	DefaultModel      *declaration.ArtifactNameReference `json:"defaultModel,omitempty"`
	Defaults          json.RawMessage                    `json:"defaults,omitempty"`
	Capabilities      json.RawMessage                    `json:"capabilities,omitempty"`
	AdapterParameters json.RawMessage                    `json:"adapterParameters,omitempty"`
}

func ModelProviderJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeModelProviderJSON(raw []byte) (ProviderDocument, error) {
	var value ProviderDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledModelProviderSchema,
		&value,
	); err != nil {
		return ProviderDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return ProviderDocument{}, err
	}
	return value, nil
}

func DecodeModelProviderEntry(
	entry declaration.Entry,
) (ProviderDocument, error) {
	var value ProviderDocument
	if err := declaration.DecodeEntryDocumentInto(
		entry,
		compiledModelProviderSchema,
		&value,
	); err != nil {
		return ProviderDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return ProviderDocument{}, err
	}
	return value, nil
}

func (v ProviderDocument) Clone() (ProviderDocument, error) {
	output := v
	output.Header = v.Header.Clone()
	output.Connection = cloneProviderRaw(v.Connection)
	output.Authentication = cloneProviderRaw(v.Authentication)
	output.Defaults = cloneProviderRaw(v.Defaults)
	output.Capabilities = cloneProviderRaw(v.Capabilities)
	output.AdapterParameters = cloneProviderRaw(v.AdapterParameters)
	if v.DefaultModel != nil {
		value := v.DefaultModel.Clone()
		output.DefaultModel = &value
	}
	return output, nil
}

func (v ProviderDocument) Canonicalize() (ProviderDocument, error) {
	return v.Clone()
}

func (v ProviderDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

func (v ProviderDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return declaration.DocumentDigest(v)
}

func (v ProviderDocument) Validate() error {
	if err := declaration.ValidateDocument(
		compiledModelProviderSchema,
		v,
	); err != nil {
		return fmt.Errorf("model provider schema: %w", err)
	}
	return v.validateFields()
}

func (v ProviderDocument) ValidateEntry() error {
	return v.Validate()
}

func (v ProviderDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: ModelProviderType,
		RequireName:  true,
	}); err != nil {
		return err
	}
	if v.Locator != nil {
		return fmt.Errorf(
			"%w: Model Provider declarations do not support locator",
			spec.ErrInvalid,
		)
	}
	if err := spec.ValidateIdentifier(
		"Model Provider adapter",
		v.Adapter,
		spec.MaxKindBytes,
	); err != nil {
		return err
	}
	if err := validateConnection(v.Connection); err != nil {
		return err
	}
	if err := validateAuthentication(v.Authentication); err != nil {
		return err
	}
	if v.DefaultModel != nil {
		if err := v.DefaultModel.Validate(); err != nil {
			return fmt.Errorf("model provider defaultModel: %w", err)
		}
	}
	if err := validateProviderPatchObject(
		"model provider defaults",
		v.Defaults,
	); err != nil {
		return err
	}
	if err := validateProviderPatchObject(
		"model provider capabilities",
		v.Capabilities,
	); err != nil {
		return err
	}
	return validateProviderNoSecretObject(
		"model provider adapterParameters",
		v.AdapterParameters,
	)
}

func cloneProviderRaw(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}

func validateConnection(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		spec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return fmt.Errorf("model provider connection: %w", err)
	}

	var value struct {
		Origin  *string `json:"origin"`
		Path    *string `json:"path"`
		Headers struct {
			Set    map[string]string `json:"set"`
			Remove []string          `json:"remove"`
		} `json:"headers"`
	}
	if err := json.Unmarshal(canonical, &value); err != nil {
		return err
	}

	if value.Origin != nil && *value.Origin != "" {
		if err := declaration.ValidateAbsoluteURL(
			"Model Provider connection origin",
			*value.Origin,
		); err != nil {
			return err
		}
	}
	if value.Path != nil && *value.Path != "" {
		if err := validateProviderConnectionPath(*value.Path); err != nil {
			return err
		}
	}
	if err := validateProviderHeaderPatch(
		value.Headers.Set,
		value.Headers.Remove,
	); err != nil {
		return err
	}
	return validateProviderNoSecretObject(
		"Model Provider connection",
		canonical,
	)
}

func validateAuthentication(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		spec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return fmt.Errorf("model provider authentication: %w", err)
	}

	var value struct {
		Mode       string  `json:"mode"`
		HeaderName string  `json:"headerName"`
		Prefix     *string `json:"prefix"`
	}
	if err := json.Unmarshal(canonical, &value); err != nil {
		return err
	}

	switch value.Mode {
	case "none":
		if value.HeaderName != "" || value.Prefix != nil {
			return fmt.Errorf(
				"%w: authentication mode none cannot override header behavior",
				spec.ErrInvalid,
			)
		}
	case "apiKeyHeader", "bearerToken":
		if err := validateProviderHeaderName(
			"Model Provider authentication headerName",
			value.HeaderName,
		); err != nil {
			return err
		}
		if value.Prefix != nil &&
			strings.ContainsAny(*value.Prefix, "\r\n\x00") {
			return fmt.Errorf(
				"%w: Model Provider authentication prefix contains CR, LF, or NUL",
				spec.ErrInvalid,
			)
		}
	default:
		return fmt.Errorf(
			"%w: unsupported Model Provider authentication mode %q",
			spec.ErrInvalid,
			value.Mode,
		)
	}
	return nil
}

func validateProviderConnectionPath(value string) error {
	if len(value) > spec.MaxURIBytes ||
		!utf8.ValidString(value) ||
		strings.TrimSpace(value) != value ||
		!strings.HasPrefix(value, "/") ||
		strings.ContainsAny(value, "?#\r\n\x00") {
		return fmt.Errorf(
			"%w: Model Provider connection path is invalid",
			spec.ErrInvalid,
		)
	}
	return nil
}

func validateProviderHeaderPatch(
	set map[string]string,
	remove []string,
) error {
	removed := make(map[string]struct{}, len(remove))
	for index, name := range remove {
		if err := validateProviderHeaderName(
			fmt.Sprintf("Model Provider connection headers.remove[%d]", index),
			name,
		); err != nil {
			return err
		}
		key := strings.ToLower(name)
		if _, duplicate := removed[key]; duplicate {
			return fmt.Errorf(
				"%w: Model Provider connection removes header %q more than once",
				spec.ErrIdentityConflict,
				name,
			)
		}
		removed[key] = struct{}{}
	}

	for name, value := range set {
		if err := validateProviderHeaderName(
			"Model Provider connection headers.set",
			name,
		); err != nil {
			return err
		}
		if providerSecretHeaderName(name) {
			return fmt.Errorf(
				"%w: Model Provider static header %q can carry credentials",
				spec.ErrInvalid,
				name,
			)
		}
		if len(value) > spec.MaxURIBytes ||
			!utf8.ValidString(value) ||
			strings.ContainsAny(value, "\r\n\x00") {
			return fmt.Errorf(
				"%w: Model Provider static header %q has an invalid value",
				spec.ErrInvalid,
				name,
			)
		}
		if _, removed := removed[strings.ToLower(name)]; removed {
			return fmt.Errorf(
				"%w: Model Provider header %q is both set and removed",
				spec.ErrInvalid,
				name,
			)
		}
	}
	return nil
}

func validateProviderHeaderName(
	label string,
	value string,
) error {
	if value == "" || strings.TrimSpace(value) != value {
		return fmt.Errorf(
			"%w: %s is invalid",
			spec.ErrInvalid,
			label,
		)
	}
	for _, character := range value {
		if character >= 'A' && character <= 'Z' ||
			character >= 'a' && character <= 'z' ||
			character >= '0' && character <= '9' {
			continue
		}
		if strings.ContainsRune("!#$%&'*+-.^_`|~", character) {
			continue
		}
		return fmt.Errorf(
			"%w: %s contains invalid character %q",
			spec.ErrInvalid,
			label,
			character,
		)
	}
	return nil
}

func providerSecretHeaderName(value string) bool {
	normalized := strings.NewReplacer(
		"-",
		"",
		"_",
		"",
		".",
		"",
	).Replace(strings.ToLower(strings.TrimSpace(value)))

	return normalized == "authorization" ||
		normalized == "proxyauthorization" ||
		strings.Contains(normalized, "apikey") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret")
}

func validateProviderPatchObject(
	label string,
	raw json.RawMessage,
) error {
	if len(raw) == 0 {
		return nil
	}
	_, err := jsonutil.CanonicalizeObject(
		raw,
		spec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	return validateProviderNoSecretObject(label, raw)
}

func validateProviderNoSecretObject(
	label string,
	raw json.RawMessage,
) error {
	if len(raw) == 0 {
		return nil
	}

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		spec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}

	var value map[string]any
	if err := json.Unmarshal(canonical, &value); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	return rejectProviderSecretValues(label, value)
}

func rejectProviderSecretValues(
	label string,
	value any,
) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if providerSecretObjectKey(key) && child != nil {
				return fmt.Errorf(
					"%w: %s contains forbidden credential field %q",
					spec.ErrInvalid,
					label,
					key,
				)
			}
			if err := rejectProviderSecretValues(label, child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := rejectProviderSecretValues(label, child); err != nil {
				return err
			}
		}
	}
	return nil
}

func providerSecretObjectKey(value string) bool {
	normalized := strings.NewReplacer(
		"-",
		"",
		"_",
		"",
		".",
		"",
	).Replace(strings.ToLower(strings.TrimSpace(value)))

	switch normalized {
	case
		"apikey",
		"authorization",
		"accesstoken",
		"refreshtoken",
		"clientsecret",
		"password",
		"credential",
		"credentials",
		"secret":
		return true
	default:
		return strings.HasSuffix(normalized, "apikey") ||
			strings.HasSuffix(normalized, "clientsecret")
	}
}
