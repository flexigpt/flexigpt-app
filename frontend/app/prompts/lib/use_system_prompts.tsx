import { useCallback, useEffect, useSyncExternalStore } from 'react';

import { type PromptBundle, PromptRoleEnum, type PromptTemplate, PromptTemplateKind } from '@/spec/prompt';

import { getUUIDv7 } from '@/lib/uuid_utils';

import { promptStoreAPI } from '@/apis/baseapi';
import { getAllPromptBundles, getAllPromptTemplates } from '@/apis/list_helper';

import {
	buildPromptTemplateRefKey,
	buildPromptTemplateSeriesKey,
	getPromptTemplateRef,
} from '@/prompts/lib/prompt_template_ref';

export type SystemPromptRole = PromptRoleEnum.System | PromptRoleEnum.Developer;

export type SystemPromptItem = {
	identityKey: string;
	templateID?: string;
	bundleID: string;
	templateSlug: string;
	templateVersion: string;
	displayName: string;
	prompt: string;
	role: SystemPromptRole;
	bundleDisplayName: string;
	bundleSlug: string;
	isBuiltIn: boolean;
	createdAt: string;
	modifiedAt: string;
};

export type SystemPromptDraft = {
	bundleID: string;
	displayName: string;
	slug: string;
	version: string;
	role: SystemPromptRole;
	content: string;
};

type SystemPromptStoreState = {
	prompts: SystemPromptItem[];
	bundles: PromptBundle[];
	templateVersionsBySeriesKey: Record<string, string[]>;
	preferredBundleID: string | null;
	loading: boolean;
	error: string | null;
	initialized: boolean;
};

const listeners = new Set<() => void>();

let state: SystemPromptStoreState = {
	prompts: [],
	bundles: [],
	templateVersionsBySeriesKey: {},
	preferredBundleID: null,
	loading: false,
	error: null,
	initialized: false,
};

let refreshPromise: Promise<void> | null = null;
let initialLoadRequested = false;

function emitChange() {
	for (const listener of listeners) {
		listener();
	}
}

function setState(next: SystemPromptStoreState) {
	state = next;
	emitChange();
}

function getSnapshot(): SystemPromptStoreState {
	return state;
}

function subscribe(listener: () => void): () => void {
	listeners.add(listener);
	return () => {
		listeners.delete(listener);
	};
}

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

function normalizeSystemPromptRole(role?: PromptRoleEnum): SystemPromptRole {
	return role === PromptRoleEnum.Developer ? PromptRoleEnum.Developer : PromptRoleEnum.System;
}

function flattenInstructionsPrompt(template: PromptTemplate): string {
	return template.blocks
		.filter(block => block.role === PromptRoleEnum.System || block.role === PromptRoleEnum.Developer)
		.map(block => block.content.trim())
		.filter(Boolean)
		.join('\n\n');
}

export function deriveSystemPromptRole(template: PromptTemplate): SystemPromptRole {
	const roles = template.blocks
		.filter(block => block.role === PromptRoleEnum.System || block.role === PromptRoleEnum.Developer)
		.map(block => block.role);

	return roles.length > 0 && roles.every(role => role === PromptRoleEnum.Developer)
		? PromptRoleEnum.Developer
		: PromptRoleEnum.System;
}

function sortBundles(bundles: PromptBundle[]): PromptBundle[] {
	return [...bundles].sort((a, b) => {
		if (a.isBuiltIn !== b.isBuiltIn) {
			return a.isBuiltIn ? 1 : -1;
		}

		const aLabel = (a.displayName || a.slug).toLowerCase();
		const bLabel = (b.displayName || b.slug).toLowerCase();
		const byLabel = aLabel.localeCompare(bLabel);
		if (byLabel !== 0) return byLabel;

		return a.id.localeCompare(b.id);
	});
}

function sortPrompts(prompts: SystemPromptItem[]): SystemPromptItem[] {
	return [...prompts].sort((a, b) => {
		const byDisplay = a.displayName.localeCompare(b.displayName);
		if (byDisplay !== 0) return byDisplay;

		const bySlug = a.templateSlug.localeCompare(b.templateSlug);
		if (bySlug !== 0) return bySlug;

		return a.templateVersion.localeCompare(b.templateVersion);
	});
}

function pickPreferredBundleID(bundles: PromptBundle[], current: string | null): string | null {
	const allCustomBundles = bundles.filter(bundle => !bundle.isBuiltIn);
	const writableCustomBundles = allCustomBundles.filter(bundle => bundle.isEnabled);

	if (current && allCustomBundles.some(bundle => bundle.id === current)) {
		return current;
	}

	return writableCustomBundles[0]?.id ?? allCustomBundles[0]?.id ?? null;
}

function toSystemPromptItem(template: PromptTemplate, bundle: PromptBundle | undefined): SystemPromptItem | null {
	if (template.kind !== PromptTemplateKind.InstructionsOnly || !template.isResolved) {
		return null;
	}

	const prompt = flattenInstructionsPrompt(template);
	if (!prompt) {
		return null;
	}

	const identity = getPromptTemplateRef(bundle?.id ?? '', template);

	return {
		identityKey: buildPromptTemplateRefKey(identity),
		templateID: template.id,
		bundleID: identity.bundleID,
		templateSlug: template.slug,
		templateVersion: template.version,
		displayName: template.displayName || template.slug,
		prompt,
		role: deriveSystemPromptRole(template),
		bundleDisplayName: bundle?.displayName || bundle?.slug || identity.bundleID,
		bundleSlug: bundle?.slug || '',
		isBuiltIn: template.isBuiltIn,
		createdAt: template.createdAt,
		modifiedAt: template.modifiedAt,
	};
}

async function loadSystemPromptState(): Promise<void> {
	const [bundles, allTemplateListItems, visibleSystemPromptListItems] = await Promise.all([
		getAllPromptBundles(undefined, true),
		getAllPromptTemplates(undefined, undefined, true),
		getAllPromptTemplates(undefined, undefined, false, [PromptTemplateKind.InstructionsOnly], true),
	]);

	const nextBundles = sortBundles(bundles);
	const bundleByID = new Map(nextBundles.map(bundle => [bundle.id, bundle]));

	const templateVersionsBySeriesKey: Record<string, string[]> = {};
	for (const item of allTemplateListItems) {
		const seriesKey = buildPromptTemplateSeriesKey(item.bundleID, item.templateSlug);
		const nextVersions = templateVersionsBySeriesKey[seriesKey] ?? [];
		if (!nextVersions.includes(item.templateVersion)) {
			nextVersions.push(item.templateVersion);
			nextVersions.sort((a, b) => a.localeCompare(b));
			templateVersionsBySeriesKey[seriesKey] = nextVersions;
		}
	}

	const visibleTemplates = await Promise.all(
		visibleSystemPromptListItems.map(item =>
			promptStoreAPI.getPromptTemplate(item.bundleID, item.templateSlug, item.templateVersion)
		)
	);

	const nextPrompts = sortPrompts(
		visibleTemplates
			.map((template, index) => {
				if (!template) return null;
				return toSystemPromptItem(template, bundleByID.get(visibleSystemPromptListItems[index].bundleID));
			})
			.filter(Boolean) as SystemPromptItem[]
	);

	setState({
		prompts: nextPrompts,
		bundles: nextBundles,
		templateVersionsBySeriesKey,
		preferredBundleID: pickPreferredBundleID(nextBundles, state.preferredBundleID),
		loading: false,
		error: null,
		initialized: true,
	});
}

async function refreshSystemPromptStore(): Promise<void> {
	if (refreshPromise) {
		return refreshPromise;
	}

	setState({
		...state,
		loading: !state.initialized,
		error: null,
	});

	refreshPromise = loadSystemPromptState()
		.catch((error: unknown) => {
			setState({
				...state,
				loading: false,
				error: getErrorMessage(error, 'Failed to load system prompts.'),
				initialized: true,
			});
			throw error;
		})
		.finally(() => {
			refreshPromise = null;
		});

	return refreshPromise;
}

function ensureInitialLoad() {
	if (initialLoadRequested || state.initialized || refreshPromise) {
		return;
	}

	initialLoadRequested = true;
	void refreshSystemPromptStore().catch((error: unknown) => {
		console.error('Failed to initialize system prompts:', error);
	});
}

function setPreferredBundleID(bundleID: string | null) {
	setState({
		...state,
		preferredBundleID: bundleID,
	});
}

async function createSystemPrompt(draft: SystemPromptDraft): Promise<SystemPromptItem> {
	if (!state.initialized) {
		await refreshSystemPromptStore();
	}

	const bundle = state.bundles.find(item => item.id === draft.bundleID);
	if (!bundle) {
		throw new Error('Prompt bundle not found.');
	}
	if (bundle.isBuiltIn) {
		throw new Error('Built-in bundles cannot be used here. Use a custom bundle.');
	}
	if (!bundle.isEnabled) {
		throw new Error('Selected bundle is disabled. Enable it from Prompt Bundles first.');
	}

	const displayName = draft.displayName.trim();
	const slug = draft.slug.trim();
	const version = draft.version.trim();
	const content = draft.content.trim();

	if (!displayName) {
		throw new Error('Display name is required.');
	}
	if (!slug) {
		throw new Error('Slug is required.');
	}
	if (!version) {
		throw new Error('Version is required.');
	}
	if (!content) {
		throw new Error('Prompt content is required.');
	}

	const seriesKey = buildPromptTemplateSeriesKey(bundle.id, slug);
	const existingVersions = state.templateVersionsBySeriesKey[seriesKey] ?? [];
	if (existingVersions.includes(version)) {
		throw new Error(`Version "${version}" already exists for slug "${slug}" in this bundle.`);
	}

	await promptStoreAPI.putPromptTemplate(
		PromptTemplateKind.InstructionsOnly,
		bundle.id,
		slug,
		displayName,
		true,
		[
			{
				id: getUUIDv7(),
				role: normalizeSystemPromptRole(draft.role),
				content,
			},
		],
		version,
		true,
		undefined,
		undefined,
		[]
	);

	setPreferredBundleID(bundle.id);
	await refreshSystemPromptStore();

	const identityKey = buildPromptTemplateRefKey({
		bundleID: bundle.id,
		templateSlug: slug,
		templateVersion: version,
	});

	const created = state.prompts.find(item => item.identityKey === identityKey);
	if (!created) {
		throw new Error('Prompt was created but could not be reloaded.');
	}

	return created;
}

export function useSystemPrompts() {
	useEffect(() => {
		ensureInitialLoad();
	}, []);

	const snapshot = useSyncExternalStore(subscribe, getSnapshot, getSnapshot);

	const refreshPrompts = useCallback(() => {
		return refreshSystemPromptStore();
	}, []);

	const addPrompt = useCallback((draft: SystemPromptDraft) => {
		return createSystemPrompt(draft);
	}, []);

	const getExistingVersions = useCallback(
		(bundleID: string, slug: string) => {
			return snapshot.templateVersionsBySeriesKey[buildPromptTemplateSeriesKey(bundleID, slug)] ?? [];
		},
		[snapshot.templateVersionsBySeriesKey]
	);

	return {
		prompts: snapshot.prompts,
		bundles: snapshot.bundles,
		preferredBundleID: snapshot.preferredBundleID,
		loading: snapshot.loading,
		error: snapshot.error,
		refreshPrompts,
		addPrompt,
		getExistingVersions,
	};
}
