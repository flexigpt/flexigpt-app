package workflowv1

import "strings"

// RootDeclarationPath is the canonical root declaration path. Contract
// admission owns the grammar of every descendant path.
const RootDeclarationPath = "root"

// JoinDeclarationPath creates one canonical declaration path. Callers pass
// trusted declaration segments. Runtime receives the resulting path as opaque
// diagnostic and authorization identity.
func JoinDeclarationPath(
	base string,
	segments ...string,
) string {
	parts := make([]string, 0, len(segments)+1)

	appendValue := func(value string) {
		for part := range strings.SplitSeq(
			strings.Trim(value, "/"),
			"/",
		) {
			if part != "" {
				parts = append(parts, part)
			}
		}
	}

	if strings.TrimSpace(base) == "" {
		appendValue(RootDeclarationPath)
	} else {
		appendValue(base)
	}
	for _, segment := range segments {
		appendValue(segment)
	}
	return strings.Join(parts, "/")
}

func DeclarationStepPath(
	blockPath string,
	stepID string,
) string {
	return JoinDeclarationPath(blockPath, "steps", stepID)
}
