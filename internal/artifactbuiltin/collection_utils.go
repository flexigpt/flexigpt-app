package artifactbuiltin

type ContentRef struct {
	Locator            string  `json:"locator,omitempty"`
	URI                string  `json:"uri,omitempty"`
	SubresourceLocator string  `json:"subresourceLocator,omitempty"`
	Digest             *string `json:"digest,omitempty"`
	MediaType          string  `json:"mediaType,omitempty"`
	Role               string  `json:"role,omitempty"`
}

func (v ContentRef) Clone() ContentRef {
	output := v
	if v.Digest != nil {
		digest := *v.Digest
		output.Digest = &digest
	}
	return output
}
