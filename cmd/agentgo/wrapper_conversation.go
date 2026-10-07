package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/conversation"
	conversationSpec "github.com/flexigpt/flexigpt-app/internal/conversation/spec"
)

type ConversationCollectionWrapper struct {
	store *conversation.ConversationCollection
}

func InitConversationCollectionWrapper(
	c *ConversationCollectionWrapper,
	conversationDir string,
) error {
	conversationStoreAPI, err := conversation.NewConversationCollection(
		conversationDir,
		conversation.WithFTS(true),
	)
	if err != nil {
		return err
	}
	c.store = conversationStoreAPI
	return nil
}

func (ccw *ConversationCollectionWrapper) PutConversation(
	req *conversationSpec.PutConversationRequest,
) (*conversationSpec.PutConversationResponse, error) {
	return withRecoveryResp(func() (*conversationSpec.PutConversationResponse, error) {
		return ccw.store.PutConversation(context.Background(), req)
	})
}

func (ccw *ConversationCollectionWrapper) DeleteConversation(
	req *conversationSpec.DeleteConversationRequest,
) (*conversationSpec.DeleteConversationResponse, error) {
	return withRecoveryResp(func() (*conversationSpec.DeleteConversationResponse, error) {
		return ccw.store.DeleteConversation(context.Background(), req)
	})
}

func (ccw *ConversationCollectionWrapper) GetConversation(
	req *conversationSpec.GetConversationRequest,
) (*conversationSpec.GetConversationResponse, error) {
	return withRecoveryResp(func() (*conversationSpec.GetConversationResponse, error) {
		return ccw.store.GetConversation(context.Background(), req)
	})
}

func (ccw *ConversationCollectionWrapper) ListConversations(
	req *conversationSpec.ListConversationsRequest,
) (*conversationSpec.ListConversationsResponse, error) {
	return withRecoveryResp(func() (*conversationSpec.ListConversationsResponse, error) {
		return ccw.store.ListConversations(context.Background(), req)
	})
}

func (ccw *ConversationCollectionWrapper) SearchConversations(
	req *conversationSpec.SearchConversationsRequest,
) (*conversationSpec.SearchConversationsResponse, error) {
	return withRecoveryResp(func() (*conversationSpec.SearchConversationsResponse, error) {
		return ccw.store.SearchConversations(context.Background(), req)
	})
}

func (ccw *ConversationCollectionWrapper) PutMessagesToConversation(
	req *conversationSpec.PutMessagesToConversationRequest,
) (*conversationSpec.PutMessagesToConversationResponse, error) {
	return withRecoveryResp(func() (*conversationSpec.PutMessagesToConversationResponse, error) {
		return ccw.store.PutMessagesToConversation(context.Background(), req)
	})
}

func (ccw *ConversationCollectionWrapper) close() {
	if ccw == nil || ccw.store == nil {
		return
	}
	ccw.store.Close()
}
