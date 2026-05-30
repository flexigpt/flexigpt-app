package runtime

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func TestDiscoveryPageTokenEncodeDecodeAndPaginationEdges(t *testing.T) {
	t.Run("round trip token", func(t *testing.T) {
		in := spec.MCPDiscoveryPageToken{
			ServerID:       "server",
			SnapshotDigest: "digest",
			Kind:           discoveryPageKindTools,
			PageSize:       25,
			Index:          25,
		}

		raw, err := encodeDiscoveryPageToken(in)
		if err != nil {
			t.Fatalf("encodeDiscoveryPageToken: %v", err)
		}
		got, err := decodeDiscoveryPageToken(raw)
		if err != nil {
			t.Fatalf("decodeDiscoveryPageToken: %v", err)
		}
		if got != in {
			t.Fatalf("round trip = %#v, want %#v", got, in)
		}
	})

	t.Run("decode token failures", func(t *testing.T) {
		if _, err := decodeDiscoveryPageToken("not-base64"); err == nil {
			t.Fatalf("decodeDiscoveryPageToken succeeded, want error")
		}

		raw := base64.URLEncoding.EncodeToString([]byte("not-json"))
		if _, err := decodeDiscoveryPageToken(raw); err == nil {
			t.Fatalf("decodeDiscoveryPageToken invalid json succeeded, want error")
		}
	})

	items := func(n int) []int {
		out := make([]int, n)
		for i := range out {
			out[i] = i + 1
		}
		return out
	}

	digest := "snapshot-digest"

	tests := []struct {
		name            string
		serverID        spec.MCPServerID
		kind            string
		items           []int
		pageSize        int
		pageToken       string
		want            []int
		wantNext        bool
		wantErrContains string
	}{
		{
			name:     "empty items",
			serverID: "server",
			kind:     discoveryPageKindTools,
			items:    nil,
			pageSize: 0,
			want:     []int{},
			wantNext: false,
		},
		{
			name:     "default page size",
			serverID: "server",
			kind:     discoveryPageKindTools,
			items:    items(30),
			pageSize: 0,
			want:     items(25),
			wantNext: true,
		},
		{
			name:     "clamped max page size",
			serverID: "server",
			kind:     discoveryPageKindTools,
			items:    items(300),
			pageSize: spec.MaxMCPServerPageSize + 10,
			want:     items(spec.MaxMCPServerPageSize),
			wantNext: true,
		},
		{
			name:     "exact end no next",
			serverID: "server",
			kind:     discoveryPageKindTools,
			items:    items(3),
			pageSize: 3,
			want:     items(3),
			wantNext: false,
		},
		{
			name:     "continue with token",
			serverID: "server",
			kind:     discoveryPageKindTools,
			items:    items(3),
			pageSize: 2,
			pageToken: mustDiscoveryPageTokenForTest(t, spec.MCPDiscoveryPageToken{
				ServerID:       "server",
				SnapshotDigest: digest,
				Kind:           discoveryPageKindTools,
				PageSize:       2,
				Index:          2,
			}),
			want:     []int{3},
			wantNext: false,
		},
		{
			name:            "bad token",
			serverID:        "server",
			kind:            discoveryPageKindTools,
			items:           items(3),
			pageSize:        2,
			pageToken:       "bad-token",
			wantErrContains: "bad pageToken",
		},
		{
			name:     "stale server",
			serverID: "server-a",
			kind:     discoveryPageKindTools,
			items:    items(3),
			pageSize: 2,
			pageToken: mustDiscoveryPageTokenForTest(
				t,
				spec.MCPDiscoveryPageToken{
					ServerID:       "server-b",
					SnapshotDigest: digest,
					Kind:           discoveryPageKindTools,
					PageSize:       2,
					Index:          2,
				},
			),
			wantErrContains: "stale pageToken",
		},
		{
			name:     "stale digest",
			serverID: "server",
			kind:     discoveryPageKindTools,
			items:    items(3),
			pageSize: 2,
			pageToken: mustDiscoveryPageTokenForTest(
				t,
				spec.MCPDiscoveryPageToken{
					ServerID:       "server",
					SnapshotDigest: "other",
					Kind:           discoveryPageKindTools,
					PageSize:       2,
					Index:          2,
				},
			),
			wantErrContains: "stale pageToken",
		},
		{
			name:     "stale kind",
			serverID: "server",
			kind:     discoveryPageKindResources,
			items:    items(3),
			pageSize: 2,
			pageToken: mustDiscoveryPageTokenForTest(
				t,
				spec.MCPDiscoveryPageToken{
					ServerID:       "server",
					SnapshotDigest: digest,
					Kind:           discoveryPageKindTools,
					PageSize:       2,
					Index:          2,
				},
			),
			wantErrContains: "stale pageToken",
		},
		{
			name:     "stale page size",
			serverID: "server",
			kind:     discoveryPageKindTools,
			items:    items(3),
			pageSize: 2,
			pageToken: mustDiscoveryPageTokenForTest(
				t,
				spec.MCPDiscoveryPageToken{
					ServerID:       "server",
					SnapshotDigest: digest,
					Kind:           discoveryPageKindTools,
					PageSize:       0,
					Index:          2,
				},
			),
			wantErrContains: "stale pageToken",
		},
		{
			name:     "index out of range",
			serverID: "server",
			kind:     discoveryPageKindTools,
			items:    items(3),
			pageSize: 2,
			pageToken: mustDiscoveryPageTokenForTest(
				t,
				spec.MCPDiscoveryPageToken{
					ServerID:       "server",
					SnapshotDigest: digest,
					Kind:           discoveryPageKindTools,
					PageSize:       2,
					Index:          99,
				},
			),
			wantErrContains: "stale pageToken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, next, err := paginateDiscoveryItems(tt.serverID, digest, tt.kind, tt.items, tt.pageSize, tt.pageToken)
			if tt.wantErrContains != "" {
				if err == nil {
					t.Fatalf("paginateDiscoveryItems succeeded, want error containing %q", tt.wantErrContains)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("err = %q, want substring %q", err.Error(), tt.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("paginateDiscoveryItems: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("got[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
			if (next != nil) != tt.wantNext {
				t.Fatalf("next present = %v, want %v", next != nil, tt.wantNext)
			}
		})
	}
}

func mustDiscoveryPageTokenForTest(t *testing.T, tok spec.MCPDiscoveryPageToken) string {
	t.Helper()
	raw, err := encodeDiscoveryPageToken(tok)
	if err != nil {
		t.Fatalf("encodeDiscoveryPageToken: %v", err)
	}
	return raw
}
