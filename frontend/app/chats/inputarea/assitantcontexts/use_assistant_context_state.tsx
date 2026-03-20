import { type Dispatch, type SetStateAction, useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { type OutputVerbosity, type ReasoningLevel, ReasoningType } from '@/spec/inference';
import { DefaultUIChatOptions, type IncludePreviousMessages, type UIChatOption } from '@/spec/modelpreset';
import { type PromptBundle, PromptRoleEnum } from '@/spec/prompt';

import { DEFAULT_SEMVER, suggestNextMinorVersion } from '@/lib/version_utils';

import {
	getSupportedReasoningLevels,
	sanitizeUIChatOptionByCapabilities,
	supportsOutputVerbosity,
} from '@/chats/inputarea/assitantcontexts/capabilities_override_helper';
import { getChatInputOptions } from '@/chats/inputarea/assitantcontexts/context_uichatoption_helper';
import {
	type SetSystemPromptForChatDetail,
	useSetSystemPromptForChat,
} from '@/chats/inputarea/events/set_system_prompt';
import { buildPromptTemplateRefKey } from '@/prompts/lib/prompt_template_ref';
import { buildEffectiveSystemPrompt } from '@/prompts/lib/system_prompt_utils';
import type { SystemPromptDraft } from '@/prompts/lib/use_system_prompts';
import { type SystemPromptItem, useSystemPrompts } from '@/prompts/lib/use_system_prompts';

function normalizeSystemPromptRole(
	role?: PromptRoleEnum.System | PromptRoleEnum.Developer
): PromptRoleEnum.System | PromptRoleEnum.Developer {
	return role === PromptRoleEnum.Developer ? PromptRoleEnum.Developer : PromptRoleEnum.System;
}

function buildSystemPromptDisplayName(prompt: string, role: PromptRoleEnum.System | PromptRoleEnum.Developer): string {
	const firstLine = prompt
		.split(/\r\n?|\n/g)
		.map(line => line.trim())
		.find(Boolean);

	const fallback = role === PromptRoleEnum.Developer ? 'Developer Prompt' : 'System Prompt';
	const label = firstLine || fallback;
	return label.length > 64 ? `${label.slice(0, 64)}…` : label;
}

function slugifySystemPromptName(input: string): string {
	const slug = input
		.toLowerCase()
		.trim()
		.replace(/[^a-z0-9]+/g, '-')
		.replace(/^-+|-+$/g, '')
		.replace(/-{2,}/g, '-');

	return slug || 'system-prompt';
}

function pickWritableSystemPromptBundleID(
	bundles: PromptBundle[],
	preferredBundleID: string | null,
	sourceBundleID?: string
): string | undefined {
	const writableBundles = bundles.filter(bundle => !bundle.isBuiltIn && bundle.isEnabled);

	if (sourceBundleID && writableBundles.some(bundle => bundle.id === sourceBundleID)) {
		return sourceBundleID;
	}

	if (preferredBundleID && writableBundles.some(bundle => bundle.id === preferredBundleID)) {
		return preferredBundleID;
	}

	return writableBundles[0]?.id;
}

function suggestSystemPromptVersion(baseVersion: string | undefined, existingVersions: string[]): string {
	return suggestNextMinorVersion(baseVersion?.trim() || DEFAULT_SEMVER, existingVersions).suggested;
}

function isHybridReasoningModel(model: UIChatOption): boolean {
	return model.reasoning?.type === ReasoningType.HybridWithTokens;
}

function buildFinalOptions(
	selectedModel: UIChatOption,
	includePreviousMessages: IncludePreviousMessages,
	isHybridReasoningEnabled: boolean,
	includeModelDefault: boolean,
	selectedPromptKeys: string[],
	promptsByKey: Map<string, SystemPromptItem>
): UIChatOption {
	const base: UIChatOption = {
		...selectedModel,
		includePreviousMessages,
		systemPrompt: buildEffectiveSystemPrompt({
			modelDefaultPrompt: selectedModel.systemPrompt,
			includeModelDefault,
			selectedPromptKeys,
			promptsByKey,
		}),
	};

	if (selectedModel.reasoning?.type === ReasoningType.HybridWithTokens && !isHybridReasoningEnabled) {
		const modifiedOptions = { ...base };
		delete modifiedOptions.reasoning;

		if (modifiedOptions.temperature === undefined) {
			modifiedOptions.temperature = DefaultUIChatOptions.temperature;
		}

		return sanitizeUIChatOptionByCapabilities(modifiedOptions);
	}

	return sanitizeUIChatOptionByCapabilities(base);
}

export type AssistantContextController = {
	chatOptions: UIChatOption;

	selectedModel: UIChatOption;
	allOptions: UIChatOption[];

	isHybridReasoningEnabled: boolean;
	includePreviousMessages: IncludePreviousMessages;
	prompts: SystemPromptItem[];
	selectedPromptKeys: string[];
	systemPromptBundles: PromptBundle[];
	preferredSystemPromptBundleID: string | null;
	systemPromptsLoading: boolean;
	systemPromptError: string | null;
	includeModelDefault: boolean;

	handleSetSelectedModel: Dispatch<SetStateAction<UIChatOption>>;
	handleSetIsHybridReasoningEnabled: Dispatch<SetStateAction<boolean>>;
	setIncludePreviousMessages: Dispatch<SetStateAction<IncludePreviousMessages>>;
	setIncludeModelDefault: Dispatch<SetStateAction<boolean>>;

	setTemperature: (temp: number) => void;
	setReasoningLevel: (level: ReasoningLevel) => void;
	setHybridTokens: (tokens: number) => void;
	setOutputVerbosity: (verbosity?: OutputVerbosity) => void;

	togglePromptSelection: (identityKey: string) => void;
	addAndSelectPrompt: (draft: SystemPromptDraft) => Promise<void>;
	clearSelectedPromptSources: () => void;
	refreshSystemPrompts: () => Promise<void>;
	getExistingSystemPromptVersions: (bundleID: string, slug: string) => string[];

	applyAdvancedModel: (updatedModel: UIChatOption) => void;

	verbosityEnabled: boolean;
	reasoningLevelOptions: ReasoningLevel[];
};

export function useAssistantContextState({ active }: { active: boolean }): AssistantContextController {
	const [selectedModel, setSelectedModel] = useState<UIChatOption>(DefaultUIChatOptions);
	const [allOptions, setAllOptions] = useState<UIChatOption[]>([DefaultUIChatOptions]);

	const [isHybridReasoningEnabled, setIsHybridReasoningEnabled] = useState(true);
	const [includePreviousMessages, setIncludePreviousMessages] = useState<IncludePreviousMessages>(
		DefaultUIChatOptions.includePreviousMessages
	);
	const [rawSelectedPromptKeys, setRawSelectedPromptKeys] = useState<string[]>([]);
	const [includeModelDefault, setIncludeModelDefault] = useState<boolean>(
		Boolean(DefaultUIChatOptions.systemPrompt.trim())
	);

	const selectedModelRef = useRef(selectedModel);
	const isHybridReasoningEnabledRef = useRef(isHybridReasoningEnabled);

	useEffect(() => {
		selectedModelRef.current = selectedModel;
	}, [selectedModel]);

	useEffect(() => {
		isHybridReasoningEnabledRef.current = isHybridReasoningEnabled;
	}, [isHybridReasoningEnabled]);

	const {
		prompts,
		bundles: systemPromptBundles,
		preferredBundleID: preferredSystemPromptBundleID,
		loading: systemPromptsLoading,
		error: systemPromptError,
		addPrompt,
		refreshPrompts,
		getExistingVersions,
	} = useSystemPrompts();

	const promptsByKey = useMemo(() => new Map(prompts.map(item => [item.identityKey, item])), [prompts]);

	const selectedPromptKeys = useMemo(
		() => rawSelectedPromptKeys.filter(key => promptsByKey.has(key)),
		[promptsByKey, rawSelectedPromptKeys]
	);

	const chatOptions = useMemo(
		() =>
			buildFinalOptions(
				selectedModel,
				includePreviousMessages,
				isHybridReasoningEnabled,
				includeModelDefault,
				selectedPromptKeys,
				promptsByKey
			),
		[
			includeModelDefault,
			includePreviousMessages,
			isHybridReasoningEnabled,
			promptsByKey,
			selectedModel,
			selectedPromptKeys,
		]
	);

	const applySelectedModel = useCallback(
		(
			action: SetStateAction<UIChatOption>,
			options?: {
				syncHybridFromModel?: boolean;
			}
		) => {
			const currentSelectedModel = selectedModelRef.current;
			const currentIsHybridReasoningEnabled = isHybridReasoningEnabledRef.current;

			const nextSelectedModel =
				typeof action === 'function'
					? (action as (prevState: UIChatOption) => UIChatOption)(currentSelectedModel)
					: action;

			const nextIsHybridReasoningEnabled = options?.syncHybridFromModel
				? isHybridReasoningModel(nextSelectedModel)
				: currentIsHybridReasoningEnabled;

			setSelectedModel(nextSelectedModel);

			if (nextIsHybridReasoningEnabled !== currentIsHybridReasoningEnabled) {
				setIsHybridReasoningEnabled(nextIsHybridReasoningEnabled);
			}
		},
		[]
	);

	useEffect(() => {
		let cancelled = false;

		void (async () => {
			const r = await getChatInputOptions();
			if (cancelled) return;

			const nextSelectedModel = sanitizeUIChatOptionByCapabilities(r.default);
			const nextIsHybridReasoningEnabled = isHybridReasoningModel(nextSelectedModel);

			setSelectedModel(nextSelectedModel);
			setAllOptions(r.allOptions);
			setIsHybridReasoningEnabled(nextIsHybridReasoningEnabled);
			setIncludeModelDefault(Boolean(nextSelectedModel.systemPrompt.trim()));
		})();

		return () => {
			cancelled = true;
		};
	}, []);

	const handleSetSelectedModel = useCallback(
		(action: SetStateAction<UIChatOption>) => {
			applySelectedModel(action, {
				syncHybridFromModel: true,
			});
		},
		[applySelectedModel]
	);

	const handleSetIsHybridReasoningEnabled = useCallback((action: SetStateAction<boolean>) => {
		const currentIsHybridReasoningEnabled = isHybridReasoningEnabledRef.current;
		const nextIsHybridReasoningEnabled =
			typeof action === 'function' ? action(currentIsHybridReasoningEnabled) : action;
		setIsHybridReasoningEnabled(nextIsHybridReasoningEnabled);
	}, []);

	const setTemperature = useCallback(
		(temp: number) => {
			const clampedTemp = Math.max(0, Math.min(1, temp));
			applySelectedModel(prev => ({ ...prev, temperature: clampedTemp }));
		},
		[applySelectedModel]
	);

	const setReasoningLevel = useCallback(
		(newLevel: ReasoningLevel) => {
			applySelectedModel(prev => ({
				...prev,
				reasoning: {
					type: ReasoningType.SingleWithLevels,
					level: newLevel,
					tokens: 1024,
					summaryStyle: prev.reasoning?.summaryStyle,
				},
			}));
		},
		[applySelectedModel]
	);

	const setHybridTokens = useCallback(
		(tokens: number) => {
			applySelectedModel(prev => {
				if (!prev.reasoning || prev.reasoning.type !== ReasoningType.HybridWithTokens) return prev;
				return { ...prev, reasoning: { ...prev.reasoning, tokens } };
			});
		},
		[applySelectedModel]
	);

	const setOutputVerbosity = useCallback(
		(verbosity?: OutputVerbosity) => {
			applySelectedModel(prev => {
				const next = { ...(prev.outputParam ?? {}) };

				if (verbosity === undefined) {
					delete next.verbosity;
				} else {
					next.verbosity = verbosity;
				}

				const hasAny = !!next.verbosity || !!next.format;

				return {
					...prev,
					outputParam: hasAny ? next : undefined,
				};
			});
		},
		[applySelectedModel]
	);

	const togglePromptSelection = useCallback((identityKey: string) => {
		setRawSelectedPromptKeys(prev =>
			prev.includes(identityKey) ? prev.filter(item => item !== identityKey) : [...prev, identityKey]
		);
	}, []);

	const addAndSelectPrompt = useCallback(
		async (draft: SystemPromptDraft) => {
			const item = await addPrompt(draft);
			setRawSelectedPromptKeys(prev => (prev.includes(item.identityKey) ? prev : [...prev, item.identityKey]));
		},
		[addPrompt]
	);

	const clearSelectedPromptSources = useCallback(() => {
		setIncludeModelDefault(false);
		setRawSelectedPromptKeys([]);
	}, []);

	const handleSetSystemPromptForChat = useCallback(
		(detail: SetSystemPromptForChatDetail) => {
			if (!active) return;

			const trimmed = detail.prompt.trim();
			if (!trimmed) return;

			const role = normalizeSystemPromptRole(detail.role);
			const sourceIdentityKey = detail.sourceTemplate ? buildPromptTemplateRefKey(detail.sourceTemplate) : undefined;
			const exactExisting = sourceIdentityKey ? promptsByKey.get(sourceIdentityKey) : undefined;

			if (exactExisting && exactExisting.prompt === trimmed && exactExisting.role === role) {
				setIncludeModelDefault(false);
				setRawSelectedPromptKeys([exactExisting.identityKey]);
				return;
			}

			const targetBundleID = pickWritableSystemPromptBundleID(
				systemPromptBundles,
				preferredSystemPromptBundleID,
				detail.sourceTemplate?.bundleID
			);

			if (!targetBundleID) {
				console.error('No writable custom prompt bundle is available for saving a system prompt.');
				return;
			}

			const displayName =
				detail.displayName?.trim() || exactExisting?.displayName || buildSystemPromptDisplayName(trimmed, role);
			const slug = detail.sourceTemplate?.templateSlug || slugifySystemPromptName(displayName);
			const version = suggestSystemPromptVersion(
				detail.sourceTemplate?.templateVersion,
				getExistingVersions(targetBundleID, slug)
			);

			void addPrompt({
				bundleID: targetBundleID,
				displayName,
				slug,
				version,
				role,
				content: trimmed,
			})
				.then(item => {
					setIncludeModelDefault(false);
					setRawSelectedPromptKeys([item.identityKey]);
				})
				.catch((error: unknown) => {
					console.error('Failed to save system prompt from event:', error);
				});
		},
		[active, addPrompt, getExistingVersions, preferredSystemPromptBundleID, promptsByKey, systemPromptBundles]
	);

	useSetSystemPromptForChat(handleSetSystemPromptForChat);

	const applyAdvancedModel = useCallback(
		(updatedModel: UIChatOption) => {
			applySelectedModel(sanitizeUIChatOptionByCapabilities(updatedModel));
		},
		[applySelectedModel]
	);

	const verbosityEnabled = supportsOutputVerbosity(selectedModel.capabilitiesOverride);
	const reasoningLevelOptions = getSupportedReasoningLevels(selectedModel.capabilitiesOverride);

	return {
		chatOptions,

		selectedModel,
		allOptions,

		isHybridReasoningEnabled,
		includePreviousMessages,
		prompts,
		selectedPromptKeys,
		systemPromptBundles,
		preferredSystemPromptBundleID,
		systemPromptsLoading,
		systemPromptError,
		includeModelDefault,

		handleSetSelectedModel,
		handleSetIsHybridReasoningEnabled,
		setIncludePreviousMessages,
		setIncludeModelDefault,

		setTemperature,
		setReasoningLevel,
		setHybridTokens,
		setOutputVerbosity,

		togglePromptSelection,
		addAndSelectPrompt,
		clearSelectedPromptSources,
		refreshSystemPrompts: refreshPrompts,
		getExistingSystemPromptVersions: getExistingVersions,

		applyAdvancedModel,
		verbosityEnabled,
		reasoningLevelOptions,
	};
}
