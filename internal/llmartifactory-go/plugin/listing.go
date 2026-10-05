package plugin

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/pluginv1"
)

// ListRequest is the public Collection listing request.
//
// It contains only domain-owned listing choices. Generic Artifact Store query
// details remain inside the Collection API.
type ListRequest struct {
	RootID rootModel.RootID `json:"rootID"`
}

// ListItem is a lightweight Collection projection.
//
// Members are not returned in ordinary listings. The member count is enough
// for management pages to determine whether deletion can be offered. Read
// returns CollectionView when a caller explicitly selects one Collection.
type ListItem struct {
	Ref      artifactModel.ArtifactRef `json:"ref"`
	SourceID sourceModel.SourceID      `json:"sourceID"`

	Name        spec.LogicalName `json:"name"`
	DisplayName string           `json:"displayName"`
	Description string           `json:"description,omitempty"`

	State    artifactModel.State `json:"state"`
	Enabled  bool                `json:"enabled"`
	Revision uint64              `json:"revision"`

	MemberCount int  `json:"memberCount"`
	BuiltIn     bool `json:"builtIn"`
	Editable    bool `json:"editable"`
	Deletable   bool `json:"deletable"`
	Baseline    bool `json:"baseline"`
}

type collectionMemberShape struct {
	Type declaration.Type
	Form declaration.MemberForm
}

type collectionProjection struct {
	document pluginv1.PluginDocument
	members  []collectionMemberShape
}

func (a *API) listCollections(
	ctx context.Context,
	request ListRequest,
	domainOnly bool,
) ([]ListItem, error) {
	if a == nil || a.artifacts == nil {
		return nil, spec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Collection list context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := request.RootID.Validate(); err != nil {
		return nil, err
	}

	entries, err := a.cat.ListByRoot(
		ctx,
		request.RootID,
		catalogModel.ListOptions{Kind: artifactModel.ArtifactKind(pluginv1.PluginType)},
	)
	if err != nil {
		return nil, err
	}

	documents, err := a.collectionDocuments(ctx, entries)
	if err != nil {
		return nil, err
	}

	output := make([]ListItem, 0)
	for _, entry := range entries {
		if entry.Kind != artifactModel.ArtifactKind(pluginv1.PluginType) ||
			entry.State != artifactModel.StateAvailable ||
			entry.Binding.SubresourceLocator != "" ||
			entry.Definition == nil {
			continue
		}

		key := definitionModel.Key{
			RootID: entry.RootID,
			Digest: entry.Definition.Digest,
		}
		var document *definitionModel.Definition
		if value, found := documents[key]; found {
			copyValue := value.Clone()
			document = &copyValue
		}
		projection, err := a.collectionProjectionFor(entry, document)
		if err != nil {
			return nil, err
		}
		if domainOnly {
			visible, err := a.collectionVisibleInList(
				entry,
				projection,
			)
			if err != nil {
				return nil, err
			}
			if !visible {
				continue
			}
		}

		if a.domain != nil && a.domain.ValidateDocument != nil {
			document, err := projection.document.Clone()
			if err != nil {
				return nil, err
			}
			if err := a.domain.ValidateDocument(
				document,
			); err != nil {
				return nil, err
			}
		}

		editable, baseline := a.collectionListEditability(
			entry,
			projection,
		)

		displayName := projection.document.DisplayName
		if displayName == "" {
			displayName = entry.DisplayName
		}

		item := ListItem{
			Ref:         entry.Ref(),
			SourceID:    entry.Source.ID,
			Name:        entry.LogicalName,
			DisplayName: displayName,
			Description: projection.document.Description,
			State:       entry.State,
			Enabled:     entry.Enabled,
			Revision:    entry.Revision,
			MemberCount: len(projection.members),
			BuiltIn:     entry.Ref().RootID == topology.BuiltinRootID(),
			Editable:    editable,
			Deletable:   editable && !baseline && len(projection.members) == 0,
			Baseline:    baseline,
		}
		output = append(output, item)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		return output[left].Ref.ArtifactID <
			output[right].Ref.ArtifactID
	})
	return output, nil
}

func (a *API) collectionDocuments(
	ctx context.Context,
	entries []catalogModel.Entry,
) (map[definitionModel.Key]definitionModel.Definition, error) {
	keys := make([]definitionModel.Key, 0)
	for _, entry := range entries {
		if entry.State != artifactModel.StateAvailable ||
			entry.Binding.SubresourceLocator != "" ||
			entry.Definition == nil {
			continue
		}
		key := definitionModel.Key{
			RootID: entry.RootID,
			Digest: entry.Definition.Digest,
		}
		if _, found := a.catalogProjections.Get(key); found {
			continue
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return map[definitionModel.Key]definitionModel.Definition{}, nil
	}

	values, err := a.definitions.GetDefinitions(ctx, keys)
	if err != nil {
		return nil, err
	}
	if len(values) != len(keys) {
		return nil, fmt.Errorf(
			"%w: Collection Definition batch is incomplete",
			spec.ErrDefinitionNotFound,
		)
	}

	output := make(map[definitionModel.Key]definitionModel.Definition, len(values))
	for index, value := range values {
		if value.Digest != keys[index].Digest {
			return nil, fmt.Errorf(
				"%w: Collection Definition batch returned another digest",
				spec.ErrDigestMismatch,
			)
		}
		output[keys[index]] = value.Clone()
	}
	return output, nil
}

func (a *API) collectionProjectionFor(
	entry catalogModel.Entry,
	loaded *definitionModel.Definition,
) (collectionProjection, error) {
	if entry.Definition == nil {
		return collectionProjection{}, fmt.Errorf(
			"%w: Collection Definition is unavailable",
			spec.ErrDefinitionNotFound,
		)
	}

	key := definitionModel.Key{
		RootID: entry.RootID,
		Digest: entry.Definition.Digest,
	}
	if cached, found := a.catalogProjections.Get(key); found {
		if cached.document.Name != string(entry.LogicalName) {
			return collectionProjection{}, fmt.Errorf(
				"%w: cached Plugin identity differs from catalog identity",
				spec.ErrDigestMismatch,
			)
		}
		return cached, nil
	}
	if loaded == nil {
		return collectionProjection{}, fmt.Errorf(
			"%w: Collection listing requires an admitted Definition document",
			spec.ErrDefinitionNotFound,
		)
	}

	return a.catalogProjections.GetOrLoad(
		key,
		len(loaded.Body),
		func() (collectionProjection, error) {
			document, err := pluginv1.FromDefinition(*loaded)
			if err != nil {
				return collectionProjection{}, err
			}
			if document.Name != string(entry.LogicalName) {
				return collectionProjection{}, fmt.Errorf(
					"%w: Collection Definition identity differs from catalog identity",
					spec.ErrDigestMismatch,
				)
			}
			projection := collectionProjection{
				document: document,
				members:  make([]collectionMemberShape, 0, len(document.Members)),
			}
			for _, member := range document.Members {
				form, err := member.MemberForm()
				if err != nil {
					return collectionProjection{}, err
				}
				projection.members = append(
					projection.members,
					collectionMemberShape{
						Type: member.Header().Type,
						Form: form,
					},
				)
			}
			return projection, nil
		},
	)
}

func (a *API) collectionVisibleInList(
	entry catalogModel.Entry,
	projection collectionProjection,
) (bool, error) {
	if a.domain == nil {
		return true, nil
	}

	if a.domain.ReadOnly {
		if entry.Source.Kind != managedfs.Kind ||
			entry.Binding.SubresourceLocator != "" {
			return false, nil
		}
		address, err := a.managedCollectionAddressFromLocator(
			entry.Binding.Locator,
		)
		if err != nil || address.Name != entry.LogicalName {
			//nolint:nilerr // Ok.
			return false, nil
		}
		return true, a.validateEditableProjection(projection)
	}

	if entry.Source.StorageKey == a.domain.SourceStorageKey {
		return true, a.validateEditableProjection(projection)
	}
	if len(projection.members) == 0 {
		return false, nil
	}
	for _, member := range projection.members {
		if !a.domain.allows(member.Type) {
			return false, nil
		}
	}
	return true, nil
}

func (a *API) collectionListEditability(
	entry catalogModel.Entry,
	projection collectionProjection,
) (editable, baseline bool) {
	if entry.Ref().RootID == topology.BuiltinRootID() ||
		entry.Source.Kind != managedfs.Kind ||
		!entry.Source.Enabled ||
		entry.Binding.SubresourceLocator != "" ||
		(a.domain != nil &&
			(a.domain.ReadOnly ||
				entry.Source.StorageKey != a.domain.SourceStorageKey)) {
		return false, false
	}

	address, err := a.managedCollectionAddressFromLocator(
		entry.Binding.Locator,
	)
	if err != nil || address.Name != entry.LogicalName ||
		a.validateEditableProjection(projection) != nil {
		return false, false
	}

	if a.domain == nil {
		return true, false
	}
	return true,
		entry.LogicalName == a.domain.BaselineName &&
			address.Name == a.domain.BaselineName
}

func (a *API) validateEditableProjection(
	projection collectionProjection,
) error {
	for _, member := range projection.members {
		if member.Form != declaration.MemberNamed {
			return fmt.Errorf(
				"%w: editable Collection has a non-named member",
				spec.ErrUnsupported,
			)
		}
		if a.domain != nil &&
			(!a.domain.allows(member.Type) ||
				!a.domain.allowsMemberForm(member.Form)) {
			return fmt.Errorf(
				"%w: Collection member is outside the domain policy",
				spec.ErrUnsupported,
			)
		}
	}
	return nil
}
