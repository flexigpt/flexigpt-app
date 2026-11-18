package workflowruntime

import (
	"context"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

// WorkflowCapabilityFactRequest is a narrow target-domain request. It is
// evaluated during execution preparation, before Runtime creates WorkItems.
type WorkflowCapabilityFactRequest struct {
	TargetName   string
	TargetKind   sessionv3Runtime.WorkflowTargetKind
	Artifact     *sessionv3Runtime.ArtifactPin
	Direct       *sessionv3Runtime.DirectCapabilityPlan
	Requirements []sessionv3Runtime.WorkflowCapabilityRequirement
}

// WorkflowCapabilityFactProvider is implemented by an owning domain adapter
// when that domain supports named callable capability requirements.
type WorkflowCapabilityFactProvider interface {
	ResolveWorkflowCapabilityFacts(
		ctx context.Context,
		request WorkflowCapabilityFactRequest,
	) ([]sessionv3Runtime.WorkflowCapabilityFact, error)
}
