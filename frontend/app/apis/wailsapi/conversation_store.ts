import type { ConversationSearchItem, StoreConversation, StoreConversationMessage } from '@/spec/conversation';

import type { IConversationStoreAPI } from '@/apis/interface';
import type { spec as wailsSpec } from '@/apis/wailsjs/go/models';
import {
	optionalWailsBody,
	optionalWailsString,
	requireWailsBody,
	requireWailsString,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	DeleteConversation,
	GetConversation,
	ListConversations,
	PutConversation,
	PutMessagesToConversation,
	SearchConversations,
} from '@/apis/wailsjs/go/main/ConversationCollectionWrapper';

function searchItemsFromWails(
	values: Array<wailsSpec.ConversationListItem>,
	operation: string
): ConversationSearchItem[] {
	return values.map((value, index) => ({
		id: requireWailsString(value.id, `${operation}[${index}].id`),
		// This is the only projection retained because the Go DTO has the
		// historical misspelled transport field, while the app contract uses
		// the correct `title` name.
		title: requireWailsString(value.sanatizedTitle, `${operation}[${index}].sanatizedTitle`),
		modifiedAt: optionalWailsString(value.modifiedAt, `${operation}[${index}].modifiedAt`),
	}));
}

export class WailsConversationStoreAPI implements IConversationStoreAPI {
	async putConversation(conversation: StoreConversation): Promise<void> {
		const { id, schemaVersion: _schemaVersion, ...body } = conversation;

		await PutConversation({
			ID: id,
			Body: body as wailsSpec.PutConversationRequestBody,
		} as wailsSpec.PutConversationRequest);
	}

	async putMessagesToConversation(id: string, title: string, messages: StoreConversationMessage[]): Promise<void> {
		await PutMessagesToConversation({
			ID: id,
			Body: {
				title,
				messages: messages as wailsSpec.ConversationMessage[],
			} as wailsSpec.PutMessagesToConversationRequestBody,
		} as wailsSpec.PutMessagesToConversationRequest);
	}

	async deleteConversation(id: string, title: string): Promise<void> {
		await DeleteConversation({
			ID: id,
			Title: title,
		});
	}

	async getConversation(id: string, title: string, forceFetch?: boolean): Promise<StoreConversation | null> {
		const response = await GetConversation({
			ID: id,
			Title: title,
			ForceFetch: forceFetch ?? false,
		});

		const body = optionalWailsBody(response.Body, 'GetConversation');
		return body === undefined ? null : (body as StoreConversation);
	}

	async listConversations(
		token?: string,
		pageSize?: number
	): Promise<{ conversations: ConversationSearchItem[]; nextToken?: string }> {
		const response = await ListConversations({
			PageToken: token ?? '',
			PageSize: pageSize ?? 20,
		});

		const body = requireWailsBody(response.Body, 'ListConversations');

		return {
			conversations: searchItemsFromWails(
				wailsObjectArrayOrEmpty<wailsSpec.ConversationListItem>(
					body.conversationListItems,
					'ListConversations.conversationListItems'
				),
				'ListConversations.conversationListItems'
			),
			nextToken: optionalWailsString(body.nextPageToken, 'ListConversations.nextPageToken') || undefined,
		};
	}

	async searchConversations(
		query: string,
		token?: string,
		pageSize?: number
	): Promise<{ conversations: ConversationSearchItem[]; nextToken?: string }> {
		const response = await SearchConversations({
			Query: query,
			PageToken: token ?? '',
			PageSize: pageSize ?? 10,
		});

		const body = requireWailsBody(response.Body, 'SearchConversations');

		return {
			conversations: searchItemsFromWails(
				wailsObjectArrayOrEmpty<wailsSpec.ConversationListItem>(
					body.conversationListItems,
					'SearchConversations.conversationListItems'
				),
				'SearchConversations.conversationListItems'
			),
			nextToken: optionalWailsString(body.nextPageToken, 'SearchConversations.nextPageToken') || undefined,
		};
	}
}
