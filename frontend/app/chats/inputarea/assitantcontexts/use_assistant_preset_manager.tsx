import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import {
	BASE_ASSISTANT_PRESET_BUNDLEID,
	BASE_ASSISTANT_PRESET_SLUG,
	BASE_ASSISTANT_PRESET_VERSION,
} from '@/spec/assistantpreset';

import {
	type AssistantPresetModificationSummary,
	type AssistantPresetOptionItem,
	type AssistantPresetPreparedApplication,
	type AssistantPresetRuntimeSnapshot,
	EMPTY_ASSISTANT_PRESET_MODIFICATION_SUMMARY,
	findBaseAssistantPresetOption,
	findDefaultAssistantPresetOption,
	getAssistantPresetModificationSummary,
} from '@/chats/inputarea/assitantcontexts/assistant_preset_runtime';
import type { AssistantContextController } from '@/chats/inputarea/assitantcontexts/use_assistant_context_state';

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

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
	runtimeSnapshot: AssistantPresetRuntimeSnapshot;
	applyRuntimeSelections: (prepared: AssistantPresetPreparedApplication) => void;
}): AssistantPresetManagerState {
	const { context, runtimeSnapshot, applyRuntimeSelections } = args;
	const {
		assistantPresetOptions,
		assistantPresetsLoading,
		assistantPresetError,
		modelOptionsLoaded,
		selectedModel,
		includeModelDefault,
		selectedPromptKeys,
		prepareAssistantPresetApplication,
		applyPreparedAssistantPreset,
	} = context;

	const [selectionState, setSelectionState] = useState<AssistantPresetSelectionState>({
		selectedPresetKey: null,
		appliedPresetApplication: null,
		actionError: null,
	});
	const [isApplying, setIsApplying] = useState(false);
	const applyRequestSeqRef = useRef(0);
	const noSelectablePresetError = `No selectable assistant preset is available. Expected a "${BASE_ASSISTANT_PRESET_SLUG}" preset or another selectable fallback.`;

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

	const suppressInvariantRef = useRef(false);
	const pendingDefaultTrackingRef = useRef(false);
	const autoAppliedInvariantPresetKeyRef = useRef<string | null>(null);
	const cancelPendingDefaultTracking = useCallback(() => {
		pendingDefaultTrackingRef.current = false;
		suppressInvariantRef.current = false;
	}, []);
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
	}, [appliedPresetApplication, includeModelDefault, selectedModel, selectedPromptKeys, runtimeSnapshot]);

	const applyPresetByKey = useCallback(
		async (presetKey: string): Promise<boolean> => {
			cancelPendingDefaultTracking();
			const requestSeq = applyRequestSeqRef.current + 1;
			applyRequestSeqRef.current = requestSeq;
			setIsApplying(true);
			clearSelectionActionError();
			try {
				const prepared = await prepareAssistantPresetApplication(presetKey);
				if (applyRequestSeqRef.current !== requestSeq) {
					return false;
				}
				if (!prepared) {
					setSelectionActionError('Assistant preset not found.');
					return false;
				}

				applyPreparedAssistantPreset(prepared);
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
			applyRuntimeSelections,
			cancelPendingDefaultTracking,
			clearSelectionActionError,
			prepareAssistantPresetApplication,
			setSelectionActionError,
		]
	);

	const resolveDefaultPresetTrackingWithoutApplying = useCallback(async (): Promise<boolean> => {
		if (assistantPresetsLoading || !modelOptionsLoaded) {
			return false;
		}

		const defaultPresetKey = invariantFallbackPresetKey;
		if (!defaultPresetKey) {
			pendingDefaultTrackingRef.current = false;
			suppressInvariantRef.current = false;
			setSelectionActionError(noSelectablePresetError);

			return false;
		}

		const requestSeq = applyRequestSeqRef.current + 1;
		applyRequestSeqRef.current = requestSeq;
		clearSelectionActionError();

		try {
			const prepared = await prepareAssistantPresetApplication(defaultPresetKey);
			if (applyRequestSeqRef.current !== requestSeq) {
				return false;
			}

			if (!prepared) {
				setSelectionActionError('Assistant preset not found.');
				return false;
			}

			autoAppliedInvariantPresetKeyRef.current = null;
			setSelectionState({
				selectedPresetKey: defaultPresetKey,
				appliedPresetApplication: prepared,
				actionError: null,
			});
			return true;
		} catch (error) {
			if (applyRequestSeqRef.current !== requestSeq) {
				return false;
			}
			console.error('Failed to track default assistant preset:', error);
			setSelectionActionError(getErrorMessage(error, 'Failed to prepare default assistant preset.'));
			return false;
		} finally {
			if (applyRequestSeqRef.current === requestSeq) {
				pendingDefaultTrackingRef.current = false;
				suppressInvariantRef.current = false;
			}
		}
	}, [
		assistantPresetsLoading,
		clearSelectionActionError,
		invariantFallbackPresetKey,
		modelOptionsLoaded,
		noSelectablePresetError,
		prepareAssistantPresetApplication,
		setSelectionActionError,
	]);

	const trackDefaultPresetWithoutApplying = useCallback(async (): Promise<boolean> => {
		pendingDefaultTrackingRef.current = true;
		suppressInvariantRef.current = true;

		if (assistantPresetsLoading || !modelOptionsLoaded) {
			clearSelectionActionError();
			return false;
		}

		return resolveDefaultPresetTrackingWithoutApplying();
	}, [
		assistantPresetsLoading,
		clearSelectionActionError,
		modelOptionsLoaded,
		resolveDefaultPresetTrackingWithoutApplying,
	]);

	const reapplySelectedPreset = useCallback(async (): Promise<boolean> => {
		const presetKey = selectedPresetKey ?? appliedPresetApplication?.presetKey;
		if (!presetKey) {
			return false;
		}

		return applyPresetByKey(presetKey);
	}, [appliedPresetApplication?.presetKey, applyPresetByKey, selectedPresetKey]);

	const resetToBasePreset = useCallback(async (): Promise<boolean> => {
		const targetPresetKey = (basePreset?.isSelectable ? basePreset.key : null) ?? invariantFallbackPresetKey;
		if (!targetPresetKey) {
			if (assistantPresetsLoading || !modelOptionsLoaded) {
				clearSelectionActionError();
				return false;
			}

			setSelectionActionError(noSelectablePresetError);

			return false;
		}

		return applyPresetByKey(targetPresetKey);
	}, [
		applyPresetByKey,
		assistantPresetsLoading,
		basePreset?.isSelectable,
		basePreset?.key,
		clearSelectionActionError,
		invariantFallbackPresetKey,
		modelOptionsLoaded,
		noSelectablePresetError,
		setSelectionActionError,
	]);

	useEffect(() => {
		autoAppliedInvariantPresetKeyRef.current = null;
	}, [assistantPresetOptions, modelOptionsLoaded]);

	const clearSelectedPreset = useCallback(() => {
		void resetToBasePreset();
	}, [resetToBasePreset]);

	useEffect(() => {
		if (!pendingDefaultTrackingRef.current) {
			return;
		}

		if (assistantPresetsLoading || !modelOptionsLoaded) {
			return;
		}

		void resolveDefaultPresetTrackingWithoutApplying();
	}, [assistantPresetsLoading, modelOptionsLoaded, resolveDefaultPresetTrackingWithoutApplying]);

	useEffect(() => {
		// eslint-disable-next-line react-you-might-not-need-an-effect/no-event-handler
		if (activePresetKey) {
			autoAppliedInvariantPresetKeyRef.current = null;
			return;
		}

		// eslint-disable-next-line react-you-might-not-need-an-effect/no-event-handler
		if (suppressInvariantRef.current || assistantPresetsLoading || !modelOptionsLoaded || isApplying) {
			return;
		}

		if (!invariantFallbackPresetKey) {
			if (assistantPresetOptions.length === 0) {
				return;
			}

			// eslint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change, react-you-might-not-need-an-effect/no-chain-state-updates
			setSelectionActionError(noSelectablePresetError);
			return;
		}

		if (autoAppliedInvariantPresetKeyRef.current === invariantFallbackPresetKey) {
			return;
		}

		autoAppliedInvariantPresetKeyRef.current = invariantFallbackPresetKey;
		void applyPresetByKey(invariantFallbackPresetKey);
	}, [
		activePresetKey,
		applyPresetByKey,
		assistantPresetOptions.length,
		assistantPresetsLoading,
		assistantPresetOptions,
		modelOptionsLoaded,
		invariantFallbackPresetKey,
		isApplying,
		noSelectablePresetError,
		setSelectionActionError,
	]);

	return {
		presetOptions: assistantPresetOptions,
		loading: assistantPresetsLoading,
		error: assistantPresetError,
		actionError,
		isApplying,
		basePresetKey: basePreset?.key ?? null,
		isBasePresetSelected,

		selectedPresetKey,
		selectedPreset,
		appliedPresetApplication,
		runtimeSnapshot,
		modificationSummary,

		selectPreset: applyPresetByKey,
		resetToBasePreset,

		reapplySelectedPreset,
		clearSelectedPreset,
		trackDefaultPresetWithoutApplying,
	};
}
