import type { Dispatch, SetStateAction, SubmitEventHandler } from 'react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { FiAlertCircle, FiHelpCircle } from 'react-icons/fi';

import type { CacheControlKind } from '@/spec/inference';
import type { ModelOutputFormat, UIModelOption } from '@/spec/model';
import { OutputFormatKind, ReasoningSummaryStyle } from '@/spec/inference';
import { ModelRequestClearField } from '@/spec/model';

import { focusTextInputAtEnd } from '@/lib/focus_input';
import { MAX_JSON_SCHEMA_INPUT_CHARS, tryParseJSONObject } from '@/lib/jsonschema_utils';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { Dropdown } from '@/components/dropdown';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

import type { CacheControlTTLSelection } from '@/models/lib/cache_control';
import {
	buildCacheControlFromForm,
	buildCacheControlKindDropdownItems,
	buildCacheControlTTLDropdownItems,
	getInitialCacheControlKind,
	getInitialCacheControlTTLSelection,
	resolveSupportedCacheControlKinds,
	resolveSupportedCacheControlTTLs,
} from '@/models/lib/cache_control';
import {
	getEffectiveCacheCapabilities,
	getStopSequencesPolicy,
	getSupportedOutputFormats,
	getTopLevelCacheControlCapabilities,
	supportsReasoningSummaryStyle,
} from '@/models/lib/capabilities';
import { parseOptionalPositiveInteger } from '@/models/lib/model_runtime_defaults';
import {
	clearRequestDefault,
	editRequestDefaults,
	inheritRequestDefault,
	removeRequestClear,
} from '@/models/lib/request_preferences';

interface AdvancedParamsModalProps {
	isOpen: boolean;
	onClose: () => void;
	currentModel: UIModelOption;
	effectiveReasoningEnabled?: boolean;
	onSave: (updatedModel: UIModelOption) => void;
}

type OutputFormatChoice = 'default' | 'text' | 'jsonSchema';
type SummaryStyleChoice = '' | ReasoningSummaryStyle;
type StreamChoice = 'default' | 'enabled' | 'disabled';
type CacheControlMode = 'default' | 'enabled' | 'disabled';
type StrictChoice = '' | 'true' | 'false';

type ErrorKey = 'maxPromptLength' | 'maxOutputLength' | 'timeout' | 'stopSequences' | 'jsonSchemaName' | 'jsonSchema';

type AdvancedParamsModalInnerProps = Omit<AdvancedParamsModalProps, 'isOpen' | 'onClose'>;

const streamChoiceItems: Record<StreamChoice, { isEnabled: boolean; displayName: string }> = {
	default: { isEnabled: true, displayName: 'Default' },
	enabled: { isEnabled: true, displayName: 'Enabled' },
	disabled: { isEnabled: true, displayName: 'Disabled' },
};

const strictChoiceItems: Record<StrictChoice, { isEnabled: boolean; displayName: string }> = {
	'': { isEnabled: true, displayName: 'Default' },
	true: { isEnabled: true, displayName: 'Strict' },
	false: { isEnabled: true, displayName: 'Non-strict' },
};

function getInitialOutputFormatChoice(kind: OutputFormatKind | undefined): OutputFormatChoice {
	if (kind === OutputFormatKind.Text) {
		return 'text';
	}
	if (kind === OutputFormatKind.JSONSchema) {
		return 'jsonSchema';
	}
	return 'default';
}

function getInitialReasoningSummaryStyle(summaryStyle: ReasoningSummaryStyle | undefined): SummaryStyleChoice {
	return summaryStyle ?? '';
}

function parseRequestStopSequences(raw: string): string[] | undefined {
	const values = raw.split(/\r?\n/g).filter(value => value.length > 0);
	return values.length > 0 ? values : undefined;
}

function strictChoiceFrom(value: boolean | undefined): StrictChoice {
	if (value === true) {
		return 'true';
	}
	if (value === false) {
		return 'false';
	}
	return '';
}

function strictChoiceToValue(value: StrictChoice): boolean | undefined {
	if (value === 'true') {
		return true;
	}
	if (value === 'false') {
		return false;
	}
	return undefined;
}

function HelpHint({ content }: { content: string }) {
	return (
		<span className="tooltip tooltip-right ml-1 inline-flex cursor-help" data-tip={content}>
			<FiHelpCircle size={12} />
		</span>
	);
}

const validateNumberField = (field: 'maxPromptLength' | 'maxOutputLength' | 'timeout', value: string) => {
	const n = parseOptionalPositiveInteger(value);
	if (n === undefined) {
		return undefined;
	}
	if (!Number.isFinite(n)) {
		return `${field} must be a positive integer.`;
	}
	return undefined;
};

function AdvancedParamsModalInner({ currentModel, effectiveReasoningEnabled, onSave }: AdvancedParamsModalInnerProps) {
	const { requestClose } = useModalDialogController();
	const maxPromptLengthInputRef = useRef<HTMLInputElement | null>(null);

	const requestDefaults = currentModel.requestPatch?.defaults;
	const requestClear = currentModel.requestPatch?.clear ?? [];

	const supportedOutputFormats = useMemo(
		() => getSupportedOutputFormats(currentModel.capabilities),
		[currentModel.capabilities]
	);

	const outputFormatItems: Record<OutputFormatChoice, { isEnabled: boolean; displayName: string }> = useMemo(() => {
		const supportsText = !supportedOutputFormats || supportedOutputFormats.includes(OutputFormatKind.Text);
		const supportsSchema = !supportedOutputFormats || supportedOutputFormats.includes(OutputFormatKind.JSONSchema);

		return {
			default: { isEnabled: true, displayName: 'Default' },
			text: { isEnabled: supportsText, displayName: 'Text' },
			jsonSchema: { isEnabled: supportsSchema, displayName: 'JSON (schema)' },
		};
	}, [supportedOutputFormats]);

	const effectiveCacheCapabilities = useMemo(
		() => getEffectiveCacheCapabilities(currentModel.providerSDKType, currentModel.capabilities),
		[currentModel.capabilities, currentModel.providerSDKType]
	);
	const topLevelCacheCapabilities = useMemo(
		() => getTopLevelCacheControlCapabilities(currentModel.providerSDKType, currentModel.capabilities),
		[currentModel.capabilities, currentModel.providerSDKType]
	);
	const supportedCacheKinds = useMemo(
		() => resolveSupportedCacheControlKinds(topLevelCacheCapabilities?.supportedKinds, requestDefaults?.cacheControl),
		[requestDefaults?.cacheControl, topLevelCacheCapabilities?.supportedKinds]
	);
	const supportedCacheTTLs = useMemo(
		() => resolveSupportedCacheControlTTLs(topLevelCacheCapabilities?.supportedTTLs, requestDefaults?.cacheControl),
		[requestDefaults?.cacheControl, topLevelCacheCapabilities?.supportedTTLs]
	);
	const supportsManualCacheControl = supportedCacheKinds.length > 0;
	const supportsCacheKey = topLevelCacheCapabilities?.supportsKey === true;
	const supportsAutomaticProviderCaching = effectiveCacheCapabilities?.supportsAutomaticCaching === true;

	const summaryStyleSupported = supportsReasoningSummaryStyle(currentModel.capabilities);
	const reasoningEnabled =
		!requestClear.includes(ModelRequestClearField.Reasoning) &&
		(effectiveReasoningEnabled ?? Boolean(currentModel.reasoning));

	const stopPolicy = useMemo(() => getStopSequencesPolicy(currentModel.capabilities), [currentModel.capabilities]);
	const stopSequencesDisabledBecauseReasoning = stopPolicy.disallowedWithReasoning && reasoningEnabled;

	const reasoningSummaryStyleItems: Record<SummaryStyleChoice, { isEnabled: boolean; displayName: string }> = useMemo(
		() => ({
			'': { isEnabled: true, displayName: 'Default' },
			[ReasoningSummaryStyle.Auto]: {
				isEnabled: reasoningEnabled && summaryStyleSupported,
				displayName: 'Auto',
			},
			[ReasoningSummaryStyle.Concise]: {
				isEnabled: reasoningEnabled && summaryStyleSupported,
				displayName: 'Concise',
			},
			[ReasoningSummaryStyle.Detailed]: {
				isEnabled: reasoningEnabled && summaryStyleSupported,
				displayName: 'Detailed',
			},
			[ReasoningSummaryStyle.Omitted]: {
				isEnabled: reasoningEnabled && summaryStyleSupported,
				displayName: 'Concise (legacy default)',
			},
		}),
		[reasoningEnabled, summaryStyleSupported]
	);

	const cacheControlKindItems = useMemo(
		() => buildCacheControlKindDropdownItems(supportedCacheKinds),
		[supportedCacheKinds]
	);
	const cacheControlTTLItems = useMemo(
		() => buildCacheControlTTLDropdownItems(supportedCacheTTLs),
		[supportedCacheTTLs]
	);
	const cacheControlModeItems = useMemo<Record<CacheControlMode, { isEnabled: boolean; displayName: string }>>(
		() => ({
			default: { isEnabled: true, displayName: 'Default' },
			enabled: { isEnabled: supportsManualCacheControl, displayName: 'Enabled' },
			disabled: { isEnabled: true, displayName: 'Disabled' },
		}),
		[supportsManualCacheControl]
	);

	const [streamChoice, setStreamChoice] = useState<StreamChoice>(() => {
		if (requestDefaults?.stream === true) {
			return 'enabled';
		}
		if (requestDefaults?.stream === false) {
			return 'disabled';
		}
		return 'default';
	});

	const [maxPromptLength, setMaxPromptLength] = useState(() =>
		requestDefaults?.maxPromptTokens === undefined ? '' : String(requestDefaults.maxPromptTokens)
	);
	const [maxOutputLength, setMaxOutputLength] = useState(() =>
		requestDefaults?.maxOutputTokens === undefined ? '' : String(requestDefaults.maxOutputTokens)
	);
	const [timeoutSec, setTimeoutSec] = useState(() =>
		requestDefaults?.timeoutMS === undefined ? '' : String(Math.ceil(requestDefaults.timeoutMS / 1000))
	);

	const [cacheControlMode, setCacheControlMode] = useState<CacheControlMode>(() => {
		if (requestClear.includes(ModelRequestClearField.CacheControl)) {
			return 'disabled';
		}
		return requestDefaults?.cacheControl ? 'enabled' : 'default';
	});
	const [cacheControlKind, setCacheControlKind] = useState<CacheControlKind | ''>(() =>
		getInitialCacheControlKind(requestDefaults?.cacheControl, supportedCacheKinds)
	);
	const [cacheControlTTL, setCacheControlTTL] = useState<CacheControlTTLSelection>(() =>
		getInitialCacheControlTTLSelection(requestDefaults?.cacheControl, supportedCacheTTLs)
	);
	const [cacheControlKey, setCacheControlKey] = useState(() => requestDefaults?.cacheControl?.key ?? '');

	const [reasoningSummaryStyle, setReasoningSummaryStyle] = useState<SummaryStyleChoice>(() =>
		getInitialReasoningSummaryStyle(requestDefaults?.reasoning?.summaryStyle)
	);
	const [outputFormatChoice, setOutputFormatChoice] = useState<OutputFormatChoice>(() =>
		getInitialOutputFormatChoice(requestDefaults?.output?.format?.kind)
	);

	const requestJSONSchema = requestDefaults?.output?.format?.jsonSchema;
	const [jsonSchemaName, setJsonSchemaName] = useState(() => requestJSONSchema?.name ?? '');
	const [jsonSchemaDescription, setJsonSchemaDescription] = useState(() => requestJSONSchema?.description ?? '');
	const [jsonSchemaStrict, setJsonSchemaStrict] = useState<StrictChoice>(() =>
		strictChoiceFrom(requestJSONSchema?.strict)
	);
	const [jsonSchemaText, setJsonSchemaText] = useState(() =>
		requestJSONSchema?.schema ? JSON.stringify(requestJSONSchema.schema, null, 2) : ''
	);

	const [stopSequencesText, setStopSequencesText] = useState(() => (requestDefaults?.stopSequences ?? []).join('\n'));
	const [errors, setErrors] = useState<Partial<Record<ErrorKey, string>>>({});

	useEffect(() => {
		let raf1 = 0;
		let raf2 = 0;

		raf1 = window.requestAnimationFrame(() => {
			raf2 = window.requestAnimationFrame(() => {
				focusTextInputAtEnd(maxPromptLengthInputRef.current);
			});
		});

		return () => {
			window.cancelAnimationFrame(raf1);
			window.cancelAnimationFrame(raf2);
		};
	}, []);

	const updateField = (
		field: 'maxPromptLength' | 'maxOutputLength' | 'timeout',
		value: string,
		setter: Dispatch<SetStateAction<string>>
	) => {
		setter(value);
		setErrors(previous => ({
			...previous,
			[field]: validateNumberField(field, value),
		}));
	};

	const validateStopSequences = (raw: string): string | undefined => {
		if (!stopPolicy.isSupported || stopSequencesDisabledBecauseReasoning) {
			return undefined;
		}

		const parsed = parseRequestStopSequences(raw) ?? [];
		if (parsed.length === 0) {
			return undefined;
		}
		if (parsed.length > stopPolicy.maxSequences) {
			return `Too many stop sequences (max ${stopPolicy.maxSequences}).`;
		}

		const tooLong = parsed.find(value => value.length > 256);
		if (tooLong) {
			return 'A stop sequence is too long (max 256 chars per line).';
		}

		return undefined;
	};

	const validateJSONSchema = (
		choice: OutputFormatChoice,
		nameValue: string = jsonSchemaName,
		schemaTextValue: string = jsonSchemaText
	) => {
		if (choice !== 'jsonSchema') {
			return {
				nameErr: undefined as string | undefined,
				schemaErr: undefined as string | undefined,
			};
		}

		const name = nameValue.trim();
		if (!name) {
			return {
				nameErr: 'Schema name is required.',
				schemaErr: undefined,
			};
		}

		const schemaRaw = schemaTextValue.trim();
		if (!schemaRaw) {
			return {
				nameErr: undefined,
				schemaErr: 'Schema JSON is required.',
			};
		}

		const parsed = tryParseJSONObject(schemaRaw, 'Schema JSON', MAX_JSON_SCHEMA_INPUT_CHARS);
		if (!parsed.ok) {
			return {
				nameErr: undefined,
				schemaErr: parsed.error,
			};
		}

		return {
			nameErr: undefined,
			schemaErr: undefined,
		};
	};

	const maybeRefreshJSONSchemaErrors = (nextChoice: OutputFormatChoice, nextName: string, nextSchemaText: string) => {
		if (!errors.jsonSchemaName && !errors.jsonSchema) {
			return;
		}

		const { nameErr, schemaErr } = validateJSONSchema(nextChoice, nextName, nextSchemaText);
		setErrors(previous => ({
			...previous,
			jsonSchemaName: nameErr,
			jsonSchema: schemaErr,
		}));
	};

	const formHasErrors = Object.values(errors).some(Boolean);

	const handleOutputFormatChange = (choice: OutputFormatChoice) => {
		setOutputFormatChoice(choice);

		if (choice !== 'jsonSchema') {
			setErrors(previous => ({
				...previous,
				jsonSchemaName: undefined,
				jsonSchema: undefined,
			}));
			return;
		}

		maybeRefreshJSONSchemaErrors(choice, jsonSchemaName, jsonSchemaText);
	};

	const handleJsonSchemaNameChange = (value: string) => {
		setJsonSchemaName(value);
		maybeRefreshJSONSchemaErrors(outputFormatChoice, value, jsonSchemaText);
	};

	const handleJsonSchemaTextChange = (value: string) => {
		setJsonSchemaText(value);
		maybeRefreshJSONSchemaErrors(outputFormatChoice, jsonSchemaName, value);
	};

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();

		const maxPromptErr = validateNumberField('maxPromptLength', maxPromptLength);
		const maxOutputErr = validateNumberField('maxOutputLength', maxOutputLength);
		const timeoutErr = validateNumberField('timeout', timeoutSec);
		const stopErr = validateStopSequences(stopSequencesText);
		const { nameErr, schemaErr } = validateJSONSchema(outputFormatChoice);

		const nextCacheControl =
			cacheControlMode === 'enabled'
				? buildCacheControlFromForm({
						enabled: true,
						kind: cacheControlKind,
						supportedKinds: supportedCacheKinds,
						ttlSelection: cacheControlTTL,
						key: cacheControlKey,
						supportsTTL: topLevelCacheCapabilities?.supportsTTL ?? true,
						supportsKey: supportsCacheKey,
					})
				: undefined;

		const nextErrors: Partial<Record<ErrorKey, string>> = {
			maxPromptLength: maxPromptErr,
			maxOutputLength: maxOutputErr,
			timeout: timeoutErr,
			stopSequences: stopErr,
			jsonSchemaName: nameErr,
			jsonSchema: schemaErr,
		};

		if (Object.values(nextErrors).some(Boolean)) {
			setErrors(nextErrors);
			return;
		}

		let requestPatch = currentModel.requestPatch;

		requestPatch = editRequestDefaults(requestPatch, defaults => {
			if (streamChoice === 'default') {
				delete defaults.stream;
			} else {
				defaults.stream = streamChoice === 'enabled';
			}

			const promptTokens = parseOptionalPositiveInteger(maxPromptLength);
			if (promptTokens === undefined) {
				delete defaults.maxPromptTokens;
			} else {
				defaults.maxPromptTokens = promptTokens;
			}

			const outputTokens = parseOptionalPositiveInteger(maxOutputLength);
			if (outputTokens === undefined) {
				delete defaults.maxOutputTokens;
			} else {
				defaults.maxOutputTokens = outputTokens;
			}

			const timeoutSeconds = parseOptionalPositiveInteger(timeoutSec);
			if (timeoutSeconds === undefined) {
				delete defaults.timeoutMS;
			} else {
				defaults.timeoutMS = timeoutSeconds * 1000;
			}

			const reasoning = { ...defaults.reasoning };
			if (!reasoningSummaryStyle) {
				delete reasoning.summaryStyle;
			} else {
				reasoning.summaryStyle = reasoningSummaryStyle;
			}
			if (Object.keys(reasoning).length === 0) {
				delete defaults.reasoning;
			} else {
				defaults.reasoning = reasoning;
			}

			const output = { ...defaults.output };

			if (outputFormatChoice === 'default') {
				delete output.format;
			} else if (outputFormatChoice === 'text' && outputFormatItems.text.isEnabled) {
				output.format = {
					kind: OutputFormatKind.Text,
				};
			} else if (outputFormatChoice === 'jsonSchema' && outputFormatItems.jsonSchema.isEnabled) {
				const parsedSchema = tryParseJSONObject(jsonSchemaText.trim(), 'Schema JSON', MAX_JSON_SCHEMA_INPUT_CHARS);
				const strict = strictChoiceToValue(jsonSchemaStrict);

				if (parsedSchema.ok) {
					const format: ModelOutputFormat = {
						kind: OutputFormatKind.JSONSchema,
						jsonSchema: {
							name: jsonSchemaName.trim(),
							...(jsonSchemaDescription.trim() ? { description: jsonSchemaDescription.trim() } : {}),
							...(strict !== undefined ? { strict } : {}),
							schema: parsedSchema.value,
						},
					};
					output.format = format;
				}
			}

			if (Object.keys(output).length === 0) {
				delete defaults.output;
			} else {
				defaults.output = output;
			}

			if (stopPolicy.isSupported && !stopSequencesDisabledBecauseReasoning) {
				const stopSequences = parseRequestStopSequences(stopSequencesText);
				if (!stopSequences) {
					delete defaults.stopSequences;
				} else {
					defaults.stopSequences = stopSequences;
				}
			}
		});

		if (reasoningSummaryStyle) {
			requestPatch = removeRequestClear(requestPatch, ModelRequestClearField.Reasoning);
		}

		if (outputFormatChoice !== 'default') {
			requestPatch = removeRequestClear(requestPatch, ModelRequestClearField.Output);
		}

		switch (cacheControlMode) {
			case 'default':
				requestPatch = inheritRequestDefault(requestPatch, ModelRequestClearField.CacheControl);
				break;

			case 'disabled':
				requestPatch = clearRequestDefault(requestPatch, ModelRequestClearField.CacheControl);
				break;

			case 'enabled':
				if (nextCacheControl) {
					requestPatch = editRequestDefaults(requestPatch, defaults => {
						defaults.cacheControl = nextCacheControl;
					});
					requestPatch = removeRequestClear(requestPatch, ModelRequestClearField.CacheControl);
				}
				break;
		}

		const updatedModel: UIModelOption = {
			...currentModel,
			requestPatch,
		};

		onSave(updatedModel);
		requestClose();
	};

	return (
		<div className="modal-box bg-base-200 max-h-[80vh] max-w-3xl overflow-auto rounded-2xl p-0">
			<ModalHeader
				title="Advanced Model Parameters"
				onClose={() => {
					requestClose();
				}}
			/>

			<form onSubmit={handleSubmit} className="space-y-4 p-6">
				<div className="grid grid-cols-12 items-center gap-2">
					<div className="label col-span-4">
						<span className="text-sm">Streaming</span>
						<HelpHint content="Default leaves streaming to the model/runtime configuration." />
					</div>
					<div className="col-span-8">
						<Dropdown<StreamChoice>
							dropdownItems={streamChoiceItems}
							selectedKey={streamChoice}
							onChange={setStreamChoice}
							filterDisabled={false}
							title="Select streaming preference"
							getDisplayName={choice => streamChoiceItems[choice].displayName}
						/>
					</div>
				</div>

				<div className="grid grid-cols-12 items-center gap-2">
					<label htmlFor="advanced-max-prompt-length" className="label col-span-4">
						<span className="text-sm">Max Prompt Tokens</span>
						<HelpHint content="Blank leaves this value to the model/runtime configuration." />
					</label>
					<div className="col-span-8">
						<input
							id="advanced-max-prompt-length"
							ref={maxPromptLengthInputRef}
							type="text"
							value={maxPromptLength}
							onChange={event => {
								updateField('maxPromptLength', event.target.value, setMaxPromptLength);
							}}
							className={`input w-full rounded-xl ${errors.maxPromptLength ? 'input-error' : ''}`}
							placeholder="Default: model / SDK"
							spellCheck="false"
						/>
						{errors.maxPromptLength && (
							<div className="label">
								<span className="text-error flex items-center gap-1">
									<FiAlertCircle size={12} /> {errors.maxPromptLength}
								</span>
							</div>
						)}
					</div>
				</div>

				<div className="grid grid-cols-12 items-center gap-2">
					<label htmlFor="advanced-max-output-length" className="label col-span-4">
						<span className="text-sm">Max Output Tokens</span>
						<HelpHint content="Blank leaves this value to the model/runtime configuration." />
					</label>
					<div className="col-span-8">
						<input
							id="advanced-max-output-length"
							type="text"
							value={maxOutputLength}
							onChange={event => {
								updateField('maxOutputLength', event.target.value, setMaxOutputLength);
							}}
							className={`input w-full rounded-xl ${errors.maxOutputLength ? 'input-error' : ''}`}
							placeholder="Default: model / SDK"
							spellCheck="false"
						/>
						{errors.maxOutputLength && (
							<div className="label">
								<span className="text-error flex items-center gap-1">
									<FiAlertCircle size={12} /> {errors.maxOutputLength}
								</span>
							</div>
						)}
					</div>
				</div>

				<div className="grid grid-cols-12 items-center gap-2">
					<label htmlFor="advanced-timeout" className="label col-span-4">
						<span className="text-sm">Timeout (s)</span>
						<HelpHint content="Blank leaves timeout selection to the model/runtime configuration." />
					</label>
					<div className="col-span-8">
						<input
							id="advanced-timeout"
							type="text"
							value={timeoutSec}
							onChange={event => {
								updateField('timeout', event.target.value, setTimeoutSec);
							}}
							className={`input w-full rounded-xl ${errors.timeout ? 'input-error' : ''}`}
							placeholder="Default: model / SDK"
							spellCheck="false"
						/>
						{errors.timeout && (
							<div className="label">
								<span className="text-error flex items-center gap-1">
									<FiAlertCircle size={12} /> {errors.timeout}
								</span>
							</div>
						)}
					</div>
				</div>

				<div className="grid grid-cols-12 items-center gap-2">
					<div className="label col-span-4">
						<span className="text-sm">Reasoning Summary</span>
						<HelpHint
							content={
								reasoningEnabled
									? 'Default leaves reasoning summary style to source configuration and the SDK.'
									: 'Reasoning is not currently active for this model selection.'
							}
						/>
					</div>
					<div className="col-span-8">
						<Dropdown<SummaryStyleChoice>
							dropdownItems={reasoningSummaryStyleItems}
							selectedKey={reasoningSummaryStyle}
							onChange={setReasoningSummaryStyle}
							filterDisabled={false}
							title="Select reasoning summary style"
							getDisplayName={key => reasoningSummaryStyleItems[key].displayName}
						/>
					</div>
				</div>

				<div className="grid grid-cols-12 items-center gap-2">
					<div className="label col-span-4">
						<span className="text-sm">Output Format</span>
						<HelpHint content="Default leaves output format to source configuration and the SDK." />
					</div>
					<div className="col-span-8">
						<Dropdown<OutputFormatChoice>
							dropdownItems={outputFormatItems}
							selectedKey={outputFormatChoice}
							onChange={handleOutputFormatChange}
							filterDisabled={false}
							title="Select output format"
							getDisplayName={key => outputFormatItems[key].displayName}
						/>
					</div>
				</div>

				{outputFormatChoice === 'jsonSchema' && (
					<>
						<div className="grid grid-cols-12 items-center gap-2">
							<label htmlFor="advanced-json-schema-name" className="label col-span-4">
								<span className="text-sm">Schema Name</span>
							</label>
							<div className="col-span-8">
								<input
									id="advanced-json-schema-name"
									type="text"
									className={`input w-full rounded-xl ${errors.jsonSchemaName ? 'input-error' : ''}`}
									value={jsonSchemaName}
									onChange={event => {
										handleJsonSchemaNameChange(event.target.value);
									}}
									placeholder="e.g. my_response"
									spellCheck="false"
								/>
								{errors.jsonSchemaName && (
									<div className="label">
										<span className="text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.jsonSchemaName}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label htmlFor="advanced-json-schema-description" className="label col-span-4">
								<span className="text-sm">Description</span>
							</label>
							<div className="col-span-8">
								<input
									id="advanced-json-schema-description"
									type="text"
									className="input w-full rounded-xl"
									value={jsonSchemaDescription}
									onChange={event => {
										setJsonSchemaDescription(event.target.value);
									}}
									placeholder="Optional"
									spellCheck="false"
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<div className="label col-span-4">
								<span className="text-sm">Strict</span>
							</div>
							<div className="col-span-8">
								<Dropdown<StrictChoice>
									dropdownItems={strictChoiceItems}
									selectedKey={jsonSchemaStrict}
									onChange={setJsonSchemaStrict}
									filterDisabled={false}
									title="Select JSON Schema strictness"
									getDisplayName={choice => strictChoiceItems[choice].displayName}
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-start gap-2">
							<label htmlFor="advanced-json-schema" className="label col-span-4">
								<span className="text-sm">Schema JSON</span>
							</label>
							<div className="col-span-8">
								<textarea
									id="advanced-json-schema"
									className={`textarea w-full rounded-xl ${errors.jsonSchema ? 'textarea-error' : ''}`}
									rows={8}
									value={jsonSchemaText}
									onChange={event => {
										handleJsonSchemaTextChange(event.target.value);
									}}
									placeholder={`{\n  "type": "object",\n  "properties": {\n    "answer": { "type": "string" }\n  },\n  "required": ["answer"]\n}`}
									spellCheck="false"
								/>
								{errors.jsonSchema && (
									<div className="label">
										<span className="text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.jsonSchema}
										</span>
									</div>
								)}
							</div>
						</div>
					</>
				)}

				{(supportsManualCacheControl || supportsAutomaticProviderCaching) && (
					<>
						<div className="grid grid-cols-12 items-center gap-2">
							<div className="label col-span-4">
								<span className="text-sm">Cache Control</span>
								<HelpHint content="Default leaves cache behavior to source configuration and the provider." />
							</div>
							<div className="col-span-8">
								{supportsManualCacheControl ? (
									<Dropdown<CacheControlMode>
										dropdownItems={cacheControlModeItems}
										selectedKey={cacheControlMode}
										onChange={mode => {
											setCacheControlMode(mode);
											if (mode === 'enabled' && !cacheControlKind) {
												setCacheControlKind(supportedCacheKinds[0] ?? '');
											}
										}}
										filterDisabled={false}
										title="Select cache-control preference"
										getDisplayName={mode => cacheControlModeItems[mode].displayName}
									/>
								) : (
									<span className="text-sm opacity-70">Manual top-level cache control is not available.</span>
								)}
							</div>
							{supportsAutomaticProviderCaching && (
								<div className="label">
									<span className="text-xs opacity-70">
										This provider SDK also supports automatic caching behavior.
									</span>
								</div>
							)}
						</div>

						{supportsManualCacheControl && cacheControlMode === 'enabled' && (
							<>
								<div className="grid grid-cols-12 items-center gap-2">
									<div className="label col-span-4">
										<span className="text-sm">Cache Kind</span>
									</div>
									<div className="col-span-8">
										<Dropdown<CacheControlKind>
											dropdownItems={cacheControlKindItems}
											selectedKey={(cacheControlKind || supportedCacheKinds[0]) as CacheControlKind}
											onChange={setCacheControlKind}
											filterDisabled={false}
											title="Select cache kind"
											getDisplayName={key => cacheControlKindItems[key].displayName}
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<div className="label col-span-4">
										<span className="text-sm">Cache TTL</span>
									</div>
									<div className="col-span-8">
										<Dropdown<CacheControlTTLSelection>
											dropdownItems={cacheControlTTLItems}
											selectedKey={cacheControlTTL}
											onChange={setCacheControlTTL}
											filterDisabled={false}
											title="Select cache TTL"
											getDisplayName={key => cacheControlTTLItems[key].displayName}
										/>
									</div>
								</div>

								{supportsCacheKey && (
									<div className="grid grid-cols-12 items-center gap-2">
										<label htmlFor="advanced-cache-key" className="label col-span-4">
											<span className="text-sm">Cache Key</span>
										</label>
										<div className="col-span-8">
											<input
												id="advanced-cache-key"
												type="text"
												value={cacheControlKey}
												onChange={event => {
													setCacheControlKey(event.target.value);
												}}
												className="input w-full rounded-xl"
												placeholder="Optional request cache key"
												spellCheck="false"
											/>
										</div>
									</div>
								)}
							</>
						)}
					</>
				)}

				{stopPolicy.isSupported && (
					<div className="grid grid-cols-12 items-start gap-2">
						<label htmlFor="advanced-stop-sequences" className="label col-span-4">
							<span className="text-sm">Stop Sequences</span>
							<HelpHint
								content={
									stopSequencesDisabledBecauseReasoning
										? 'Stop sequences are disabled while reasoning is active for this model/provider.'
										: `One per line. Blank uses model / SDK defaults. Max ${stopPolicy.maxSequences}.`
								}
							/>
						</label>
						<div className="col-span-8">
							<textarea
								id="advanced-stop-sequences"
								disabled={stopSequencesDisabledBecauseReasoning}
								className={`textarea w-full rounded-xl ${
									errors.stopSequences ? 'textarea-error' : ''
								} ${stopSequencesDisabledBecauseReasoning ? 'cursor-not-allowed opacity-50' : ''}`}
								rows={4}
								value={stopSequencesText}
								onChange={event => {
									const value = event.target.value;
									setStopSequencesText(value);
									setErrors(previous => ({
										...previous,
										stopSequences: validateStopSequences(value),
									}));
								}}
								placeholder={'e.g.\n\nEND\n</final>'}
								spellCheck="false"
							/>
							{stopSequencesDisabledBecauseReasoning && (
								<div className="label">
									<span className="opacity-70">Disabled because reasoning is enabled for this model/provider.</span>
								</div>
							)}
							{errors.stopSequences && (
								<div className="label">
									<span className="text-error flex items-center gap-1">
										<FiAlertCircle size={12} /> {errors.stopSequences}
									</span>
								</div>
							)}
						</div>
					</div>
				)}

				<ModalActions className="-mx-6 mt-6 -mb-6">
					<button
						type="button"
						className="btn bg-base-300 rounded-xl"
						onClick={() => {
							requestClose();
						}}
					>
						Cancel
					</button>
					<button type="submit" className="btn btn-primary rounded-xl" disabled={formHasErrors}>
						Save
					</button>
				</ModalActions>
			</form>
		</div>
	);
}

export function AdvancedParamsModal({
	isOpen,
	onClose,
	currentModel,
	effectiveReasoningEnabled,
	onSave,
}: AdvancedParamsModalProps) {
	if (!isOpen || typeof document === 'undefined') {
		return null;
	}

	const modelIdentity = currentModel.model
		? `${currentModel.model.rootID}::${currentModel.model.artifactID}`
		: `${currentModel.providerName}::${currentModel.logicalName}`;

	return (
		<ModalDialog isOpen={isOpen} onClose={onClose}>
			<AdvancedParamsModalInner
				key={modelIdentity}
				currentModel={currentModel}
				effectiveReasoningEnabled={effectiveReasoningEnabled}
				onSave={onSave}
			/>
		</ModalDialog>
	);
}
