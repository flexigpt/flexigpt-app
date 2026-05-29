package sdkclient

import "net/http"

type headerRoundTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (h headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := h.base
	if base == nil {
		base = http.DefaultTransport
	}

	cloned := req.Clone(req.Context())
	if cloned.Header == nil {
		cloned.Header = make(http.Header)
	}

	for k, v := range h.headers {
		// Let SDK-required transport headers win if already present.
		if cloned.Header.Get(k) == "" {
			cloned.Header.Set(k, v)
		}
	}

	return base.RoundTrip(cloned)
}
