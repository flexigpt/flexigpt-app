package policy

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// Effective is the deterministic result of composing MCP policies. It is a
// domain value, not a Store value. Store persists policy documents, while
// Runtime consumes the resolved policy projection through Aggregate.
type Effective struct {
	Body      MCPPolicy         `json:"body"`
	Conflicts map[string]string `json:"conflicts,omitempty"`
	Digest    cryptoutil.Digest `json:"digest"`
}

func Compose(
	baseline MCPPolicy,
	policies ...MCPPolicy,
) (Effective, error) {
	composed, err := ComposeMCPPolicy(baseline, policies...)
	if err != nil {
		return Effective{}, err
	}

	digest, err := effectiveDigest(composed.Body, composed.Conflicts)
	if err != nil {
		return Effective{}, err
	}

	return Effective{
		Body:      CloneMCPPolicy(composed.Body),
		Conflicts: maps.Clone(composed.Conflicts),
		Digest:    digest,
	}, nil
}

func (e Effective) Validate() error {
	if err := ValidateMCPPolicy(e.Body); err != nil {
		return fmt.Errorf("%w: invalid effective MCP policy: %w", ErrInvalid, err)
	}
	if err := cryptoutil.ValidateDigest(e.Digest); err != nil {
		return err
	}

	calculated, err := effectiveDigest(e.Body, e.Conflicts)
	if err != nil {
		return err
	}
	if calculated != e.Digest {
		return fmt.Errorf("%w: effective MCP policy digest mismatch", ErrInvalid)
	}
	return nil
}

func effectiveDigest(
	body MCPPolicy,
	conflicts map[string]string,
) (cryptoutil.Digest, error) {
	raw, err := json.Marshal(struct {
		Body      MCPPolicy         `json:"body"`
		Conflicts map[string]string `json:"conflicts,omitempty"`
	}{
		Body:      body,
		Conflicts: conflicts,
	})
	if err != nil {
		return "", err
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return "", err
	}
	return cryptoutil.DigestBytes(canonical), nil
}
