package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/conversation/spec"
	conversationStore "github.com/flexigpt/flexigpt-app/internal/conversation/store"
)

type ConversationCollectionWrapper struct {
	store *conversationStore.ConversationCollection
}

func InitConversationCollectionWrapper(
	c *ConversationCollectionWrapper,
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

func (ccw *ConversationCollectionWrapper) PutConversation(
	req *spec.PutConversationRequest,
) (*spec.PutConversationResponse, error) {
	return withRecoveryResp(func() (*spec.PutConversationResponse, error) {
		return ccw.store.PutConversation(context.Background(), req)
	})
}

func (ccw *ConversationCollectionWrapper) DeleteConversation(
	req *spec.DeleteConversationRequest,
) (*spec.DeleteConversationResponse, error) {
	return withRecoveryResp(func() (*spec.DeleteConversationResponse, error) {
		return ccw.store.DeleteConversation(context.Background(), req)
	})
}

func (ccw *ConversationCollectionWrapper) GetConversation(
	req *spec.GetConversationRequest,
) (*spec.GetConversationResponse, error) {
	return withRecoveryResp(func() (*spec.GetConversationResponse, error) {
		return ccw.store.GetConversation(context.Background(), req)
	})
}

func (ccw *ConversationCollectionWrapper) ListConversations(
	req *spec.ListConversationsRequest,
) (*spec.ListConversationsResponse, error) {
	return withRecoveryResp(func() (*spec.ListConversationsResponse, error) {
		return ccw.store.ListConversations(context.Background(), req)
	})
}

func (ccw *ConversationCollectionWrapper) SearchConversations(
	req *spec.SearchConversationsRequest,
) (*spec.SearchConversationsResponse, error) {
	return withRecoveryResp(func() (*spec.SearchConversationsResponse, error) {
		return ccw.store.SearchConversations(context.Background(), req)
	})
}

func (ccw *ConversationCollectionWrapper) PutMessagesToConversation(
	req *spec.PutMessagesToConversationRequest,
) (*spec.PutMessagesToConversationResponse, error) {
	return withRecoveryResp(func() (*spec.PutMessagesToConversationResponse, error) {
		return ccw.store.PutMessagesToConversation(context.Background(), req)
	})
}

func (ccw *ConversationCollectionWrapper) close() {
	if ccw == nil || ccw.store == nil {
		return
	}
	ccw.store.Close()
}
