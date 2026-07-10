import { useCallback, useMemo, useRef, useState } from 'react';

import {
	BASE_ASSISTANT_PRESET_BUNDLEID,
	BASE_ASSISTANT_PRESET_SLUG,
	BASE_ASSISTANT_PRESET_VERSION,
} from '@/spec/assistantpreset';

import type {
	AssistantPresetModificationSummary,
	AssistantPresetOptionItem,
	AssistantPresetPreparedApplication,
	AssistantPresetRuntimeSnapshot,
} from '@/chats/composer/assistantpresets/assistant_preset_runtime';
import {
	buildAssistantPresetModelComparisonState,
	EMPTY_ASSISTANT_PRESET_MODIFICATION_SUMMARY,
	findBaseAssistantPresetOption,
	findDefaultAssistantPresetOption,
	getAssistantPresetModificationSummary,
	normalizeAssistantPresetMCPContext,
} from '@/chats/composer/assistantpresets/assistant_preset_runtime';
import type { AssistantContextController } from '@/chats/composer/contextarea/use_context_state';
import type { ComposerSystemPromptController } from '@/chats/composer/skills/use_composer_system_prompt';

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

const NO_SELECTABLE_PRESET_ERROR = `No selectable assistant preset is available. Expected a "${BASE_ASSISTANT_PRESET_SLUG}" preset or another selectable fallback.`;

export interface AssistantPresetManagerState {
	presetOptions: AssistantPresetOptionItem[];
	loading: boolean;
	error: string | null;
	actionError: string | null;
	isApplying: boolean;

	basePresetKey: string | null;
	isBasePresetSelected: boolean;

	selectedPresetKey: string | null;
	selectedPreset: AssistantPresetOptionItem | null;

	appliedPresetApplication: AssistantPresetPreparedApplication | null;
	runtimeSnapshot: AssistantPresetRuntimeSnapshot;
	modificationSummary: AssistantPresetModificationSummary;

	resetToBasePreset: () => Promise<boolean>;
	ensureActivePreset: () => Promise<boolean>;
	selectPreset: (presetKey: string) => Promise<boolean>;
	reapplySelectedPreset: () => Promise<boolean>;
	clearSelectedPreset: () => void;
	trackDefaultPresetWithoutApplying: () => Promise<boolean>;
}

interface AssistantPresetSelectionState {
	selectedPresetKey: string | null;
	appliedPresetApplication: AssistantPresetPreparedApplication | null;
	actionError: string | null;
}

export function useAssistantPresetManager(args: {
	context: AssistantContextController;
	systemPrompt: ComposerSystemPromptController;
	runtimeSnapshot: AssistantPresetRuntimeSnapshot;
	applyRuntimeSelections: (prepared: AssistantPresetPreparedApplication) => void;
}): AssistantPresetManagerState {
	const { context, systemPrompt, runtimeSnapshot, applyRuntimeSelections } = args;
	const {
		assistantPresetOptions,
		assistantPresetsLoading,
		assistantPresetError,
		modelOptionsLoaded,
		selectedModel,
		prepareAssistantPresetApplication,
		applyPreparedAssistantPreset,
	} = context;
	const {
		includeModelDefault,
		selectedPromptKeys,
		prepareAssistantPresetSelections,
		applyPreparedAssistantPresetSelections,
	} = systemPrompt;

	const [selectionState, setSelectionState] = useState<AssistantPresetSelectionState>({
		selectedPresetKey: null,
		appliedPresetApplication: null,
		actionError: null,
	});
	const [isApplying, setIsApplying] = useState(false);
	const applyRequestSeqRef = useRef(0);

	const setSelectionActionError = useCallback((message: string | null) => {
		setSelectionState(current => {
			if (current.actionError === message) {
				return current;
			}
			return {
				...current,
				actionError: message,
			};
		});
	}, []);

	const clearSelectionActionError = useCallback(() => {
		setSelectionActionError(null);
	}, [setSelectionActionError]);

	const basePreset = useMemo(
		() =>
			findBaseAssistantPresetOption(
				assistantPresetOptions,
				BASE_ASSISTANT_PRESET_BUNDLEID,
				BASE_ASSISTANT_PRESET_SLUG,
				BASE_ASSISTANT_PRESET_VERSION
			),
		[assistantPresetOptions]
	);
	const isPresetLayerReady = !assistantPresetsLoading && modelOptionsLoaded;
	const basePresetKey = basePreset?.key ?? null;
	const basePresetSelectableKey = basePreset?.isSelectable ? basePreset.key : null;

	const invariantFallbackPreset = useMemo(
		() => findDefaultAssistantPresetOption(assistantPresetOptions),
		[assistantPresetOptions]
	);
	const invariantFallbackPresetKey = invariantFallbackPreset?.isSelectable ? invariantFallbackPreset.key : null;

	const hasMissingSelectedPreset =
		selectionState.selectedPresetKey !== null &&
		!assistantPresetOptions.some(option => option.key === selectionState.selectedPresetKey);
	const selectedPresetKey = hasMissingSelectedPreset ? null : selectionState.selectedPresetKey;
	const appliedPresetApplication = hasMissingSelectedPreset ? null : selectionState.appliedPresetApplication;
	const actionError = hasMissingSelectedPreset ? null : selectionState.actionError;
	const activePresetKey = selectedPresetKey ?? appliedPresetApplication?.presetKey ?? null;
	const isBasePresetSelected = activePresetKey !== null && basePreset?.key === activePresetKey;

	const selectedPreset = useMemo(() => {
		if (!selectedPresetKey) {
			return appliedPresetApplication?.option ?? null;
		}

		return (
			assistantPresetOptions.find(option => option.key === selectedPresetKey) ??
			appliedPresetApplication?.option ??
			null
		);
	}, [appliedPresetApplication?.option, assistantPresetOptions, selectedPresetKey]);

	const modificationSummary = useMemo(() => {
		if (!appliedPresetApplication) {
			return EMPTY_ASSISTANT_PRESET_MODIFICATION_SUMMARY;
		}

		return getAssistantPresetModificationSummary({
			preparedApplication: appliedPresetApplication,
			currentSelectedModel: selectedModel,
			currentIncludeModelSystemPrompt: includeModelDefault,
			currentSelectedPromptKeys: selectedPromptKeys,
			currentRuntimeSnapshot: runtimeSnapshot,
		});
	}, [appliedPresetApplication, includeModelDefault, runtimeSnapshot, selectedModel, selectedPromptKeys]);

	const buildPreparedApplication = useCallback(
		async (presetKey: string): Promise<AssistantPresetPreparedApplication | null> => {
			const basePrepared = await prepareAssistantPresetApplication(presetKey);
			if (!basePrepared) {
				return null;
			}

			const preparedSystemPromptSelections = await prepareAssistantPresetSelections(basePrepared.preset);

			const normalizedPresetMCPContext = normalizeAssistantPresetMCPContext(basePrepared.preset.startingMCPContext);
			const runtimeSelections = {
				...basePrepared.runtimeSelections,
				hasMCPSelection: basePrepared.runtimeSelections.hasMCPSelection ?? Boolean(normalizedPresetMCPContext),
				mcpContext: basePrepared.runtimeSelections.mcpContext ?? normalizedPresetMCPContext,
			};
			const comparisonMCPContext = runtimeSelections.hasMCPSelection
				? normalizeAssistantPresetMCPContext(runtimeSelections.mcpContext ?? normalizedPresetMCPContext)
				: undefined;

			return {
				...basePrepared,
				runtimeSelections,
				hasIncludeModelSystemPromptSelection: preparedSystemPromptSelections.hasIncludeModelSystemPromptSelection,
				nextIncludeModelSystemPrompt: preparedSystemPromptSelections.nextIncludeModelSystemPrompt,
				hasInstructionTemplateSelection: preparedSystemPromptSelections.hasInstructionTemplateSelection,
				nextSelectedPromptKeys: preparedSystemPromptSelections.nextSelectedPromptKeys,
				comparisonState: {
					...basePrepared.comparisonState,
					model: buildAssistantPresetModelComparisonState(
						basePrepared.preset,
						basePrepared.nextSelectedModel,
						preparedSystemPromptSelections.nextIncludeModelSystemPrompt
					),
					instructions: preparedSystemPromptSelections.hasInstructionTemplateSelection
						? [...preparedSystemPromptSelections.nextSelectedPromptKeys]
						: undefined,
					mcp: comparisonMCPContext,
				},
			};
		},
		[prepareAssistantPresetApplication, prepareAssistantPresetSelections]
	);

	const applyPresetByKey = useCallback(
		async (presetKey: string): Promise<boolean> => {
			const requestSeq = applyRequestSeqRef.current + 1;
			applyRequestSeqRef.current = requestSeq;
			setIsApplying(true);
			clearSelectionActionError();
			try {
				const prepared = await buildPreparedApplication(presetKey);
				if (applyRequestSeqRef.current !== requestSeq) {
					return false;
				}
				if (!prepared) {
					setSelectionActionError('Assistant preset not found.');
					return false;
				}

				applyPreparedAssistantPreset(prepared);
				applyPreparedAssistantPresetSelections(prepared);
				applyRuntimeSelections(prepared);

				setSelectionState({
					selectedPresetKey: presetKey,
					appliedPresetApplication: prepared,
					actionError: null,
				});
				return true;
			} catch (error) {
				if (applyRequestSeqRef.current !== requestSeq) {
					return false;
				}
				console.error('Failed to apply assistant preset:', error);
				setSelectionActionError(getErrorMessage(error, 'Failed to apply assistant preset.'));

				return false;
			} finally {
				if (applyRequestSeqRef.current === requestSeq) {
					setIsApplying(false);
				}
			}
		},
		[
			applyPreparedAssistantPreset,
			applyPreparedAssistantPresetSelections,
			applyRuntimeSelections,
			buildPreparedApplication,
			clearSelectionActionError,
			setSelectionActionError,
		]
	);

	const trackPresetWithoutApplying = useCallback(
		async (presetKey: string): Promise<boolean> => {
			const requestSeq = applyRequestSeqRef.current + 1;
			applyRequestSeqRef.current = requestSeq;
			clearSelectionActionError();
			try {
				const prepared = await buildPreparedApplication(presetKey);
				if (applyRequestSeqRef.current !== requestSeq) {
					return false;
				}

				if (!prepared) {
					setSelectionActionError('Assistant preset not found.');
					return false;
				}

				setSelectionState({
					selectedPresetKey: presetKey,
					appliedPresetApplication: prepared,
					actionError: null,
				});
				return true;
			} catch (error) {
				if (applyRequestSeqRef.current !== requestSeq) {
					return false;
				}
				console.error('Failed to track assistant preset without applying:', error);
				setSelectionActionError(getErrorMessage(error, 'Failed to prepare assistant preset.'));
				return false;
			}
		},
		[buildPreparedApplication, clearSelectionActionError, setSelectionActionError]
	);

	const trackDefaultPresetWithoutApplying = useCallback(async (): Promise<boolean> => {
		if (!isPresetLayerReady) {
			clearSelectionActionError();
			return false;
		}
		if (!invariantFallbackPresetKey) {
			if (assistantPresetOptions.length > 0) {
				setSelectionActionError(NO_SELECTABLE_PRESET_ERROR);
			} else {
				clearSelectionActionError();
			}
			return false;
		}

		return trackPresetWithoutApplying(invariantFallbackPresetKey);
	}, [
		assistantPresetOptions.length,
		clearSelectionActionError,
		invariantFallbackPresetKey,
		isPresetLayerReady,
		setSelectionActionError,
		trackPresetWithoutApplying,
	]);

	const reapplySelectedPreset = useCallback(async (): Promise<boolean> => {
		const presetKey = selectedPresetKey ?? appliedPresetApplication?.presetKey;
		if (!presetKey) {
			return false;
		}

		return applyPresetByKey(presetKey);
	}, [appliedPresetApplication?.presetKey, applyPresetByKey, selectedPresetKey]);

	const ensureActivePreset = useCallback(async (): Promise<boolean> => {
		if (activePresetKey || isApplying) {
			return true;
		}

		if (!isPresetLayerReady) {
			clearSelectionActionError();
			return false;
		}

		if (!invariantFallbackPresetKey) {
			if (assistantPresetOptions.length > 0) {
				setSelectionActionError(NO_SELECTABLE_PRESET_ERROR);
			} else {
				clearSelectionActionError();
			}
			return false;
		}

		return applyPresetByKey(invariantFallbackPresetKey);
	}, [
		activePresetKey,
		applyPresetByKey,
		assistantPresetOptions.length,
		clearSelectionActionError,
		invariantFallbackPresetKey,
		isApplying,
		isPresetLayerReady,
		setSelectionActionError,
	]);

	const resetToBasePreset = useCallback(async (): Promise<boolean> => {
		const targetPresetKey = basePresetSelectableKey ?? invariantFallbackPresetKey;

		if (!targetPresetKey) {
			if (!isPresetLayerReady) {
				clearSelectionActionError();
				return false;
			}

			setSelectionActionError(NO_SELECTABLE_PRESET_ERROR);

			return false;
		}

		return applyPresetByKey(targetPresetKey);
	}, [
		applyPresetByKey,
		basePresetSelectableKey,
		clearSelectionActionError,
		invariantFallbackPresetKey,
		isPresetLayerReady,
		setSelectionActionError,
	]);

	const clearSelectedPreset = useCallback(() => {
		void resetToBasePreset();
	}, [resetToBasePreset]);

	return useMemo(
		() => ({
			presetOptions: assistantPresetOptions,
			loading: assistantPresetsLoading,
			error: assistantPresetError,
			actionError,
			isApplying,
			basePresetKey,
			isBasePresetSelected,

			selectedPresetKey,
			selectedPreset,
			appliedPresetApplication,
			runtimeSnapshot,
			modificationSummary,

			selectPreset: applyPresetByKey,
			resetToBasePreset,
			ensureActivePreset,
			reapplySelectedPreset,
			clearSelectedPreset,
			trackDefaultPresetWithoutApplying,
		}),
		[
			actionError,
			appliedPresetApplication,
			applyPresetByKey,
			assistantPresetError,
			assistantPresetOptions,
			assistantPresetsLoading,
			basePresetKey,
			clearSelectedPreset,
			ensureActivePreset,
			isApplying,
			isBasePresetSelected,
			modificationSummary,
			reapplySelectedPreset,
			resetToBasePreset,
			runtimeSnapshot,
			selectedPreset,
			selectedPresetKey,
			trackDefaultPresetWithoutApplying,
		]
	);
}
