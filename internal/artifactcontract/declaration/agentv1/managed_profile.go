package agentv1

import (
	_ "embed"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
)

const ManagedAgentImportProfileID = "managed-agent-import"

//go:embed agent-managed-restrictions.schema.json
var managedAgentRestrictionsSchema []byte

func ManagedAgentImportRestrictionsJSONSchema() []byte {
	return append([]byte(nil), managedAgentRestrictionsSchema...)
}

// ManagedAgentImportProfileDescriptor uses the existing Agent portable schema.
// It does not add a schema key, schema revision, declaration field, or codec.
func ManagedAgentImportProfileDescriptor() declaration.ManagedProfilePolicyDescriptor {
	return declaration.ManagedProfilePolicyDescriptor{
		ID:               ManagedAgentImportProfileID,
		DeclarationType:  AgentType,
		BaseSchemaJSON:   AgentJSONSchema(),
		RestrictionsJSON: ManagedAgentImportRestrictionsJSONSchema(),
	}
}
