package internal

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
)

type Synchronization struct {
	Creates     []artifactModel.Artifact
	Updates     []artifact.SourceStateUpdate
	Diagnostics []diagnostic.Diagnostic
}

func (s Synchronization) Clone() Synchronization {
	output := s
	output.Creates = make([]artifactModel.Artifact, len(s.Creates))
	for index, value := range s.Creates {
		output.Creates[index] = value.Clone()
	}
	output.Updates = make(
		[]artifact.SourceStateUpdate,
		len(s.Updates),
	)
	for index, value := range s.Updates {
		output.Updates[index] = value.Clone()
	}
	output.Diagnostics = diagnostic.Clone(s.Diagnostics)
	return output
}
