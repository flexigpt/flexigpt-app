import { type SetStateAction, useCallback, useEffect, useState } from 'react';

import { FiSliders } from 'react-icons/fi';

import { type OutputVerbosity, type ReasoningLevel, ReasoningType } from '@/spec/inference';
import { DefaultUIChatOptions, type UIChatOption } from '@/spec/modelpreset';

import { AdvancedParamsModal } from '@/chats/assitantcontexts/advanced_params_modal';
import {
	getSupportedReasoningLevels,
	sanitizeUIChatOptionByCapabilities,
	supportsOutputVerbosity,
} from '@/chats/assitantcontexts/capabilities_override_helper';
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

function isHybridReasoningModel(model: UIChatOption): boolean {
	return model.reasoning?.type === ReasoningType.HybridWithTokens;
}

function buildFinalOptions(
	selectedModel: UIChatOption,
	disablePreviousMessages: boolean,
	isHybridReasoningEnabled: boolean
): UIChatOption {
	const base = { ...selectedModel, disablePreviousMessages };

	if (selectedModel.reasoning?.type === ReasoningType.HybridWithTokens && !isHybridReasoningEnabled) {
		const modifiedOptions = { ...base };
		delete modifiedOptions.reasoning;

		if (modifiedOptions.temperature === undefined) {
			modifiedOptions.temperature = DefaultUIChatOptions.temperature;
		}

		return sanitizeUIChatOptionByCapabilities(modifiedOptions);
	}

	return sanitizeUIChatOptionByCapabilities(base);
}

function upsertSystemPromptItem(
	items: SystemPromptItem[],
	prompt: string,
	options?: { locked?: boolean }
): SystemPromptItem[] {
	const trimmedPrompt = prompt.trim();
	if (!trimmedPrompt) return items;
	if (items.some(i => i.prompt === trimmedPrompt)) return items;

	return [...items, createSystemPromptItem(trimmedPrompt, options?.locked ? { locked: true } : undefined)];
}

export function AssistantContextBar({ onOptionsChange }: AssistantContextBarProps) {
	const [selectedModel, setSelectedModel] = useState<UIChatOption>(DefaultUIChatOptions);
	const [allOptions, setAllOptions] = useState<UIChatOption[]>([DefaultUIChatOptions]);

	const [isHybridReasoningEnabled, setIsHybridReasoningEnabled] = useState(true);
	const [disablePreviousMessages, setDisablePreviousMessages] = useState(false);
	const [systemPrompts, setSystemPrompts] = useState<SystemPromptItem[]>([]);

	const [isModelDropdownOpen, setIsModelDropdownOpen] = useState(false);
	const [isSecondaryDropdownOpen, setIsSecondaryDropdownOpen] = useState(false);
	const [isVerbosityDropdownOpen, setIsVerbosityDropdownOpen] = useState(false);
	const [isSystemDropdownOpen, setIsSystemDropdownOpen] = useState(false);
	const [isAdvancedModalOpen, setIsAdvancedModalOpen] = useState(false);

	const emitOptionsChange = useCallback(
		(nextSelectedModel: UIChatOption, nextDisablePreviousMessages: boolean, nextIsHybridReasoningEnabled: boolean) => {
			onOptionsChange(buildFinalOptions(nextSelectedModel, nextDisablePreviousMessages, nextIsHybridReasoningEnabled));
		},
		[onOptionsChange]
	);

	const addSystemPromptIfMissing = useCallback((prompt: string, options?: { locked?: boolean }) => {
		setSystemPrompts(prev => upsertSystemPromptItem(prev, prompt, options));
	}, []);

	const applySelectedModel = useCallback(
		(
			action: SetStateAction<UIChatOption>,
			options?: {
				syncHybridFromModel?: boolean;
				lockSystemPrompt?: boolean;
			}
		) => {
			const nextSelectedModel =
				typeof action === 'function' ? (action as (prevState: UIChatOption) => UIChatOption)(selectedModel) : action;

			const nextIsHybridReasoningEnabled = options?.syncHybridFromModel
				? isHybridReasoningModel(nextSelectedModel)
				: isHybridReasoningEnabled;

			setSelectedModel(nextSelectedModel);

			if (nextIsHybridReasoningEnabled !== isHybridReasoningEnabled) {
				setIsHybridReasoningEnabled(nextIsHybridReasoningEnabled);
			}

			addSystemPromptIfMissing(nextSelectedModel.systemPrompt, {
				locked: options?.lockSystemPrompt,
			});

			emitOptionsChange(nextSelectedModel, disablePreviousMessages, nextIsHybridReasoningEnabled);
		},
		[selectedModel, isHybridReasoningEnabled, disablePreviousMessages, addSystemPromptIfMissing, emitOptionsChange]
	);

	useEffect(() => {
		let cancelled = false;

		const initialHybridEnabled = isHybridReasoningModel(DefaultUIChatOptions);
		emitOptionsChange(DefaultUIChatOptions, false, initialHybridEnabled);

		void (async () => {
			const r = await getChatInputOptions();
			if (cancelled) return;

			const nextSelectedModel = sanitizeUIChatOptionByCapabilities(r.default);
			const nextIsHybridReasoningEnabled = isHybridReasoningModel(nextSelectedModel);

			setSelectedModel(nextSelectedModel);
			setAllOptions(r.allOptions);
			setIsHybridReasoningEnabled(nextIsHybridReasoningEnabled);

			addSystemPromptIfMissing(nextSelectedModel.systemPrompt, { locked: true });
			emitOptionsChange(nextSelectedModel, false, nextIsHybridReasoningEnabled);
		})();

		return () => {
			cancelled = true;
		};
	}, [addSystemPromptIfMissing, emitOptionsChange]);

	const handleSetSelectedModel = useCallback(
		(action: SetStateAction<UIChatOption>) => {
			applySelectedModel(action, { syncHybridFromModel: true });
		},
		[applySelectedModel]
	);

	const handleSetDisablePreviousMessages = useCallback(
		(action: SetStateAction<boolean>) => {
			const nextDisablePreviousMessages = typeof action === 'function' ? action(disablePreviousMessages) : action;

			setDisablePreviousMessages(nextDisablePreviousMessages);
			emitOptionsChange(selectedModel, nextDisablePreviousMessages, isHybridReasoningEnabled);
		},
		[disablePreviousMessages, emitOptionsChange, isHybridReasoningEnabled, selectedModel]
	);

	const handleSetIsHybridReasoningEnabled = useCallback(
		(action: SetStateAction<boolean>) => {
			const nextIsHybridReasoningEnabled = typeof action === 'function' ? action(isHybridReasoningEnabled) : action;

			setIsHybridReasoningEnabled(nextIsHybridReasoningEnabled);
			emitOptionsChange(selectedModel, disablePreviousMessages, nextIsHybridReasoningEnabled);
		},
		[disablePreviousMessages, emitOptionsChange, isHybridReasoningEnabled, selectedModel]
	);

	const setTemperature = useCallback(
		(temp: number) => {
			const clampedTemp = Math.max(0, Math.min(1, temp));
			applySelectedModel(prev => ({ ...prev, temperature: clampedTemp }));
		},
		[applySelectedModel]
	);

	const setReasoningLevel = useCallback(
		(newLevel: ReasoningLevel) => {
			applySelectedModel(prev => ({
				...prev,
				reasoning: {
					type: ReasoningType.SingleWithLevels,
					level: newLevel,
					tokens: 1024,
					summaryStyle: prev.reasoning?.summaryStyle,
				},
			}));
		},
		[applySelectedModel]
	);

	const setHybridTokens = useCallback(
		(tokens: number) => {
			applySelectedModel(prev => {
				if (!prev.reasoning || prev.reasoning.type !== ReasoningType.HybridWithTokens) return prev;
				return { ...prev, reasoning: { ...prev.reasoning, tokens } };
			});
		},
		[applySelectedModel]
	);

	const setOutputVerbosity = useCallback(
		(verbosity?: OutputVerbosity) => {
			applySelectedModel(prev => {
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
		},
		[applySelectedModel]
	);

	const verbosityEnabled = supportsOutputVerbosity(selectedModel.capabilitiesOverride);

	const selectSystemPrompt = useCallback(
		(item: SystemPromptItem) => {
			applySelectedModel(prev => ({ ...prev, systemPrompt: item.prompt }));
		},
		[applySelectedModel]
	);

	const clearSystemPrompt = useCallback(() => {
		applySelectedModel(prev => ({ ...prev, systemPrompt: '' }));
	}, [applySelectedModel]);

	const editSystemPrompt = useCallback(
		(id: string, updatedPrompt: string) => {
			const updatedText = updatedPrompt.trim();
			if (!updatedText) return;

			const oldItem = systemPrompts.find(i => i.id === id);
			if (!oldItem) return;

			setSystemPrompts(prev =>
				prev.map(i =>
					i.id === id
						? {
								...i,
								prompt: updatedText,
								title: updatedText.length > 24 ? `${updatedText.slice(0, 24)}…` : updatedText || '(empty)',
							}
						: i
				)
			);

			if ((selectedModel.systemPrompt || '').trim() === (oldItem.prompt || '').trim()) {
				applySelectedModel(prev => ({ ...prev, systemPrompt: updatedText }));
			}
		},
		[applySelectedModel, selectedModel.systemPrompt, systemPrompts]
	);

	const addSystemPrompt = useCallback(
		(item: SystemPromptItem) => {
			const p = item.prompt.trim();
			if (!p) return;

			setSystemPrompts(prev => (prev.some(i => i.prompt === p) ? prev : [...prev, item]));
			applySelectedModel(prev => ({ ...prev, systemPrompt: p }));
		},
		[applySelectedModel]
	);

	const removeSystemPrompt = useCallback((id: string) => {
		setSystemPrompts(prev => {
			const target = prev.find(i => i.id === id);
			if (target?.locked) return prev;
			return prev.filter(i => i.id !== id);
		});
	}, []);

	const handleSetSystemPromptForChat = useCallback(
		(prompt: string) => {
			const p = (prompt || '').trim();
			if (!p) return;

			addSystemPromptIfMissing(p);
			applySelectedModel(prev => ({ ...prev, systemPrompt: p }));
		},
		[addSystemPromptIfMissing, applySelectedModel]
	);

	useSetSystemPromptForChat(handleSetSystemPromptForChat);

	const selectedPromptId = systemPrompts.find(i => i.prompt === (selectedModel.systemPrompt.trim() || ''))?.id;

	return (
		<div className="bg-base-200 mx-4 my-0 flex items-center justify-between space-x-1">
			<ModelDropdown
				selectedModel={selectedModel}
				setSelectedModel={handleSetSelectedModel}
				allOptions={allOptions}
				isOpen={isModelDropdownOpen}
				setIsOpen={setIsModelDropdownOpen}
			/>

			{selectedModel.reasoning?.type === ReasoningType.HybridWithTokens && (
				<HybridReasoningCheckbox
					isReasoningEnabled={isHybridReasoningEnabled}
					setIsReasoningEnabled={handleSetIsHybridReasoningEnabled}
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
					levelOptions={getSupportedReasoningLevels(selectedModel.capabilitiesOverride)}
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

			{verbosityEnabled && (
				<OutputVerbosityDropdown
					verbosity={selectedModel.outputParam?.verbosity}
					setVerbosity={setOutputVerbosity}
					disabled={!verbosityEnabled}
					isOpen={isVerbosityDropdownOpen}
					setIsOpen={setIsVerbosityDropdownOpen}
				/>
			)}

			<SystemPromptDropdown
				prompts={systemPrompts}
				selectedPromptId={selectedPromptId}
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
				setDisablePreviousMessages={handleSetDisablePreviousMessages}
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
				currentModel={selectedModel}
				onSave={(updatedModel: UIChatOption) => {
					applySelectedModel(sanitizeUIChatOptionByCapabilities(updatedModel));
				}}
			/>
		</div>
	);
}
