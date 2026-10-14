package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/backgroundutil"
)

type panicTestModelExecutor func(
	context.Context,
	ModelExecutionRequest,
	ModelStream,
) (ModelExecutionResult, error)

func (f panicTestModelExecutor) ExecuteModel(
	ctx context.Context,
	request ModelExecutionRequest,
	stream ModelStream,
) (ModelExecutionResult, error) {
	return f(ctx, request, stream)
}

type panicTestToolExecutor struct {
	stage string
}

func (e panicTestToolExecutor) EvaluateTool(
	_ context.Context,
	_ ToolExecutionRequest,
) (ToolEvaluation, error) {
	if e.stage == "evaluation" {
		panic("external evaluator panic")
	}

	if e.stage == "wait" {
		return ToolEvaluation{
			Decision: ToolEvaluationWait,
			Wait: &ToolWaitSpec{
				Kind:               WaitKindApproval,
				AllowedResolutions: []WaitResolution{WaitResolutionApproveOnce},
				ExecutorReference:  "private-approval-reference",
			},
		}, nil
	}

	return ToolEvaluation{
		Decision: ToolEvaluationAllowed,
	}, nil
}

func (e panicTestToolExecutor) ExecuteTool(
	_ context.Context,
	_ ToolExecutionRequest,
) (ToolExecutionResult, error) {
	if e.stage == "execution" {
		panic("external invocation panic")
	}

	return ToolExecutionResult{
		HasResult: true,
		Kind:      ToolResultKindSuccess,
		Text:      "completed",
	}, nil
}

func (e panicTestToolExecutor) ResolveToolWait(
	_ context.Context,
	_ ToolWaitResolutionRequest,
) (ToolWaitResolution, error) {
	if e.stage == "wait" {
		panic("external approval resolution panic")
	}

	return ToolWaitResolution{
		Status: ToolWaitResolutionApproved,
	}, nil
}

// panicTestSubmissionPreparer preserves the accepted command intent without
// introducing domain/runtime-setup behavior into Session-core panic tests.
type panicTestSubmissionPreparer struct{}

func (panicTestSubmissionPreparer) PrepareSubmission(
	_ context.Context,
	request SubmissionPreparationRequest,
) (PreparedSubmission, error) {
	return PreparedSubmission{
		Configuration:        request.Configuration,
		Input:                request.Input,
		PersistConfiguration: request.PersistConfiguration,
	}, nil
}

type panicTestPreparedOperation struct{}

// panicTestOperationPreparer creates the minimum valid immutable preparation
// required by Session before dispatching model work. The test Model executor
// remains the component under test.
type panicTestOperationPreparer struct{}

func (panicTestOperationPreparer) PrepareModelOperation(
	_ context.Context,
	_ ModelOperationPreparationRequest,
) (OperationPreparation, error) {
	return OperationPreparation{
		State:       PreparationStatePrepared,
		Fingerprint: "panic-test-prepared-operation",
		Private:     panicTestPreparedOperation{},
	}, nil
}

type panicTestSkillController struct{}

func (panicTestSkillController) EnsureSkillSession(
	_ context.Context,
	request SkillSessionRequest,
) (SkillSessionResult, error) {
	sessionID := request.SessionID
	if sessionID == "" {
		sessionID = "panic-test-skill-session"
	}

	return SkillSessionResult{
		SessionID: sessionID,
		Active:    request.Active,
	}, nil
}

func (panicTestSkillController) PrepareUserMessageSkillTemplate(
	_ context.Context,
	_ UserMessageSkillTemplatePreparationRequest,
) (UserMessageSkillTemplateResult, error) {
	return UserMessageSkillTemplateResult{}, nil
}

func (panicTestSkillController) CloseSkillSession(
	_ context.Context,
	_ string,
) error {
	return nil
}

type panicTestAppExecutor struct{}

func (panicTestAppExecutor) PrepareAppTool(
	_ context.Context,
	_ AppToolPreparationRequest,
) (AppToolPreparation, error) {
	return AppToolPreparation{}, errors.New(
		"panic test does not support app tool execution",
	)
}

func (panicTestAppExecutor) ReadAppResource(
	_ context.Context,
	_ AppResourceReadRequest,
) (AppResourceReadResult, error) {
	return AppResourceReadResult{}, errors.New(
		"panic test does not support app resource reads",
	)
}

func (panicTestAppExecutor) ValidateAppContext(
	_ context.Context,
	_ AppContextStagingRequest,
) (AppContextUpdate, error) {
	return AppContextUpdate{}, errors.New(
		"panic test does not support app context staging",
	)
}

type panicTestAttachmentPreparer struct{}

func (panicTestAttachmentPreparer) PrepareAttachments(
	_ context.Context,
	references []AttachmentReference,
) ([]AttachmentReference, error) {
	return references, nil
}

type panicTestCheckpointSink struct{}

func (panicTestCheckpointSink) HandleCheckpoint(
	_ context.Context,
	_ RuntimeCheckpointView,
) error {
	return nil
}

type panicTestDiagnosticSink struct{}

func (panicTestDiagnosticSink) ReportRuntimeDiagnostic(
	_ context.Context,
	_ RuntimeDiagnostic,
) {
}

func TestModelPanicSettlesRunAndPreservesStream(t *testing.T) {
	model := panicTestModelExecutor(func(
		_ context.Context,
		_ ModelExecutionRequest,
		stream ModelStream,
	) (ModelExecutionResult, error) {
		if err := stream.OnText("visible partial output"); err != nil {
			return ModelExecutionResult{}, err
		}
		panic("provider request containing private material")
	})

	runtimeValue, events, sessionID := newPanicTestRuntime(t, model, nil)
	receipt := startPanicTestRun(t, runtimeValue, sessionID)

	awaitPanicTestEvent(t, events, receipt.RunID, RuntimeEventRunFailed)

	snapshot, err := runtimeValue.GetRunSnapshot(t.Context(), receipt.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Run.State != RunStateFailed {
		t.Fatalf("unexpected Run state: %q", snapshot.Run.State)
	}
	if snapshot.Run.ActiveModelOperationID != "" {
		t.Fatal("panicked model operation remains active")
	}
	if len(snapshot.Run.ModelOperations) != 1 {
		t.Fatalf("unexpected model operation count: %d", len(snapshot.Run.ModelOperations))
	}

	operation := snapshot.Run.ModelOperations[0]
	if operation.State != OperationStateFailed {
		t.Fatalf("unexpected operation state: %q", operation.State)
	}
	if operation.Error == nil || operation.Error.Code != backgroundutil.PanicFailureCode {
		t.Fatalf("unexpected operation error: %#v", operation.Error)
	}
	if operation.StreamedText != "visible partial output" {
		t.Fatalf("streamed text was lost: %q", operation.StreamedText)
	}
}

func TestToolEvaluationPanicSettlesRunWithoutInvocation(t *testing.T) {
	runtimeValue, events, sessionID := newPanicTestRuntime(
		t,
		panicTestToolModel(),
		panicTestToolExecutor{stage: "evaluation"},
	)
	receipt := startPanicTestRun(t, runtimeValue, sessionID)

	awaitPanicTestEvent(t, events, receipt.RunID, RuntimeEventRunFailed)

	snapshot, err := runtimeValue.GetRunSnapshot(t.Context(), receipt.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Run.State != RunStateFailed {
		t.Fatalf("unexpected Run state: %q", snapshot.Run.State)
	}
	if len(snapshot.Run.ToolCalls) != 1 {
		t.Fatalf("unexpected Tool call count: %d", len(snapshot.Run.ToolCalls))
	}

	call := snapshot.Run.ToolCalls[0]
	if call.State != ToolCallStateFailed {
		t.Fatalf("unexpected Tool call state: %q", call.State)
	}
	if len(call.Attempts) != 0 {
		t.Fatal("an invocation attempt was created after evaluation panicked")
	}
	if call.SelectedResultID != "" {
		t.Fatal("evaluation panic created a selected Tool result")
	}
}

func TestToolInvocationPanicRecordsUncertainAttempt(t *testing.T) {
	runtimeValue, events, sessionID := newPanicTestRuntime(
		t,
		panicTestToolModel(),
		panicTestToolExecutor{stage: "execution"},
	)
	receipt := startPanicTestRun(t, runtimeValue, sessionID)

	awaitPanicTestEvent(t, events, receipt.RunID, RuntimeEventRunFailed)

	snapshot, err := runtimeValue.GetRunSnapshot(t.Context(), receipt.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Run.State != RunStateFailed {
		t.Fatalf("unexpected Run state: %q", snapshot.Run.State)
	}
	if len(snapshot.Run.ToolCalls) != 1 {
		t.Fatalf("unexpected Tool call count: %d", len(snapshot.Run.ToolCalls))
	}

	call := snapshot.Run.ToolCalls[0]
	if call.State != ToolCallStateFailed {
		t.Fatalf("unexpected Tool call state: %q", call.State)
	}
	if call.SelectedResultID != "" || len(call.Results) != 0 {
		t.Fatal("invocation panic fabricated a consumable Tool result")
	}
	if len(call.Attempts) != 1 {
		t.Fatalf("unexpected attempt count: %d", len(call.Attempts))
	}

	attempt := call.Attempts[0]
	if attempt.State != ToolAttemptStateUncertain || !attempt.Uncertain {
		t.Fatalf("external outcome was not marked uncertain: %#v", attempt)
	}
	if attempt.FinishedAt.IsZero() {
		t.Fatal("uncertain attempt has no settlement timestamp")
	}
	if len(snapshot.Run.ModelOperations) != 1 {
		t.Fatal("automatic continuation occurred after an uncertain invocation")
	}
}

func TestWaitResolutionPanicInvalidatesWaitAndSettlesRun(t *testing.T) {
	runtimeValue, events, sessionID := newPanicTestRuntime(
		t,
		panicTestToolModel(),
		panicTestToolExecutor{stage: "wait"},
	)
	receipt := startPanicTestRun(t, runtimeValue, sessionID)

	awaitPanicTestEvent(t, events, receipt.RunID, RuntimeEventWaitRequested)

	snapshot, err := runtimeValue.GetRunSnapshot(t.Context(), receipt.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Run.WaitingRequests) != 1 {
		t.Fatalf("unexpected wait count: %d", len(snapshot.Run.WaitingRequests))
	}

	resolution, err := runtimeValue.ResolveWait(
		t.Context(),
		ResolveWaitCommand{
			Meta:       CommandMeta{ID: "resolve-panicking-wait"},
			SessionID:  sessionID,
			RunID:      receipt.RunID,
			WaitID:     snapshot.Run.WaitingRequests[0].ID,
			Resolution: WaitResolutionApproveOnce,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !resolution.Accepted {
		t.Fatalf("wait resolution was rejected: %s", resolution.Message)
	}

	awaitPanicTestEvent(t, events, receipt.RunID, RuntimeEventRunFailed)

	snapshot, err = runtimeValue.GetRunSnapshot(t.Context(), receipt.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Run.State != RunStateFailed {
		t.Fatalf("unexpected Run state: %q", snapshot.Run.State)
	}
	if snapshot.Run.WaitingRequests[0].Status != WaitStatusInvalid {
		t.Fatalf("wait remains dispatchable: %q", snapshot.Run.WaitingRequests[0].Status)
	}
	if snapshot.Run.ToolCalls[0].WaitID != "" {
		t.Fatal("Tool call retains an invalid wait reference")
	}
	if snapshot.Run.ToolCalls[0].SelectedResultID != "" {
		t.Fatal("approval panic fabricated a selected Tool result")
	}
}

func panicTestToolModel() panicTestModelExecutor {
	return func(
		_ context.Context,
		_ ModelExecutionRequest,
		_ ModelStream,
	) (ModelExecutionResult, error) {
		return ModelExecutionResult{
			Usable: true,
			ToolCalls: []ToolCallProposal{{
				ID:        "test-tool-call",
				Name:      "test-tool",
				ChoiceID:  "test-choice",
				Type:      "function",
				Required:  true,
				Automatic: true,
				Binding: ToolBinding{
					Kind: ToolBindingKindGo,
				},
			}},
		}, nil
	}
}

func newPanicTestRuntime(
	t *testing.T,
	model ModelExecutor,
	tools ToolExecutor,
) (*Runtime, <-chan RuntimeEvent, SessionID) {
	t.Helper()

	// The model-panic test has no Tool calls, but Runtime construction still
	// requires a concrete executor.
	if tools == nil {
		tools = panicTestToolExecutor{}
	}

	runtimeValue, err := NewRuntime(RuntimeDependencies{
		Preparation: panicTestSubmissionPreparer{},
		Operations:  panicTestOperationPreparer{},
		Model:       model,
		Tools:       tools,
		Templates:   panicTestSkillController{},
		Skills:      panicTestSkillController{},
		Apps:        panicTestAppExecutor{},
		Attachments: panicTestAttachmentPreparer{},
		Checkpoints: panicTestCheckpointSink{},
		Diagnostics: panicTestDiagnosticSink{},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		runtimeValue.Close()
	})

	events, unsubscribe := runtimeValue.Subscribe(128)
	t.Cleanup(unsubscribe)

	receipt, err := runtimeValue.CreateSession(
		t.Context(),
		CreateSessionCommand{
			Meta: CommandMeta{ID: "create-panic-test-session"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Accepted {
		t.Fatalf("Session creation was rejected: %s", receipt.Message)
	}

	return runtimeValue, events, receipt.SessionID
}

func startPanicTestRun(
	t *testing.T,
	runtimeValue *Runtime,
	sessionID SessionID,
) *CommandReceipt {
	t.Helper()

	receipt, err := runtimeValue.StartRun(
		t.Context(),
		StartRunCommand{
			Meta:      CommandMeta{ID: "start-panic-test-run"},
			SessionID: sessionID,
			Input:     UserInput{Text: "execute"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Accepted {
		t.Fatalf("Run was rejected: %s", receipt.Message)
	}

	return receipt
}

func awaitPanicTestEvent(
	t *testing.T,
	events <-chan RuntimeEvent,
	runID RunID,
	kind RuntimeEventKind,
) {
	t.Helper()

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	for {
		select {
		case event, open := <-events:
			if !open {
				t.Fatal("runtime event subscription closed before settlement")
			}
			if event.RunID == runID && event.Kind == kind {
				return
			}
		case <-timer.C:
			t.Fatalf("Run %q did not emit %q", runID, kind)
		}
	}
}
