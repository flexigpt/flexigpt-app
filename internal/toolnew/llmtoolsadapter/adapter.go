package llmtoolsadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/flexigpt/llmtools-go"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	toolnewDomain "github.com/flexigpt/flexigpt-app/internal/toolnew/store/domain"
)

var nonAutoGoFunctions = map[string]struct{}{
	"github.com/flexigpt/llmtools-go/fstool/writefile.WriteFile":                 {},
	"github.com/flexigpt/llmtools-go/fstool/deletefile.DeleteFile":               {},
	"github.com/flexigpt/llmtools-go/exectool/shellcommand.ShellCommand":         {},
	"github.com/flexigpt/llmtools-go/exectool/runscript.RunScript":               {},
	"github.com/flexigpt/llmtools-go/texttool/inserttext.InsertText":             {},
	"github.com/flexigpt/llmtools-go/texttool/replacetext.ReplaceText":           {},
	"github.com/flexigpt/llmtools-go/texttool/deletetext.DeleteText":             {},
	"github.com/flexigpt/llmtools-go/texttool/applyunifieddiff.ApplyUnifiedDiff": {},
	"github.com/flexigpt/llmtools-go/gittool/createtag.CreateTag":                {},
	"github.com/flexigpt/llmtools-go/gittool/deletetag.DeleteTag":                {},
	"github.com/flexigpt/llmtools-go/gittool/add.Add":                            {},
	"github.com/flexigpt/llmtools-go/gittool/reset.Reset":                        {},
	"github.com/flexigpt/llmtools-go/gittool/commit.Commit":                      {},
	"github.com/flexigpt/llmtools-go/gittool/createbranch.CreateBranch":          {},
	"github.com/flexigpt/llmtools-go/gittool/checkout.Checkout":                  {},
	"github.com/flexigpt/llmtools-go/gittool/init.Init":                          {},
}

type Adapter struct {
	registry   *llmtools.Registry
	byFunction map[string]toolnewDomain.GoToolDescriptor
	byName     map[basespec.LogicalName]toolnewDomain.GoToolDescriptor
}

func New() (*Adapter, error) {
	registry, err := llmtools.NewBuiltinRegistry()
	if err != nil {
		return nil, fmt.Errorf(
			"create llmtools-go builtin registry: %w",
			err,
		)
	}

	byFunction := make(
		map[string]toolnewDomain.GoToolDescriptor,
		len(registry.Tools()),
	)
	byName := make(
		map[basespec.LogicalName]toolnewDomain.GoToolDescriptor,
		len(registry.Tools()),
	)

	for _, value := range registry.Tools() {
		descriptor, err := descriptorFor(value)
		if err != nil {
			return nil, err
		}
		if _, duplicate := byFunction[descriptor.Function]; duplicate {
			return nil, fmt.Errorf(
				"%w: Go Tool function %q is repeated",
				basespec.ErrIdentityConflict,
				descriptor.Function,
			)
		}
		if _, duplicate := byName[descriptor.Name]; duplicate {
			return nil, fmt.Errorf(
				"%w: Go Tool name %q is repeated",
				basespec.ErrIdentityConflict,
				descriptor.Name,
			)
		}

		byFunction[descriptor.Function] = descriptor
		byName[descriptor.Name] = descriptor
	}

	return &Adapter{
		registry:   registry,
		byFunction: byFunction,
		byName:     byName,
	}, nil
}

func (a *Adapter) LookupGoTool(
	ctx context.Context,
	function string,
) (toolnewDomain.GoToolDescriptor, error) {
	if a == nil || a.registry == nil {
		return toolnewDomain.GoToolDescriptor{}, basespec.ErrClosed
	}
	if ctx == nil {
		return toolnewDomain.GoToolDescriptor{}, fmt.Errorf(
			"%w: Go Tool lookup context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return toolnewDomain.GoToolDescriptor{}, err
	}

	function = strings.TrimSpace(function)
	if function == "" {
		return toolnewDomain.GoToolDescriptor{}, fmt.Errorf(
			"%w: Go Tool function is required",
			basespec.ErrInvalid,
		)
	}

	value, found := a.byFunction[function]
	if !found {
		return toolnewDomain.GoToolDescriptor{}, fmt.Errorf(
			"%w: Go Tool function %q is not registered",
			basespec.ErrReferenceUnresolved,
			function,
		)
	}
	return cloneDescriptor(value), nil
}

func (a *Adapter) LookupGoToolByName(
	ctx context.Context,
	name basespec.LogicalName,
) (toolnewDomain.GoToolDescriptor, error) {
	if a == nil || a.registry == nil {
		return toolnewDomain.GoToolDescriptor{}, basespec.ErrClosed
	}
	if err := name.Validate(); err != nil {
		return toolnewDomain.GoToolDescriptor{}, err
	}
	if err := ctx.Err(); err != nil {
		return toolnewDomain.GoToolDescriptor{}, err
	}

	value, found := a.byName[name]
	if !found {
		return toolnewDomain.GoToolDescriptor{}, fmt.Errorf(
			"%w: Go Tool %q is not registered",
			basespec.ErrReferenceUnresolved,
			name,
		)
	}
	return cloneDescriptor(value), nil
}

func (a *Adapter) CallGoTool(
	ctx context.Context,
	function string,
	args json.RawMessage,
	timeout time.Duration,
) ([]llmtoolsSpec.ToolOutputUnion, error) {
	if _, err := a.LookupGoTool(ctx, function); err != nil {
		return nil, err
	}

	options := make([]llmtools.CallOption, 0, 1)
	if timeout > 0 {
		options = append(
			options,
			llmtools.WithCallTimeout(timeout),
		)
	}

	return llmtoolsutil.CallUsingRegistry(
		ctx,
		a.registry,
		function,
		args,
		options...,
	)
}

func descriptorFor(
	value llmtoolsSpec.Tool,
) (toolnewDomain.GoToolDescriptor, error) {
	function := strings.TrimSpace(string(value.GoImpl.FuncID))
	_, manual := nonAutoGoFunctions[function]

	descriptor := toolnewDomain.GoToolDescriptor{
		Function:    function,
		Name:        basespec.LogicalName(value.Slug),
		Version:     basespec.LogicalVersion(value.Version),
		DisplayName: value.DisplayName,
		Description: value.Description,
		Tags:        append([]string(nil), value.Tags...),
		AutoExecute: !manual,
		InputSchema: append(json.RawMessage(nil), value.ArgSchema...),
	}
	if err := descriptor.Validate(); err != nil {
		return toolnewDomain.GoToolDescriptor{}, err
	}
	return descriptor, nil
}

func cloneDescriptor(
	value toolnewDomain.GoToolDescriptor,
) toolnewDomain.GoToolDescriptor {
	output := value
	output.Tags = append([]string(nil), value.Tags...)
	output.InputSchema = append(
		json.RawMessage(nil),
		value.InputSchema...,
	)
	return output
}

func (a *Adapter) GoTools() []toolnewDomain.GoToolDescriptor {
	if a == nil {
		return nil
	}

	output := make(
		[]toolnewDomain.GoToolDescriptor,
		0,
		len(a.byFunction),
	)
	for _, value := range a.byFunction {
		output = append(output, cloneDescriptor(value))
	}
	sort.Slice(output, func(left, right int) bool {
		return output[left].Name < output[right].Name
	})
	return output
}
