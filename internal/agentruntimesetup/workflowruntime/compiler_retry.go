package workflowruntime

import (
	"bytes"
	"encoding/json"

	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

func workflowStepSupportsRetry(
	document workflowv1.WorkflowDocument,
	step workflowv1.Step,
) bool {
	switch {
	case step.Transform != nil,
		step.Set != nil,
		step.Wait != nil,
		step.Emit != nil:
		return true

	case step.Call != nil:
		return workflowCallSupportsRetry(document, *step.Call)

	default:
		return false
	}
}

func workflowCallSupportsRetry(
	document workflowv1.WorkflowDocument,
	call workflowv1.Call,
) bool {
	targetNames, known := workflowCallTargetNames(document, call)
	if !known || len(targetNames) == 0 {
		return false
	}

	for _, name := range targetNames {
		target, found := document.Targets[name]
		if !found {
			return false
		}
		switch target.Kind {
		case workflowv1.TargetTool,
			workflowv1.TargetService,
			workflowv1.TargetActivity:
		default:
			return false
		}
	}
	return true
}

func workflowCallTargetNames(
	document workflowv1.WorkflowDocument,
	call workflowv1.Call,
) ([]string, bool) {
	raw := bytes.TrimSpace(call.Target)
	var literal string
	if err := json.Unmarshal(raw, &literal); err == nil && literal != "" {
		return []string{literal}, true
	}

	if call.Allowed == "" {
		return nil, false
	}
	set, found := document.TargetSets[call.Allowed]
	if !found || set.Resolved != nil || len(set.Members) == 0 {
		return nil, false
	}

	output := make([]string, 0, len(set.Members))
	seen := make(map[string]struct{}, len(set.Members))
	for _, name := range set.Members {
		if _, duplicate := seen[name]; duplicate {
			continue
		}
		seen[name] = struct{}{}
		output = append(output, name)
	}
	return output, true
}
