import { useState } from 'react';
import { FiSettings, FiSliders } from 'react-icons/fi';

import { Link } from 'react-router';

import type { UIModelOption } from '@/spec/model';
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
import { getHybridReasoningPreference } from '@/models/lib/request_preferences';

interface ContextBarProps {
	context: ComposerContextController;
	agent: AgentManagerState;
}

function ModelSetupAction({
	unavailableReason,
	hasCatalogError,
}: {
	unavailableReason: ComposerContextController['modelCatalogUnavailableReason'];
	hasCatalogError: boolean;
}) {
	const noProvider = unavailableReason === 'no-runnable-provider';
	const noModel = unavailableReason === 'no-runnable-model';
	const destination = '/models/';
	const label = noProvider ? 'No providers configured' : noModel ? 'No enabled model configured' : 'Models unavailable';
	const description = noProvider
		? 'Configure an enabled provider API key in Models before sending a request.'
		: noModel
			? 'Enable or add a model in Models before sending a request.'
			: hasCatalogError
				? 'Open Models to review provider and model configuration.'
				: 'Configure a provider and model before sending a request.';

	return (
		<div className="flex w-full justify-center">
			<HoverTip content={description} placement="top" wrapperElement="div" wrapperClassName="w-full">
				<Link to={destination} className={`${actionTriggerChipButtonClasses} w-full justify-center`}>
					<ActionTriggerChipContent
						icon={<FiSettings size={14} />}
						label={label}
						showChevron={false}
						className="w-full justify-center"
						labelClassName="min-w-0 truncate text-center text-xs font-normal"
					/>
				</Link>
			</HoverTip>
		</div>
	);
}

export function ContextBar({ context, agent }: ContextBarProps) {
	const [isAdvancedModalOpen, setIsAdvancedModalOpen] = useState(false);
	const hybridPreference = getHybridReasoningPreference(context.selectedModel.requestPatch);

	if (!context.hasRunnableModel) {
		return (
			<div className="bg-base-200 mx-2 my-0 flex items-center justify-between gap-2 p-1 xl:mx-4">
				<ModelSetupAction
					unavailableReason={context.modelCatalogUnavailableReason}
					hasCatalogError={context.modelCatalogError !== null}
				/>
			</div>
		);
	}

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
					preference={hybridPreference}
					setIsReasoningEnabled={context.handleSetIsHybridReasoningEnabled}
				/>
			)}

			{context.selectedModel.reasoning?.type === ReasoningType.HybridWithTokens ? (
				context.isHybridReasoningEnabled ? (
					<ReasoningTokensDropdown
						tokens={context.selectedModel.requestPatch?.defaults?.reasoning?.tokens}
						setTokens={t => {
							context.setHybridTokens(t);
						}}
					/>
				) : (
					<TemperatureDropdown
						temperature={context.selectedModel.requestPatch?.defaults?.temperature}
						setTemperature={t => {
							context.setTemperature(t);
						}}
					/>
				)
			) : context.selectedModel.reasoning?.type === ReasoningType.SingleWithLevels ? (
				<SingleReasoningDropdown
					reasoningLevel={context.selectedModel.requestPatch?.defaults?.reasoning?.level || undefined}
					setReasoningLevel={r => {
						context.setReasoningLevel(r);
					}}
					levelOptions={context.reasoningLevelOptions}
				/>
			) : (
				<TemperatureDropdown
					temperature={context.selectedModel.requestPatch?.defaults?.temperature}
					setTemperature={t => {
						context.setTemperature(t);
					}}
				/>
			)}

			{context.verbosityEnabled ? (
				<OutputVerbosityDropdown
					sdkType={context.selectedModel.providerSDKType}
					verbosity={context.selectedModel.requestPatch?.defaults?.output?.verbosity}
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
				onSave={(updatedModel: UIModelOption) => {
					context.applyAdvancedModel(updatedModel);
				}}
			/>
		</div>
	);
}
