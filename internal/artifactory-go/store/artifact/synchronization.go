package artifact

import (
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
)

// Synchronization is Artifact-owned publication intent derived from admitted
// Ingest observations and current source-backed Artifact state.
type Synchronization struct {
	Creates     []artifactModel.Artifact
	Updates     []SourceStateUpdate
	Diagnostics []diagnostic.Diagnostic
}

func (s Synchronization) Clone() Synchronization {
	output := s
	output.Creates = make([]artifactModel.Artifact, len(s.Creates))
	for index, value := range s.Creates {
		output.Creates[index] = value.Clone()
	}
	output.Updates = make([]SourceStateUpdate, len(s.Updates))
	for index, value := range s.Updates {
		output.Updates[index] = value.Clone()
	}
	output.Diagnostics = diagnostic.Clone(s.Diagnostics)
	return output
}
