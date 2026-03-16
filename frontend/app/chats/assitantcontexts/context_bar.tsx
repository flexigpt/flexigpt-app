import { useState } from 'react';

import { FiSliders } from 'react-icons/fi';

import { ReasoningType } from '@/spec/inference';
import { type UIChatOption } from '@/spec/modelpreset';

import { AdvancedParamsModal } from '@/chats/assitantcontexts/advanced_params_modal';
import { ModelDropdown } from '@/chats/assitantcontexts/model_dropdown';
import { OutputVerbosityDropdown } from '@/chats/assitantcontexts/output_verbosity_dropdown';
import { PreviousMessagesDropdown } from '@/chats/assitantcontexts/previous_messages_dropdown';
import { HybridReasoningCheckbox } from '@/chats/assitantcontexts/reasoning_hybrid_checkbox';
import { SingleReasoningDropdown } from '@/chats/assitantcontexts/reasoning_levels_dropdown';
import { ReasoningTokensDropdown } from '@/chats/assitantcontexts/reasoning_tokens_dropdown';
import { SystemPromptDropdown } from '@/chats/assitantcontexts/system_prompt_dropdown';
import { TemperatureDropdown } from '@/chats/assitantcontexts/temperature_dropdown';
import type { AssistantContextController } from '@/chats/assitantcontexts/use_assistant_context_state';

type AssistantContextBarProps = {
	context: AssistantContextController;
};

export function AssistantContextBar({ context }: AssistantContextBarProps) {
	const [isModelDropdownOpen, setIsModelDropdownOpen] = useState(false);
	const [isSecondaryDropdownOpen, setIsSecondaryDropdownOpen] = useState(false);
	const [isVerbosityDropdownOpen, setIsVerbosityDropdownOpen] = useState(false);
	const [isSystemDropdownOpen, setIsSystemDropdownOpen] = useState(false);
	const [isPreviousMessagesDropdownOpen, setIsPreviousMessagesDropdownOpen] = useState(false);
	const [isAdvancedModalOpen, setIsAdvancedModalOpen] = useState(false);

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
				selectedPromptIds={context.selectedPromptIds}
				modelDefaultPrompt={context.selectedModel.systemPrompt}
				includeModelDefault={context.includeModelDefault}
				onTogglePrompt={context.togglePromptSelection}
				onToggleModelDefault={next => {
					context.setIncludeModelDefault(next);
				}}
				onAddPrompt={context.addAndSelectPrompt}
				onDeletePrompt={context.removeSavedPrompt}
				onClearSelected={context.clearSelectedPromptSources}
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
				onSave={(updatedModel: UIChatOption) => {
					context.applyAdvancedModel(updatedModel);
				}}
			/>
		</div>
	);
}
