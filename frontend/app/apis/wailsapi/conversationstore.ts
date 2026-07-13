import type { ConversationSearchItem, StoreConversation, StoreConversationMessage } from '@/spec/conversation';

import { parseAnyToTime } from '@/lib/date_utils';
import { extractTimeFromUUIDv7Str } from '@/lib/uuid_utils';

import type { IConversationStoreAPI } from '@/apis/interface';
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
		return c.Body as StoreConversation;
	}

	async listConversations(
		token?: string,
		pageSize?: number
	): Promise<{ conversations: ConversationSearchItem[]; nextToken?: string }> {
		const req = { PageToken: token || '', PageSize: pageSize ?? 20 };
		const resp = await ListConversations(req as wailsSpec.ListConversationsRequest);
		return {
			conversations: mapConversationsToSearchItems(resp.Body ? resp.Body.conversationListItems : []),
			nextToken: resp.Body?.nextPageToken,
		};
	}

	async searchConversations(
		query: string,
		token?: string,
		pageSize?: number
	): Promise<{ conversations: ConversationSearchItem[]; nextToken?: string }> {
		const req = { Query: query, PageToken: token || '', PageSize: pageSize || 10 };
		const resp = await SearchConversations(req as wailsSpec.SearchConversationsRequest);

		return {
			conversations: mapConversationsToSearchItems(resp.Body ? resp.Body.conversationListItems : []),
			nextToken: resp.Body?.nextPageToken,
		};
	}
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
