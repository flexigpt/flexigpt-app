package spec

import "errors"

var (
	ErrInvalidWorkspace           = errors.New("workspace: invalid")
	ErrNotWorkspace               = errors.New("workspace: collection is not a Workspace")
	ErrPrimarySourceImmutable     = errors.New("workspace: primary source is immutable")
	ErrReferenceUnresolved        = errors.New("workspace: reference unresolved")
	ErrWorkspaceDefinitionInvalid = errors.New("workspace: descriptor invalid")
)
