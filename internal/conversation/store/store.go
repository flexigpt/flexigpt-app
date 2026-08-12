package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/conversation/spec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	"github.com/flexigpt/flexigpt-app/internal/workspace/selection"
	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/dirpartition"
	"github.com/flexigpt/mapstore-go/ftsengine"
	"github.com/flexigpt/mapstore-go/jsonencdec"
	"github.com/flexigpt/mapstore-go/uuidv7filename"
)

type ConversationCollection struct {
	baseDir   string
	enableFTS bool
	store     *mapstore.MapDirectoryStore
	fts       *ftsengine.Engine
	pp        mapstore.PartitionProvider

	ftsRebuildCtx    context.Context
	ftsRebuildCancel context.CancelFunc
	ftsRebuildWG     sync.WaitGroup
}

type Option func(*ConversationCollection) error

func WithPartitionProvider(pp mapstore.PartitionProvider) Option {
	return func(cc *ConversationCollection) error {
		cc.pp = pp
		return nil
	}
}

func WithFTS(enabled bool) Option {
	return func(cc *ConversationCollection) error {
		cc.enableFTS = enabled
		return nil
	}
}

// NewConversationCollection creates a collection with sensible defaults
// (UUID-v7 file names under yyyyMM partitions).  Callers may override either
// strategy via the Option functions above.
//
//	baseDir is the root directory for the map-directory store.
func NewConversationCollection(baseDir string, opts ...Option) (*ConversationCollection, error) {
	defPP := dirpartition.MonthPartitionProvider{
		TimeFn: func(fileKey mapstore.FileKey) (time.Time, error) {
			u, err := uuidv7filename.Parse(fileKey.FileName)
			if err != nil {
				return time.Time{}, err
			}
			return u.Time, nil
		},
	}

	cc := &ConversationCollection{
		baseDir: filepath.Clean(baseDir),
		pp:      &defPP,
	}

	for _, o := range opts {
		if err := o(cc); err != nil {
			return nil, err
		}
	}

	// Optional full-text engine.
	if cc.enableFTS {
		var err error
		cc.fts, err = ftsengine.NewEngine(ftsengine.Config{
			BaseDir:    baseDir,
			DBFileName: "conversations.fts.sqlite",
			Table:      "conversations",
			Columns: []ftsengine.Column{
				{Name: "title", Weight: 1},
				{Name: "system", Weight: 2},
				{Name: "user", Weight: 3},
				{Name: "assistant", Weight: 4},
				{Name: "mtime", Unindexed: true},
			},
		}, ftsengine.WithLogger(slog.Default()))
		if err != nil {
			return nil, err
		}
		// Start rebuild with a cancellable context so we can shutdown cleanly.
		cc.ftsRebuildCtx, cc.ftsRebuildCancel = context.WithCancel(context.Background())
		cc.ftsRebuildWG.Go(func() {
			stat, _ := ftsengine.SyncDirToFTS(
				cc.ftsRebuildCtx,
				cc.fts,
				baseDir,
				"mtime",
				1000,
				processFTSDataForFile,
			)
			if stat != nil {
				slog.Info("conversation fts rebuild", "stat", stat)
			}
		})
	}

	optsDir := []mapstore.DirOption{mapstore.WithDirLogger(slog.Default())}
	if cc.fts != nil {
		optsDir = append(optsDir, mapstore.WithDirFileListeners(NewFTSListner(cc.fts)))
	}
	store, err := mapstore.NewMapDirectoryStore(baseDir, true, cc.pp, jsonencdec.JSONEncoderDecoder{}, optsDir...)
	if err != nil {
		return nil, err
	}
	cc.store = store
	return cc, nil
}

// Close releases resources.
func (cc *ConversationCollection) Close() (err error) {
	// Stop rebuild goroutine first (avoid it using the engine while we close it).
	if cc.ftsRebuildCancel != nil {
		cc.ftsRebuildCancel()
		cc.ftsRebuildWG.Wait()
		cc.ftsRebuildCancel = nil
	}

	if cc.store != nil {
		err = cc.store.CloseAll()
	}

	// Close sqlite handle.
	if cc.fts != nil {
		err = cc.fts.Close()
		cc.fts = nil
	}
	return err
}

func (cc *ConversationCollection) PutConversation(
	ctx context.Context,
	req *spec.PutConversationRequest,
) (*spec.PutConversationResponse, error) {
	if req == nil || req.Body == nil || req.ID == "" || req.Body.Title == "" {
		return nil, errors.New("request or request body cannot be nil")
	}
	if req.ID == "" || req.Body.Title == "" {
		return nil, errors.New("request ID an title are required")
	}

	// Get filename from info.
	info, err := uuidv7filename.Build(req.ID, req.Body.Title, spec.ConversationFileExtension)
	if err != nil {
		return nil, err
	}
	filename := info.FileName
	partitionDirName, err := cc.pp.GetPartitionDir(mapstore.FileKey{FileName: filename})
	if err != nil {
		return nil, err
	}

	currentConversation := &spec.Conversation{
		SchemaVersion: spec.ConversationSchemaVersion,
		ID:            req.ID,
		Title:         req.Body.Title,
		CreatedAt:     req.Body.CreatedAt,
		ModifiedAt:    req.Body.ModifiedAt,
		Messages:      req.Body.Messages,
		Meta:          req.Body.Meta,
	}
	if err := validateConversationV1(currentConversation); err != nil {
		return nil, err
	}
	data, err := jsonencdec.StructWithJSONTagsToMap(currentConversation)
	if err != nil {
		return nil, err
	}

	// Check if there are files with same id as prefix
	// We don't iterate as we expect only 1 file max with the id prefix of uuid.
	fileEntries, _, err := cc.store.ListFiles(
		mapstore.ListingConfig{
			FilenamePrefix:   req.ID,
			PageSize:         10,
			FilterPartitions: []string{partitionDirName},
		},
		"",
	)
	if err != nil {
		return nil, err
	}

	// Persist the valid replacement first. A title change can then remove the
	// prior filename without risking loss of the previous conversation when
	// validation or the replacement write fails.
	if err := cc.store.SetFileData(
		mapstore.FileKey{FileName: filename},
		data,
	); err != nil {
		return nil, err
	}
	for idx := range fileEntries {
		existing := filepath.Base(fileEntries[idx].BaseRelativePath)
		if existing == filename {
			continue
		}
		err := cc.store.DeleteFile(
			mapstore.FileKey{FileName: existing},
		)
		if err != nil {
			slog.Warn("put conversation remove existing file", "error", err)
		}
	}
	return &spec.PutConversationResponse{}, nil
}

func (cc *ConversationCollection) PutMessagesToConversation(
	ctx context.Context,
	req *spec.PutMessagesToConversationRequest,
) (*spec.PutMessagesToConversationResponse, error) {
	if req == nil || req.Body == nil || req.Body.Messages == nil || len(req.Body.Messages) == 0 {
		return nil, errors.New("request or request body cannot be nil")
	}

	convoResp, err := cc.GetConversation(ctx,
		&spec.GetConversationRequest{ID: req.ID, Title: req.Body.Title, ForceFetch: false})
	if err != nil {
		return nil, err
	}

	currentConversation := convoResp.Body
	currentConversation.ModifiedAt = time.Now().UTC()
	currentConversation.Messages = req.Body.Messages

	if err := validateConversationV1(currentConversation); err != nil {
		return nil, err
	}
	filename, err := cc.fileNameFromConversation(*currentConversation)
	if err != nil {
		return nil, err
	}

	data, err := jsonencdec.StructWithJSONTagsToMap(currentConversation)
	if err != nil {
		return nil, err
	}
	if err := cc.store.SetFileData(mapstore.FileKey{FileName: filename}, data); err != nil {
		return nil, err
	}

	return &spec.PutMessagesToConversationResponse{}, nil
}

func (cc *ConversationCollection) DeleteConversation(
	ctx context.Context,
	req *spec.DeleteConversationRequest,
) (*spec.DeleteConversationResponse, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	info, err := uuidv7filename.Build(req.ID, req.Title, spec.ConversationFileExtension)
	if err != nil {
		return nil, err
	}
	filename := info.FileName

	if err := cc.store.DeleteFile(mapstore.FileKey{FileName: filename}); err != nil {
		return nil, err
	}
	slog.Info("delete conversation", "file", filename)
	return &spec.DeleteConversationResponse{}, nil
}

func (cc *ConversationCollection) GetConversation(
	ctx context.Context,
	req *spec.GetConversationRequest,
) (*spec.GetConversationResponse, error) {
	if req == nil || req.Title == "" || req.ID == "" {
		return nil, errors.New("request or request body cannot be nil")
	}
	info, err := uuidv7filename.Build(req.ID, req.Title, spec.ConversationFileExtension)
	if err != nil {
		return nil, err
	}
	filename := info.FileName

	raw, err := cc.store.GetFileData(mapstore.FileKey{FileName: filename}, req.ForceFetch)
	if err != nil {
		return nil, err
	}
	if schemaVersion, _ := stringField(raw, "schemaVersion"); schemaVersion !=
		spec.ConversationSchemaVersion {
		return nil, errors.New("unsupported schema version for conversation")
	}

	var convo spec.Conversation
	if err := jsonencdec.MapToStructWithJSONTags(raw, &convo); err != nil {
		return nil, err
	}
	if convo.SchemaVersion != spec.ConversationSchemaVersion {
		return nil, errors.New("unsupported schema version for conversation")
	}
	if err := validateConversationV1(&convo); err != nil {
		return nil, err
	}

	return &spec.GetConversationResponse{Body: &convo}, nil
}

func (cc *ConversationCollection) ListConversations(
	ctx context.Context,
	req *spec.ListConversationsRequest,
) (*spec.ListConversationsResponse, error) {
	token := ""
	pageSize := spec.DefaultPageSize
	if req != nil {
		token = req.PageToken
		if req.PageSize > 0 && req.PageSize <= spec.MaxPageSize {
			pageSize = req.PageSize
		}
	}

	fileEntries, next, err := cc.store.ListFiles(
		mapstore.ListingConfig{SortOrder: mapstore.SortOrderDescending, PageSize: pageSize},
		token,
	)
	if err != nil {
		return nil, err
	}

	items := make([]spec.ConversationListItem, 0, len(fileEntries))
	for _, f := range fileEntries {
		filename := filepath.Base(f.BaseRelativePath)
		info, err := uuidv7filename.Parse(filename)
		if err != nil {
			// Corrupted/foreign file skip.
			continue
		}

		// Load JSON to check schemaVersion.
		raw, err := cc.store.GetFileData(mapstore.FileKey{FileName: filename}, false)
		if err != nil {
			// If we can't read it, treat as corrupted/legacy; skip.
			continue
		}
		var convo spec.Conversation
		if err := jsonencdec.MapToStructWithJSONTags(raw, &convo); err != nil {
			continue
		}
		if convo.SchemaVersion != spec.ConversationSchemaVersion {
			// Older conversations are intentionally not interpreted as v1.
			continue
		}
		if err := validateConversationV1(&convo); err != nil {
			continue
		}

		fileModTime := f.FileInfo.ModTime()
		items = append(items, spec.ConversationListItem{
			ID:             info.ID,
			SanatizedTitle: info.Suffix,
			ModifiedAt:     &fileModTime,
		})
	}

	return &spec.ListConversationsResponse{
		Body: &spec.ListConversationsResponseBody{
			ConversationListItems: items,
			NextPageToken:         &next,
		},
	}, nil
}

func (cc *ConversationCollection) SearchConversations(
	ctx context.Context,
	req *spec.SearchConversationsRequest,
) (*spec.SearchConversationsResponse, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if cc.fts == nil {
		return nil, errors.New("full-text search is disabled")
	}
	pageSize := spec.DefaultPageSize
	if req.PageSize > 0 && req.PageSize <= spec.MaxPageSize {
		pageSize = req.PageSize
	}

	hits, next, err := cc.fts.Search(ctx, req.Query, req.PageToken, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]spec.ConversationListItem, 0, len(hits))
	for _, h := range hits {
		info, err := uuidv7filename.Parse(filepath.Base(h.ID))
		if err != nil {
			continue
		}
		items = append(items, spec.ConversationListItem{
			ID:             info.ID,
			SanatizedTitle: info.Suffix,
		})
	}
	return &spec.SearchConversationsResponse{
		Body: &spec.SearchConversationsResponseBody{
			ConversationListItems: items,
			NextPageToken:         &next,
		},
	}, nil
}

func (cc *ConversationCollection) fileNameFromConversation(c spec.Conversation) (string, error) {
	info, err := uuidv7filename.Build(c.ID, c.Title, spec.ConversationFileExtension)
	if err != nil {
		return "", err
	}
	return info.FileName, nil
}

func validateConversationV1(value *spec.Conversation) error {
	if value == nil {
		return errors.New("conversation is nil")
	}
	if value.SchemaVersion != spec.ConversationSchemaVersion {
		return errors.New("unsupported schema version for conversation")
	}
	if strings.TrimSpace(value.ID) == "" {
		return errors.New("conversation ID is empty")
	}

	for index := range value.Messages {
		message := &value.Messages[index]
		if err := validateConversationArtifactRefs(
			fmt.Sprintf("messages[%d].enabledSkillRefs", index),
			message.EnabledSkillRefs,
		); err != nil {
			return err
		}
		if err := validateConversationArtifactRefs(
			fmt.Sprintf("messages[%d].activeSkillRefs", index),
			message.ActiveSkillRefs,
		); err != nil {
			return err
		}
		if message.MCPContext != nil {
			if err := runtime.ValidateMCPConversationContext(
				*message.MCPContext,
			); err != nil {
				return fmt.Errorf("messages[%d].mcpContext: %w", index, err)
			}
		}
		if len(message.MCPToolMappings) != 0 {
			if message.MCPContext == nil {
				return fmt.Errorf(
					"messages[%d].mcpToolMappings require mcpContext",
					index,
				)
			}
			if err := runtime.ValidateMCPProviderToolMappingsForContext(
				*message.MCPContext,
				message.MCPToolMappings,
			); err != nil {
				return fmt.Errorf("messages[%d].mcpToolMappings: %w", index, err)
			}
		}
		if len(message.MCPAppContextUpdates) != 0 {
			if message.MCPContext == nil {
				return fmt.Errorf(
					"messages[%d].mcpAppContextUpdates require mcpContext",
					index,
				)
			}
			if err := runtime.ValidateMCPAppContextUpdatesForContext(
				*message.MCPContext,
				message.MCPAppContextUpdates,
			); err != nil {
				return fmt.Errorf("messages[%d].mcpAppContextUpdates: %w", index, err)
			}
		}
		if message.WorkspaceSelection == nil {
			continue
		}

		selected := message.WorkspaceSelection
		if err := selected.Workspace.Validate(); err != nil {
			return fmt.Errorf(
				"messages[%d].workspaceSelection.workspace: %w",
				index,
				err,
			)
		}
		if err := validateConversationSelectionRefs(
			fmt.Sprintf(
				"messages[%d].workspaceSelection.contextRefs",
				index,
			),
			selected.ContextRefs,
			selected.Workspace.RootID,
		); err != nil {
			return err
		}
		if err := validateConversationSelectionRefs(
			fmt.Sprintf(
				"messages[%d].workspaceSelection.skillRefs",
				index,
			),
			selected.SkillRefs,
			selected.Workspace.RootID,
		); err != nil {
			return err
		}
	}
	return nil
}

func validateConversationArtifactRefs(
	field string,
	refs []artifact.ArtifactRef,
) error {
	seen := make(map[string]struct{}, len(refs))
	for index, ref := range refs {
		if err := ref.Validate(); err != nil {
			return fmt.Errorf("%s[%d]: %w", field, index, err)
		}
		key := string(ref.RootID) + "\x00" + string(ref.ArtifactID)
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("%s[%d]: duplicate ArtifactRef", field, index)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateConversationSelectionRefs(
	field string,
	refs []selection.ConversationResourceSelectionRef,
	rootID basespec.RootID,
) error {
	seen := make(map[string]struct{}, len(refs))
	for index, ref := range refs {
		if err := ref.Artifact.Validate(); err != nil {
			return fmt.Errorf("%s[%d].artifact: %w", field, index, err)
		}
		if ref.Artifact.RootID != rootID {
			return fmt.Errorf(
				"%s[%d].artifact: Artifact belongs to another Root",
				field,
				index,
			)
		}
		key := string(ref.Artifact.RootID) + "\x00" +
			string(ref.Artifact.ArtifactID)
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("%s[%d]: duplicate ArtifactRef", field, index)
		}
		seen[key] = struct{}{}
	}
	return nil
}
