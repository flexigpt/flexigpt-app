package main

import (
	"context"

	conversationSpec "github.com/flexigpt/flexigpt-app/internal/conversation/spec"
	conversationStore "github.com/flexigpt/flexigpt-app/internal/conversation/store"
)

type ConversationPluginWrapper struct {
	store *conversationStore.ConversationCollection
}

func InitConversationPluginWrapper(
	c *ConversationPluginWrapper,
	conversationDir string,
) error {
	conversationStoreAPI, err := conversationStore.NewConversationCollection(
		conversationDir,
		conversationStore.WithFTS(true),
	)
	if err != nil {
		return err
	}
	c.store = conversationStoreAPI
	return nil
}

func (ccw *ConversationPluginWrapper) PutConversation(
	req *conversationSpec.PutConversationRequest,
) (*conversationSpec.PutConversationResponse, error) {
	return withRecoveryResp(func() (*conversationSpec.PutConversationResponse, error) {
		return ccw.store.PutConversation(context.Background(), req)
	})
}

func (ccw *ConversationPluginWrapper) DeleteConversation(
	req *conversationSpec.DeleteConversationRequest,
) (*conversationSpec.DeleteConversationResponse, error) {
	return withRecoveryResp(func() (*conversationSpec.DeleteConversationResponse, error) {
		return ccw.store.DeleteConversation(context.Background(), req)
	})
}

func (ccw *ConversationPluginWrapper) GetConversation(
	req *conversationSpec.GetConversationRequest,
) (*conversationSpec.GetConversationResponse, error) {
	return withRecoveryResp(func() (*conversationSpec.GetConversationResponse, error) {
		return ccw.store.GetConversation(context.Background(), req)
	})
}

func (ccw *ConversationPluginWrapper) ListConversations(
	req *conversationSpec.ListConversationsRequest,
) (*conversationSpec.ListConversationsResponse, error) {
	return withRecoveryResp(func() (*conversationSpec.ListConversationsResponse, error) {
		return ccw.store.ListConversations(context.Background(), req)
	})
}

func (ccw *ConversationPluginWrapper) SearchConversations(
	req *conversationSpec.SearchConversationsRequest,
) (*conversationSpec.SearchConversationsResponse, error) {
	return withRecoveryResp(func() (*conversationSpec.SearchConversationsResponse, error) {
		return ccw.store.SearchConversations(context.Background(), req)
	})
}

func (ccw *ConversationPluginWrapper) PutMessagesToConversation(
	req *conversationSpec.PutMessagesToConversationRequest,
) (*conversationSpec.PutMessagesToConversationResponse, error) {
	return withRecoveryResp(func() (*conversationSpec.PutMessagesToConversationResponse, error) {
		return ccw.store.PutMessagesToConversation(context.Background(), req)
	})
}

func (ccw *ConversationPluginWrapper) close() {
	if ccw == nil || ccw.store == nil {
		return
	}
	ccw.store.Close()
}
