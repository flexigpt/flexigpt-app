package local

import (
	"context"
	"testing"

	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
)

type closeTrackingValueStore struct {
	closeCalls int
}

func (*closeTrackingValueStore) Name() string { return "test-values" }
func (*closeTrackingValueStore) Put(context.Context, secretModel.Ref, string) error {
	return nil
}

func (*closeTrackingValueStore) Get(context.Context, secretModel.Ref) (string, error) {
	return "", nil
}

func (*closeTrackingValueStore) Delete(context.Context, secretModel.Ref) error {
	return nil
}

func (s *closeTrackingValueStore) Close() error {
	s.closeCalls++
	return nil
}

func TestDeploymentResourcesTransferSecretOwnershipOnlyOnSuccess(t *testing.T) {
	t.Parallel()

	values := &closeTrackingValueStore{}
	resources := &deploymentResources{secretValues: values}
	if err := resources.closeAfterFailure(); err != nil {
		t.Fatalf("closeAfterFailure: %v", err)
	}
	if values.closeCalls != 0 {
		t.Fatalf("failure cleanup closed caller-owned values %d times", values.closeCalls)
	}

	resources.transferSecretOwnership()
	if err := resources.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := resources.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if values.closeCalls != 1 {
		t.Fatalf("successful deployment close calls=%d, want 1", values.closeCalls)
	}
}
