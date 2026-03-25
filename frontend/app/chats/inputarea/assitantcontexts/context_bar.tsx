import { type SetStateAction, useCallback, useRef, useState } from 'react';

import { FiSliders } from 'react-icons/fi';

import { ReasoningType } from '@/spec/inference';
import { type UIChatOption } from '@/spec/modelpreset';

import { HoverTip } from '@/components/ariakit_hover_tip';

import { AdvancedParamsModal } from '@/chats/inputarea/assitantcontexts/advanced_params_modal';
import { AssistantPresetDropdown } from '@/chats/inputarea/assitantcontexts/assistant_preset_dropdown';
import type {
	AssistantPresetOptionItem,
	AssistantPresetPreparedApplication,
} from '@/chats/inputarea/assitantcontexts/assistant_preset_runtime';
import { AssistantPresetViewModal } from '@/chats/inputarea/assitantcontexts/assistant_preset_view_modal';
import { ModelDropdown } from '@/chats/inputarea/assitantcontexts/model_dropdown';
import { OutputVerbosityDropdown } from '@/chats/inputarea/assitantcontexts/output_verbosity_dropdown';
import { PreviousMessagesDropdown } from '@/chats/inputarea/assitantcontexts/previous_messages_dropdown';
import { HybridReasoningCheckbox } from '@/chats/inputarea/assitantcontexts/reasoning_hybrid_checkbox';
import { SingleReasoningDropdown } from '@/chats/inputarea/assitantcontexts/reasoning_levels_dropdown';
import { ReasoningTokensDropdown } from '@/chats/inputarea/assitantcontexts/reasoning_tokens_dropdown';
import { SystemPromptDropdown } from '@/chats/inputarea/assitantcontexts/system_prompt_dropdown';
import { TemperatureDropdown } from '@/chats/inputarea/assitantcontexts/temperature_dropdown';
import type { AssistantContextController } from '@/chats/inputarea/assitantcontexts/use_assistant_context_state';
import type { AssistantPresetManagerState } from '@/chats/inputarea/assitantcontexts/use_assistant_preset_manager';

type AssistantContextBarProps = {
	context: AssistantContextController;
	assistantPreset: AssistantPresetManagerState;
};

type ContextBarMenuKey = 'assistant' | 'model' | 'secondary' | 'verbosity' | 'system' | 'previous' | null;

type AssistantPresetViewState = {
	option: AssistantPresetOptionItem | null;
	preparedApplication: AssistantPresetPreparedApplication | null;
	isActivePreset: boolean;
};

export function AssistantContextBar({ context, assistantPreset }: AssistantContextBarProps) {
	const [openMenu, setOpenMenu] = useState<ContextBarMenuKey>(null);
	const [isAdvancedModalOpen, setIsAdvancedModalOpen] = useState(false);
	const [isAssistantViewModalOpen, setIsAssistantViewModalOpen] = useState(false);

	const setMenuOpen = useCallback((menuKey: Exclude<ContextBarMenuKey, null>, action: SetStateAction<boolean>) => {
		setOpenMenu(prevOpenMenu => {
			const currentIsOpen = prevOpenMenu === menuKey;
			const nextIsOpen = typeof action === 'function' ? action(currentIsOpen) : action;

			if (nextIsOpen) return menuKey;
			return currentIsOpen ? null : prevOpenMenu;
		});
	}, []);

	const isAssistantDropdownOpen = openMenu === 'assistant';
	const isModelDropdownOpen = openMenu === 'model';
	const isSecondaryDropdownOpen = openMenu === 'secondary';
	const isVerbosityDropdownOpen = openMenu === 'verbosity';
	const isSystemDropdownOpen = openMenu === 'system';
	const isPreviousMessagesDropdownOpen = openMenu === 'previous';

	const setIsAssistantDropdownOpen = useCallback(
		(action: SetStateAction<boolean>) => {
			setMenuOpen('assistant', action);
		},
		[setMenuOpen]
	);

	const setIsModelDropdownOpen = useCallback(
		(action: SetStateAction<boolean>) => {
			setMenuOpen('model', action);
		},
		[setMenuOpen]
	);
	const setIsSecondaryDropdownOpen = useCallback(
		(action: SetStateAction<boolean>) => {
			setMenuOpen('secondary', action);
		},
		[setMenuOpen]
	);
	const setIsVerbosityDropdownOpen = useCallback(
		(action: SetStateAction<boolean>) => {
			setMenuOpen('verbosity', action);
		},
		[setMenuOpen]
	);
	const setIsSystemDropdownOpen = useCallback(
		(action: SetStateAction<boolean>) => {
			setMenuOpen('system', action);
		},
		[setMenuOpen]
	);
	const setIsPreviousMessagesDropdownOpen = useCallback(
		(action: SetStateAction<boolean>) => {
			setMenuOpen('previous', action);
		},
		[setMenuOpen]
	);

	const [assistantPresetViewState, setAssistantPresetViewState] = useState<AssistantPresetViewState>({
		option: null,
		preparedApplication: null,
		isActivePreset: false,
	});
	const assistantPresetViewRequestSeqRef = useRef(0);

	const openAssistantPresetView = useCallback(
		(option: AssistantPresetOptionItem) => {
			setOpenMenu(null);

			const activePresetKey =
				assistantPreset.selectedPresetKey ?? assistantPreset.appliedPresetApplication?.presetKey ?? null;
			const isActivePreset = activePresetKey === option.key;
			const initialPreparedApplication =
				isActivePreset && assistantPreset.appliedPresetApplication?.presetKey === option.key
					? assistantPreset.appliedPresetApplication
					: null;

			setAssistantPresetViewState({
				option,
				preparedApplication: initialPreparedApplication,
				isActivePreset,
			});
			setIsAssistantViewModalOpen(true);

			if (initialPreparedApplication || !option.isSelectable) {
				assistantPresetViewRequestSeqRef.current += 1;
				return;
			}

			const requestSeq = assistantPresetViewRequestSeqRef.current + 1;
			assistantPresetViewRequestSeqRef.current = requestSeq;

			void (async () => {
				try {
					const prepared = await context.prepareAssistantPresetApplication(option.key);
					if (!prepared || assistantPresetViewRequestSeqRef.current !== requestSeq) return;
					setAssistantPresetViewState(current =>
						current.option?.key === option.key ? { ...current, preparedApplication: prepared } : current
					);
				} catch (error) {
					console.error('Failed to prepare assistant preset preview:', error);
				}
			})();
		},
		[assistantPreset.appliedPresetApplication, assistantPreset.selectedPresetKey, context]
	);

	return (
		<div className="bg-base-200 mx-2 my-0 flex items-center justify-between gap-2 xl:mx-4">
			<AssistantPresetDropdown
				presetOptions={assistantPreset.presetOptions}
				selectedPresetKey={assistantPreset.selectedPresetKey}
				selectedPreset={assistantPreset.selectedPreset}
				loading={assistantPreset.loading}
				error={assistantPreset.error}
				actionError={assistantPreset.actionError}
				isApplying={assistantPreset.isApplying}
				basePresetKey={assistantPreset.basePresetKey}
				selectedPresetModifiedLabels={assistantPreset.modificationSummary.modifiedLabels}
				canResetToBasePreset={
					assistantPreset.presetOptions.some(option => option.isSelectable) && !assistantPreset.isBasePresetSelected
				}
				isOpen={isAssistantDropdownOpen}
				setIsOpen={setIsAssistantDropdownOpen}
				onViewPreset={openAssistantPresetView}
				onReapplySelectedPreset={() => {
					setOpenMenu(null);
					return assistantPreset.reapplySelectedPreset();
				}}
				onResetToBasePreset={() => {
					setOpenMenu(null);
					return assistantPreset.resetToBasePreset();
				}}
				onSelectPreset={assistantPreset.selectPreset}
			/>

			<ModelDropdown
				selectedModel={context.selectedModel}
				setSelectedModel={context.handleSetSelectedModel}
				allOptions={context.allOptions}
				isOpen={isModelDropdownOpen}
				setIsOpen={setIsModelDropdownOpen}
			/>

			{context.selectedModel.reasoning?.type === ReasoningType.HybridWithTokens && (
				<HybridReasoningCheckbox
					isReasoningEnabled={context.isHybridReasoningEnabled}
					setIsReasoningEnabled={context.handleSetIsHybridReasoningEnabled}
				/>
			)}

			{context.selectedModel.reasoning?.type === ReasoningType.HybridWithTokens ? (
				context.isHybridReasoningEnabled ? (
					<ReasoningTokensDropdown
						tokens={context.selectedModel.reasoning.tokens}
						setTokens={context.setHybridTokens}
						isOpen={isSecondaryDropdownOpen}
						setIsOpen={setIsSecondaryDropdownOpen}
					/>
				) : (
					<TemperatureDropdown
						temperature={context.selectedModel.temperature ?? 0.1}
						setTemperature={context.setTemperature}
						isOpen={isSecondaryDropdownOpen}
						setIsOpen={setIsSecondaryDropdownOpen}
					/>
				)
			) : context.selectedModel.reasoning?.type === ReasoningType.SingleWithLevels ? (
				<SingleReasoningDropdown
					reasoningLevel={context.selectedModel.reasoning.level}
					setReasoningLevel={context.setReasoningLevel}
					levelOptions={context.reasoningLevelOptions}
					isOpen={isSecondaryDropdownOpen}
					setIsOpen={setIsSecondaryDropdownOpen}
				/>
			) : (
				<TemperatureDropdown
					temperature={context.selectedModel.temperature ?? 0.1}
					setTemperature={context.setTemperature}
					isOpen={isSecondaryDropdownOpen}
					setIsOpen={setIsSecondaryDropdownOpen}
				/>
			)}

			{context.verbosityEnabled && (
				<OutputVerbosityDropdown
					verbosity={context.selectedModel.outputParam?.verbosity}
					setVerbosity={context.setOutputVerbosity}
					disabled={!context.verbosityEnabled}
					isOpen={isVerbosityDropdownOpen}
					setIsOpen={setIsVerbosityDropdownOpen}
				/>
			)}

			<SystemPromptDropdown
				prompts={context.prompts}
				bundles={context.systemPromptBundles}
				selectedPromptKeys={context.selectedPromptKeys}
				preferredBundleID={context.preferredSystemPromptBundleID}
				loading={context.systemPromptsLoading}
				error={context.systemPromptError}
				modelDefaultPrompt={context.selectedModel.systemPrompt}
				includeModelDefault={context.includeModelDefault}
				onTogglePrompt={context.togglePromptSelection}
				onToggleModelDefault={next => {
					context.setIncludeModelDefault(next);
				}}
				onAddPrompt={context.addAndSelectPrompt}
				onClearSelected={context.clearSelectedPromptSources}
				onRefreshPrompts={context.refreshSystemPrompts}
				getExistingVersions={context.getExistingSystemPromptVersions}
				isOpen={isSystemDropdownOpen}
				setIsOpen={setIsSystemDropdownOpen}
			/>

			<PreviousMessagesDropdown
				value={context.includePreviousMessages}
				setValue={context.setIncludePreviousMessages}
				isOpen={isPreviousMessagesDropdownOpen}
				setIsOpen={setIsPreviousMessagesDropdownOpen}
			/>

			<div className="flex items-center justify-center">
				<HoverTip
					content="Advanced parameters: streaming, token limits, output format, stop sequences, raw JSON"
					placement="left"
				>
					<button
						type="button"
						className="btn btn-xs btn-ghost text-neutral-custom m-1"
						onClick={() => {
							setOpenMenu(null);
							setIsAdvancedModalOpen(true);
						}}
					>
						<FiSliders size={14} />
					</button>
				</HoverTip>
			</div>

			<AdvancedParamsModal
				isOpen={isAdvancedModalOpen}
				onClose={() => {
					setIsAdvancedModalOpen(false);
				}}
				currentModel={context.selectedModel}
				effectiveReasoningEnabled={
					context.selectedModel.reasoning?.type === ReasoningType.HybridWithTokens
						? context.isHybridReasoningEnabled
						: !!context.selectedModel.reasoning
				}
				onSave={(updatedModel: UIChatOption) => {
					context.applyAdvancedModel(updatedModel);
				}}
			/>

			<AssistantPresetViewModal
				isOpen={isAssistantViewModalOpen && assistantPresetViewState.option !== null}
				onClose={() => {
					assistantPresetViewRequestSeqRef.current += 1;
					setIsAssistantViewModalOpen(false);
					setAssistantPresetViewState({
						option: null,
						preparedApplication: null,
						isActivePreset: false,
					});
				}}
				viewedPreset={assistantPresetViewState.option}
				viewedPresetApplication={assistantPresetViewState.preparedApplication}
				isActivePresetView={assistantPresetViewState.isActivePreset}
				currentRuntimeSnapshot={assistantPreset.runtimeSnapshot}
				currentModel={context.selectedModel}
				currentIncludeModelSystemPrompt={context.includeModelDefault}
				currentSelectedPromptKeys={context.selectedPromptKeys}
				promptItems={context.prompts}
				modificationSummary={assistantPreset.modificationSummary}
			/>
		</div>
	);
}
