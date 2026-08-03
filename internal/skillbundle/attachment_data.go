package skillbundle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const AttachmentDataSchemaVersion = "v1"

// AttachmentData is Collection-owned local mount configuration. It is never
// portable Collection payload data. ExpectedMemberDigests keys are relative to
// DiscoveryRoot, not to the Source root.
type AttachmentData struct {
	SchemaVersion         string                                 `json:"schemaVersion"`
	DiscoveryRoot         basespec.Locator                       `json:"discoveryRoot"`
	ExpectedMemberDigests map[basespec.Locator]cryptoutil.Digest `json:"expectedMemberDigests,omitempty"`
}

func NewAttachmentData(
	discoveryRoot basespec.Locator,
	expectedMemberDigests map[basespec.Locator]cryptoutil.Digest,
) (AttachmentData, error) {
	if discoveryRoot == "" {
		discoveryRoot = "."
	}
	value := AttachmentData{
		SchemaVersion:         AttachmentDataSchemaVersion,
		DiscoveryRoot:         discoveryRoot,
		ExpectedMemberDigests: maps.Clone(expectedMemberDigests),
	}
	if err := value.Validate(); err != nil {
		return AttachmentData{}, err
	}
	return value, nil
}

func (d AttachmentData) Clone() AttachmentData {
	output := d
	output.ExpectedMemberDigests = maps.Clone(d.ExpectedMemberDigests)
	return output
}

func (d AttachmentData) Validate() error {
	if d.SchemaVersion != AttachmentDataSchemaVersion {
		return fmt.Errorf(
			"%w: unsupported Skill attachment data schema %q",
			basespec.ErrInvalid,
			d.SchemaVersion,
		)
	}
	if err := basespec.ValidateLocator(d.DiscoveryRoot, true); err != nil {
		return err
	}
	if len(d.ExpectedMemberDigests) > basespec.MaxDiscoveryCandidates {
		return fmt.Errorf(
			"%w: expected Skill member digest count exceeds limit",
			basespec.ErrInvalid,
		)
	}
	for member, digest := range d.ExpectedMemberDigests {
		if err := basespec.ValidatePortableLocator(member, false); err != nil {
			return err
		}
		if err := cryptoutil.ValidateDigest(digest); err != nil {
			return err
		}
	}
	return nil
}

func EncodeAttachmentData(
	value AttachmentData,
) (json.RawMessage, error) {
	value = value.Clone()
	if err := value.Validate(); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

func DecodeAttachmentData(
	raw json.RawMessage,
) (AttachmentData, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return AttachmentData{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()

	var value AttachmentData
	if err := decoder.Decode(&value); err != nil {
		return AttachmentData{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("skill attachment data contains trailing JSON values")
		}
		return AttachmentData{}, err
	}
	if err := value.Validate(); err != nil {
		return AttachmentData{}, err
	}
	return value.Clone(), nil
}

// SourceExpectedContentDigests maps Collection-relative expected members into
// Source-relative locators used by generic Artifact Store discovery.
func (d AttachmentData) SourceExpectedContentDigests() (
	map[basespec.Locator]cryptoutil.Digest,
	error,
) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	if len(d.ExpectedMemberDigests) == 0 {
		//nolint:nilnil // Explicit.
		return nil, nil
	}
	output := make(
		map[basespec.Locator]cryptoutil.Digest,
		len(d.ExpectedMemberDigests),
	)
	for member, digest := range d.ExpectedMemberDigests {
		locator := basespec.Locator(
			path.Join(string(d.DiscoveryRoot), string(member)),
		)
		if err := basespec.ValidateLocator(locator, false); err != nil {
			return nil, err
		}
		output[locator] = digest
	}
	return output, nil
}
