package sdkclient

import (
	"errors"
	"fmt"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

func TestDiscoveryListError(t *testing.T) {
	missing := fmt.Errorf("wrapped: %w", &jsonrpc.Error{
		Code:    jsonrpc.CodeMethodNotFound,
		Message: "method not found",
	})
	denied := errors.New("HTTP 401")

	tests := []struct {
		name       string
		err        error
		advertised bool
		wantError  bool
	}{
		{"success", nil, true, false},
		{"optional method absent", missing, false, false},
		{"advertised method absent", missing, true, true},
		{"optional auth failure", denied, false, true},
		{"advertised auth failure", denied, true, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := discoveryListError("tools/list", test.err, test.advertised)
			if (err != nil) != test.wantError {
				t.Fatalf("got %v; want error=%t", err, test.wantError)
			}
			if test.wantError && !errors.Is(err, test.err) {
				t.Fatalf("error lost its cause: %v", err)
			}
		})
	}
}
