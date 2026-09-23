import { useState } from 'react';
import { FiSliders } from 'react-icons/fi';

import type { UIChatOption } from '@/spec/modelpreset';
import { ReasoningType } from '@/spec/inference';

import { actionTriggerChipButtonClasses, ActionTriggerChipContent } from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/hover_tip';

import type { AgentManagerState } from '@/chats/composer/agents/use_agent_manager';
import type { ComposerContextController } from '@/chats/composer/contextarea/use_context_state';
import { AdvancedParamsModal } from '@/chats/composer/advancedparams/advanced_params_modal';
import { AgentDropdown } from '@/chats/composer/agents/agent_dropdown';
import { ModelDropdown } from '@/chats/composer/models/model_dropdown';
import { OutputVerbosityDropdown } from '@/chats/composer/outputverbosities/output_verbosity_dropdown';
import { PreviousMessagesDropdown } from '@/chats/composer/previousmessages/previous_messages_dropdown';
import { HybridReasoningCheckbox } from '@/chats/composer/reasoningparams/reasoning_hybrid_checkbox';
import { SingleReasoningDropdown } from '@/chats/composer/reasoningparams/reasoning_levels_dropdown';
import { ReasoningTokensDropdown } from '@/chats/composer/reasoningparams/reasoning_tokens_dropdown';
import { TemperatureDropdown } from '@/chats/composer/temperatures/temperature_dropdown';

interface ContextBarProps {
	context: ComposerContextController;
	agent: AgentManagerState;
}

export function ContextBar({ context, agent }: ContextBarProps) {
	const [isAdvancedModalOpen, setIsAdvancedModalOpen] = useState(false);

	return (
		<div className="bg-base-200 mx-2 my-0 flex items-center justify-between gap-2 p-1 xl:mx-4">
			<AgentDropdown manager={agent} />

			<ModelDropdown
				selectedModel={context.selectedModel}
				setSelectedModel={context.handleSetSelectedModel}
				allOptions={context.allOptions}
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
						setTokens={t => {
							context.setHybridTokens(t);
						}}
					/>
				) : (
					<TemperatureDropdown
						temperature={context.selectedModel.temperature ?? 0.1}
						setTemperature={t => {
							context.setTemperature(t);
						}}
					/>
				)
			) : context.selectedModel.reasoning?.type === ReasoningType.SingleWithLevels ? (
				<SingleReasoningDropdown
					reasoningLevel={context.selectedModel.reasoning.level}
					setReasoningLevel={r => {
						context.setReasoningLevel(r);
					}}
					levelOptions={context.reasoningLevelOptions}
				/>
			) : (
				<TemperatureDropdown
					temperature={context.selectedModel.temperature ?? 0.1}
					setTemperature={t => {
						context.setTemperature(t);
					}}
				/>
			)}

			{context.verbosityEnabled ? (
				<OutputVerbosityDropdown
					sdkType={context.selectedModel.providerSDKType}
					verbosity={context.selectedModel.outputParam?.verbosity}
					setVerbosity={o => {
						context.setOutputVerbosity(o);
					}}
				/>
			) : null}

			<PreviousMessagesDropdown value={context.includePreviousMessages} setValue={context.setIncludePreviousMessages} />

			<div className="flex items-center justify-center">
				<HoverTip
					content="Advanced parameters: streaming, token limits, output format, stop sequences, raw JSON"
					placement="left"
				>
					<button
						type="button"
						className={`${actionTriggerChipButtonClasses} justify-center p-0`}
						onClick={() => {
							setIsAdvancedModalOpen(true);
						}}
					>
						<ActionTriggerChipContent
							icon={<FiSliders size={14} />}
							label=""
							showChevron={false}
							className="justify-center"
							labelClassName="truncate text-xs font-normal"
						/>
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
						: Boolean(context.selectedModel.reasoning)
				}
				onSave={(updatedModel: UIChatOption) => {
					context.applyAdvancedModel(updatedModel);
				}}
			/>
		</div>
	);
}
