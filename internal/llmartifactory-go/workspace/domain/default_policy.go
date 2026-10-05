package domain

import (
	"bytes"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	workspacev1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/contract/v1"
)

// DefaultPolicy is application-supplied Workspace policy content. The
// Workspace family validates its immutable structure but does not select an
// embedded filesystem, provider key, policy identity, or application topology.
type DefaultPolicy struct {
	ID            string                        `json:"policyID"`
	Version       string                        `json:"policyVersion"`
	Digest        cryptoutil.Digest             `json:"policyDigest"`
	RawYAML       []byte                        `json:"-"`
	CanonicalJSON []byte                        `json:"-"`
	Document      workspacev1.WorkspaceDocument `json:"-"`
}

func (p DefaultPolicy) Clone() (DefaultPolicy, error) {
	document, err := p.Document.Clone()
	if err != nil {
		return DefaultPolicy{}, err
	}
	output := p
	output.RawYAML = append([]byte(nil), p.RawYAML...)
	output.CanonicalJSON = append([]byte(nil), p.CanonicalJSON...)
	output.Document = document
	return output, nil
}

func (p DefaultPolicy) Validate() error {
	if err := spec.ValidateRequiredText(
		"Workspace default policy ID",
		p.ID,
		spec.MaxLogicalNameBytes,
	); err != nil {
		return err
	}
	if err := spec.ValidateRequiredText(
		"Workspace default policy version",
		p.Version,
		spec.MaxVersionBytes,
	); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(p.Digest); err != nil {
		return err
	}
	if len(p.RawYAML) == 0 || len(p.CanonicalJSON) == 0 {
		return fmt.Errorf(
			"%w: Workspace default policy content is empty",
			spec.ErrInvalid,
		)
	}
	if err := p.Document.Validate(); err != nil {
		return err
	}
	if p.Document.Name != p.ID {
		return fmt.Errorf(
			"%w: Workspace default policy document identity is invalid",
			spec.ErrInvalid,
		)
	}

	canonical, err := p.Document.CanonicalJSON()
	if err != nil {
		return err
	}
	if !bytes.Equal(canonical, p.CanonicalJSON) ||
		cryptoutil.DigestBytes(canonical) != p.Digest {
		return fmt.Errorf(
			"%w: Workspace default policy bytes or digest do not match its document",
			spec.ErrDigestMismatch,
		)
	}
	return nil
}
