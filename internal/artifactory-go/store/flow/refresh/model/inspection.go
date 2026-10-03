package model

type Inspection struct {
	State                   State `json:"state"`
	SourceRevisionChanged   bool  `json:"sourceRevisionChanged"`
	DiscoveryChanged        bool  `json:"discoveryChanged"`
	DecoderChanged          bool  `json:"decoderChanged"`
	SourceGenerationChanged bool  `json:"sourceGenerationChanged"`
}

func (i Inspection) Validate() error {
	return i.State.Validate()
}

func (i Inspection) Clone() Inspection {
	output := i
	output.State = i.State.Clone()
	return output
}

func (i Inspection) IsCurrent() bool {
	return !i.SourceRevisionChanged &&
		!i.DiscoveryChanged &&
		!i.DecoderChanged &&
		!i.SourceGenerationChanged
}
