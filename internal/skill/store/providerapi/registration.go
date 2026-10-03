package providerapi

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
)

type Registration struct {
	decoders []ingest.Decoder
}

func NewRegistration() (*Registration, error) {
	decoders, err := ingest.NormalizeDecoders([]ingest.Decoder{
		NewDecoder(),
	})
	if err != nil {
		return nil, err
	}

	return &Registration{
		decoders: decoders,
	}, nil
}

func (r *Registration) Decoders() []ingest.Decoder {
	if r == nil {
		return nil
	}
	return append([]ingest.Decoder(nil), r.decoders...)
}
