package sessionv3runtime

import (
	"context"
	"errors"

	attachmentRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/attachment"
	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/attachmentruntime"
)

// delegate is the canonical Attachment-domain admission implementation.
// The remaining setup type exists only for package compatibility with callers
// that were compiled before attachmentruntime became the V3 owner.
type v3AttachmentAdmitterDelegate interface {
	AdmitAttachments(
		ctx context.Context,
		request sessionv3Runtime.AttachmentAdmissionRequest,
	) ([]sessionv3Runtime.Attachment, error)
}

type V3AttachmentAdmitter struct {
	delegate              v3AttachmentAdmitterDelegate
	maximumDirectoryFiles int
}

func NewV3AttachmentAdmitter() *V3AttachmentAdmitter {
	return &V3AttachmentAdmitter{
		delegate:              attachmentruntime.NewSessionV3Admitter(),
		maximumDirectoryFiles: attachmentRuntime.DefaultDirectoryWalkMaxFiles,
	}
}

func (a *V3AttachmentAdmitter) AdmitAttachments(
	ctx context.Context,
	request sessionv3Runtime.AttachmentAdmissionRequest,
) ([]sessionv3Runtime.Attachment, error) {
	if a == nil || a.delegate == nil {
		return nil, errors.New("v3 attachment admitter is unavailable")
	}
	return a.delegate.AdmitAttachments(ctx, request)
}
