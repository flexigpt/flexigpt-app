import { useCallback, useMemo, useRef, useState } from 'react';

import {
	type AssistantPresetModificationSummary,
	type AssistantPresetOptionItem,
	type AssistantPresetPreparedApplication,
	type AssistantPresetRuntimeSnapshot,
	EMPTY_ASSISTANT_PRESET_MODIFICATION_SUMMARY,
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

	selectedPresetKey: string | null;
	selectedPreset: AssistantPresetOptionItem | null;
	appliedPresetApplication: AssistantPresetPreparedApplication | null;
	runtimeSnapshot: AssistantPresetRuntimeSnapshot;
	modificationSummary: AssistantPresetModificationSummary;

	selectPreset: (presetKey: string) => Promise<boolean>;
	reapplySelectedPreset: () => Promise<boolean>;
	clearSelectedPreset: () => void;
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

	const [selectionState, setSelectionState] = useState<AssistantPresetSelectionState>({
		selectedPresetKey: null,
		appliedPresetApplication: null,
		actionError: null,
	});
	const [isApplying, setIsApplying] = useState(false);
	const applyRequestSeqRef = useRef(0);
	const invalidatePendingApply = useCallback(() => {
		applyRequestSeqRef.current += 1;
		setIsApplying(false);
	}, []);

	const hasMissingSelectedPreset =
		selectionState.selectedPresetKey !== null &&
		!context.assistantPresetOptions.some(option => option.key === selectionState.selectedPresetKey);
	const selectedPresetKey = hasMissingSelectedPreset ? null : selectionState.selectedPresetKey;
	const appliedPresetApplication = hasMissingSelectedPreset ? null : selectionState.appliedPresetApplication;
	const actionError = hasMissingSelectedPreset ? null : selectionState.actionError;

	const selectedPreset = useMemo(() => {
		if (!selectedPresetKey) {
			return appliedPresetApplication?.option ?? null;
		}

		return (
			context.assistantPresetOptions.find(option => option.key === selectedPresetKey) ??
			appliedPresetApplication?.option ??
			null
		);
	}, [appliedPresetApplication?.option, context.assistantPresetOptions, selectedPresetKey]);

	const modificationSummary = useMemo(() => {
		if (!appliedPresetApplication) {
			return EMPTY_ASSISTANT_PRESET_MODIFICATION_SUMMARY;
		}

		return getAssistantPresetModificationSummary({
			preparedApplication: appliedPresetApplication,
			currentSelectedModel: context.selectedModel,
			currentIncludeModelSystemPrompt: context.includeModelDefault,
			currentSelectedPromptKeys: context.selectedPromptKeys,
			currentRuntimeSnapshot: runtimeSnapshot,
		});
	}, [
		appliedPresetApplication,
		context.includeModelDefault,
		context.selectedModel,
		context.selectedPromptKeys,
		runtimeSnapshot,
	]);

	const applyPresetByKey = useCallback(
		async (presetKey: string): Promise<boolean> => {
			const requestSeq = applyRequestSeqRef.current + 1;
			applyRequestSeqRef.current = requestSeq;
			setIsApplying(true);
			setSelectionState(current =>
				current.actionError === null
					? current
					: {
							...current,
							actionError: null,
						}
			);
			try {
				const prepared = await context.prepareAssistantPresetApplication(presetKey);
				if (applyRequestSeqRef.current !== requestSeq) {
					return false;
				}
				if (!prepared) {
					setSelectionState(current => ({
						...current,
						actionError: 'Assistant preset not found.',
					}));
					return false;
				}

				context.applyPreparedAssistantPreset(prepared);
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
				setSelectionState(current => ({
					...current,
					actionError: getErrorMessage(error, 'Failed to apply assistant preset.'),
				}));
				return false;
			} finally {
				if (applyRequestSeqRef.current === requestSeq) {
					setIsApplying(false);
				}
			}
		},
		[applyRuntimeSelections, context]
	);

	const reapplySelectedPreset = useCallback(async (): Promise<boolean> => {
		const presetKey = selectedPresetKey ?? appliedPresetApplication?.presetKey;
		if (!presetKey) {
			return false;
		}

		return applyPresetByKey(presetKey);
	}, [appliedPresetApplication?.presetKey, applyPresetByKey, selectedPresetKey]);

	const clearSelectedPreset = useCallback(() => {
		invalidatePendingApply();
		setSelectionState({
			selectedPresetKey: null,
			appliedPresetApplication: null,
			actionError: null,
		});
	}, [invalidatePendingApply]);

	return {
		presetOptions: context.assistantPresetOptions,
		loading: context.assistantPresetsLoading,
		error: context.assistantPresetError,
		actionError,
		isApplying,

		selectedPresetKey,
		selectedPreset,
		appliedPresetApplication,
		runtimeSnapshot,
		modificationSummary,

		selectPreset: applyPresetByKey,
		reapplySelectedPreset,
		clearSelectedPreset,
	};
}
