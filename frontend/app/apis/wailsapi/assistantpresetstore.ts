import type {
	AssistantPreset,
	AssistantPresetBundle,
	AssistantPresetListItem,
	PutAssistantPresetPayload,
} from '@/spec/assistantpreset';

import type { IAssistantPresetStoreAPI } from '@/apis/interface';
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

function omitUndefined<T extends Record<string, unknown>>(obj: T): Partial<T> {
	return Object.fromEntries(Object.entries(obj).filter(([, value]) => value !== undefined)) as Partial<T>;
}

function toRequiredDate(value: unknown, fieldName: string): Date {
	if (value instanceof Date) return value;

	if (typeof value === 'string' || typeof value === 'number') {
		const d = new Date(value);
		if (!Number.isNaN(d.getTime())) return d;
	}

	throw new Error(`Invalid or missing date for ${fieldName}`);
}

function toOptionalDate(value: unknown, fieldName: string): Date | undefined {
	if (value === undefined || value === null || value === '') return undefined;
	return toRequiredDate(value, fieldName);
}

function normalizeAssistantPreset(preset: AssistantPreset): AssistantPreset {
	return {
		...preset,
		createdAt: toRequiredDate(preset.createdAt, 'assistantPreset.createdAt'),
		modifiedAt: toRequiredDate(preset.modifiedAt, 'assistantPreset.modifiedAt'),
	};
}

function normalizeAssistantPresetBundle(bundle: AssistantPresetBundle): AssistantPresetBundle {
	return {
		...bundle,
		createdAt: toRequiredDate(bundle.createdAt, 'assistantPresetBundle.createdAt'),
		modifiedAt: toRequiredDate(bundle.modifiedAt, 'assistantPresetBundle.modifiedAt'),
		softDeletedAt: toOptionalDate(bundle.softDeletedAt, 'assistantPresetBundle.softDeletedAt'),
	};
}

function normalizeAssistantPresetListItem(item: AssistantPresetListItem): AssistantPresetListItem {
	return {
		...item,
		modifiedAt: toOptionalDate(item.modifiedAt, 'assistantPresetListItem.modifiedAt'),
	};
}

/**
 * @public
 */
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

		return {
			assistantPresetBundles: ((resp.Body?.assistantPresetBundles ?? []) as AssistantPresetBundle[]).map(
				normalizeAssistantPresetBundle
			),
			nextPageToken: resp.Body?.nextPageToken ?? undefined,
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

		return {
			assistantPresetListItems: ((resp.Body?.assistantPresetListItems ?? []) as AssistantPresetListItem[]).map(
				normalizeAssistantPresetListItem
			),
			nextPageToken: resp.Body?.nextPageToken ?? undefined,
		};
	}

	async putAssistantPreset(
		bundleID: string,
		assistantPresetSlug: string,
		version: string,
		payload: PutAssistantPresetPayload
	): Promise<void> {
		const rawPatch = payload.startingModelPresetPatch as
			| { systemPrompt?: unknown; capabilitiesOverride?: unknown }
			| undefined;

		if (rawPatch?.systemPrompt !== undefined) {
			throw new Error('startingModelPresetPatch.systemPrompt is not allowed for assistant presets');
		}
		if (rawPatch?.capabilitiesOverride !== undefined) {
			throw new Error('startingModelPresetPatch.capabilitiesOverride is not allowed for assistant presets');
		}

		const body = omitUndefined({
			displayName: payload.displayName,
			description: payload.description,
			isEnabled: payload.isEnabled,
			startingModelPresetRef: payload.startingModelPresetRef,
			startingModelPresetPatch: payload.startingModelPresetPatch,
			startingIncludeModelSystemPrompt: payload.startingIncludeModelSystemPrompt,
			startingInstructionTemplateRefs: payload.startingInstructionTemplateRefs,
			startingToolSelections: payload.startingToolSelections,
			startingSkillSelections: payload.startingSkillSelections,
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
		return resp.Body ? normalizeAssistantPreset(resp.Body as AssistantPreset) : undefined;
	}
}
