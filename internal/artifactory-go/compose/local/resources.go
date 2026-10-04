package local

import (
	"errors"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/sqlite"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/value"
)

// deploymentResources has one deployment owner. SQLite is opened locally;
// SecretValues remains caller-owned until local.Open successfully returns the
// assembled store handle.
type deploymentResources struct {
	metadata     *sqlite.Store
	secretValues value.ValueStore

	secretOwnershipTransferred bool
	closeOnce                  sync.Once
	closeErr                   error
}

func (r *deploymentResources) Close() error {
	if r == nil {
		return nil
	}
	r.closeOnce.Do(func() {
		if r.metadata != nil {
			r.closeErr = errors.Join(r.closeErr, r.metadata.Close())
			r.metadata = nil
		}
		if r.secretOwnershipTransferred && r.secretValues != nil {
			r.closeErr = errors.Join(r.closeErr, r.secretValues.Close())
			r.secretValues = nil
		}
	})
	return r.closeErr
}

func (r *deploymentResources) closeAfterFailure() error {
	if r == nil || r.metadata == nil {
		return nil
	}
	err := r.metadata.Close()
	r.metadata = nil
	return err
}

func (r *deploymentResources) transferSecretOwnership() {
	if r != nil {
		r.secretOwnershipTransferred = true
	}
}

func joinOpenFailure(primary, cleanup error) error {
	return errors.Join(primary, cleanup)
}
