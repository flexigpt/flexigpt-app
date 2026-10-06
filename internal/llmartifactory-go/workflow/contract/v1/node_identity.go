package workflowv1

import (
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

const workflowNodeIdentityDigestLength = 16

// NodeRelationshipSegment converts a Workflow node ID into one portable
// relationship path segment. Workflow owns this because Workflow owns node
// identity and node-to-member relationship semantics.
func NodeRelationshipSegment(
	id string,
) (string, error) {
	if err := declaration.ValidateWorkflowID(
		"Workflow node ID",
		id,
	); err != nil {
		return "", err
	}

	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes([]byte(id))),
		cryptoutil.DigestSHA256Prefix,
	)
	if len(digest) < workflowNodeIdentityDigestLength {
		return "", fmt.Errorf(
			"%w: Workflow node identity digest is invalid",
			spec.ErrInvalid,
		)
	}

	segment := "node-" + digest[:workflowNodeIdentityDigestLength]
	if err := spec.SubresourceLocator(segment).Validate(); err != nil {
		return "", err
	}
	return segment, nil
}
