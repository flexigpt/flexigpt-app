package sessionstore

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

func validateIdentity(subject, value string) error {
	if value == "" ||
		strings.TrimSpace(value) != value ||
		!utf8.ValidString(value) ||
		strings.ContainsRune(value, 0) ||
		len(value) > 16<<10 {
		return fmt.Errorf("%w: invalid %s", ErrInvalidDocument, subject)
	}
	return nil
}

func validateSessionID(id sessionRuntime.SessionID) error {
	if err := uuidutil.ValidateUUIDv7(string(id)); err != nil {
		return fmt.Errorf("%w: Session ID: %w", ErrInvalidDocument, err)
	}
	return nil
}

func validateCapabilityTarget(target sessionRuntime.CapabilityTarget) error {
	if err := validateIdentity("capability name", target.Name); err != nil {
		return err
	}
	switch target.Form {
	case "artifact":
		if target.Artifact == nil ||
			target.ProviderIdentity != "" ||
			target.ProviderLocalID != "" {
			return fmt.Errorf("%w: malformed Artifact capability target", ErrInvalidDocument)
		}
		if err := validateIdentity("Artifact Root ID", target.Artifact.RootID); err != nil {
			return err
		}
		return validateIdentity("Artifact ID", target.Artifact.ArtifactID)
	case "direct":
		if target.Artifact != nil {
			return fmt.Errorf("%w: direct capability contains an Artifact", ErrInvalidDocument)
		}
		if err := validateIdentity("direct provider identity", target.ProviderIdentity); err != nil {
			return err
		}
		return validateIdentity("direct provider-local identity", target.ProviderLocalID)
	default:
		return fmt.Errorf("%w: unsupported capability target form %q", ErrInvalidDocument, target.Form)
	}
}

func validateStoredSession(document storedSession) error {
	if document.SchemaVersion != SessionSchemaVersion {
		return fmt.Errorf("%w: %q", ErrUnsupportedStore, document.SchemaVersion)
	}

	switch document.Kind {
	case documentKindDeleted:
		if document.Deletion == nil || document.Session != nil || len(document.Runs) != 0 {
			return fmt.Errorf("%w: malformed deletion record", ErrInvalidDocument)
		}
		if err := validateSessionID(document.Deletion.SessionID); err != nil {
			return err
		}
		if document.Deletion.DeletedAt.IsZero() {
			return fmt.Errorf("%w: deletion time is missing", ErrInvalidDocument)
		}
		return nil

	case documentKindSession:
		if document.Session == nil || document.Deletion != nil {
			return fmt.Errorf("%w: malformed Session envelope", ErrInvalidDocument)
		}
	default:
		return fmt.Errorf("%w: unsupported document kind %q", ErrInvalidDocument, document.Kind)
	}

	sessionValue := document.Session
	if err := validateSessionID(sessionValue.ID); err != nil {
		return err
	}
	if sessionValue.Revision == 0 ||
		sessionValue.CreatedAt.IsZero() ||
		sessionValue.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: Session revision or timestamps are missing", ErrInvalidDocument)
	}
	if len(sessionValue.Transcript) > maximumTranscriptMessages ||
		len(document.Runs) > maximumAssociatedRuns {
		return fmt.Errorf("%w: Session record limit exceeded", ErrInvalidDocument)
	}
	if err := validateConfiguration(sessionValue.Configuration); err != nil {
		return err
	}

	messages := make(map[string]struct{}, len(sessionValue.Transcript))
	for _, message := range sessionValue.Transcript {
		if err := validateMessage(message); err != nil {
			return err
		}
		if _, duplicate := messages[message.ID]; duplicate {
			return fmt.Errorf("%w: duplicate transcript message %q", ErrInvalidDocument, message.ID)
		}
		messages[message.ID] = struct{}{}
	}

	runs := make(map[sessionRuntime.RunID]struct{}, len(document.Runs))
	for _, run := range document.Runs {
		if err := validateRun(run, sessionValue.ID); err != nil {
			return err
		}
		if _, duplicate := runs[run.ID]; duplicate {
			return fmt.Errorf("%w: duplicate Run %q", ErrInvalidDocument, run.ID)
		}
		runs[run.ID] = struct{}{}
	}

	associated := make(map[sessionRuntime.RunID]struct{}, len(sessionValue.RunIDs))
	for _, id := range sessionValue.RunIDs {
		if _, found := runs[id]; !found {
			return fmt.Errorf("%w: associated Run %q is missing", ErrInvalidDocument, id)
		}
		if _, duplicate := associated[id]; duplicate {
			return fmt.Errorf("%w: duplicate associated Run %q", ErrInvalidDocument, id)
		}
		associated[id] = struct{}{}
	}
	if len(associated) != len(runs) {
		return fmt.Errorf("%w: Run association set differs from stored Runs", ErrInvalidDocument)
	}

	return nil
}

func validateMessage(message sessionRuntime.TranscriptMessage) error {
	if err := validateIdentity("message ID", message.ID); err != nil {
		return err
	}
	if message.CreatedAt.IsZero() {
		return fmt.Errorf("%w: message timestamp is missing", ErrInvalidDocument)
	}
	switch message.Role {
	case sessionRuntime.MessageRoleSystem,
		sessionRuntime.MessageRoleDeveloper,
		sessionRuntime.MessageRoleUser,
		sessionRuntime.MessageRoleAssistant,
		sessionRuntime.MessageRoleTool:
	default:
		return fmt.Errorf("%w: invalid message role %q", ErrInvalidDocument, message.Role)
	}
	if len(message.ProtocolInputs) != 0 || message.AttachmentsHydrated {
		return fmt.Errorf("%w: hydrated inputs reached persistence", ErrInvalidDocument)
	}
	for _, reference := range message.Attachments {
		if err := validateAttachment(reference, 0); err != nil {
			return err
		}
	}
	return nil
}

func validateRun(run sessionRuntime.Run, sessionID sessionRuntime.SessionID) error {
	if err := validateIdentity("Run ID", string(run.ID)); err != nil {
		return err
	}
	if run.SessionID != sessionID || run.Revision == 0 ||
		run.CreatedAt.IsZero() || run.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: invalid Run identity or revision", ErrInvalidDocument)
	}
	switch run.State {
	case sessionRuntime.RunStateQueued,
		sessionRuntime.RunStateRunning,
		sessionRuntime.RunStateWaiting,
		sessionRuntime.RunStatePaused,
		sessionRuntime.RunStateCancelling,
		sessionRuntime.RunStateCompleted,
		sessionRuntime.RunStateFailed,
		sessionRuntime.RunStateCancelled:
	default:
		return fmt.Errorf("%w: unsupported Run state %q", ErrInvalidDocument, run.State)
	}
	if run.SkillSession.ID != "" || run.SkillSession.EstablishedGeneration != 0 {
		return fmt.Errorf("%w: private Skill session state reached persistence", ErrInvalidDocument)
	}
	if err := validateConfiguration(run.Configuration); err != nil {
		return err
	}
	if run.Continuation.Configuration != nil {
		if err := validateConfiguration(*run.Continuation.Configuration); err != nil {
			return err
		}
	}

	enabled := make(map[sessionRuntime.ArtifactReference]struct{}, len(run.SkillSession.Enabled))
	for _, reference := range run.SkillSession.Enabled {
		enabled[reference] = struct{}{}
	}
	for _, reference := range run.SkillSession.Active {
		if _, found := enabled[reference]; !found {
			return fmt.Errorf("%w: active Skill is not enabled", ErrInvalidDocument)
		}
	}

	operations := make(map[sessionRuntime.OperationID]struct{}, len(run.ModelOperations))
	for _, operation := range run.ModelOperations {
		if err := validateIdentity("operation ID", string(operation.ID)); err != nil {
			return err
		}
		if _, duplicate := operations[operation.ID]; duplicate {
			return fmt.Errorf("%w: duplicated model operation", ErrInvalidDocument)
		}
		operations[operation.ID] = struct{}{}
		if operation.Preparation.Private != nil {
			return fmt.Errorf("%w: private prepared operation reached persistence", ErrInvalidDocument)
		}
		if err := validateMessage(operation.Request.Current); err != nil {
			return err
		}
		for _, message := range operation.Request.History {
			if err := validateMessage(message); err != nil {
				return err
			}
		}
	}

	calls := make(map[sessionRuntime.ToolCallID]struct{}, len(run.ToolCalls))
	for _, call := range run.ToolCalls {
		if err := validateIdentity("Tool call ID", string(call.ID)); err != nil {
			return err
		}
		if _, duplicate := calls[call.ID]; duplicate {
			return fmt.Errorf("%w: duplicated Tool call", ErrInvalidDocument)
		}
		calls[call.ID] = struct{}{}
		if _, found := operations[call.OriginOperationID]; !found {
			return fmt.Errorf("%w: Tool call has no originating operation", ErrInvalidDocument)
		}
		if len(call.Authorization) != 0 {
			return fmt.Errorf("%w: Tool authorization reached persistence", ErrInvalidDocument)
		}
		if call.SelectedResultID != "" {
			found := slices.ContainsFunc(call.Results, func(result sessionRuntime.ToolResult) bool {
				return result.ID == call.SelectedResultID && result.CallID == call.ID
			})
			if !found {
				return fmt.Errorf("%w: selected Tool result is missing", ErrInvalidDocument)
			}
		}
	}

	for _, app := range run.Apps {
		if app.Route != nil {
			return fmt.Errorf("%w: private App route reached persistence", ErrInvalidDocument)
		}
	}

	for _, wait := range run.WaitingRequests {
		if err := validateIdentity("wait ID", string(wait.ID)); err != nil {
			return err
		}
		if wait.ToolCallID != "" {
			if _, found := calls[wait.ToolCallID]; !found {
				return fmt.Errorf("%w: wait has no owning Tool call", ErrInvalidDocument)
			}
		}
	}
	return nil
}

func validateConfiguration(
	value sessionRuntime.SessionConfiguration,
) error {
	validateSelection := func(selection *sessionRuntime.ArtifactSelection) error {
		if selection == nil {
			return nil
		}
		if err := validateIdentity("Artifact Root ID", selection.RootID); err != nil {
			return err
		}
		return validateIdentity("Artifact ID", selection.ArtifactID)
	}
	if err := validateSelection(value.Agent); err != nil {
		return err
	}
	if err := validateSelection(value.Model); err != nil {
		return err
	}
	if value.Workspace != nil {
		if err := validateSelection(&value.Workspace.Workspace); err != nil {
			return err
		}
	}
	for _, override := range value.Tools {
		if override.Value == nil || override.Value.Capability.Artifact == nil {
			continue
		}
		if err := validateSelection(override.Value.Capability.Artifact); err != nil {
			return err
		}
	}
	for _, override := range value.Skills {
		if override.Value == nil {
			continue
		}
		if err := validateSelection(override.Value); err != nil {
			return err
		}
	}
	return nil
}

func validateAttachment(reference sessionRuntime.AttachmentReference, depth int) error {
	if depth > maximumAttachmentDepth {
		return fmt.Errorf("%w: attachment nesting limit exceeded", ErrInvalidDocument)
	}

	count := 0
	for _, present := range []bool{
		reference.File != nil,
		reference.Image != nil,
		reference.URL != nil,
		reference.Generic != nil,
		reference.Directory != nil,
	} {
		if present {
			count++
		}
	}
	if count != 1 {
		return fmt.Errorf("%w: attachment must contain exactly one reference", ErrInvalidDocument)
	}

	switch reference.Kind {
	case sessionRuntime.AttachmentReferenceFile:
		if reference.File == nil {
			return fmt.Errorf("%w: file reference is missing", ErrInvalidDocument)
		}
		return validateLocalAttachment(*reference.File)

	case sessionRuntime.AttachmentReferenceImage:
		if reference.Image == nil {
			return fmt.Errorf("%w: image reference is missing", ErrInvalidDocument)
		}
		return validateLocalAttachment(*reference.Image)

	case sessionRuntime.AttachmentReferenceURL:
		if reference.URL == nil {
			return fmt.Errorf("%w: URL reference is missing", ErrInvalidDocument)
		}
		value, err := url.Parse(reference.URL.URL)
		if err != nil || value.User != nil || value.Hostname() == "" ||
			(value.Scheme != "http" && value.Scheme != "https") {
			return fmt.Errorf("%w: invalid attachment URL", ErrInvalidDocument)
		}

	case sessionRuntime.AttachmentReferenceGeneric:
		if reference.Generic == nil {
			return fmt.Errorf("%w: generic reference is missing", ErrInvalidDocument)
		}
		return validateIdentity("generic attachment handle", reference.Generic.Handle)

	case sessionRuntime.AttachmentReferenceDirectory:
		if reference.Directory == nil ||
			len(reference.Directory.Expanded) == 0 ||
			reference.Directory.ScannedFileCount != len(reference.Directory.Expanded) {
			return fmt.Errorf("%w: directory snapshot is incomplete", ErrInvalidDocument)
		}
		for _, child := range reference.Directory.Expanded {
			if err := validateAttachment(child, depth+1); err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("%w: unsupported attachment kind %q", ErrInvalidDocument, reference.Kind)
	}
	return nil
}

func validateLocalAttachment(reference sessionRuntime.LocalAttachmentReference) error {
	if reference.IsDirectory || reference.Size < 0 || reference.OriginalSize < 0 {
		return fmt.Errorf("%w: invalid local attachment snapshot", ErrInvalidDocument)
	}
	if err := validateIdentity("local attachment path", reference.Path); err != nil {
		return err
	}
	if err := validateIdentity("original attachment path", reference.OriginalPath); err != nil {
		return err
	}
	if _, err := time.Parse(time.RFC3339Nano, reference.OriginalMod); err != nil {
		return fmt.Errorf("%w: original attachment timestamp: %w", ErrInvalidDocument, err)
	}
	return nil
}
