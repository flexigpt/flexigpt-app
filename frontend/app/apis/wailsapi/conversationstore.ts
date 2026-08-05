// oxlint-disable typescript/no-misused-spread
import type { ConversationSearchItem, StoreConversation, StoreConversationMessage } from '@/spec/conversation';
import { RoleEnum, Status } from '@/spec/inference';

import { parseAnyToTime } from '@/lib/date_utils';
import { extractTimeFromUUIDv7Str } from '@/lib/uuid_utils';

import type { IConversationStoreAPI } from '@/apis/interface';
import {
	enumFromWails,
	optionalWailsBody,
	requireWailsArray,
	requireWailsBody,
	toFrontendDate,
} from '@/apis/wailsapi/transport';
import {
	DeleteConversation,
	GetConversation,
	ListConversations,
	PutConversation,
	PutMessagesToConversation,
	SearchConversations,
} from '@/apis/wailsjs/go/main/ConversationCollectionWrapper';
import type { spec as wailsSpec } from '@/apis/wailsjs/go/models';

export class WailsConversationStoreAPI implements IConversationStoreAPI {
	async putConversation(conversation: StoreConversation): Promise<void> {
		const req = {
			ID: conversation.id,
			Body: {
				title: conversation.title,
				createdAt: conversation.createdAt,
				modifiedAt: conversation.modifiedAt,
				messages: conversation.messages as wailsSpec.ConversationMessage[],
			} as wailsSpec.PutConversationRequestBody,
		};

		await PutConversation(req as wailsSpec.PutConversationRequest);
	}

	async putMessagesToConversation(id: string, title: string, messages: StoreConversationMessage[]): Promise<void> {
		const req = {
			ID: id,
			Body: {
				title: title,
				messages: messages as wailsSpec.ConversationMessage[],
			} as wailsSpec.PutMessagesToConversationRequestBody,
		};

		await PutMessagesToConversation(req as wailsSpec.PutMessagesToConversationRequest);
	}

	async deleteConversation(id: string, title: string): Promise<void> {
		const req = { ID: id, Title: title };
		await DeleteConversation(req as wailsSpec.DeleteConversationRequest);
	}

	async getConversation(id: string, title: string, forceFetch?: boolean): Promise<StoreConversation | null> {
		const req = { ID: id, Title: title, ForceFetch: forceFetch ?? false };
		const c = await GetConversation(req as wailsSpec.GetConversationRequest);
		const body = optionalWailsBody(c.Body);
		return body === undefined ? null : conversationFromWails(body);
	}

	async listConversations(
		token?: string,
		pageSize?: number
	): Promise<{ conversations: ConversationSearchItem[]; nextToken?: string }> {
		const req = { PageToken: token || '', PageSize: pageSize ?? 20 };
		const resp = await ListConversations(req as wailsSpec.ListConversationsRequest);
		const body = requireWailsBody(resp.Body, 'ListConversations');
		return {
			conversations: mapConversationsToSearchItems(
				requireWailsArray(body.conversationListItems, 'ListConversations.conversationListItems')
			),
			nextToken: body.nextPageToken || undefined,
		};
	}

	async searchConversations(
		query: string,
		token?: string,
		pageSize?: number
	): Promise<{ conversations: ConversationSearchItem[]; nextToken?: string }> {
		const req = { Query: query, PageToken: token || '', PageSize: pageSize || 10 };
		const resp = await SearchConversations(req as wailsSpec.SearchConversationsRequest);
		const body = requireWailsBody(resp.Body, 'SearchConversations');

		return {
			conversations: mapConversationsToSearchItems(
				requireWailsArray(body.conversationListItems, 'SearchConversations.conversationListItems')
			),
			nextToken: body.nextPageToken || undefined,
		};
	}
}

function conversationFromWails(conversation: wailsSpec.Conversation): StoreConversation {
	return {
		...conversation,
		createdAt: toFrontendDate(conversation.createdAt, 'conversation.createdAt'),
		modifiedAt: toFrontendDate(conversation.modifiedAt, 'conversation.modifiedAt'),
		messages: requireWailsArray<wailsSpec.ConversationMessage>(conversation.messages, 'conversation.messages').map(
			(message, index) =>
				Object.assign(message, {
					createdAt: toFrontendDate(message.createdAt, `conversation.messages[${index}].createdAt`),
					role: enumFromWails(message.role, RoleEnum, `conversation.messages[${index}].role`),
					status: enumFromWails(message.status, Status, `conversation.messages[${index}].status`),
				})
		),
	} as StoreConversation;
}

function mapConversationsToSearchItems(conversations: Array<wailsSpec.ConversationListItem>): ConversationSearchItem[] {
	return conversations.map(conv => {
		const idDate = extractTimeFromUUIDv7Str(conv.id);
		const modifiedAtDate = parseAnyToTime(conv.modifiedAt) ?? idDate;

		return {
			id: conv.id,
			title: conv.sanatizedTitle,
			idDate: idDate,
			modifiedAt: modifiedAtDate,
		} as ConversationSearchItem;
	});
}
