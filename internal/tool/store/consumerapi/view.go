package consumerapi

import (
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
)

type ToolImplementationView struct {
	Kind        toolv1.ImplementationKind `json:"kind"`
	Function    string                    `json:"function,omitempty"`
	SDKType     string                    `json:"sdkType,omitempty"`
	SDKToolType toolv1.SDKToolType        `json:"sdkToolType,omitempty"`
}

type ToolView struct {
	Artifact         artifact.Artifact       `json:"artifact"`
	DefinitionDigest cryptoutil.Digest       `json:"definitionDigest"`
	Name             basespec.LogicalName    `json:"name"`
	Version          basespec.LogicalVersion `json:"version"`
	DisplayName      string                  `json:"displayName"`
	Description      string                  `json:"description,omitempty"`
	Tags             []string                `json:"tags,omitempty"`
	AutoExecute      bool                    `json:"autoExecute"`
	BuiltIn          bool                    `json:"builtIn"`

	InputSchema   json.RawMessage  `json:"inputSchema"`
	UserArgSchema *json.RawMessage `json:"userArgSchema,omitempty"`
	OutputSchema  *json.RawMessage `json:"outputSchema,omitempty"`

	Implementation ToolImplementationView `json:"implementation"`
}

type ResolvedToolView struct {
	Tool       ToolView                  `json:"tool"`
	Collection collection.CollectionView `json:"collection"`
}

func (v ResolvedToolView) Enabled() bool {
	return v.Tool.Artifact.State == artifact.StateAvailable &&
		v.Collection.Artifact.State == artifact.StateAvailable &&
		v.Tool.Artifact.Enabled &&
		v.Collection.Artifact.Enabled
}

func toolView(value toolDomain.Tool) ToolView {
	document := value.Document
	displayName := document.DisplayName
	if displayName == "" {
		displayName = document.Name
	}

	return ToolView{
		Artifact:         value.Artifact.Clone(),
		DefinitionDigest: value.Definition.Digest,
		Name:             value.Artifact.LogicalName,
		Version:          document.Version,
		DisplayName:      displayName,
		Description:      document.Description,
		Tags:             append([]string(nil), document.Tags...),
		AutoExecute:      document.AutoExecute,
		BuiltIn:          true,
		InputSchema:      append(json.RawMessage(nil), document.InputSchema...),
		UserArgSchema:    cloneOptionalJSON(document.UserArgSchema),
		OutputSchema:     cloneOptionalJSON(document.OutputSchema),
		Implementation: ToolImplementationView{
			Kind:        document.Implementation.Kind,
			Function:    document.Implementation.Function,
			SDKType:     document.Implementation.SDKType,
			SDKToolType: document.Implementation.SDKToolType,
		},
	}
}

func cloneOptionalJSON(value *json.RawMessage) *json.RawMessage {
	if value == nil {
		return nil
	}
	cloned := append(json.RawMessage(nil), (*value)...)
	return &cloned
}
