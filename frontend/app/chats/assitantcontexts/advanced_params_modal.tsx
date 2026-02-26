import {
	type Dispatch,
	type SetStateAction,
	type SubmitEventHandler,
	useEffect,
	useMemo,
	useRef,
	useState,
} from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiX } from 'react-icons/fi';

import { type JSONSchemaParam, OutputFormatKind, type OutputParam, ReasoningSummaryStyle } from '@/spec/inference';
import type { UIChatOption } from '@/spec/modelpreset';

import { Dropdown } from '@/components/dropdown';

type AdvancedParamsModalProps = {
	isOpen: boolean;
	onClose: () => void;
	currentModel: UIChatOption;
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

const MAX_STOP_SEQUENCES = 16;
const MAX_JSON_CHARS = 50_000;

const outputFormatItems: Record<OutputFormatChoice, { isEnabled: boolean; displayName: string }> = {
	default: { isEnabled: true, displayName: 'Default' },
	text: { isEnabled: true, displayName: 'Text' },
	jsonSchema: { isEnabled: true, displayName: 'JSON (schema)' },
};

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

export function AdvancedParamsModal({ isOpen, onClose, currentModel, onSave }: AdvancedParamsModalProps) {
	// Reset only when opening OR switching to a different model preset identity.
	const modelIdentity = useMemo(
		() => `${currentModel.providerName}::${currentModel.modelPresetID}`,
		[currentModel.providerName, currentModel.modelPresetID]
	);

	/* local form state (strings for easy blank entry) */
	const [stream, setStream] = useState(false);
	const [maxPromptLength, setMaxPromptLength] = useState('');
	const [maxOutputLength, setMaxOutputLength] = useState('');
	const [timeoutSec, setTimeoutSec] = useState('');

	const [reasoningSummaryStyle, setReasoningSummaryStyle] = useState<SummaryStyleChoice>('');

	// output format (verbosity is controlled via bar dropdown)
	const [outputFormatChoice, setOutputFormatChoice] = useState<OutputFormatChoice>('default');
	const [jsonSchemaName, setJsonSchemaName] = useState('');
	const [jsonSchemaDescription, setJsonSchemaDescription] = useState('');
	const [jsonSchemaStrict, setJsonSchemaStrict] = useState(true);
	const [jsonSchemaText, setJsonSchemaText] = useState('');

	// stop sequences + raw additional JSON
	const [stopSequencesText, setStopSequencesText] = useState('');
	const [additionalParametersRawJSON, setAdditionalParametersRawJSON] = useState('');

	/* validation errors */
	const [errors, setErrors] = useState<Partial<Record<ErrorKey, string>>>({});

	const dialogRef = useRef<HTMLDialogElement | null>(null);

	// Whether reasoning is active for this model (controls summary style availability)
	const reasoningEnabled = !!currentModel.reasoning;

	const reasoningSummaryStyleItems: Record<SummaryStyleChoice, { isEnabled: boolean; displayName: string }> = useMemo(
		() => ({
			'': { isEnabled: true, displayName: 'Default' },
			[ReasoningSummaryStyle.Auto]: { isEnabled: reasoningEnabled, displayName: 'Auto' },
			[ReasoningSummaryStyle.Concise]: { isEnabled: reasoningEnabled, displayName: 'Concise' },
			[ReasoningSummaryStyle.Detailed]: { isEnabled: reasoningEnabled, displayName: 'Detailed' },
		}),
		[reasoningEnabled]
	);

	useEffect(() => {
		if (!isOpen) return;

		setStream(currentModel.stream);
		setMaxPromptLength(String(currentModel.maxPromptLength));
		setMaxOutputLength(String(currentModel.maxOutputLength));
		setTimeoutSec(String(currentModel.timeout));

		// summary style seed (only meaningful if reasoning exists)
		setReasoningSummaryStyle((currentModel.reasoning?.summaryStyle as SummaryStyleChoice) ?? '');

		// output format seed
		const kind = currentModel.outputParam?.format?.kind;
		if (kind === OutputFormatKind.Text) setOutputFormatChoice('text');
		else if (kind === OutputFormatKind.JSONSchema) setOutputFormatChoice('jsonSchema');
		else setOutputFormatChoice('default');

		const jsp = currentModel.outputParam?.format?.jsonSchemaParam;
		setJsonSchemaName(jsp?.name ?? '');
		setJsonSchemaDescription(jsp?.description ?? '');
		setJsonSchemaStrict(jsp?.strict ?? true);
		setJsonSchemaText(jsp?.schema ? JSON.stringify(jsp.schema, null, 2) : '');

		// stop sequences
		setStopSequencesText((currentModel.stopSequences ?? []).join('\n'));

		// raw JSON
		setAdditionalParametersRawJSON(currentModel.additionalParametersRawJSON ?? '');

		setErrors({});
	}, [isOpen, modelIdentity, currentModel]);

	// Open the dialog natively when isOpen becomes true
	useEffect(() => {
		if (!isOpen) return;

		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) {
			dialog.showModal();
		}

		return () => {
			// If the component unmounts while the dialog is still open, close it.
			if (dialog.open) {
				dialog.close();
			}
		};
	}, [isOpen]);

	// Sync parent state whenever the dialog is closed (Esc or dialog.close()).
	const handleDialogClose = () => {
		onClose();
	};

	const validateNumberField = (field: 'maxPromptLength' | 'maxOutputLength' | 'timeout', value: string) => {
		const n = parsePositiveIntAllowBlank(value);
		if (n === undefined) return undefined; // blank means "leave as-is"
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
		const parsed = parseStopSequences(raw);
		if (!parsed) return undefined;

		if (parsed.length > MAX_STOP_SEQUENCES) return `Too many stop sequences (max ${MAX_STOP_SEQUENCES}).`;
		const tooLong = parsed.find(s => s.length > 256);
		if (tooLong) return 'A stop sequence is too long (max 256 chars per line).';
		return undefined;
	};

	const validateAdditionalRawJSON = (raw: string): string | undefined => {
		const s = raw.trim();
		if (!s) return undefined;
		if (s.length > MAX_JSON_CHARS) return `JSON too large (max ${MAX_JSON_CHARS} chars).`;

		const parsed = safeJSONParse(s);
		if (!parsed.ok) return `Invalid JSON: ${parsed.value}`;

		// hardened: require object (provider "additional params" is expected to be a JSON object)
		if (!isPlainObject(parsed.value)) return 'Additional params must be a JSON object (e.g. { "top_p": 0.9 }).';

		return undefined;
	};

	const validateJSONSchema = (choice: OutputFormatChoice) => {
		if (choice !== 'jsonSchema') {
			return { nameErr: undefined as string | undefined, schemaErr: undefined as string | undefined };
		}

		const name = jsonSchemaName.trim();
		if (!name) return { nameErr: 'Schema name is required.', schemaErr: undefined };

		const schemaRaw = jsonSchemaText.trim();
		if (!schemaRaw) return { nameErr: undefined, schemaErr: 'Schema JSON is required.' };
		if (schemaRaw.length > MAX_JSON_CHARS) {
			return { nameErr: undefined, schemaErr: `Schema JSON too large (max ${MAX_JSON_CHARS} chars).` };
		}

		const parsed = safeJSONParse(schemaRaw);
		if (!parsed.ok) return { nameErr: undefined, schemaErr: `Invalid schema JSON: ${parsed.value}` };

		if (!isPlainObject(parsed.value)) return { nameErr: undefined, schemaErr: 'Schema must be a JSON object.' };

		return { nameErr: undefined, schemaErr: undefined };
	};

	const formHasErrors = Object.values(errors).some(Boolean);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();

		const maxPromptErr = validateNumberField('maxPromptLength', maxPromptLength);
		const maxOutputErr = validateNumberField('maxOutputLength', maxOutputLength);
		const timeoutErr = validateNumberField('timeout', timeoutSec);

		const stopErr = validateStopSequences(stopSequencesText);
		const addJsonErr = validateAdditionalRawJSON(additionalParametersRawJSON);

		const { nameErr, schemaErr } = validateJSONSchema(outputFormatChoice);

		const nextErrors: Partial<Record<ErrorKey, string>> = {
			maxPromptLength: maxPromptErr,
			maxOutputLength: maxOutputErr,
			timeout: timeoutErr,
			stopSequences: stopErr,
			additionalParametersRawJSON: addJsonErr,
			jsonSchemaName: nameErr,
			jsonSchema: schemaErr,
		};

		if (Object.values(nextErrors).some(Boolean)) {
			setErrors(nextErrors);
			return;
		}

		// Build outputParam.format (preserve existing verbosity from bar dropdown)
		const baseOutputParam: OutputParam | undefined = currentModel.outputParam
			? { ...currentModel.outputParam }
			: undefined;

		let nextFormat: OutputParam['format'] = undefined;

		if (outputFormatChoice === 'text') {
			nextFormat = { kind: OutputFormatKind.Text };
		} else if (outputFormatChoice === 'jsonSchema') {
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

		const mergedOutputParam: OutputParam | undefined = (() => {
			const merged: OutputParam = { ...(baseOutputParam ?? {}) };

			if (nextFormat === undefined) delete merged.format;
			else merged.format = nextFormat;

			// Keep verbosity untouched here (bar controls it)
			const hasAny = !!merged.verbosity || !!merged.format;
			return hasAny ? merged : undefined;
		})();

		const mergedReasoning = (() => {
			if (!currentModel.reasoning) return currentModel.reasoning; // undefined
			const r = { ...currentModel.reasoning };
			if (!reasoningSummaryStyle) {
				delete r.summaryStyle;
			} else {
				r.summaryStyle = reasoningSummaryStyle;
			}
			return r as UIChatOption['reasoning'];
		})();

		const updatedModel: UIChatOption = {
			...currentModel,

			stream,
			maxPromptLength: parsePositiveIntAllowBlank(maxPromptLength) ?? currentModel.maxPromptLength,
			maxOutputLength: parsePositiveIntAllowBlank(maxOutputLength) ?? currentModel.maxOutputLength,
			timeout: parsePositiveIntAllowBlank(timeoutSec) ?? currentModel.timeout,
			reasoning: mergedReasoning,
			outputParam: mergedOutputParam,
			stopSequences: parseStopSequences(stopSequencesText),
			additionalParametersRawJSON: additionalParametersRawJSON.trim() ? additionalParametersRawJSON.trim() : undefined,
		};

		onSave(updatedModel);
		dialogRef.current?.close();
	};

	if (!isOpen) return null;

	return createPortal(
		<dialog ref={dialogRef} className="modal" onClose={handleDialogClose}>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-3xl overflow-auto rounded-2xl">
				<div className="mb-4 flex items-center justify-between">
					<h3 className="text-lg font-bold">Advanced Model Parameters</h3>
					<button
						type="button"
						className="btn btn-sm btn-circle bg-base-300"
						onClick={() => dialogRef.current?.close()}
						aria-label="Close"
					>
						<FiX size={12} />
					</button>
				</div>

				<form onSubmit={handleSubmit} className="space-y-4">
					{/* stream */}
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-4 cursor-pointer">
							<span className="label-text text-sm">Streaming</span>
							<span className="label-text-alt tooltip tooltip-right" data-tip="Stream data continuously.">
								<FiHelpCircle size={12} />
							</span>
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

					{/* max prompt */}
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-4">
							<span className="label-text text-sm">Max Prompt Tokens</span>
							<span className="label-text-alt tooltip tooltip-right" data-tip="Maximum tokens for input prompt">
								<FiHelpCircle size={12} />
							</span>
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

					{/* max output */}
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-4">
							<span className="label-text text-sm">Max Output Tokens</span>
							<span className="label-text-alt tooltip tooltip-right" data-tip="Maximum tokens for model output">
								<FiHelpCircle size={12} />
							</span>
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

					{/* timeout */}
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-4">
							<span className="label-text text-sm">Timeout (s)</span>
							<span
								className="label-text-alt tooltip tooltip-right"
								data-tip="Maximum time a request can take (seconds)"
							>
								<FiHelpCircle size={12} />
							</span>
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

					{/* Reasoning summary style (converted to inbuilt Dropdown) */}
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-4">
							<span className="label-text text-sm">Reasoning Summary</span>
							<span
								className="label-text-alt tooltip tooltip-right"
								data-tip={
									reasoningEnabled
										? 'Controls whether the model produces a reasoning summary (when supported).'
										: 'This model is currently using temperature mode (no reasoning params active).'
								}
							>
								<FiHelpCircle size={12} />
							</span>
						</label>
						<div className="col-span-8">
							<Dropdown<SummaryStyleChoice>
								dropdownItems={reasoningSummaryStyleItems}
								selectedKey={reasoningSummaryStyle}
								onChange={k => {
									// If reasoning isn't enabled, only "Default" is enabled in the items list.
									setReasoningSummaryStyle(k);
								}}
								filterDisabled={false}
								title="Select Reasoning Summary Style"
								getDisplayName={k => reasoningSummaryStyleItems[k].displayName}
							/>
						</div>
					</div>

					{/* Output format (converted to inbuilt Dropdown) */}
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-4">
							<span className="label-text text-sm">Output Format</span>
							<span
								className="label-text-alt tooltip tooltip-right"
								data-tip="Controls output formatting. Verbosity is set in the top bar."
							>
								<FiHelpCircle size={12} />
							</span>
						</label>
						<div className="col-span-8">
							<Dropdown<OutputFormatChoice>
								dropdownItems={outputFormatItems}
								selectedKey={outputFormatChoice}
								onChange={k => {
									setOutputFormatChoice(k);
								}}
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
											setJsonSchemaName(e.target.value);
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
											setJsonSchemaText(e.target.value);
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

					{/* stop sequences */}
					<div className="grid grid-cols-12 items-start gap-2">
						<label className="label col-span-4">
							<span className="label-text text-sm">Stop Sequences</span>
							<span className="label-text-alt tooltip tooltip-right" data-tip="One per line. Empty = none.">
								<FiHelpCircle size={12} />
							</span>
						</label>
						<div className="col-span-8">
							<textarea
								className={`textarea textarea-bordered w-full rounded-xl ${errors.stopSequences ? 'textarea-error' : ''}`}
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
							{errors.stopSequences && (
								<div className="label">
									<span className="label-text-alt text-error flex items-center gap-1">
										<FiAlertCircle size={12} /> {errors.stopSequences}
									</span>
								</div>
							)}
						</div>
					</div>

					{/* additional raw JSON */}
					<div className="grid grid-cols-12 items-start gap-2">
						<label className="label col-span-4">
							<span className="label-text text-sm">Additional Params JSON</span>
							<span
								className="label-text-alt tooltip tooltip-right"
								data-tip="Raw provider-specific JSON object. Must be valid JSON."
							>
								<FiHelpCircle size={12} />
							</span>
						</label>
						<div className="col-span-8">
							<textarea
								className={`textarea textarea-bordered w-full rounded-xl ${
									errors.additionalParametersRawJSON ? 'textarea-error' : ''
								}`}
								rows={6}
								value={additionalParametersRawJSON}
								onChange={e => {
									const v = e.target.value;
									setAdditionalParametersRawJSON(v);
									setErrors(prev => ({ ...prev, additionalParametersRawJSON: validateAdditionalRawJSON(v) }));
								}}
								placeholder={'{\n  "top_p": 0.9,\n  "presence_penalty": 0.2\n}'}
								spellCheck="false"
							/>
							{errors.additionalParametersRawJSON && (
								<div className="label">
									<span className="label-text-alt text-error flex items-center gap-1">
										<FiAlertCircle size={12} /> {errors.additionalParametersRawJSON}
									</span>
								</div>
							)}
						</div>
					</div>

					<div className="modal-action">
						<button type="button" className="btn bg-base-300 rounded-xl" onClick={() => dialogRef.current?.close()}>
							Cancel
						</button>
						<button type="submit" className="btn btn-primary rounded-xl" disabled={formHasErrors}>
							Save
						</button>
					</div>
				</form>
			</div>
		</dialog>,
		document.body
	);
}
