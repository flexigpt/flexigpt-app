package store

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/flexigpt/flexigpt-app/internal/conversation/spec"
	inferencegoSpec "github.com/flexigpt/inference-go/spec"
)

func TestFTSSearchHappyPath(t *testing.T) {
	dir := t.TempDir()
	cc := newCollection(t, dir, true)

	c1 := newConv(t, "Banana split")
	_, _ = cc.PutConversation(
		t.Context(),
		getNewPutRequestFromConversation(c1),
	)

	resp, err := cc.SearchConversations(t.Context(), &spec.SearchConversationsRequest{
		Query:     "banana",
		PageToken: "",
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(resp.Body.ConversationListItems) != 1 {
		t.Fatalf("want 1 hit, got %d", len(resp.Body.ConversationListItems))
	}
	if resp.Body.ConversationListItems[0].ID != c1.ID {
		t.Fatalf("unexpected hit %q", resp.Body.ConversationListItems[0].ID)
	}
}

func TestFTSRankingTitleVsBody(t *testing.T) {
	dir := t.TempDir()
	cc := newCollection(t, dir, true)

	// A: query term in title.
	a := newConv(t, "Alpha winner")
	_, _ = cc.PutConversation(t.Context(), getNewPutRequestFromConversation(a))

	// B: query term only inside a message.
	b := newConv(t, "No match in title")
	b.Messages = []spec.ConversationMessage{
		newTextTurn("m1", inferencegoSpec.RoleUser, "alpha appears in body"),
	}
	_, _ = cc.PutConversation(t.Context(), getNewPutRequestFromConversation(b))

	resp, _ := cc.SearchConversations(t.Context(),
		&spec.SearchConversationsRequest{Query: "alpha"})
	if len(resp.Body.ConversationListItems) != 2 ||
		resp.Body.ConversationListItems[0].ID != a.ID {
		t.Fatalf("ranking wrong, hits=%v", resp.Body.ConversationListItems)
	}
}

func TestFTSPagination(t *testing.T) {
	dir := t.TempDir()
	cc := newCollection(t, dir, true)

	for i := range 14 {
		c := newConv(t, "Kiwi talk "+strconv.Itoa(i))
		c.Messages = []spec.ConversationMessage{
			newTextTurn("m", inferencegoSpec.RoleUser, "kiwi everywhere"),
		}
		_, _ = cc.PutConversation(t.Context(), getNewPutRequestFromConversation(c))
	}

	var (
		token string
		total int
	)
	for {
		res, err := cc.SearchConversations(t.Context(),
			&spec.SearchConversationsRequest{Query: "kiwi", PageToken: token})
		if err != nil {
			t.Fatalf("search page: %v", err)
		}
		total += len(res.Body.ConversationListItems)
		if res.Body.NextPageToken == nil || *res.Body.NextPageToken == "" {
			if len(res.Body.ConversationListItems) != 2 {
				t.Fatalf("last page size want 2, got %d", len(res.Body.ConversationListItems))
			}
			break
		}
		if len(res.Body.ConversationListItems) != 12 {
			t.Fatalf("full page size want 12 got %d", len(res.Body.ConversationListItems))
		}
		token = *res.Body.NextPageToken
	}
	if total != 14 {
		t.Fatalf("expected 14 hits over all pages, got %d", total)
	}
}

func TestFTSDeletePurgesIndex(t *testing.T) {
	dir := t.TempDir()
	cc := newCollection(t, dir, true)

	c := newConv(t, "Cherry pie")
	_, _ = cc.PutConversation(t.Context(), getNewPutRequestFromConversation(c))

	// Ensure hit exists.
	_, err := cc.SearchConversations(t.Context(),
		&spec.SearchConversationsRequest{Query: "cherry"})
	if err != nil {
		t.Fatalf("initial search: %v", err)
	}

	// Delete conversation.
	_, _ = cc.DeleteConversation(t.Context(),
		&spec.DeleteConversationRequest{ID: c.ID, Title: c.Title})

	// Expect zero hits.
	res, _ := cc.SearchConversations(t.Context(),
		&spec.SearchConversationsRequest{Query: "cherry"})
	if len(res.Body.ConversationListItems) != 0 {
		t.Fatalf("expected 0 hits after delete, got %d", len(res.Body.ConversationListItems))
	}
}

func TestFTSAddMessageUpdatesIndex(t *testing.T) {
	dir := t.TempDir()
	cc := newCollection(t, dir, true)

	c := newConv(t, "Silent")
	_, _ = cc.PutConversation(t.Context(), getNewPutRequestFromConversation(c))

	// No hit yet.
	res, _ := cc.SearchConversations(t.Context(),
		&spec.SearchConversationsRequest{Query: "whisper"})
	if len(res.Body.ConversationListItems) != 0 {
		t.Fatal("should have no hits before message added")
	}

	msg := newTextTurn("m1", inferencegoSpec.RoleAssistant, "let me whisper a secret")
	_, _ = cc.PutMessagesToConversation(t.Context(),
		&spec.PutMessagesToConversationRequest{
			ID: c.ID,
			Body: &spec.PutMessagesToConversationRequestBody{
				Title:    c.Title,
				Messages: []spec.ConversationMessage{msg},
			},
		})

	// Now should hit.
	res, _ = cc.SearchConversations(t.Context(),
		&spec.SearchConversationsRequest{Query: "whisper"})
	if len(res.Body.ConversationListItems) != 1 {
		t.Fatalf("want 1 hit after adding message, got %d", len(res.Body.ConversationListItems))
	}
}

func TestFTSDisabled(t *testing.T) {
	dir := t.TempDir()
	cc := newCollection(t, dir, false)

	_, err := cc.SearchConversations(t.Context(),
		&spec.SearchConversationsRequest{Query: "x"})
	if err == nil {
		t.Fatal("expected error when FTS disabled")
	}
}

func TestFTSScaleSearch(t *testing.T) {
	dir := t.TempDir()
	cc := newCollection(t, dir, true)

	const (
		nConvos   = 100
		nMessages = 20
		pageSz    = 15
	)

	// Create & save conversations.
	for i := range nConvos {
		c := newConv(t, fmt.Sprintf("Scale test fruit%d", i))
		for m := range nMessages {
			role := inferencegoSpec.RoleUser
			if m%2 == 1 {
				role = inferencegoSpec.RoleAssistant
			}
			c.Messages = append(
				c.Messages,
				newTextTurn(fmt.Sprintf("m%02d", m), role, fmt.Sprintf("This is message %d with common keyword", m)),
			)
		}
		if _, err := cc.PutConversation(t.Context(),
			getNewPutRequestFromConversation(c)); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}

	// Single-doc search.
	res, err := cc.SearchConversations(t.Context(),
		&spec.SearchConversationsRequest{Query: "fruit42", PageSize: pageSz})
	if err != nil {
		t.Fatalf("search fruit42: %v", err)
	}
	if len(res.Body.ConversationListItems) != 1 {
		t.Fatalf("expect 1 hit for fruit42, got %d", len(res.Body.ConversationListItems))
	}

	// Bulk search with pagination.
	token := ""
	seen := map[string]bool{}
	total := 0
	for page := 0; ; page++ {
		r, err := cc.SearchConversations(t.Context(),
			&spec.SearchConversationsRequest{
				Query:     "common",
				PageToken: token,
				PageSize:  pageSz,
			})
		if err != nil {
			t.Fatalf("page %d search: %v", page, err)
		}
		for _, it := range r.Body.ConversationListItems {
			if seen[it.ID] {
				t.Fatalf("duplicate id %s on page %d", it.ID, page)
			}
			seen[it.ID] = true
		}
		total += len(r.Body.ConversationListItems)

		if r.Body.NextPageToken == nil || *r.Body.NextPageToken == "" {
			break
		}
		if len(r.Body.ConversationListItems) != pageSz {
			t.Fatalf("full page expected %d items, got %d", pageSz,
				len(r.Body.ConversationListItems))
		}
		token = *r.Body.NextPageToken
	}

	if total != nConvos {
		t.Fatalf("expected %d total hits for 'common', got %d", nConvos, total)
	}
}

func newCollection(t *testing.T, dir string, withFTS bool) *ConversationCollection {
	t.Helper()
	cc, err := NewConversationCollection(
		dir,
		WithFTS(withFTS),
	)
	if err != nil {
		t.Fatalf("NewConversationCollection: %v", err)
	}
	t.Cleanup(func() { _ = cc.Close() })
	return cc
}

func newConv(t *testing.T, title string) *spec.Conversation {
	t.Helper()
	id, _ := uuid.NewV7()
	sec, nsec := id.Time().UnixTime()
	createdAt := time.Unix(sec, nsec).UTC()
	return &spec.Conversation{
		ID:         id.String(),
		Title:      title,
		CreatedAt:  createdAt,
		ModifiedAt: createdAt,
		Messages:   []spec.ConversationMessage{},
	}
}
