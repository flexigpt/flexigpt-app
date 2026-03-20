import { type SetStateAction, useCallback, useState } from 'react';

import { FiSliders } from 'react-icons/fi';

import { ReasoningType } from '@/spec/inference';
import { type UIChatOption } from '@/spec/modelpreset';

import { AdvancedParamsModal } from '@/chats/inputarea/assitantcontexts/advanced_params_modal';
import { ModelDropdown } from '@/chats/inputarea/assitantcontexts/model_dropdown';
import { OutputVerbosityDropdown } from '@/chats/inputarea/assitantcontexts/output_verbosity_dropdown';
import { PreviousMessagesDropdown } from '@/chats/inputarea/assitantcontexts/previous_messages_dropdown';
import { HybridReasoningCheckbox } from '@/chats/inputarea/assitantcontexts/reasoning_hybrid_checkbox';
import { SingleReasoningDropdown } from '@/chats/inputarea/assitantcontexts/reasoning_levels_dropdown';
import { ReasoningTokensDropdown } from '@/chats/inputarea/assitantcontexts/reasoning_tokens_dropdown';
import { SystemPromptDropdown } from '@/chats/inputarea/assitantcontexts/system_prompt_dropdown';
import { TemperatureDropdown } from '@/chats/inputarea/assitantcontexts/temperature_dropdown';
import type { AssistantContextController } from '@/chats/inputarea/assitantcontexts/use_assistant_context_state';

type AssistantContextBarProps = {
	context: AssistantContextController;
};

type ContextBarMenuKey = 'model' | 'secondary' | 'verbosity' | 'system' | 'previous' | null;

export function AssistantContextBar({ context }: AssistantContextBarProps) {
	const [openMenu, setOpenMenu] = useState<ContextBarMenuKey>(null);
	const [isAdvancedModalOpen, setIsAdvancedModalOpen] = useState(false);

	const setMenuOpen = useCallback((menuKey: Exclude<ContextBarMenuKey, null>, action: SetStateAction<boolean>) => {
		setOpenMenu(prevOpenMenu => {
			const currentIsOpen = prevOpenMenu === menuKey;
			const nextIsOpen = typeof action === 'function' ? action(currentIsOpen) : action;

			if (nextIsOpen) return menuKey;
			return currentIsOpen ? null : prevOpenMenu;
		});
	}, []);

	const isModelDropdownOpen = openMenu === 'model';
	const isSecondaryDropdownOpen = openMenu === 'secondary';
	const isVerbosityDropdownOpen = openMenu === 'verbosity';
	const isSystemDropdownOpen = openMenu === 'system';
	const isPreviousMessagesDropdownOpen = openMenu === 'previous';

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
		<div className="bg-base-200 mx-4 my-0 flex items-center justify-between space-x-1">
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
		</div>
	);
}
