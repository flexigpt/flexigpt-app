package runtime

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	discoveryPageKindTools             = "tools"
	discoveryPageKindResources         = "resources"
	discoveryPageKindResourceTemplates = "resourceTemplates"
	discoveryPageKindPrompts           = "prompts"
)

func paginateDiscoveryItems[T any](
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	snapshotDigest string,
	kind string,
	items []T,
	pageSize int,
	pageToken string,
) (out []T, next *string, err error) {
	start := 0

	if pageToken != "" {
		tok, err := decodeDiscoveryPageToken(pageToken)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: bad pageToken", spec.ErrMCPInvalidRequest)
		}
		if tok.BundleID != bundleID || tok.ServerID != serverID ||
			tok.SnapshotDigest != snapshotDigest || tok.Kind != kind {
			return nil, nil, fmt.Errorf("%w: stale pageToken", spec.ErrMCPInvalidRequest)
		}
		if tok.PageSize <= 0 || tok.PageSize > spec.MaxMCPServerPageSize {
			return nil, nil, fmt.Errorf("%w: stale pageToken", spec.ErrMCPInvalidRequest)
		}
		start = tok.Index
		pageSize = tok.PageSize
		if start < 0 || start > len(items) {
			return nil, nil, fmt.Errorf("%w: stale pageToken", spec.ErrMCPInvalidRequest)
		}
	} else {
		if pageSize <= 0 {
			pageSize = spec.DefaultMCPPageSize
		}
		if pageSize > spec.MaxMCPServerPageSize {
			pageSize = spec.MaxMCPServerPageSize
		}
	}

	end := min(start+pageSize, len(items))
	out = append([]T(nil), items[start:end]...)
	if out == nil {
		out = []T{}
	}

	if end < len(items) {
		raw, err := encodeDiscoveryPageToken(spec.MCPDiscoveryPageToken{
			BundleID:       bundleID,
			ServerID:       serverID,
			SnapshotDigest: snapshotDigest,
			Kind:           kind,
			PageSize:       pageSize,
			Index:          end,
		})
		if err != nil {
			return nil, nil, err
		}
		next = &raw
	}

	return out, next, nil
}

func encodeDiscoveryPageToken(tok spec.MCPDiscoveryPageToken) (string, error) {
	raw, err := json.Marshal(tok)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(raw), nil
}

func decodeDiscoveryPageToken(token string) (spec.MCPDiscoveryPageToken, error) {
	raw, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return spec.MCPDiscoveryPageToken{}, err
	}
	var tok spec.MCPDiscoveryPageToken
	if err := json.Unmarshal(raw, &tok); err != nil {
		return spec.MCPDiscoveryPageToken{}, err
	}
	return tok, nil
}
