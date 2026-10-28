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

// IsContentCurrent reports whether the physical Source snapshot still matches
// the generation admitted by the latest refresh.
//
// Source revision, discovery, and decoder changes affect declaration
// admission. They do not by themselves change source bytes. Callers that
// need only verified source content should use this after InspectSource;
// callers that require declaration re-admission should continue to use
// IsCurrent.
func (i Inspection) IsContentCurrent() bool {
	return !i.SourceGenerationChanged
}
