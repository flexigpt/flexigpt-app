package definition

import definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"

// Admission is the narrow Definition canonicalization capability used at a
// decoder or trusted-generated-document boundary.
type Admission interface {
	Admit(def definitionModel.Definition) (definitionModel.Definition, error)
}

type admissionService struct{}

func (admissionService) Admit(value definitionModel.Definition) (definitionModel.Definition, error) {
	return Admit(value)
}

// NewAdmission returns Definition's stateless admission capability without
// exposing persistence reads or repository mechanics.
func NewAdmission() Admission { return admissionService{} }
