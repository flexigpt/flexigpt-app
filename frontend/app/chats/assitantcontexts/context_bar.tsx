import { useCallback, useEffect, useState } from 'react';

import { FiSliders } from 'react-icons/fi';

import { type OutputVerbosity, type ReasoningLevel, ReasoningType } from '@/spec/inference';
import { DefaultUIChatOptions, type UIChatOption } from '@/spec/modelpreset';

import { AdvancedParamsModal } from '@/chats/assitantcontexts/advanced_params_modal';
import { getChatInputOptions } from '@/chats/assitantcontexts/context_uichatoption_helper';
import { DisablePreviousMessagesCheckbox } from '@/chats/assitantcontexts/disable_checkbox';
import { ModelDropdown } from '@/chats/assitantcontexts/model_dropdown';
import { OutputVerbosityDropdown } from '@/chats/assitantcontexts/output_verbosity_dropdown';
import { HybridReasoningCheckbox } from '@/chats/assitantcontexts/reasoning_hybrid_checkbox';
import { SingleReasoningDropdown } from '@/chats/assitantcontexts/reasoning_levels_dropdown';
import { ReasoningTokensDropdown } from '@/chats/assitantcontexts/reasoning_tokens_dropdown';
import {
	createSystemPromptItem,
	SystemPromptDropdown,
	type SystemPromptItem,
} from '@/chats/assitantcontexts/system_prompt_dropdown';
import { TemperatureDropdown } from '@/chats/assitantcontexts/temperature_dropdown';
import { useSetSystemPromptForChat } from '@/chats/events/set_system_prompt';

type AssistantContextBarProps = {
	onOptionsChange: (options: UIChatOption) => void;
};

export function AssistantContextBar({ onOptionsChange }: AssistantContextBarProps) {
	const [selectedModel, setSelectedModel] = useState<UIChatOption>(DefaultUIChatOptions);
	const [allOptions, setAllOptions] = useState<UIChatOption[]>([DefaultUIChatOptions]);

	const [isHybridReasoningEnabled, setIsHybridReasoningEnabled] = useState(true);
	const [disablePreviousMessages, setDisablePreviousMessages] = useState(false);
	const [systemPrompts, setSystemPrompts] = useState<SystemPromptItem[]>([]);

	const loadInitialItems = useCallback(async () => {
		const r = await getChatInputOptions();
		setSelectedModel(r.default);
		setAllOptions(r.allOptions);
		// Seed system prompts with current default if available
		const initialSP = r.default.systemPrompt.trim();
		if (initialSP) {
			setSystemPrompts(prev =>
				prev.some(i => i.prompt === initialSP) ? prev : [...prev, createSystemPromptItem(initialSP, { locked: true })]
			);
		}
	}, []);

	useEffect(() => {
		loadInitialItems();
	}, [loadInitialItems]);

	useEffect(() => {
		setIsHybridReasoningEnabled(selectedModel.reasoning?.type === ReasoningType.HybridWithTokens);
	}, [selectedModel.reasoning?.type]);

	useEffect(() => {
		const sp = selectedModel.systemPrompt.trim();
		if (sp) {
			setSystemPrompts(prev => (prev.some(i => i.prompt === sp) ? prev : [...prev, createSystemPromptItem(sp)]));
		}
	}, [selectedModel.systemPrompt]);

	const buildFinalOptions = useCallback((): UIChatOption => {
		const base = { ...selectedModel, disablePreviousMessages };

		// remove reasoning when hybrid reasoning is toggled off
		if (selectedModel.reasoning?.type === ReasoningType.HybridWithTokens && !isHybridReasoningEnabled) {
			const modifiedOptions = { ...base };
			delete modifiedOptions.reasoning;
			if (modifiedOptions.temperature === undefined) {
				modifiedOptions.temperature = DefaultUIChatOptions.temperature;
			}
			return modifiedOptions;
		}

		return base;
	}, [selectedModel, disablePreviousMessages, isHybridReasoningEnabled]);

	useEffect(() => {
		onOptionsChange(buildFinalOptions());
	}, [buildFinalOptions, onOptionsChange]);

	const [isModelDropdownOpen, setIsModelDropdownOpen] = useState(false);
	const [isSecondaryDropdownOpen, setIsSecondaryDropdownOpen] = useState(false);
	const [isVerbosityDropdownOpen, setIsVerbosityDropdownOpen] = useState(false);
	const [isSystemDropdownOpen, setIsSystemDropdownOpen] = useState(false);

	const [isAdvancedModalOpen, setIsAdvancedModalOpen] = useState(false);

	const setTemperature = useCallback((temp: number) => {
		const clampedTemp = Math.max(0, Math.min(1, temp));
		setSelectedModel(prev => ({ ...prev, temperature: clampedTemp }));
	}, []);

	const setReasoningLevel = useCallback((newLevel: ReasoningLevel) => {
		setSelectedModel(prev => ({
			...prev,
			reasoning: {
				type: ReasoningType.SingleWithLevels,
				level: newLevel,
				tokens: 1024,
				summaryStyle: prev.reasoning?.summaryStyle,
			},
		}));
	}, []);

	const setHybridTokens = useCallback((tokens: number) => {
		setSelectedModel(prev => {
			if (!prev.reasoning || prev.reasoning.type !== ReasoningType.HybridWithTokens) return prev;
			return { ...prev, reasoning: { ...prev.reasoning, tokens } };
		});
	}, []);

	const setOutputVerbosity = useCallback((verbosity?: OutputVerbosity) => {
		setSelectedModel(prev => {
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
	}, []);

	const selectSystemPrompt = useCallback((item: SystemPromptItem) => {
		setSelectedModel(prev => ({ ...prev, systemPrompt: item.prompt }));
	}, []);

	const clearSystemPrompt = useCallback(() => {
		setSelectedModel(prev => ({ ...prev, systemPrompt: '' }));
	}, []);

	const editSystemPrompt = useCallback(
		(id: string, updatedPrompt: string) => {
			const updatedText = updatedPrompt.trim();
			if (!updatedText) return;
			setSystemPrompts(prev => {
				const oldItem = prev.find(i => i.id === id);
				if (!oldItem) return prev;
				const updated = prev.map(i =>
					i.id === id
						? {
								...i,
								prompt: updatedText,
								title: updatedText.length > 24 ? `${updatedText.slice(0, 24)}…` : updatedText || '(empty)',
							}
						: i
				);
				if ((selectedModel.systemPrompt || '').trim() === (oldItem.prompt || '').trim()) {
					setSelectedModel(sel => ({ ...sel, systemPrompt: updatedText }));
				}
				return updated;
			});
		},
		[selectedModel.systemPrompt]
	);

	const addSystemPrompt = useCallback(
		(item: SystemPromptItem) => {
			const p = item.prompt.trim();
			if (!p) return;
			setSystemPrompts(prev => (prev.some(i => i.prompt === p) ? prev : [...prev, item]));
			setSelectedModel(prev => ({ ...prev, systemPrompt: p }));
		},
		[setSelectedModel]
	);

	const removeSystemPrompt = useCallback((id: string) => {
		setSystemPrompts(prev => {
			const target = prev.find(i => i.id === id);
			if (target?.locked) return prev; // don't remove locked default
			return prev.filter(i => i.id !== id);
		});
	}, []);

	useSetSystemPromptForChat(prompt => {
		const p = (prompt || '').trim();
		if (!p) return;
		setSystemPrompts(prev => (prev.some(i => i.prompt === p) ? prev : [...prev, createSystemPromptItem(p)]));
		setSelectedModel(prev => ({ ...prev, systemPrompt: p }));
	});

	return (
		<div className="bg-base-200 mx-4 my-0 flex items-center justify-between space-x-1">
			<ModelDropdown
				selectedModel={selectedModel}
				setSelectedModel={setSelectedModel}
				allOptions={allOptions}
				isOpen={isModelDropdownOpen}
				setIsOpen={setIsModelDropdownOpen}
			/>

			{selectedModel.reasoning?.type === ReasoningType.HybridWithTokens && (
				<HybridReasoningCheckbox
					isReasoningEnabled={isHybridReasoningEnabled}
					setIsReasoningEnabled={setIsHybridReasoningEnabled}
				/>
			)}

			{selectedModel.reasoning?.type === ReasoningType.HybridWithTokens ? (
				isHybridReasoningEnabled ? (
					<ReasoningTokensDropdown
						tokens={selectedModel.reasoning.tokens}
						setTokens={setHybridTokens}
						isOpen={isSecondaryDropdownOpen}
						setIsOpen={setIsSecondaryDropdownOpen}
					/>
				) : (
					<TemperatureDropdown
						temperature={selectedModel.temperature ?? 0.1}
						setTemperature={setTemperature}
						isOpen={isSecondaryDropdownOpen}
						setIsOpen={setIsSecondaryDropdownOpen}
					/>
				)
			) : selectedModel.reasoning?.type === ReasoningType.SingleWithLevels ? (
				<SingleReasoningDropdown
					reasoningLevel={selectedModel.reasoning.level}
					setReasoningLevel={setReasoningLevel}
					isOpen={isSecondaryDropdownOpen}
					setIsOpen={setIsSecondaryDropdownOpen}
				/>
			) : (
				<TemperatureDropdown
					temperature={selectedModel.temperature ?? 0.1}
					setTemperature={setTemperature}
					isOpen={isSecondaryDropdownOpen}
					setIsOpen={setIsSecondaryDropdownOpen}
				/>
			)}

			{/* Verbosity dropdown */}
			<OutputVerbosityDropdown
				verbosity={selectedModel.outputParam?.verbosity}
				setVerbosity={setOutputVerbosity}
				isOpen={isVerbosityDropdownOpen}
				setIsOpen={setIsVerbosityDropdownOpen}
			/>

			<SystemPromptDropdown
				prompts={systemPrompts}
				selectedPromptId={systemPrompts.find(i => i.prompt === (selectedModel.systemPrompt.trim() || ''))?.id}
				onSelect={selectSystemPrompt}
				onAdd={addSystemPrompt}
				onEdit={editSystemPrompt}
				onRemove={removeSystemPrompt}
				onClear={clearSystemPrompt}
				isOpen={isSystemDropdownOpen}
				setIsOpen={setIsSystemDropdownOpen}
			/>

			<DisablePreviousMessagesCheckbox
				disablePreviousMessages={disablePreviousMessages}
				setDisablePreviousMessages={setDisablePreviousMessages}
			/>

			{/* Advanced params button */}
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

			{/* Advanced params modal */}
			<AdvancedParamsModal
				isOpen={isAdvancedModalOpen}
				onClose={() => {
					setIsAdvancedModalOpen(false);
				}}
				currentModel={selectedModel}
				onSave={(updatedModel: UIChatOption) => {
					setSelectedModel(updatedModel);
				}}
			/>
		</div>
	);
}
