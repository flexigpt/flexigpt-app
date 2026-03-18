package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/conversation/spec"
	"github.com/google/uuid"
	"github.com/ppipada/mapstore-go/uuidv7filename"

	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func TestInitConversation(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		expectedTitle string
	}{
		{"Empty title", "", "New Conversation"},
		{"Short title", "Chat with AI", "Chat with AI"},
		{
			"Long title",
			"This is a very long title that should be truncated to fit within the maximum allowed length of 64 characters",
			"This is a very long title that should be truncated to fit within",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			convo, err := initConversation(tt.title)
			if err != nil {
				t.Errorf("Expected valid convo, got error: %v", err)
			}
			if convo.Title != tt.expectedTitle {
				t.Errorf("Expected title %s, got %s", tt.expectedTitle, convo.Title)
			}
			if len(convo.Title) > 64 {
				t.Errorf(
					"Expected title to be truncated to 64 characters, got %d",
					len(convo.Title),
				)
			}
			if _, err := uuid.Parse(convo.ID); err != nil {
				t.Errorf("Expected valid UUID, got error: %v", err)
			}
		})
	}
}

func TestConversationCollection(t *testing.T) {
	baseDir := t.TempDir()
	defer os.RemoveAll(baseDir)
	cc, err := NewConversationCollection(baseDir)
	if err != nil {
		t.Fatalf("Failed to create conversation collection: %v", err)
	}

	t.Run("Save and Get Conversation", func(t *testing.T) {
		validConvo, err := initConversation("Test Conversation")
		if err != nil {
			t.Errorf("Failed to init conversation: %v", err)
		}
		emptyConvo, err := initConversation("")
		if err != nil {
			t.Errorf("Failed to init conversation: %v", err)
		}
		tests := []struct {
			name         string
			conversation spec.Conversation
			expectError  bool
		}{
			{"Valid conversation", *validConvo, false},
			{"Empty title", *emptyConvo, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx := t.Context()
				_, err := cc.PutConversation(
					ctx,
					getNewPutRequestFromConversation(&tt.conversation),
				)

				if (err != nil) != tt.expectError {
					t.Fatalf("Expected error: %v, got: %v", tt.expectError, err)
				}

				retrievedConvo, err := cc.GetConversation(
					ctx,
					&spec.GetConversationRequest{
						ID:    tt.conversation.ID,
						Title: tt.conversation.Title,
					},
				)
				if err != nil {
					t.Fatalf("Failed to get conversation: %v", err)
				}

				if retrievedConvo.Body.ID != tt.conversation.ID ||
					retrievedConvo.Body.Title != tt.conversation.Title {
					t.Errorf("Retrieved conversation does not match saved conversation")
				}
			})
		}
	})

	t.Run("Delete Conversation", func(t *testing.T) {
		ctx := t.Context()
		convo, err := initConversation("To Be Deleted")
		if err != nil {
			t.Errorf("Failed to init conversation: %v", err)
		}
		_, err = cc.PutConversation(ctx, getNewPutRequestFromConversation(convo))
		if err != nil {
			t.Errorf("Failed to save conversation: %v", err)
		}

		_, err = cc.DeleteConversation(
			ctx,
			&spec.DeleteConversationRequest{ID: convo.ID, Title: convo.Title},
		)
		if err != nil {
			t.Errorf("Failed to delete conversation: %v", err)
		}

		_, err = cc.GetConversation(
			ctx,
			&spec.GetConversationRequest{ID: convo.ID, Title: convo.Title},
		)
		if err == nil {
			t.Errorf("Expected error when getting deleted conversation, got none")
		}
	})

	t.Run("Add Message to Conversation", func(t *testing.T) {
		ctx := t.Context()
		convo, err := initConversation("Message Test")
		if err != nil {
			t.Errorf("Failed to init conversation: %v", err)
		}
		_, err = cc.PutConversation(ctx, getNewPutRequestFromConversation(convo))
		if err != nil {
			t.Errorf("Failed to save conversation: %v", err)
		}
		tests := []struct {
			name        string
			message     spec.ConversationMessage
			expectError bool
		}{
			{
				"Valid message",
				newTextTurn("msg1", inferenceSpec.RoleUser, "Hello"),
				false,
			},
			{
				"Empty content",
				newTextTurn("msg2", inferenceSpec.RoleUser, ""),
				false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx := t.Context()
				_, err := cc.PutMessagesToConversation(
					ctx,
					&spec.PutMessagesToConversationRequest{
						ID: convo.ID,
						Body: &spec.PutMessagesToConversationRequestBody{
							Title:    convo.Title,
							Messages: []spec.ConversationMessage{tt.message},
						},
					},
				)
				if (err != nil) != tt.expectError {
					t.Fatalf("Expected error: %v, got: %v", tt.expectError, err)
				}

				retrievedConvo, err := cc.GetConversation(
					ctx,
					&spec.GetConversationRequest{ID: convo.ID, Title: convo.Title},
				)
				if err != nil {
					t.Fatalf("Failed to get conversation: %v", err)
				}

				if len(retrievedConvo.Body.Messages) == 0 ||
					len(retrievedConvo.Body.Messages[len(retrievedConvo.Body.Messages)-1].Inputs) == 0 ||
					retrievedConvo.Body.Messages[len(retrievedConvo.Body.Messages)-1].Inputs[0].InputMessage == nil ||
					len(
						retrievedConvo.Body.Messages[len(retrievedConvo.Body.Messages)-1].Inputs[0].InputMessage.Contents,
					) == 0 ||
					retrievedConvo.Body.Messages[len(retrievedConvo.Body.Messages)-1].Inputs[0].InputMessage.Contents[0].TextItem.Text != tt.message.Inputs[0].InputMessage.Contents[0].TextItem.Text {

					t.Errorf("Message not added correctly to conversation")
				}
			})
		}
	})

	// NEW: Overwrite/replace scenarios.
	t.Run("Overwrite Conversation With Same ID", func(t *testing.T) {
		ctx := t.Context()
		convo, err := initConversation("Original Title")
		if err != nil {
			t.Fatalf("Failed to init conversation: %v", err)
		}
		_, err = cc.PutConversation(ctx, getNewPutRequestFromConversation(convo))
		if err != nil {
			t.Fatalf("Failed to save conversation: %v", err)
		}

		// Overwrite with new title.
		convo.Title = "Updated Title"
		_, err = cc.PutConversation(ctx, getNewPutRequestFromConversation(convo))
		if err != nil {
			t.Fatalf("Failed to overwrite conversation: %v", err)
		}

		retrieved, err := cc.GetConversation(
			ctx,
			&spec.GetConversationRequest{ID: convo.ID, Title: convo.Title},
		)
		if err != nil {
			t.Fatalf("Failed to get overwritten conversation: %v", err)
		}
		if retrieved.Body.Title != "Updated Title" {
			t.Errorf("Expected updated title, got %s", retrieved.Body.Title)
		}
	})

	// NEW: Error handling for PutConversation.
	t.Run("PutConversation Error Cases", func(t *testing.T) {
		ctx := t.Context()
		_, err := cc.PutConversation(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil request")
		}
		_, err = cc.PutConversation(ctx, &spec.PutConversationRequest{})
		if err == nil {
			t.Error("Expected error for missing ID and body")
		}
		_, err = cc.PutConversation(ctx, &spec.PutConversationRequest{ID: "id", Body: nil})
		if err == nil {
			t.Error("Expected error for nil body")
		}
		_, err = cc.PutConversation(
			ctx,
			&spec.PutConversationRequest{
				ID:   "",
				Body: &spec.PutConversationRequestBody{Title: "t"},
			},
		)
		if err == nil {
			t.Error("Expected error for missing ID")
		}
		_, err = cc.PutConversation(
			ctx,
			&spec.PutConversationRequest{
				ID:   "id",
				Body: &spec.PutConversationRequestBody{Title: ""},
			},
		)
		if err == nil {
			t.Error("Expected error for missing title")
		}
	})

	// NEW: Error handling for PutMessagesToConversation.
	t.Run("PutMessagesToConversation Error Cases", func(t *testing.T) {
		ctx := t.Context()
		_, err := cc.PutMessagesToConversation(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil request")
		}
		_, err = cc.PutMessagesToConversation(ctx, &spec.PutMessagesToConversationRequest{})
		if err == nil {
			t.Error("Expected error for nil body")
		}
		_, err = cc.PutMessagesToConversation(
			ctx,
			&spec.PutMessagesToConversationRequest{
				Body: &spec.PutMessagesToConversationRequestBody{},
			},
		)
		if err == nil {
			t.Error("Expected error for nil messages")
		}
		_, err = cc.PutMessagesToConversation(
			ctx,
			&spec.PutMessagesToConversationRequest{
				Body: &spec.PutMessagesToConversationRequestBody{
					Messages: []spec.ConversationMessage{},
				},
			},
		)
		if err == nil {
			t.Error("Expected error for empty messages")
		}
	})

	// NEW: Error handling for DeleteConversation.
	t.Run("DeleteConversation Error Cases", func(t *testing.T) {
		ctx := t.Context()
		_, err := cc.DeleteConversation(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil request")
		}
		// Non-existent conversation.
		_, err = cc.DeleteConversation(
			ctx,
			&spec.DeleteConversationRequest{ID: "nonexistent", Title: "nonexistent"},
		)
		if err == nil {
			t.Error("Expected error for deleting non-existent conversation")
		}
	})

	// NEW: Error handling for GetConversation.
	t.Run("GetConversation Error Cases", func(t *testing.T) {
		ctx := t.Context()
		_, err := cc.GetConversation(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil request")
		}
		_, err = cc.GetConversation(
			ctx,
			&spec.GetConversationRequest{ID: "nonexistent", Title: "nonexistent"},
		)
		if err == nil {
			t.Error("Expected error for getting non-existent conversation")
		}
	})

	// NEW: Add messages to non-existent conversation.
	t.Run("PutMessagesToNonExistentConversation", func(t *testing.T) {
		ctx := t.Context()
		_, err := cc.PutMessagesToConversation(ctx, &spec.PutMessagesToConversationRequest{
			ID: "nonexistent",
			Body: &spec.PutMessagesToConversationRequestBody{
				Title: "nonexistent",
				Messages: []spec.ConversationMessage{
					newTextTurn("msg", inferenceSpec.RoleUser, "hi"),
				},
			},
		})
		if err == nil {
			t.Error("Expected error for adding messages to non-existent conversation")
		}
	})

	// NEW: Multiple messages update.
	t.Run("MultipleMessagesUpdate", func(t *testing.T) {
		ctx := t.Context()
		convo, err := initConversation("MultiMsg")
		if err != nil {
			t.Fatalf("Failed to init conversation: %v", err)
		}
		_, err = cc.PutConversation(ctx, getNewPutRequestFromConversation(convo))
		if err != nil {
			t.Fatalf("Failed to save conversation: %v", err)
		}
		msgs := []spec.ConversationMessage{
			newTextTurn("m1", inferenceSpec.RoleUser, "hi"),
			newTextTurn("m2", inferenceSpec.RoleUser, "hello"),
		}
		_, err = cc.PutMessagesToConversation(ctx, &spec.PutMessagesToConversationRequest{
			ID: convo.ID,
			Body: &spec.PutMessagesToConversationRequestBody{
				Title:    convo.Title,
				Messages: msgs,
			},
		})
		if err != nil {
			t.Fatalf("Failed to add messages: %v", err)
		}
		retrieved, err := cc.GetConversation(
			ctx,
			&spec.GetConversationRequest{ID: convo.ID, Title: convo.Title},
		)
		if err != nil {
			t.Fatalf("Failed to get conversation: %v", err)
		}
		if len(retrieved.Body.Messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(retrieved.Body.Messages))
		}
	})

	// NEW: Partitioning logic (simulate different months).
	t.Run("PartitioningDifferentMonths", func(t *testing.T) {
		ctx := t.Context()
		// Jan.
		convo1 := &spec.Conversation{}
		u, err := uuidv7filename.NewUUIDv7String()
		if err != nil {
			t.Fatalf("Failed to get uuid: %v", err)
		}
		convo1.ID = u
		convo1.Title = "Jan"
		convo1.CreatedAt = time.Date(2023, 1, 15, 10, 0, 0, 0, time.UTC)
		convo1.ModifiedAt = time.Now().UTC()
		convo1.Messages = []spec.ConversationMessage{}

		// Feb.
		u, err = uuidv7filename.NewUUIDv7String()
		if err != nil {
			t.Fatalf("Failed to get uuid: %v", err)
		}

		convo2 := &spec.Conversation{}
		convo2.ID = u
		convo2.Title = "Feb"
		convo2.CreatedAt = time.Date(2023, 2, 15, 10, 0, 0, 0, time.UTC)
		convo2.ModifiedAt = time.Now().UTC()
		convo2.Messages = []spec.ConversationMessage{}

		_, err = cc.PutConversation(ctx, getNewPutRequestFromConversation(convo1))
		if err != nil {
			t.Fatalf("Failed to save Jan conversation: %v", err)
		}
		_, err = cc.PutConversation(ctx, getNewPutRequestFromConversation(convo2))
		if err != nil {
			t.Fatalf("Failed to save Feb conversation: %v", err)
		}
		resp, err := cc.ListConversations(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to list conversations: %v", err)
		}
		foundJan, foundFeb := false, false
		for _, item := range resp.Body.ConversationListItems {
			if item.SanatizedTitle == "Jan" {
				foundJan = true
			}
			if item.SanatizedTitle == "Feb" {
				foundFeb = true
			}
		}
		if !foundJan || !foundFeb {
			t.Errorf("Expected both Jan and Feb conversations to be listed")
		}
	})

	// NEW: Corrupted/invalid file handling.
	t.Run("CorruptedFileHandling", func(t *testing.T) {
		// Place a file with invalid name in the directory.
		partitionDir := filepath.Join(baseDir, time.Now().UTC().Format("200601"))
		_ = os.MkdirAll(partitionDir, 0o755)
		badFile := filepath.Join(partitionDir, "not-a-valid-convo-file.txt")
		_ = os.WriteFile(badFile, []byte("garbage"), 0o600)

		ctx := t.Context()
		resp, err := cc.ListConversations(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to list conversations: %v", err)
		}
		// Should not panic or include the bad file.
		for _, item := range resp.Body.ConversationListItems {
			if item.ID == "" || item.SanatizedTitle == "" {
				t.Errorf("Corrupted file should not be included in results")
			}
		}
	})
}

func TestConversationCollectionListing(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "conversationstore_test_list")
	defer os.RemoveAll(baseDir)

	cc, err := NewConversationCollection(baseDir)
	if err != nil {
		t.Fatalf("Failed to create conversation collection: %v", err)
	}

	t.Run("List Conversations", func(t *testing.T) {
		ctx := t.Context()
		convo1, err := initConversation("First Conversation")
		if err != nil {
			t.Errorf("Failed to init conversation: %v", err)
		}
		convo2, err := initConversation("Second Conversation")
		if err != nil {
			t.Errorf("Failed to init conversation: %v", err)
		}
		_, err = cc.PutConversation(ctx, getNewPutRequestFromConversation(convo1))
		if err != nil {
			t.Errorf("Failed to save conversation: %v", err)
		}
		_, err = cc.PutConversation(ctx, getNewPutRequestFromConversation(convo2))
		if err != nil {
			t.Errorf("Failed to save conversation: %v", err)
		}
		convoItems, err := cc.ListConversations(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to list conversations: %v", err)
		}

		if len(convoItems.Body.ConversationListItems) != 2 {
			t.Errorf("Expected 2 conversations, got %d", len(convoItems.Body.ConversationListItems))
		}
	})

	// NEW: Paging (NextPageToken).
	t.Run("ListConversationsPaging", func(t *testing.T) {
		ctx := t.Context()
		// Add 15 conversations.
		for range 15 {
			u, err := uuidv7filename.NewUUIDv7String()
			if err != nil {
				t.Fatalf("Failed to init conversation: %v", err)
			}
			convo, err := initConversation("Paged " + u)
			if err != nil {
				t.Fatalf("Failed to init conversation: %v", err)
			}
			_, err = cc.PutConversation(ctx, getNewPutRequestFromConversation(convo))
			if err != nil {
				t.Fatalf("Failed to save conversation: %v", err)
			}
		}
		// List with page size 10.
		resp, err := cc.ListConversations(ctx, &spec.ListConversationsRequest{})
		if err != nil {
			t.Fatalf("Failed to list conversations: %v", err)
		}
		if len(resp.Body.ConversationListItems) < 10 {
			t.Errorf(
				"Expected at least 10 conversations, got %d",
				len(resp.Body.ConversationListItems),
			)
		}
		// If NextPageToken is present, fetch next page.
		if resp.Body.NextPageToken != nil && *resp.Body.NextPageToken != "" {
			resp2, err := cc.ListConversations(
				ctx,
				&spec.ListConversationsRequest{PageToken: *resp.Body.NextPageToken},
			)
			if err != nil {
				t.Fatalf("Failed to list next page: %v", err)
			}
			if len(resp2.Body.ConversationListItems) == 0 {
				t.Errorf("Expected more conversations in next page")
			}
		}
	})
}

func getNewPutRequestFromConversation(c *spec.Conversation) *spec.PutConversationRequest {
	return &spec.PutConversationRequest{
		ID: c.ID,
		Body: &spec.PutConversationRequestBody{
			Title:      c.Title,
			CreatedAt:  c.CreatedAt,
			ModifiedAt: c.ModifiedAt,
			Messages:   c.Messages,
		},
	}
}

func newTextTurn(id string, role inferenceSpec.RoleEnum, txt string) spec.ConversationMessage {
	now := time.Now().UTC()
	return spec.ConversationMessage{
		ID:        id,
		CreatedAt: now,
		Role:      role,
		Status:    inferenceSpec.StatusCompleted,
		Inputs: []inferenceSpec.InputUnion{
			{
				Kind: inferenceSpec.InputKindInputMessage,
				InputMessage: &inferenceSpec.InputOutputContent{
					ID:     id + ":0",
					Role:   role,
					Status: inferenceSpec.StatusCompleted,
					Contents: []inferenceSpec.InputOutputContentItemUnion{{
						Kind:     inferenceSpec.ContentItemKindText,
						TextItem: &inferenceSpec.ContentItemText{Text: txt},
					}},
				},
			},
		},
	}
}

func initConversation(title string) (*spec.Conversation, error) {
	if title == "" {
		title = "New Conversation"
	}
	if len(title) > 64 {
		title = title[:64]
	}

	c := spec.Conversation{
		SchemaVersion: spec.ConversationSchemaVersion,
		Messages:      []spec.ConversationMessage{},
	}
	u, err := uuidv7filename.NewUUIDv7String()
	if err != nil {
		return nil, err
	}
	c.ID = u
	c.Title = title
	c.CreatedAt = time.Now().UTC()
	c.ModifiedAt = time.Now().UTC()

	return &c, nil
}
