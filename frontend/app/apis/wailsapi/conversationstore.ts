import type { ConversationSearchItem, StoreConversation, StoreConversationMessage } from '@/spec/conversation';

import { parseAnyToTime } from '@/lib/date_utils';
import { extractTimeFromUUIDv7Str } from '@/lib/uuid_utils';

import type { IConversationStoreAPI } from '@/apis/interface';
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
import type { spec as wailsSpec } from '@/apis/wailsjs/go/models';

function requireWailsDate(value: unknown, field: string): Date {
	const parsed = parseAnyToTime(value);
	if (!parsed || Number.isNaN(parsed.getTime())) {
		throw new TypeError(`${field} returned an invalid date.`);
	}

	return parsed;
}

function conversationFromWails(value: unknown): StoreConversation {
	const body = requireWailsBody(value as Record<string, unknown> | null | undefined, 'GetConversation');
	// oxlint-disable-next-line oxc/no-map-spread
	const messages = wailsObjectArrayOrEmpty(body.messages, 'GetConversation.messages').map((message, index) => {
		return {
			...message,
			createdAt: requireWailsDate(message.createdAt, `GetConversation.messages[${index}].createdAt`),
		} as unknown as StoreConversationMessage;
	});

	return {
		...body,
		id: requireWailsString(body.id, 'GetConversation.id'),
		title: requireWailsString(body.title, 'GetConversation.title'),
		createdAt: requireWailsDate(body.createdAt, 'GetConversation.createdAt'),
		modifiedAt: requireWailsDate(body.modifiedAt, 'GetConversation.modifiedAt'),
		messages,
	} as unknown as StoreConversation;
}

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
		const body = optionalWailsBody(c.Body, 'GetConversation');
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
				wailsObjectArrayOrEmpty(body.conversationListItems, 'ListConversations.conversationListItems')
			),
			nextToken: optionalWailsString(body.nextPageToken, 'ListConversations.nextPageToken') || undefined,
		};
	}

	async searchConversations(
		query: string,
		token?: string,
		pageSize?: number
	): Promise<{ conversations: ConversationSearchItem[]; nextToken?: string }> {
		const req = { Query: query, PageToken: token || '', PageSize: pageSize ?? 10 };
		const resp = await SearchConversations(req as wailsSpec.SearchConversationsRequest);
		const body = requireWailsBody(resp.Body, 'SearchConversations');

		return {
			conversations: mapConversationsToSearchItems(
				wailsObjectArrayOrEmpty(body.conversationListItems, 'SearchConversations.conversationListItems')
			),
			nextToken: optionalWailsString(body.nextPageToken, 'SearchConversations.nextPageToken') || undefined,
		};
	}
}

function mapConversationsToSearchItems(conversations: Array<wailsSpec.ConversationListItem>): ConversationSearchItem[] {
	return conversations.map((conv, index) => {
		const id = requireWailsString(conv.id, `conversationListItems[${index}].id`);
		const title = requireWailsString(conv.sanatizedTitle, `conversationListItems[${index}].sanatizedTitle`);
		const idDate = extractTimeFromUUIDv7Str(id);
		const modifiedAtDate = parseAnyToTime(conv.modifiedAt) ?? idDate;

		return {
			id,
			title,
			idDate: idDate,
			modifiedAt: modifiedAtDate,
		} as ConversationSearchItem;
	});
}
