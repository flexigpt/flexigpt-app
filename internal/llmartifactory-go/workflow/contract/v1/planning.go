package workflowv1

import "sort"

// StaticTargetSetMember identifies one statically declared target-set member.
// Resolver-backed target sets intentionally produce no values because their
// membership is not known from source planning.
type StaticTargetSetMember struct {
	TargetSet string `json:"targetSet"`
	Target    string `json:"target"`
}

// StaticTargetSetMembers returns explicit target-set membership in stable
// source order. It does not resolve targets, invoke a target-set resolver, or
// apply runtime target-selection policy.
func (v WorkflowDocument) StaticTargetSetMembers() []StaticTargetSetMember {
	setNames := make([]string, 0, len(v.TargetSets))
	for name := range v.TargetSets {
		setNames = append(setNames, name)
	}
	sort.Strings(setNames)

	output := make([]StaticTargetSetMember, 0)
	for _, setName := range setNames {
		set := v.TargetSets[setName]
		if set.Resolved != nil {
			continue
		}

		members := append([]string(nil), set.Members...)
		sort.Strings(members)

		seen := make(map[string]struct{}, len(members))
		for _, target := range members {
			if _, duplicate := seen[target]; duplicate {
				continue
			}
			seen[target] = struct{}{}
			output = append(output, StaticTargetSetMember{
				TargetSet: setName,
				Target:    target,
			})
		}
	}
	return output
}
