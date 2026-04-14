import {
	type Dispatch,
	type SetStateAction,
	type SubmitEventHandler,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiX } from 'react-icons/fi';

import {
	type CacheControlKind,
	type JSONSchemaParam,
	OutputFormatKind,
	type OutputParam,
	ReasoningSummaryStyle,
} from '@/spec/inference';
import type { UIChatOption } from '@/spec/modelpreset';

import { Dropdown } from '@/components/dropdown';

import {
	buildCacheControlFromForm,
	buildCacheControlKindDropdownItems,
	buildCacheControlTTLDropdownItems,
	type CacheControlTTLSelection,
	getInitialCacheControlKind,
	getInitialCacheControlTTLSelection,
	resolveSupportedCacheControlKinds,
	resolveSupportedCacheControlTTLs,
} from '@/modelpresets/lib/cache_control_utils';
import {
	getEffectiveCacheCapabilities,
	getStopSequencesPolicy,
	getSupportedOutputFormats,
	getTopLevelCacheControlCapabilities,
	supportsReasoningSummaryStyle,
} from '@/modelpresets/lib/capabilities_override';

type AdvancedParamsModalProps = {
	isOpen: boolean;
	onClose: () => void;
	currentModel: UIChatOption;
	effectiveReasoningEnabled?: boolean;
	onSave: (updatedModel: UIChatOption) => void;
};

type OutputFormatChoice = 'default' | 'text' | 'jsonSchema';
type SummaryStyleChoice = '' | ReasoningSummaryStyle;

type ErrorKey =
	| 'maxPromptLength'
	| 'maxOutputLength'
	| 'timeout'
	| 'stopSequences'
	| 'additionalParametersRawJSON'
	| 'jsonSchemaName'
	| 'jsonSchema';

type AdvancedParamsModalInnerProps = Omit<AdvancedParamsModalProps, 'isOpen'>;

const MAX_JSON_CHARS = 50_000;

function parsePositiveIntAllowBlank(v: string): number | undefined {
	const s = v.trim();
	if (!s) return undefined;
	const n = Number(s);
	if (!Number.isFinite(n) || !Number.isInteger(n) || n <= 0) return NaN;
	return n;
}

function parseStopSequences(raw: string): string[] | undefined {
	const lines = raw
		.split(/\r?\n/g)
		.map(s => s.trim())
		.filter(Boolean);

	if (lines.length === 0) return undefined;

	const seen = new Set<string>();
	const out: string[] = [];
	for (const s of lines) {
		if (seen.has(s)) continue;
		seen.add(s);
		out.push(s);
	}
	return out.length > 0 ? out : undefined;
}

function safeJSONParse(raw: string): { ok: boolean; value: any } {
	try {
		return { ok: true, value: JSON.parse(raw) };
	} catch (e) {
		return { ok: false, value: (e as Error)?.message ?? 'Invalid JSON' };
	}
}

function isPlainObject(v: any): boolean {
	return typeof v === 'object' && v !== null && !Array.isArray(v);
}

function getInitialOutputFormatChoice(
	currentModel: UIChatOption,
	supportedOutputFormats: OutputFormatKind[] | undefined
): OutputFormatChoice {
	const kind = currentModel.outputParam?.format?.kind;
	const kindIsSupported = !supportedOutputFormats || (kind ? supportedOutputFormats.includes(kind) : true);

	if (!kindIsSupported) return 'default';
	if (kind === OutputFormatKind.Text) return 'text';
	if (kind === OutputFormatKind.JSONSchema) return 'jsonSchema';
	return 'default';
}

function getInitialReasoningSummaryStyle(
	currentModel: UIChatOption,
	summaryStyleSupported: boolean
): SummaryStyleChoice {
	return summaryStyleSupported ? ((currentModel.reasoning?.summaryStyle as SummaryStyleChoice) ?? '') : '';
}

function closeDialogSafely(dialog: HTMLDialogElement | null): boolean {
	if (!dialog?.open) return false;

	try {
		dialog.close();
		return true;
	} catch {
		return false;
	}
}

function HelpHint({ content }: { content: string }) {
	return (
		<span className="label-text-alt tooltip tooltip-right ml-1 inline-flex cursor-help" data-tip={content}>
			<FiHelpCircle size={12} />
		</span>
	);
}

function AdvancedParamsModalInner({
	onClose,
	currentModel,
	effectiveReasoningEnabled,
	onSave,
}: AdvancedParamsModalInnerProps) {
	const dialogRef = useRef<HTMLDialogElement | null>(null);

	const supportedOutputFormats = useMemo(
		() => getSupportedOutputFormats(currentModel.capabilitiesOverride),
		[currentModel.capabilitiesOverride]
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

	const reasoningEnabled = effectiveReasoningEnabled ?? !!currentModel.reasoning;

	const effectiveCacheCapabilities = useMemo(
		() => getEffectiveCacheCapabilities(currentModel.providerSDKType, currentModel.capabilitiesOverride),
		[currentModel.capabilitiesOverride, currentModel.providerSDKType]
	);
	const topLevelCacheCapabilities = useMemo(
		() => getTopLevelCacheControlCapabilities(currentModel.providerSDKType, currentModel.capabilitiesOverride),
		[currentModel.capabilitiesOverride, currentModel.providerSDKType]
	);
	const supportedCacheKinds = useMemo(
		() => resolveSupportedCacheControlKinds(topLevelCacheCapabilities?.supportedKinds, currentModel.cacheControl),
		[currentModel.cacheControl, topLevelCacheCapabilities?.supportedKinds]
	);
	const supportedCacheTTLs = useMemo(
		() => resolveSupportedCacheControlTTLs(topLevelCacheCapabilities?.supportedTTLs, currentModel.cacheControl),
		[currentModel.cacheControl, topLevelCacheCapabilities?.supportedTTLs]
	);
	const supportsManualCacheControl = supportedCacheKinds.length > 0;
	const supportsCacheKey =
		topLevelCacheCapabilities?.supportsKey === true || Boolean(currentModel.cacheControl?.key?.trim());
	const supportsAutomaticProviderCaching = effectiveCacheCapabilities?.supportsAutomaticCaching === true;

	const summaryStyleSupported = supportsReasoningSummaryStyle(currentModel.capabilitiesOverride);

	const stopPolicy = useMemo(
		() => getStopSequencesPolicy(currentModel.capabilitiesOverride),
		[currentModel.capabilitiesOverride]
	);

	const stopSequencesDisabledBecauseReasoning = stopPolicy.disallowedWithReasoning && reasoningEnabled;

	const reasoningSummaryStyleItems: Record<SummaryStyleChoice, { isEnabled: boolean; displayName: string }> = useMemo(
		() => ({
			'': { isEnabled: true, displayName: 'Default' },
			[ReasoningSummaryStyle.Auto]: { isEnabled: reasoningEnabled && summaryStyleSupported, displayName: 'Auto' },
			[ReasoningSummaryStyle.Concise]: { isEnabled: reasoningEnabled && summaryStyleSupported, displayName: 'Concise' },
			[ReasoningSummaryStyle.Detailed]: {
				isEnabled: reasoningEnabled && summaryStyleSupported,
				displayName: 'Detailed',
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

	const initialJSONSchema = currentModel.outputParam?.format?.jsonSchemaParam;

	const [stream, setStream] = useState(() => currentModel.stream);
	const [maxPromptLength, setMaxPromptLength] = useState(() => String(currentModel.maxPromptLength));
	const [maxOutputLength, setMaxOutputLength] = useState(() => String(currentModel.maxOutputLength));
	const [timeoutSec, setTimeoutSec] = useState(() => String(currentModel.timeout));
	const [cacheControlEnabled, setCacheControlEnabled] = useState(() => Boolean(currentModel.cacheControl));
	const [cacheControlKind, setCacheControlKind] = useState<CacheControlKind | ''>(() =>
		getInitialCacheControlKind(currentModel.cacheControl, supportedCacheKinds)
	);
	const [cacheControlTTL, setCacheControlTTL] = useState<CacheControlTTLSelection>(() =>
		getInitialCacheControlTTLSelection(currentModel.cacheControl, supportedCacheTTLs)
	);
	const [cacheControlKey, setCacheControlKey] = useState(() => currentModel.cacheControl?.key ?? '');

	const [reasoningSummaryStyle, setReasoningSummaryStyle] = useState<SummaryStyleChoice>(() =>
		getInitialReasoningSummaryStyle(currentModel, summaryStyleSupported)
	);

	const [outputFormatChoice, setOutputFormatChoice] = useState<OutputFormatChoice>(() =>
		getInitialOutputFormatChoice(currentModel, supportedOutputFormats)
	);

	const [jsonSchemaName, setJsonSchemaName] = useState(() => initialJSONSchema?.name ?? '');
	const [jsonSchemaDescription, setJsonSchemaDescription] = useState(() => initialJSONSchema?.description ?? '');
	const [jsonSchemaStrict, setJsonSchemaStrict] = useState(() => initialJSONSchema?.strict ?? true);
	const [jsonSchemaText, setJsonSchemaText] = useState(() =>
		initialJSONSchema?.schema ? JSON.stringify(initialJSONSchema.schema, null, 2) : ''
	);

	const [stopSequencesText, setStopSequencesText] = useState(() =>
		(currentModel.stopSequences ?? []).slice(0, stopPolicy.maxSequences).join('\n')
	);

	const [errors, setErrors] = useState<Partial<Record<ErrorKey, string>>>({});

	useEffect(() => {
		const dialog = dialogRef.current;
		if (!dialog) return;

		try {
			if (!dialog.open) {
				dialog.showModal();
			}
		} catch {
			// Ignore showModal errors if the dialog is already open or not ready.
		}
	}, []);

	const handleDialogClose = useCallback(() => {
		onClose();
	}, [onClose]);

	const requestClose = useCallback(() => {
		if (!closeDialogSafely(dialogRef.current)) {
			onClose();
		}
	}, [onClose]);

	const validateNumberField = (field: 'maxPromptLength' | 'maxOutputLength' | 'timeout', value: string) => {
		const n = parsePositiveIntAllowBlank(value);
		if (n === undefined) return undefined;
		if (!Number.isFinite(n)) return `${field} must be a positive integer.`;
		return undefined;
	};

	const updateField = (
		field: 'maxPromptLength' | 'maxOutputLength' | 'timeout',
		value: string,
		setter: Dispatch<SetStateAction<string>>
	) => {
		setter(value);
		setErrors(prev => ({ ...prev, [field]: validateNumberField(field, value) }));
	};

	const validateStopSequences = (raw: string): string | undefined => {
		if (!stopPolicy.isSupported) return undefined;
		if (stopSequencesDisabledBecauseReasoning) return undefined;

		const parsed = parseStopSequences(raw);
		if (!parsed) return undefined;

		if (parsed.length > stopPolicy.maxSequences) {
			return `Too many stop sequences (max ${stopPolicy.maxSequences}).`;
		}

		const tooLong = parsed.find(s => s.length > 256);
		if (tooLong) return 'A stop sequence is too long (max 256 chars per line).';

		return undefined;
	};

	const validateJSONSchema = (
		choice: OutputFormatChoice,
		nameValue: string = jsonSchemaName,
		schemaTextValue: string = jsonSchemaText
	) => {
		if (choice !== 'jsonSchema') {
			return { nameErr: undefined as string | undefined, schemaErr: undefined as string | undefined };
		}

		const name = nameValue.trim();
		if (!name) return { nameErr: 'Schema name is required.', schemaErr: undefined };

		const schemaRaw = schemaTextValue.trim();
		if (!schemaRaw) return { nameErr: undefined, schemaErr: 'Schema JSON is required.' };

		if (schemaRaw.length > MAX_JSON_CHARS) {
			return { nameErr: undefined, schemaErr: `Schema JSON too large (max ${MAX_JSON_CHARS} chars).` };
		}

		const parsed = safeJSONParse(schemaRaw);
		if (!parsed.ok) return { nameErr: undefined, schemaErr: `Invalid schema JSON: ${parsed.value}` };

		if (!isPlainObject(parsed.value)) {
			return { nameErr: undefined, schemaErr: 'Schema must be a JSON object.' };
		}

		return { nameErr: undefined, schemaErr: undefined };
	};

	const maybeRefreshJSONSchemaErrors = (nextChoice: OutputFormatChoice, nextName: string, nextSchemaText: string) => {
		if (!errors.jsonSchemaName && !errors.jsonSchema) return;

		const { nameErr, schemaErr } = validateJSONSchema(nextChoice, nextName, nextSchemaText);
		setErrors(prev => ({
			...prev,
			jsonSchemaName: nameErr,
			jsonSchema: schemaErr,
		}));
	};

	const formHasErrors = Object.values(errors).some(Boolean);

	const handleOutputFormatChange = (choice: OutputFormatChoice) => {
		setOutputFormatChoice(choice);

		if (choice !== 'jsonSchema') {
			setErrors(prev => ({
				...prev,
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

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();

		const maxPromptErr = validateNumberField('maxPromptLength', maxPromptLength);
		const maxOutputErr = validateNumberField('maxOutputLength', maxOutputLength);
		const timeoutErr = validateNumberField('timeout', timeoutSec);

		const stopErr = validateStopSequences(stopSequencesText);

		const { nameErr, schemaErr } = validateJSONSchema(outputFormatChoice);

		const nextCacheControl = buildCacheControlFromForm({
			enabled: cacheControlEnabled,
			kind: cacheControlKind,
			supportedKinds: supportedCacheKinds,
			ttlSelection: cacheControlTTL,
			key: cacheControlKey,
			supportsKey: supportsCacheKey,
		});

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

		const baseOutputParam: OutputParam | undefined = currentModel.outputParam
			? { ...currentModel.outputParam }
			: undefined;

		let nextFormat: OutputParam['format'] = undefined;

		if (outputFormatChoice === 'text') {
			if (outputFormatItems.text.isEnabled) {
				nextFormat = { kind: OutputFormatKind.Text };
			}
		} else if (outputFormatChoice === 'jsonSchema') {
			if (!outputFormatItems.jsonSchema.isEnabled) {
				nextFormat = undefined;
			} else {
				const parsedSchema = safeJSONParse(jsonSchemaText.trim());
				if (parsedSchema.ok) {
					const schemaObj = parsedSchema.value as Record<string, any>;

					nextFormat = {
						kind: OutputFormatKind.JSONSchema,
						jsonSchemaParam: {
							name: jsonSchemaName.trim(),
							description: jsonSchemaDescription.trim() || undefined,
							strict: jsonSchemaStrict,
							schema: schemaObj,
						} as JSONSchemaParam,
					};
				}
			}
		}

		const mergedOutputParam: OutputParam | undefined = (() => {
			const merged: OutputParam = { ...(baseOutputParam ?? {}) };

			if (nextFormat === undefined) delete merged.format;
			else merged.format = nextFormat;

			const hasAny = !!merged.verbosity || !!merged.format;
			return hasAny ? merged : undefined;
		})();

		const mergedReasoning = (() => {
			if (!currentModel.reasoning) return currentModel.reasoning;

			const r = { ...currentModel.reasoning };
			if (!reasoningSummaryStyle) {
				delete r.summaryStyle;
			} else {
				r.summaryStyle = reasoningSummaryStyle;
			}
			return r as UIChatOption['reasoning'];
		})();

		const nextStopSequences = (() => {
			if (!stopPolicy.isSupported) return undefined;
			if (stopSequencesDisabledBecauseReasoning) return undefined;

			const parsed = parseStopSequences(stopSequencesText);
			return parsed ? parsed.slice(0, stopPolicy.maxSequences) : undefined;
		})();

		const updatedModel: UIChatOption = {
			...currentModel,
			stream,
			maxPromptLength: parsePositiveIntAllowBlank(maxPromptLength) ?? currentModel.maxPromptLength,
			maxOutputLength: parsePositiveIntAllowBlank(maxOutputLength) ?? currentModel.maxOutputLength,
			timeout: parsePositiveIntAllowBlank(timeoutSec) ?? currentModel.timeout,
			cacheControl: nextCacheControl,
			reasoning: mergedReasoning,
			outputParam: mergedOutputParam,
			stopSequences: nextStopSequences,
		};

		onSave(updatedModel);
		requestClose();
	};

	return (
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleDialogClose}
			onCancel={e => {
				e.preventDefault();
				requestClose();
			}}
		>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-3xl overflow-auto rounded-2xl">
				<div className="mb-4 flex items-center justify-between">
					<h3 className="text-lg font-bold">Advanced Model Parameters</h3>
					<button type="button" className="btn btn-sm btn-circle bg-base-300" onClick={requestClose} aria-label="Close">
						<FiX size={12} />
					</button>
				</div>

				<form onSubmit={handleSubmit} className="space-y-4">
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-4 cursor-pointer">
							<span className="label-text text-sm">Streaming</span>
							<HelpHint content="Stream data continuously." />
						</label>
						<div className="col-span-8">
							<input
								type="checkbox"
								checked={stream}
								onChange={e => {
									setStream(e.target.checked);
								}}
								className="toggle toggle-accent"
							/>
						</div>
					</div>

					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-4">
							<span className="label-text text-sm">Max Prompt Tokens</span>
							<HelpHint content="Maximum tokens for input prompt" />
						</label>
						<div className="col-span-8">
							<input
								type="text"
								value={maxPromptLength}
								onChange={e => {
									updateField('maxPromptLength', e.target.value, setMaxPromptLength);
								}}
								className={`input input-bordered w-full rounded-xl ${errors.maxPromptLength ? 'input-error' : ''}`}
								placeholder={`Default: ${currentModel.maxPromptLength}`}
								spellCheck="false"
							/>
							{errors.maxPromptLength && (
								<div className="label">
									<span className="label-text-alt text-error flex items-center gap-1">
										<FiAlertCircle size={12} /> {errors.maxPromptLength}
									</span>
								</div>
							)}
						</div>
					</div>

					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-4">
							<span className="label-text text-sm">Max Output Tokens</span>
							<HelpHint content="Maximum tokens for model output" />
						</label>
						<div className="col-span-8">
							<input
								type="text"
								value={maxOutputLength}
								onChange={e => {
									updateField('maxOutputLength', e.target.value, setMaxOutputLength);
								}}
								className={`input input-bordered w-full rounded-xl ${errors.maxOutputLength ? 'input-error' : ''}`}
								placeholder={`Default: ${currentModel.maxOutputLength}`}
								spellCheck="false"
							/>
							{errors.maxOutputLength && (
								<div className="label">
									<span className="label-text-alt text-error flex items-center gap-1">
										<FiAlertCircle size={12} /> {errors.maxOutputLength}
									</span>
								</div>
							)}
						</div>
					</div>

					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-4">
							<span className="label-text text-sm">Timeout (s)</span>
							<HelpHint content="Maximum time a request can take (seconds)" />
						</label>
						<div className="col-span-8">
							<input
								type="text"
								value={timeoutSec}
								onChange={e => {
									updateField('timeout', e.target.value, setTimeoutSec);
								}}
								className={`input input-bordered w-full rounded-xl ${errors.timeout ? 'input-error' : ''}`}
								placeholder={`Default: ${currentModel.timeout}`}
								spellCheck="false"
							/>
							{errors.timeout && (
								<div className="label">
									<span className="label-text-alt text-error flex items-center gap-1">
										<FiAlertCircle size={12} /> {errors.timeout}
									</span>
								</div>
							)}
						</div>
					</div>

					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-4">
							<span className="label-text text-sm">Reasoning Summary</span>
							<HelpHint
								content={
									reasoningEnabled
										? 'Controls whether the model produces a reasoning summary (when supported).'
										: 'This model is currently using temperature mode (no reasoning params active).'
								}
							/>
						</label>
						<div className="col-span-8">
							<Dropdown<SummaryStyleChoice>
								dropdownItems={reasoningSummaryStyleItems}
								selectedKey={reasoningSummaryStyle}
								onChange={k => {
									setReasoningSummaryStyle(k);
								}}
								filterDisabled={false}
								title="Select Reasoning Summary Style"
								getDisplayName={k => reasoningSummaryStyleItems[k].displayName}
							/>
						</div>
					</div>

					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-4">
							<span className="label-text text-sm">Output Format</span>
							<HelpHint content="Controls output formatting. Verbosity is set in the top bar." />
						</label>
						<div className="col-span-8">
							<Dropdown<OutputFormatChoice>
								dropdownItems={outputFormatItems}
								selectedKey={outputFormatChoice}
								onChange={handleOutputFormatChange}
								filterDisabled={false}
								title="Select Output Format"
								getDisplayName={k => outputFormatItems[k].displayName}
							/>
						</div>
					</div>

					{outputFormatChoice === 'jsonSchema' && (
						<>
							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-4">
									<span className="label-text text-sm">Schema Name</span>
								</label>
								<div className="col-span-8">
									<input
										type="text"
										className={`input input-bordered w-full rounded-xl ${errors.jsonSchemaName ? 'input-error' : ''}`}
										value={jsonSchemaName}
										onChange={e => {
											handleJsonSchemaNameChange(e.target.value);
										}}
										placeholder="e.g. my_response"
										spellCheck="false"
									/>
									{errors.jsonSchemaName && (
										<div className="label">
											<span className="label-text-alt text-error flex items-center gap-1">
												<FiAlertCircle size={12} /> {errors.jsonSchemaName}
											</span>
										</div>
									)}
								</div>
							</div>

							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-4">
									<span className="label-text text-sm">Description</span>
								</label>
								<div className="col-span-8">
									<input
										type="text"
										className="input input-bordered w-full rounded-xl"
										value={jsonSchemaDescription}
										onChange={e => {
											setJsonSchemaDescription(e.target.value);
										}}
										placeholder="Optional"
										spellCheck="false"
									/>
								</div>
							</div>

							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-4 cursor-pointer">
									<span className="label-text text-sm">Strict</span>
								</label>
								<div className="col-span-8">
									<input
										type="checkbox"
										checked={jsonSchemaStrict}
										onChange={e => {
											setJsonSchemaStrict(e.target.checked);
										}}
										className="toggle toggle-accent"
									/>
								</div>
							</div>

							<div className="grid grid-cols-12 items-start gap-2">
								<label className="label col-span-4">
									<span className="label-text text-sm">Schema JSON</span>
								</label>
								<div className="col-span-8">
									<textarea
										className={`textarea textarea-bordered w-full rounded-xl ${errors.jsonSchema ? 'textarea-error' : ''}`}
										rows={8}
										value={jsonSchemaText}
										onChange={e => {
											handleJsonSchemaTextChange(e.target.value);
										}}
										placeholder={`{\n  "type": "object",\n  "properties": {\n    "answer": { "type": "string" }\n  },\n  "required": ["answer"]\n}`}
										spellCheck="false"
									/>
									{errors.jsonSchema && (
										<div className="label">
											<span className="label-text-alt text-error flex items-center gap-1">
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
								<label className="label col-span-4 cursor-pointer">
									<span className="label-text text-sm">Cache Control</span>
									<HelpHint content="Top-level request cache control, when supported by the selected provider SDK." />
								</label>
								<div className="col-span-8">
									{supportsManualCacheControl ? (
										<input
											type="checkbox"
											checked={cacheControlEnabled}
											onChange={e => {
												const nextEnabled = e.target.checked;
												setCacheControlEnabled(nextEnabled);
												if (nextEnabled && !cacheControlKind) {
													setCacheControlKind(supportedCacheKinds[0] ?? '');
												}
											}}
											className="toggle toggle-accent"
										/>
									) : (
										<span className="text-sm opacity-70">Manual top-level cache control is not available.</span>
									)}
									{supportsAutomaticProviderCaching && (
										<div className="label">
											<span className="label-text-alt opacity-70">
												This provider SDK also supports automatic caching behavior.
											</span>
										</div>
									)}
								</div>
							</div>

							{supportsManualCacheControl && cacheControlEnabled && (
								<>
									<div className="grid grid-cols-12 items-center gap-2">
										<label className="label col-span-4">
											<span className="label-text text-sm">Cache Kind</span>
										</label>
										<div className="col-span-8">
											<Dropdown<CacheControlKind>
												dropdownItems={cacheControlKindItems}
												selectedKey={(cacheControlKind || supportedCacheKinds[0]) as CacheControlKind}
												onChange={kind => {
													setCacheControlKind(kind);
												}}
												filterDisabled={false}
												title="Select Cache Kind"
												getDisplayName={k => cacheControlKindItems[k].displayName}
											/>
										</div>
									</div>

									<div className="grid grid-cols-12 items-center gap-2">
										<label className="label col-span-4">
											<span className="label-text text-sm">Cache TTL</span>
										</label>
										<div className="col-span-8">
											<Dropdown<CacheControlTTLSelection>
												dropdownItems={cacheControlTTLItems}
												selectedKey={cacheControlTTL}
												onChange={ttl => {
													setCacheControlTTL(ttl);
												}}
												filterDisabled={false}
												title="Select Cache TTL"
												getDisplayName={k => cacheControlTTLItems[k].displayName}
											/>
										</div>
									</div>

									{supportsCacheKey && (
										<div className="grid grid-cols-12 items-center gap-2">
											<label className="label col-span-4">
												<span className="label-text text-sm">Cache Key</span>
											</label>
											<div className="col-span-8">
												<input
													type="text"
													value={cacheControlKey}
													onChange={e => {
														setCacheControlKey(e.target.value);
													}}
													className="input input-bordered w-full rounded-xl"
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
							<label className="label col-span-4">
								<span className="label-text text-sm">Stop Sequences</span>
								<HelpHint
									content={
										stopSequencesDisabledBecauseReasoning
											? 'Stop sequences are disabled when reasoning is enabled for this model.'
											: `One per line. Empty = none. Max ${stopPolicy.maxSequences}.`
									}
								/>
							</label>
							<div className="col-span-8">
								<textarea
									disabled={stopSequencesDisabledBecauseReasoning}
									className={`textarea textarea-bordered w-full rounded-xl ${
										errors.stopSequences ? 'textarea-error' : ''
									} ${stopSequencesDisabledBecauseReasoning ? 'cursor-not-allowed opacity-50' : ''}`}
									rows={4}
									value={stopSequencesText}
									onChange={e => {
										const v = e.target.value;
										setStopSequencesText(v);
										setErrors(prev => ({ ...prev, stopSequences: validateStopSequences(v) }));
									}}
									placeholder={'e.g.\n\nEND\n</final>'}
									spellCheck="false"
								/>
								{stopSequencesDisabledBecauseReasoning && (
									<div className="label">
										<span className="label-text-alt opacity-70">
											Disabled because reasoning is enabled for this model/provider.
										</span>
									</div>
								)}
								{errors.stopSequences && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.stopSequences}
										</span>
									</div>
								)}
							</div>
						</div>
					)}

					<div className="modal-action">
						<button type="button" className="btn bg-base-300 rounded-xl" onClick={requestClose}>
							Cancel
						</button>
						<button type="submit" className="btn btn-primary rounded-xl" disabled={formHasErrors}>
							Save
						</button>
					</div>
				</form>
			</div>
		</dialog>
	);
}

export function AdvancedParamsModal({
	isOpen,
	onClose,
	currentModel,
	effectiveReasoningEnabled,
	onSave,
}: AdvancedParamsModalProps) {
	if (!isOpen || typeof document === 'undefined') return null;

	const modelIdentity = `${currentModel.providerName}::${currentModel.modelPresetID}`;

	return createPortal(
		<AdvancedParamsModalInner
			key={modelIdentity}
			onClose={onClose}
			currentModel={currentModel}
			effectiveReasoningEnabled={effectiveReasoningEnabled}
			onSave={onSave}
		/>,
		document.body
	);
}
