package consumerapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

const (
	DefaultPageSize = 50
	MaxPageSize     = 200

	collectionPageKind = "collections"
	serverPageKind     = "servers"
)

type CollectionPage struct {
	Items         []collection.CollectionView `json:"items"`
	NextPageToken string                      `json:"nextPageToken,omitempty"`
}

type ServerPage struct {
	Items         []artifact.Artifact `json:"items"`
	NextPageToken string              `json:"nextPageToken,omitempty"`
}

type RootStore interface {
	List(ctx context.Context) ([]root.Root, error)
}

type Store interface {
	ListMCPCollections(
		ctx context.Context,
		rootID root.RootID,
	) ([]collection.CollectionView, error)

	ListServers(
		ctx context.Context,
		rootID root.RootID,
	) ([]artifact.Artifact, error)
}

type MCPListService struct {
	roots RootStore
	store Store
}

func NewMCPListService(
	roots RootStore,
	store Store,
) (*MCPListService, error) {
	if roots == nil || store == nil {
		return nil, fmt.Errorf(
			"%w: MCP management dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &MCPListService{
		roots: roots,
		store: store,
	}, nil
}

type pageCursor struct {
	Kind      string `json:"kind"`
	PageSize  int    `json:"pageSize"`
	AfterRoot string `json:"afterRoot,omitempty"`
	AfterName string `json:"afterName,omitempty"`
	AfterID   string `json:"afterID,omitempty"`
}

type pageKey struct {
	rootID root.RootID
	name   basespec.LogicalName
	id     artifact.ArtifactID
}

func (s *MCPListService) ListCollectionsPage(
	ctx context.Context,
	pageSize int,
	pageToken string,
) (CollectionPage, error) {
	cursor, err := decodeCursor(
		collectionPageKind,
		pageSize,
		pageToken,
	)
	if err != nil {
		return CollectionPage{}, err
	}

	roots, err := s.orderedRoots(ctx)
	if err != nil {
		return CollectionPage{}, err
	}
	items, next, err := pageAcrossRoots(
		ctx,
		roots,
		cursor,
		func(ctx context.Context, rootID root.RootID) ([]collection.CollectionView, error) {
			return s.store.ListMCPCollections(ctx, rootID)
		},
		func(value collection.CollectionView) pageKey {
			return pageKey{
				rootID: value.Artifact.RootID,
				name:   value.Name,
				id:     value.Artifact.ID,
			}
		},
	)
	if err != nil {
		return CollectionPage{}, err
	}
	if items == nil {
		items = []collection.CollectionView{}
	}
	return CollectionPage{
		Items:         items,
		NextPageToken: next,
	}, nil
}

func (s *MCPListService) ListServersPage(
	ctx context.Context,
	pageSize int,
	pageToken string,
) (ServerPage, error) {
	cursor, err := decodeCursor(
		serverPageKind,
		pageSize,
		pageToken,
	)
	if err != nil {
		return ServerPage{}, err
	}

	roots, err := s.orderedRoots(ctx)
	if err != nil {
		return ServerPage{}, err
	}
	items, next, err := pageAcrossRoots(
		ctx,
		roots,
		cursor,
		func(ctx context.Context, rootID root.RootID) ([]artifact.Artifact, error) {
			return s.store.ListServers(ctx, rootID)
		},
		func(value artifact.Artifact) pageKey {
			return pageKey{
				rootID: value.RootID,
				name:   value.LogicalName,
				id:     value.ID,
			}
		},
	)
	if err != nil {
		return ServerPage{}, err
	}
	if items == nil {
		items = []artifact.Artifact{}
	}
	return ServerPage{
		Items:         items,
		NextPageToken: next,
	}, nil
}

func (s *MCPListService) orderedRoots(
	ctx context.Context,
) ([]root.Root, error) {
	if s == nil || s.roots == nil {
		return nil, basespec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: MCP management context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	values, err := s.roots.List(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(values, func(left, right int) bool {
		return values[left].ID < values[right].ID
	})
	return values, nil
}

func pageAcrossRoots[T any](
	ctx context.Context,
	roots []root.Root,
	cursor pageCursor,
	list func(context.Context, root.RootID) ([]T, error),
	key func(T) pageKey,
) (items []T, next string, err error) {
	output := make([]T, 0, cursor.PageSize)

	for _, rootValue := range roots {
		// Earlier Roots cannot contribute another item after the cursor. Skip
		// their Store list calls entirely instead of relisting them on every
		// subsequent management page.
		if cursor.AfterRoot != "" &&
			rootValue.ID < root.RootID(cursor.AfterRoot) {
			continue
		}

		values, err := list(ctx, rootValue.ID)
		if err != nil {
			return nil, "", err
		}

		sort.Slice(values, func(left, right int) bool {
			return lessPageKey(key(values[left]), key(values[right]))
		})

		for _, value := range values {
			if err := ctx.Err(); err != nil {
				return nil, "", err
			}
			if !afterCursor(key(value), cursor) {
				continue
			}
			if len(output) == cursor.PageSize {
				nextCursor := cursor
				last := key(output[len(output)-1])
				nextCursor.AfterRoot = string(last.rootID)
				nextCursor.AfterName = string(last.name)
				nextCursor.AfterID = string(last.id)

				token, err := encodeCursor(nextCursor)
				if err != nil {
					return nil, "", err
				}
				return output, token, nil
			}
			output = append(output, value)
		}
	}
	return output, "", nil
}

func lessPageKey(left, right pageKey) bool {
	if left.rootID != right.rootID {
		return left.rootID < right.rootID
	}
	if left.name != right.name {
		return left.name < right.name
	}
	return left.id < right.id
}

func afterCursor(value pageKey, cursor pageCursor) bool {
	if cursor.AfterRoot == "" {
		return true
	}
	return lessPageKey(
		pageKey{
			rootID: root.RootID(cursor.AfterRoot),
			name:   basespec.LogicalName(cursor.AfterName),
			id:     artifact.ArtifactID(cursor.AfterID),
		},
		value,
	)
}

func decodeCursor(
	kind string,
	pageSize int,
	pageToken string,
) (pageCursor, error) {
	if pageToken == "" {
		return pageCursor{
			Kind:     kind,
			PageSize: normalizePageSize(pageSize),
		}, nil
	}

	raw, err := base64.RawURLEncoding.DecodeString(pageToken)
	if err != nil {
		return pageCursor{}, fmt.Errorf(
			"%w: invalid MCP management page token",
			basespec.ErrInvalid,
		)
	}

	var cursor pageCursor
	if err := json.Unmarshal(raw, &cursor); err != nil {
		return pageCursor{}, fmt.Errorf(
			"%w: invalid MCP management page token",
			basespec.ErrInvalid,
		)
	}
	if cursor.Kind != kind ||
		cursor.PageSize <= 0 ||
		cursor.PageSize > MaxPageSize {
		return pageCursor{}, fmt.Errorf(
			"%w: stale MCP management page token",
			basespec.ErrConflict,
		)
	}

	if cursor.AfterRoot == "" &&
		cursor.AfterName == "" &&
		cursor.AfterID == "" {
		return cursor, nil
	}
	if cursor.AfterRoot == "" ||
		cursor.AfterName == "" ||
		cursor.AfterID == "" {
		return pageCursor{}, fmt.Errorf(
			"%w: invalid MCP management page token",
			basespec.ErrInvalid,
		)
	}
	if err := root.RootID(cursor.AfterRoot).Validate(); err != nil {
		return pageCursor{}, err
	}
	if err := basespec.LogicalName(cursor.AfterName).Validate(); err != nil {
		return pageCursor{}, err
	}
	if err := artifact.ArtifactID(cursor.AfterID).Validate(); err != nil {
		return pageCursor{}, err
	}
	return cursor, nil
}

func encodeCursor(
	cursor pageCursor,
) (string, error) {
	raw, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func normalizePageSize(value int) int {
	if value <= 0 {
		return DefaultPageSize
	}
	if value > MaxPageSize {
		return MaxPageSize
	}
	return value
}
