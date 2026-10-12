package attachment

import (
	"context"
	"strings"
)

// GenericRef preserves legacy handle-style references (PRs, doc indexes, etc.).
type GenericRef struct {
	Handle string `json:"handle"`

	OrigHandle string `json:"origHandle"`
}

func (ref *GenericRef) PopulateRef(ctx context.Context, replaceOrig bool) error {
	h := strings.TrimSpace(ref.Handle)
	if ref.OrigHandle == "" || replaceOrig {
		ref.OrigHandle = h
	}
	ref.Handle = h

	return nil
}

func (ref *GenericRef) IsModified() bool {
	if ref == nil {
		return false
	}
	if strings.TrimSpace(ref.OrigHandle) == "" {
		return false
	}

	if ref.Handle != ref.OrigHandle {
		return true
	}

	return false
}
