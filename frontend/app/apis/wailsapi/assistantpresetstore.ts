import type {
	AssistantPreset,
	AssistantPresetBundle,
	AssistantPresetListItem,
	PutAssistantPresetPayload,
} from '@/spec/assistantpreset';

import type { IAssistantPresetStoreAPI } from '@/apis/interface';
import {
	omitUndefined,
	optionalWailsBody,
	optionalWailsString,
	requireWailsBody,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	DeleteAssistantPreset,
	DeleteAssistantPresetBundle,
	GetAssistantPreset,
	ListAssistantPresetBundles,
	ListAssistantPresets,
	PatchAssistantPreset,
	PatchAssistantPresetBundle,
	PutAssistantPreset,
	PutAssistantPresetBundle,
} from '@/apis/wailsjs/go/main/AssistantPresetStoreWrapper';
import type { spec as wailsSpec } from '@/apis/wailsjs/go/models';

export class WailsAssistantPresetStoreAPI implements IAssistantPresetStoreAPI {
	async listAssistantPresetBundles(
		bundleIDs?: string[],
		includeDisabled?: boolean,
		pageSize?: number,
		pageToken?: string
	): Promise<{ assistantPresetBundles: AssistantPresetBundle[]; nextPageToken?: string }> {
		const req: wailsSpec.ListAssistantPresetBundlesRequest = {
			BundleIDs: bundleIDs ?? [],
			IncludeDisabled: includeDisabled ?? false,
			PageSize: pageSize ?? 0,
			PageToken: pageToken ?? '',
		};

		const resp = await ListAssistantPresetBundles(req);
		const body = requireWailsBody(resp.Body, 'ListAssistantPresetBundles');

		return {
			assistantPresetBundles: wailsObjectArrayOrEmpty<AssistantPresetBundle>(
				body.assistantPresetBundles,
				'ListAssistantPresetBundles.assistantPresetBundles'
			),
			nextPageToken: optionalWailsString(body.nextPageToken, 'ListAssistantPresetBundles.nextPageToken') || undefined,
		};
	}

	async putAssistantPresetBundle(
		bundleID: string,
		slug: string,
		displayName: string,
		isEnabled: boolean,
		description?: string
	): Promise<void> {
		const req = {
			BundleID: bundleID,
			Body: {
				slug,
				displayName,
				description,
				isEnabled,
			} as wailsSpec.PutAssistantPresetBundleRequestBody,
		};

		await PutAssistantPresetBundle(req as wailsSpec.PutAssistantPresetBundleRequest);
	}

	async patchAssistantPresetBundle(bundleID: string, isEnabled: boolean): Promise<void> {
		const req = {
			BundleID: bundleID,
			Body: {
				isEnabled,
			} as wailsSpec.PatchAssistantPresetBundleRequestBody,
		};

		await PatchAssistantPresetBundle(req as wailsSpec.PatchAssistantPresetBundleRequest);
	}

	async deleteAssistantPresetBundle(bundleID: string): Promise<void> {
		const req: wailsSpec.DeleteAssistantPresetBundleRequest = {
			BundleID: bundleID,
		};

		await DeleteAssistantPresetBundle(req);
	}

	async listAssistantPresets(
		bundleIDs?: string[],
		includeDisabled?: boolean,
		recommendedPageSize?: number,
		pageToken?: string
	): Promise<{ assistantPresetListItems: AssistantPresetListItem[]; nextPageToken?: string }> {
		const req: wailsSpec.ListAssistantPresetsRequest = {
			BundleIDs: bundleIDs ?? [],
			IncludeDisabled: includeDisabled ?? false,
			RecommendedPageSize: recommendedPageSize ?? 0,
			PageToken: pageToken ?? '',
		};

		const resp = await ListAssistantPresets(req);
		const body = requireWailsBody(resp.Body, 'ListAssistantPresets');

		return {
			assistantPresetListItems: wailsObjectArrayOrEmpty<AssistantPresetListItem>(
				body.assistantPresetListItems,
				'ListAssistantPresets.assistantPresetListItems'
			),
			nextPageToken: optionalWailsString(body.nextPageToken, 'ListAssistantPresets.nextPageToken') || undefined,
		};
	}

	async putAssistantPreset(
		bundleID: string,
		assistantPresetSlug: string,
		version: string,
		payload: PutAssistantPresetPayload
	): Promise<void> {
		const body = omitUndefined({
			displayName: payload.displayName,
			description: payload.description,
			isEnabled: payload.isEnabled,
			startingText: payload.startingText,
			startingModelPresetRef: payload.startingModelPresetRef,
			startingIncludeModelSystemPrompt: payload.startingIncludeModelSystemPrompt,
			startingToolSelections: payload.startingToolSelections,
			startingSkillSelections: payload.startingSkillSelections,
			startingMCPContext: payload.startingMCPContext,
		}) as wailsSpec.PutAssistantPresetRequestBody;

		const req = {
			BundleID: bundleID,
			AssistantPresetSlug: assistantPresetSlug,
			Version: version,
			Body: body,
		};

		await PutAssistantPreset(req as wailsSpec.PutAssistantPresetRequest);
	}

	async patchAssistantPreset(
		bundleID: string,
		assistantPresetSlug: string,
		version: string,
		isEnabled: boolean
	): Promise<void> {
		const req = {
			BundleID: bundleID,
			AssistantPresetSlug: assistantPresetSlug,
			Version: version,
			Body: {
				isEnabled,
			} as wailsSpec.PatchAssistantPresetRequestBody,
		};

		await PatchAssistantPreset(req as wailsSpec.PatchAssistantPresetRequest);
	}

	async deleteAssistantPreset(bundleID: string, assistantPresetSlug: string, version: string): Promise<void> {
		const req: wailsSpec.DeleteAssistantPresetRequest = {
			BundleID: bundleID,
			AssistantPresetSlug: assistantPresetSlug,
			Version: version,
		};

		await DeleteAssistantPreset(req);
	}

	async getAssistantPreset(
		bundleID: string,
		assistantPresetSlug: string,
		version: string
	): Promise<AssistantPreset | undefined> {
		const req: wailsSpec.GetAssistantPresetRequest = {
			BundleID: bundleID,
			AssistantPresetSlug: assistantPresetSlug,
			Version: version,
		};

		const resp = await GetAssistantPreset(req);
		const body = optionalWailsBody(resp.Body, 'GetAssistantPreset');
		return body ? (body as AssistantPreset) : undefined;
	}
}
