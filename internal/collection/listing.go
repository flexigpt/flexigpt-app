package collection

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
)

// ListRequest is the public Collection listing request.
//
// It contains only domain-owned listing choices. Generic Artifact Store query
// details remain inside the Collection API.
type ListRequest struct {
	RootID root.RootID `json:"rootID"`
}

// ListItem is a lightweight Collection projection.
//
// Members are not returned in ordinary listings. The member count is enough
// for management pages to determine whether deletion can be offered. Read
// returns CollectionView when a caller explicitly selects one Collection.
type ListItem struct {
	Ref      artifact.ArtifactRef `json:"ref"`
	SourceID source.SourceID      `json:"sourceID"`

	Name        model.LogicalName `json:"name"`
	DisplayName string            `json:"displayName"`
	Description string            `json:"description,omitempty"`

	State    artifact.State `json:"state"`
	Enabled  bool           `json:"enabled"`
	Revision uint64         `json:"revision"`

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
		return nil, model.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Collection list context is nil",
			model.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := request.RootID.Validate(); err != nil {
		return nil, err
	}

	entries, err := a.artifacts.ListByRoot(
		ctx,
		request.RootID,
		catalog.ListOptions{Kind: artifact.ArtifactKind(pluginv1.PluginType)},
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
		if entry.Kind != artifact.ArtifactKind(pluginv1.PluginType) ||
			entry.State != artifact.StateAvailable ||
			entry.Binding.SubresourceLocator != "" ||
			entry.Definition == nil {
			continue
		}

		key := definition.Key{
			RootID: entry.RootID,
			Digest: entry.Definition.Digest,
		}
		var document *definition.Definition
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
			if err := a.domain.ValidateDocument(
				projection.document,
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
			BuiltIn:     entry.Ref().RootID == documentTopology.BuiltinRootID(),
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
	entries []catalog.Entry,
) (map[definition.Key]definition.Definition, error) {
	keys := make([]definition.Key, 0)
	for _, entry := range entries {
		if entry.State != artifact.StateAvailable ||
			entry.Binding.SubresourceLocator != "" ||
			entry.Definition == nil {
			continue
		}
		key := definition.Key{
			RootID: entry.RootID,
			Digest: entry.Definition.Digest,
		}
		if _, found := a.catalogProjections.Get(key); found {
			continue
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return map[definition.Key]definition.Definition{}, nil
	}

	values, err := a.artifacts.GetDefinitions(ctx, keys)
	if err != nil {
		return nil, err
	}
	if len(values) != len(keys) {
		return nil, fmt.Errorf(
			"%w: Collection Definition batch is incomplete",
			model.ErrDefinitionNotFound,
		)
	}

	output := make(map[definition.Key]definition.Definition, len(values))
	for index, value := range values {
		if value.Digest != keys[index].Digest {
			return nil, fmt.Errorf(
				"%w: Collection Definition batch returned another digest",
				model.ErrDigestMismatch,
			)
		}
		output[keys[index]] = value.Clone()
	}
	return output, nil
}

func (a *API) collectionProjectionFor(
	entry catalog.Entry,
	loaded *definition.Definition,
) (collectionProjection, error) {
	if entry.Definition == nil {
		return collectionProjection{}, fmt.Errorf(
			"%w: Collection Definition is unavailable",
			model.ErrDefinitionNotFound,
		)
	}

	key := definition.Key{
		RootID: entry.RootID,
		Digest: entry.Definition.Digest,
	}
	if cached, found := a.catalogProjections.Get(key); found {
		return cached, nil
	}
	if loaded == nil {
		return collectionProjection{}, fmt.Errorf(
			"%w: Collection listing requires an admitted Definition document",
			model.ErrDefinitionNotFound,
		)
	}

	return a.catalogProjections.GetOrLoad(
		key,
		len(loaded.Body),
		func() (collectionProjection, error) {
			document, err := pluginv1.DecodePluginJSON(loaded.Body)
			if err != nil {
				return collectionProjection{}, err
			}
			if document.Name != string(entry.LogicalName) {
				return collectionProjection{}, fmt.Errorf(
					"%w: Collection Definition identity differs from catalog identity",
					model.ErrDigestMismatch,
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
	entry catalog.Entry,
	projection collectionProjection,
) (bool, error) {
	if a.domain == nil {
		return true, nil
	}

	if a.domain.ReadOnly {
		if entry.Source.Kind != source.SourceKindManagedDirectory ||
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
	entry catalog.Entry,
	projection collectionProjection,
) (editable, baseline bool) {
	if entry.Ref().RootID == documentTopology.BuiltinRootID() ||
		entry.Source.Kind != source.SourceKindManagedDirectory ||
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
				model.ErrUnsupported,
			)
		}
		if a.domain != nil &&
			(!a.domain.allows(member.Type) ||
				!a.domain.allowsMemberForm(member.Form)) {
			return fmt.Errorf(
				"%w: Collection member is outside the domain policy",
				model.ErrUnsupported,
			)
		}
	}
	return nil
}
