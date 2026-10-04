package resource

import (
	"context"
	"testing"

	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
)

type sessionTestLease struct{ closes int }

func (s *sessionTestLease) Close(context.Context) error { s.closes++; return nil }

type sessionTestAPI struct{ lease *sessionTestLease }

func (s sessionTestAPI) BeginVerificationSession(
	ctx context.Context,
) (context.Context, resourceModel.VerificationSession, error) {
	return ctx, s.lease, nil
}

func TestWithVerificationSessionUsesExplicitSessionCapability(t *testing.T) {
	t.Parallel()
	lease := &sessionTestLease{}
	value, err := WithVerificationSession(
		t.Context(),
		sessionTestAPI{lease: lease},
		func(context.Context) (string, error) { return "ok", nil },
	)
	if err != nil {
		t.Fatalf("WithVerificationSession: %v", err)
	}
	if value != "ok" || lease.closes != 1 {
		t.Fatalf("value=%q closes=%d", value, lease.closes)
	}
}
