import { type SetStateAction, useCallback, useState } from 'react';

import { FiRefreshCcw, FiSliders } from 'react-icons/fi';

import { ReasoningType } from '@/spec/inference';
import { type UIChatOption } from '@/spec/modelpreset';

import { AdvancedParamsModal } from '@/chats/inputarea/assitantcontexts/advanced_params_modal';
import { AssistantPresetDropdown } from '@/chats/inputarea/assitantcontexts/assistant_preset_dropdown';
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
				isOpen={isAssistantDropdownOpen}
				setIsOpen={setIsAssistantDropdownOpen}
				onSelectPreset={assistantPreset.selectPreset}
				onClearPreset={assistantPreset.clearSelectedPreset}
			/>

			{assistantPreset.selectedPreset ? (
				<div className="flex items-center gap-1">
					<button
						type="button"
						className="btn btn-xs btn-ghost text-neutral-custom"
						onClick={() => {
							setOpenMenu(null);
							setIsAssistantViewModalOpen(true);
						}}
						title="View assistant preset details"
					>
						View
					</button>

					<div
						className="tooltip tooltip-top"
						data-tip={
							assistantPreset.modificationSummary.any
								? `Reapply preset-managed sections: ${assistantPreset.modificationSummary.modifiedLabels.join(', ')}`
								: 'Preset-managed sections are already in sync'
						}
					>
						<button
							type="button"
							className="btn btn-xs btn-ghost text-neutral-custom"
							disabled={!assistantPreset.modificationSummary.any || assistantPreset.isApplying}
							onClick={() => {
								setOpenMenu(null);
								void assistantPreset.reapplySelectedPreset();
							}}
							title="Reapply current assistant preset"
						>
							<FiRefreshCcw size={12} />
						</button>
					</div>

					{assistantPreset.modificationSummary.any ? (
						<span
							className="badge badge-warning badge-xs"
							title={`Modified sections: ${assistantPreset.modificationSummary.modifiedLabels.join(', ')}`}
						>
							Modified
						</span>
					) : null}

					{assistantPreset.actionError ? (
						<span className="badge badge-error badge-xs" title={assistantPreset.actionError}>
							Error
						</span>
					) : null}
				</div>
			) : null}

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
				<div
					className="tooltip tooltip-left"
					data-tip="Advanced parameters (streaming, token limits, output format, stop sequences, raw JSON)"
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
				</div>
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
				isOpen={isAssistantViewModalOpen && assistantPreset.appliedPresetApplication !== null}
				onClose={() => {
					setIsAssistantViewModalOpen(false);
				}}
				appliedPresetApplication={assistantPreset.appliedPresetApplication}
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
