package shareable

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type HTTPURLLoader http.Client

func (l *HTTPURLLoader) Load(url string) (any, error) {
	client := (*http.Client)(l)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", url, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%s returned status code %d", url, resp.StatusCode)
	}
	defer resp.Body.Close()

	return jsonschema.UnmarshalJSON(resp.Body)
}

func newHTTPURLLoader(insecure bool) *HTTPURLLoader {
	httpLoader := HTTPURLLoader(http.Client{
		Timeout: 15 * time.Second,
	})
	if insecure {
		httpLoader.Transport = &http.Transport{
			//nolint:gosec // InsecureSkipVerify.
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	return &httpLoader
}

type registeredCodec struct {
	codec  Codec
	schema *jsonschema.Schema
}

type Registry struct {
	codecs map[SchemaKey]registeredCodec
	keys   []SchemaKey
}

func NewRegistry(codecs ...Codec) (*Registry, error) {
	values := make(map[SchemaKey]registeredCodec, len(codecs))
	keys := make([]SchemaKey, 0, len(codecs))

	for _, codec := range codecs {
		if codec == nil {
			return nil, fmt.Errorf("%w: shareable schema codec is nil", basespec.ErrInvalid)
		}
		key := codec.Key()
		if err := key.Validate(); err != nil {
			return nil, err
		}
		compiled, err := compilePublishedJSONSchema(codec.JSONSchema())
		if err != nil {
			return nil, err
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf(
				"%w: duplicate shareable schema %q/%q/%q",
				basespec.ErrConflict,
				key.Kind,
				key.SchemaID,
				key.SchemaVersion,
			)
		}
		values[key] = registeredCodec{
			codec:  codec,
			schema: compiled,
		}
		keys = append(keys, key)
	}

	sort.Slice(keys, func(left, right int) bool {
		if keys[left].Kind != keys[right].Kind {
			return keys[left].Kind < keys[right].Kind
		}
		if keys[left].SchemaID != keys[right].SchemaID {
			return keys[left].SchemaID < keys[right].SchemaID
		}
		return keys[left].SchemaVersion < keys[right].SchemaVersion
	})
	return &Registry{codecs: values, keys: keys}, nil
}

func (r *Registry) Keys() []SchemaKey {
	if r == nil {
		return nil
	}
	return append([]SchemaKey(nil), r.keys...)
}

func (r *Registry) Canonicalize(
	ctx context.Context,
	raw []byte,
) (ParsedDocument, error) {
	if r == nil {
		return ParsedDocument{}, basespec.ErrClosed
	}
	if ctx == nil {
		return ParsedDocument{}, fmt.Errorf(
			"%w: shareable document context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return ParsedDocument{}, err
	}

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return ParsedDocument{}, err
	}

	var header struct {
		Kind          basespec.CollectionKind `json:"kind"`
		SchemaID      basespec.SchemaID       `json:"schemaID"`
		SchemaVersion string                  `json:"schemaVersion"`
	}
	if err := json.Unmarshal(canonical, &header); err != nil {
		return ParsedDocument{}, fmt.Errorf(
			"%w: decode shareable document header: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	key := SchemaKey{
		Entity:        EntityCollection,
		Kind:          header.Kind,
		SchemaID:      header.SchemaID,
		SchemaVersion: header.SchemaVersion,
	}
	if err := key.Validate(); err != nil {
		return ParsedDocument{}, err
	}
	registered, found := r.codecs[key]
	if !found {
		return ParsedDocument{}, fmt.Errorf(
			"%w: shareable collection schema %q/%q/%q",
			basespec.ErrUnsupported,
			key.Kind,
			key.SchemaID,
			key.SchemaVersion,
		)
	}

	if err := validateJSONSchemaInstance(registered.schema, canonical); err != nil {
		return ParsedDocument{}, err
	}

	value, err := registered.codec.Canonicalize(ctx, canonical)
	if err != nil {
		return ParsedDocument{}, err
	}
	if value.Key != key {
		return ParsedDocument{}, fmt.Errorf(
			"%w: shareable codec returned another schema key",
			basespec.ErrInvalid,
		)
	}
	if err := value.Validate(); err != nil {
		return ParsedDocument{}, err
	}
	return value.Clone(), nil
}

func compilePublishedJSONSchema(raw []byte) (*jsonschema.Schema, error) {
	var value struct {
		Schema string `json:"$schema"`
		ID     string `json:"$id"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf(
			"%w: published JSON Schema is invalid JSON: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if value.Schema == "" || value.ID == "" {
		return nil, fmt.Errorf(
			"%w: published JSON Schema requires $schema and $id",
			basespec.ErrInvalid,
		)
	}

	loader := jsonschema.SchemeURLLoader{
		"file":  jsonschema.FileLoader{},
		"https": newHTTPURLLoader(false),
	}

	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(loader)
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	if err := compiler.AddResource(value.ID, inst); err != nil {
		return nil, fmt.Errorf(
			"%w: register published JSON Schema %q: %w",
			basespec.ErrInvalid,
			value.ID,
			err,
		)
	}
	compiled, err := compiler.Compile(value.ID)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: compile published JSON Schema %q: %w",
			basespec.ErrInvalid,
			value.ID,
			err,
		)
	}
	return compiled, nil
}

func validateJSONSchemaInstance(
	schema *jsonschema.Schema,
	raw []byte,
) error {
	// Using jsonschema.UnmarshalJSON ensures proper float/integer typing alignment for v6.
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf(
			"%w: decode shareable JSON for schema validation: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if err := schema.Validate(value); err != nil {
		return fmt.Errorf(
			"%w: shareable document does not satisfy its JSON Schema: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	return nil
}
