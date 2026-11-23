package toolruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	inferenceSpec "github.com/flexigpt/inference-go/spec"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
)

const (
	v3GoChoicePrefix               = "v3-go"
	v3SDKChoicePrefix              = "v3-sdk"
	v3PreparedChoicePrefix         = "prepared-choice:"
	v3WorkflowProviderChoicePrefix = "workflow-provider-choice:"
	v3GoBindingVersion             = "session-v3-go-tool/v1"
	v3GoFingerprintKind            = "go"
	v3GoReceiptSystem              = "go-tool"
	goRuntimeUnavailableCode       = "go_runtime_unavailable"
	goInvalidBindingCode           = "invalid_go_binding"
	goBindingStaleCode             = "go_binding_stale"
	goInvalidArgumentsCode         = "invalid_arguments"
	goToolOutputInvalidCode        = "go_tool_output_invalid"
	goToolFailedCode               = "go_tool_failed"
	errV3ToolRuntimeUnavailable    = "session v3 tool runtime is unavailable"
)

// V3PreparedChoice is a provider Tool choice paired with its immutable v3
// semantic call binding.
type V3PreparedChoice struct {
	Choice                inferenceSpec.ToolChoice
	Call                  sessionv3Runtime.ToolCall
	ProviderConfiguration jsonutil.CanonicalValue
}

// V3Executor owns v3 source-backed Go and provider Tool preparation and Go
// Tool execution. It is intentionally independent of Session state mutation.
type V3Executor struct {
	tools  *toolAPI.Service
	caller GoToolCaller
}

// ErrProviderManagedWorkflowTool prevents a provider SDK Tool Artifact from
// being treated as a locally executable Tool. Such a Tool must be executed by
// its typed Model plan, not by a synthetic local success result.
var ErrProviderManagedWorkflowTool = errors.New(
	"provider-managed sdk tool requires a typed model execution plan",
)

func NewV3Executor(
	tools *toolAPI.Service,
	caller GoToolCaller,
) (*V3Executor, error) {
	if tools == nil || caller == nil {
		return nil, errors.New("session v3 tool runtime dependencies are incomplete")
	}

	return &V3Executor{
		tools:  tools,
		caller: caller,
	}, nil
}

// IsProviderManagedWorkflowTool verifies a pinned Workflow Tool target before
// admission. Provider SDK Tools must remain inside a typed Model plan and
// cannot be admitted as local Tool WorkItems.
func (e *V3Executor) IsProviderManagedWorkflowTool(
	ctx context.Context,
	pin sessionv3Runtime.ArtifactPin,
) (bool, error) {
	resolved, err := e.resolvePinnedTool(ctx, pin)
	if err != nil {
		return false, err
	}

	switch resolved.Tool.Implementation.Kind {
	case toolv1.ImplementationKindGo:
		return false, nil
	case toolv1.ImplementationKindSDK:
		return true, nil
	default:
		return false, fmt.Errorf(
			"unsupported tool implementation %q",
			resolved.Tool.Implementation.Kind,
		)
	}
}

// PrepareProviderManagedWorkflowTool returns a provider Tool choice for a
// typed Model child plan. It does not create a local Tool WorkItem.
func (e *V3Executor) PrepareProviderManagedWorkflowTool(
	ctx context.Context,
	pin sessionv3Runtime.ArtifactPin,
) (sessionv3Runtime.ProviderManagedToolChoice, error) {
	resolved, err := e.resolvePinnedTool(ctx, pin)
	if err != nil {
		return sessionv3Runtime.ProviderManagedToolChoice{}, err
	}
	choice, call, err := v3ChoiceForResolvedTool(resolved)
	if err != nil {
		return sessionv3Runtime.ProviderManagedToolChoice{}, err
	}
	userArgumentSchema, err := providerToolUserArgumentSchema(resolved)
	if err != nil {
		return sessionv3Runtime.ProviderManagedToolChoice{}, err
	}
	if call.Binding.Kind != sessionv3Runtime.ToolBindingKindProvider {
		return sessionv3Runtime.ProviderManagedToolChoice{}, ErrProviderManagedWorkflowTool
	}
	call.ID = v3WorkflowProviderChoicePrefix + call.ChoiceID
	call.Arguments = jsonutil.MustCanonicalValue(map[string]any{})

	raw, err := json.Marshal(choice.Arguments)
	if err != nil {
		return sessionv3Runtime.ProviderManagedToolChoice{}, err
	}
	schema, err := jsonutil.CanonicalValueFrom(raw)
	if err != nil {
		return sessionv3Runtime.ProviderManagedToolChoice{}, err
	}
	value := sessionv3Runtime.ProviderManagedToolChoice{
		Artifact:           pin,
		Type:               string(choice.Type),
		InputSchema:        schema,
		UserArgumentSchema: userArgumentSchema,
		Call:               call,
	}
	if err := value.Validate(); err != nil {
		return sessionv3Runtime.ProviderManagedToolChoice{}, err
	}
	return value, nil
}

// Choices resolves selected Tool Artifacts, verifies requested pins, and
// creates provider choices plus durable v3 bindings.
func (e *V3Executor) Choices(
	ctx context.Context,
	selections []sessionv3Runtime.ToolSelection,
) ([]V3PreparedChoice, error) {
	if len(selections) == 0 {
		return nil, nil
	}
	if e == nil || e.tools == nil {
		return nil, errors.New(errV3ToolRuntimeUnavailable)
	}

	output := make([]V3PreparedChoice, 0, len(selections))
	for index := range selections {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		selection := selections[index]
		ref := artifactModel.ArtifactRef{
			RootID:     rootModel.RootID(selection.Artifact.RootID),
			ArtifactID: artifactModel.ArtifactID(selection.Artifact.ArtifactID),
		}
		if err := ref.Validate(); err != nil {
			return nil, fmt.Errorf("selected tool %d: %w", index, err)
		}

		target, err := e.tools.MapToolTarget(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf(
				"map selected tool %s/%s: %w",
				ref.RootID,
				ref.ArtifactID,
				err,
			)
		}
		resolved, err := e.tools.ResolveToolTarget(ctx, target)
		if err != nil {
			return nil, fmt.Errorf(
				"resolve selected tool %s/%s: %w",
				ref.RootID,
				ref.ArtifactID,
				err,
			)
		}
		if err := validateV3Selection(selection.Artifact, resolved); err != nil {
			return nil, err
		}

		choice, call, configuration, err := v3ChoiceForResolvedToolWithConfiguration(
			resolved,
			selection.UserArguments,
		)
		if err != nil {
			return nil, err
		}
		if selection.AutoExecute != nil {
			autoExecute := *selection.AutoExecute
			call.Automatic = autoExecute
		}
		output = append(output, V3PreparedChoice{
			Choice:                choice,
			Call:                  call,
			ProviderConfiguration: configuration,
		})
	}
	return output, nil
}

// BindWorkflowTool resolves a Workflow Tool Artifact target to its current
// typed Tool binding. It verifies the durable Artifact pin before Session
// creates a dispatch attempt.
func (e *V3Executor) BindWorkflowTool(
	ctx context.Context,
	payload *sessionv3Runtime.ToolWork,
) (sessionv3Runtime.ToolCall, error) {
	if e == nil || e.tools == nil {
		return sessionv3Runtime.ToolCall{}, errors.New(errV3ToolRuntimeUnavailable)
	}
	if payload == nil || payload.Target == nil {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"workflow tool artifact target is missing",
		)
	}

	resolved, err := e.resolvePinnedTool(ctx, *payload.Target)
	if err != nil {
		return sessionv3Runtime.ToolCall{}, err
	}

	_, call, err := v3ChoiceForResolvedTool(resolved)
	if err != nil {
		return sessionv3Runtime.ToolCall{}, err
	}
	if call.Binding.Kind == sessionv3Runtime.ToolBindingKindProvider {
		return sessionv3Runtime.ToolCall{}, fmt.Errorf(
			"%w: tool %q",
			ErrProviderManagedWorkflowTool,
			resolved.Tool.Name,
		)
	}
	if payload.Call.Name != "" && payload.Call.Name != call.Name {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"workflow tool name changed since workflow admission",
		)
	}

	call.ID = payload.Call.ID
	call.Source = payload.Call.Source
	call.ProviderCallID = payload.Call.ProviderCallID
	call.Arguments = payload.Call.Arguments.Clone()
	call.Automatic = payload.Call.Automatic
	return call, nil
}

// BindGo verifies a current Go Tool binding and returns durable semantic
// binding facts before Session Runtime v3 persists a dispatched Attempt.
func (e *V3Executor) BindGo(
	ctx context.Context,
	payload *sessionv3Runtime.ToolWork,
) (sessionv3Runtime.SemanticBinding, error) {
	if e == nil || e.tools == nil {
		return sessionv3Runtime.SemanticBinding{}, errors.New(errV3ToolRuntimeUnavailable)
	}
	if payload == nil ||
		payload.Call.Binding.Kind != sessionv3Runtime.ToolBindingKindGo ||
		payload.Call.Binding.Go == nil {
		return sessionv3Runtime.SemanticBinding{}, errors.New(
			"go tool binding is missing",
		)
	}
	if err := payload.Call.Validate(); err != nil {
		return sessionv3Runtime.SemanticBinding{}, err
	}

	binding := payload.Call.Binding.Go
	if _, err := e.resolvePinnedGoTool(ctx, binding); err != nil {
		return sessionv3Runtime.SemanticBinding{}, err
	}

	return sessionv3Runtime.SemanticBinding{
		Version: v3GoBindingVersion,
		Artifacts: []sessionv3Runtime.ArtifactPin{
			binding.Target,
		},
		TargetFingerprint: strings.Join([]string{
			v3GoFingerprintKind,
			binding.Target.RootID,
			binding.Target.ArtifactID,
			binding.Target.Digest,
			binding.Function,
		}, ":"),
		BoundAt: time.Now().UTC(),
	}, nil
}

// ExecuteGo executes only a v3 Go Tool binding. It returns a durable result
// instead of exposing source-backed Tool or caller details to Session core.
func (e *V3Executor) ExecuteGo(
	ctx context.Context,
	payload *sessionv3Runtime.ToolWork,
	effects ...sessionv3Runtime.EffectContract,
) (sessionv3Runtime.WorkExecutionResult, error) {
	if e == nil || e.tools == nil || e.caller == nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    goRuntimeUnavailableCode,
				Message: "go tool runtime is unavailable",
			},
		}, nil
	}
	if payload == nil || payload.Call.Binding.Kind != sessionv3Runtime.ToolBindingKindGo {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    goInvalidBindingCode,
				Message: "go tool binding is missing",
			},
		}, nil
	}
	effectKind := sessionv3Runtime.EffectKindNonIdempotent
	if len(effects) != 0 && effects[0].Kind != "" {
		effectKind = effects[0].Kind
	}

	binding := payload.Call.Binding.Go
	resolved, err := e.resolvePinnedGoTool(ctx, binding)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    goBindingStaleCode,
				Message: strings.ToLower(err.Error()),
			},
		}, nil
	}

	arguments := payload.Call.Arguments.Clone()
	if len(arguments) == 0 {
		arguments = jsonutil.MustCanonicalValue(map[string]any{})
	}
	if !arguments.Valid() {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    goInvalidArgumentsCode,
				Message: "go tool arguments are not canonical json",
			},
		}, nil
	}

	outputs, invokeErr := e.caller.CallGoTool(
		ctx,
		resolved.Tool.Implementation.Function,
		json.RawMessage(arguments.Clone()),
		durationMilliseconds(binding.TimeoutMS),
	)
	adapted, conversionErr := AdaptToolOutputs(outputs)
	if conversionErr != nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    goToolOutputInvalidCode,
				Message: strings.ToLower(conversionErr.Error()),
			},
		}, nil
	}
	structured, conversionErr := jsonutil.CanonicalValueFrom(
		adapted.Structured,
	)
	if conversionErr != nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    goToolOutputInvalidCode,
				Message: strings.ToLower(conversionErr.Error()),
			},
		}, nil
	}
	if invokeErr != nil {
		if strings.TrimSpace(adapted.Text) == "" {
			adapted.Text = invokeErr.Error()
		}
		errorOutput := &sessionv3Runtime.WorkItemOutput{
			Value: structured,
			Tool: &sessionv3Runtime.ToolOutput{
				Kind:       sessionv3Runtime.ToolOutputKindError,
				Text:       adapted.Text,
				Contents:   adapted.Contents,
				Structured: structured,
				Error:      invokeErr.Error(),
			},
		}
		if v3GoConversationSource(payload.Call.Source) && ctx.Err() == nil {
			effect := sessionv3Runtime.EffectStateConfirmedNotApplied
			return sessionv3Runtime.WorkExecutionResult{
				Output:      errorOutput,
				Confirmed:   true,
				EffectState: &effect,
			}, nil
		}
		if effectKind == sessionv3Runtime.EffectKindRead && ctx.Err() == nil {
			return sessionv3Runtime.WorkExecutionResult{
				Output: errorOutput,
				Error: &sessionv3Runtime.SanitizedError{
					Code:     goToolFailedCode,
					Category: sessionv3Runtime.ErrorCategoryExecution,
					Message:  strings.ToLower(invokeErr.Error()),
				},
				Confirmed: true,
			}, nil
		}
		effect := sessionv3Runtime.EffectStateUncertain
		return sessionv3Runtime.WorkExecutionResult{
			Output: errorOutput,
			Error: &sessionv3Runtime.SanitizedError{
				Code:     goToolFailedCode,
				Category: sessionv3Runtime.ErrorCategoryUncertainty,
				Effect:   effect,
				Message:  strings.ToLower(invokeErr.Error()),
			},
			Confirmed:   false,
			EffectState: &effect,
		}, nil
	}

	return sessionv3Runtime.WorkExecutionResult{
		Output: &sessionv3Runtime.WorkItemOutput{
			Value: structured,
			Tool: &sessionv3Runtime.ToolOutput{
				Kind:       sessionv3Runtime.ToolOutputKindSuccess,
				Text:       adapted.Text,
				Contents:   adapted.Contents,
				Structured: structured,
			},
			Receipt: &sessionv3Runtime.ExternalReceipt{
				System:      v3GoReceiptSystem,
				Reference:   resolved.Tool.Implementation.Function,
				ConfirmedAt: time.Now().UTC(),
			},
		},
		Confirmed: true,
	}, nil
}

func v3GoConversationSource(source sessionv3Runtime.ToolCallSource) bool {
	return source == sessionv3Runtime.ToolCallSourceModel ||
		source == sessionv3Runtime.ToolCallSourceApp
}

// V3ToolOutputs produces the v3 durable text projection and canonical
// structured representation from one Tool Runtime response.
func V3ToolOutputs(
	outputs []llmtoolsSpec.ToolOutputUnion,
) (t string, s jsonutil.CanonicalValue, err error) {
	adapted, err := AdaptToolOutputs(outputs)
	if err != nil {
		return "", nil, err
	}
	value, err := jsonutil.CanonicalValueFrom(adapted.Structured)
	if err != nil {
		return "", nil, err
	}
	return adapted.Text, value, nil
}

func (e *V3Executor) resolvePinnedTool(
	ctx context.Context,
	pin sessionv3Runtime.ArtifactPin,
) (toolAPI.ResolvedToolView, error) {
	if e == nil || e.tools == nil {
		return toolAPI.ResolvedToolView{}, errors.New(errV3ToolRuntimeUnavailable)
	}
	if pin.RootID == "" ||
		pin.ArtifactID == "" ||
		pin.Revision == 0 ||
		strings.TrimSpace(pin.Digest) == "" {
		return toolAPI.ResolvedToolView{}, errors.New(
			"tool artifact pin is incomplete",
		)
	}

	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(pin.RootID),
		ArtifactID: artifactModel.ArtifactID(pin.ArtifactID),
	}
	if err := ref.Validate(); err != nil {
		return toolAPI.ResolvedToolView{}, err
	}

	target, err := e.tools.MapToolTarget(ctx, ref)
	if err != nil {
		return toolAPI.ResolvedToolView{}, err
	}
	resolved, err := e.tools.ResolveToolTarget(ctx, target)
	if err != nil {
		return toolAPI.ResolvedToolView{}, err
	}
	if resolved.Tool.Artifact.Ref() != ref ||
		resolved.Tool.Artifact.Revision != pin.Revision ||
		resolved.Tool.Artifact.ResolvedDefinition == nil ||
		string(*resolved.Tool.Artifact.ResolvedDefinition) != pin.Digest {
		return toolAPI.ResolvedToolView{}, errors.New(
			"tool artifact changed since work item binding",
		)
	}
	return resolved, nil
}

func (e *V3Executor) resolvePinnedGoTool(
	ctx context.Context,
	binding *sessionv3Runtime.GoToolBinding,
) (toolAPI.ResolvedToolView, error) {
	if binding == nil {
		return toolAPI.ResolvedToolView{}, errors.New("go tool binding is missing")
	}
	if binding.TimeoutMS < 0 {
		return toolAPI.ResolvedToolView{}, errors.New(
			"go tool timeout cannot be negative",
		)
	}

	resolved, err := e.resolvePinnedTool(ctx, binding.Target)
	if err != nil {
		return toolAPI.ResolvedToolView{}, err
	}
	if strings.TrimSpace(binding.LogicalName) != "" &&
		binding.LogicalName != string(resolved.Tool.Name) {
		return toolAPI.ResolvedToolView{}, errors.New(
			"go tool logical name changed since work item binding",
		)
	}
	if resolved.Tool.Implementation.Kind != toolv1.ImplementationKindGo ||
		resolved.Tool.Implementation.Function != binding.Function {
		return toolAPI.ResolvedToolView{}, errors.New(
			"go tool implementation changed since work item binding",
		)
	}
	return resolved, nil
}

func validateV3Selection(
	selection sessionv3Runtime.ArtifactSelection,
	resolved toolAPI.ResolvedToolView,
) error {
	artifact := resolved.Tool.Artifact
	if selection.ExpectedRevision != 0 &&
		selection.ExpectedRevision != artifact.Revision {
		return errors.New("selected tool revision changed")
	}
	if selection.ExpectedDigest != "" &&
		(artifact.ResolvedDefinition == nil ||
			selection.ExpectedDigest != string(*artifact.ResolvedDefinition)) {
		return errors.New("selected tool definition changed")
	}
	return nil
}

func v3ChoiceForResolvedTool(
	resolved toolAPI.ResolvedToolView,
) (inferenceSpec.ToolChoice, sessionv3Runtime.ToolCall, error) {
	name := strings.TrimSpace(string(resolved.Tool.Name))
	if name == "" {
		return inferenceSpec.ToolChoice{},
			sessionv3Runtime.ToolCall{},
			errors.New("resolved tool has no logical name")
	}

	arguments, err := InputSchema(resolved.Tool.InputSchema)
	if err != nil {
		return inferenceSpec.ToolChoice{},
			sessionv3Runtime.ToolCall{},
			err
	}

	digest := ""
	if resolved.Tool.Artifact.ResolvedDefinition != nil {
		digest = string(*resolved.Tool.Artifact.ResolvedDefinition)
	}
	if strings.TrimSpace(digest) == "" {
		return inferenceSpec.ToolChoice{},
			sessionv3Runtime.ToolCall{},
			errors.New("resolved tool has no definition digest")
	}

	pin := sessionv3Runtime.ArtifactPin{
		RootID:     string(resolved.Tool.Artifact.RootID),
		ArtifactID: string(resolved.Tool.Artifact.ID),
		Revision:   resolved.Tool.Artifact.Revision,
		Digest:     digest,
	}

	switch resolved.Tool.Implementation.Kind {
	case toolv1.ImplementationKindGo:
		function := strings.TrimSpace(resolved.Tool.Implementation.Function)
		if function == "" {
			return inferenceSpec.ToolChoice{},
				sessionv3Runtime.ToolCall{},
				errors.New("resolved go tool has no function")
		}

		choiceID := StableChoiceID(
			v3GoChoicePrefix,
			pin.RootID,
			pin.ArtifactID,
			name,
			pin.Digest,
		)
		choice := inferenceSpec.ToolChoice{
			Type:        inferenceSpec.ToolTypeFunction,
			ID:          choiceID,
			Name:        name,
			Description: resolved.Tool.Description,
			Arguments:   arguments,
		}
		call := sessionv3Runtime.ToolCall{
			ID:          v3PreparedChoicePrefix + choiceID,
			Name:        name,
			ChoiceID:    choiceID,
			Type:        choice.Type,
			DisplayName: resolved.Tool.DisplayName,
			Description: resolved.Tool.Description,
			Source:      sessionv3Runtime.ToolCallSourceModel,
			Binding: sessionv3Runtime.ToolBinding{
				Kind: sessionv3Runtime.ToolBindingKindGo,
				Go: &sessionv3Runtime.GoToolBinding{
					Target:      pin,
					LogicalName: name,
					Function:    function,
				},
			},
		}
		return choice, call, nil

	case toolv1.ImplementationKindSDK:
		sdkType := strings.TrimSpace(
			string(resolved.Tool.Implementation.SDKToolType),
		)
		if sdkType == "" {
			return inferenceSpec.ToolChoice{},
				sessionv3Runtime.ToolCall{},
				errors.New("resolved sdk tool has no type")
		}

		choiceID := StableChoiceID(
			v3SDKChoicePrefix,
			pin.RootID,
			pin.ArtifactID,
			name,
			pin.Digest,
		)
		choice := inferenceSpec.ToolChoice{
			Type:        inferenceSpec.ToolType(sdkType),
			ID:          choiceID,
			Name:        name,
			Description: resolved.Tool.Description,
			Arguments:   arguments,
		}
		call := sessionv3Runtime.ToolCall{
			ID:          v3PreparedChoicePrefix + choiceID,
			Name:        name,
			ChoiceID:    choiceID,
			Type:        choice.Type,
			DisplayName: resolved.Tool.DisplayName,
			Description: resolved.Tool.Description,
			Source:      sessionv3Runtime.ToolCallSourceModel,
			Binding: sessionv3Runtime.ToolBinding{
				Kind: sessionv3Runtime.ToolBindingKindProvider,
				Provider: &sessionv3Runtime.ProviderToolBinding{
					Artifact: &pin,
					Type:     sdkType,
					Name:     name,
				},
			},
		}
		return choice, call, nil

	default:
		return inferenceSpec.ToolChoice{},
			sessionv3Runtime.ToolCall{},
			fmt.Errorf(
				"unsupported tool implementation %q",
				resolved.Tool.Implementation.Kind,
			)
	}
}
