package store

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func requireMCPBundleID(bundleID bundleitemutils.BundleID) error {
	if bundleID == "" {
		return fmt.Errorf("%w: bundleID required", spec.ErrMCPInvalidRequest)
	}
	return nil
}

func requireMCPBundleServerIDs(bundleID bundleitemutils.BundleID, serverID spec.MCPServerID) error {
	switch {
	case bundleID == "" && serverID == "":
		return fmt.Errorf("%w: bundleID and serverID required", spec.ErrMCPInvalidRequest)
	case bundleID == "":
		return fmt.Errorf("%w: bundleID required", spec.ErrMCPInvalidRequest)
	case serverID == "":
		return fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	default:
		return nil
	}
}
