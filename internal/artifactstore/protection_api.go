package artifactstore

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
)

func (a *API) IsProtectedRoot(rootID basespec.RootID) bool {
	return a != nil &&
		a.components != nil &&
		a.components.RootMutationPolicy() != nil &&
		a.components.RootMutationPolicy().IsProtectedRoot(rootID)
}

func (a *API) RequirePrivilegedInstaller(ctx context.Context) error {
	if err := a.check(ctx); err != nil {
		return err
	}
	return protection.RequirePrivilegedInstaller(ctx)
}
