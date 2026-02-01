import { type FormEvent, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiRefreshCcw, FiTool, FiX } from 'react-icons/fi';

import {
	buildExampleFromDraft7Schema,
	getJSONObject,
	getPropertiesFromJSONSchema,
	getRequiredFromJSONSchema,
	type JSONObject,
	type JSONSchema,
} from '@/lib/jsonschema_utils';

import { ModalBackdrop } from '@/components/modal_backdrop';

import { MessageContentCard } from '@/chats/messages/message_content_card';
import { computeToolUserArgsStatus } from '@/chats/tools/tool_editor_utils';

interface ToolUserArgsModalProps {
	isOpen: boolean;
	onClose: () => void;

	toolLabel: string;
	schema: JSONSchema | undefined;
	existingInstance?: string;
	onSave: (newInstance: string) => void;
}

type FormState = {
	rawJson: string;
};

// Shared type for "what args editor is currently open?"
export type ToolArgsTarget =
	| { kind: 'attached'; selectionID: string }
	| { kind: 'conversation'; key: string }
	| { kind: 'webSearch' };

function toPrettyJSON(value: unknown): string {
	try {
		return JSON.stringify(value, null, 2);
	} catch {
		return String(value);
	}
}

function asJsonMarkdownBlock(json: string): string {
	return `\`\`\`json\n${json}\n\`\`\``;
}

export function ToolUserArgsModal({
	isOpen,
	onClose,
	toolLabel,
	schema,
	existingInstance,
	onSave,
}: ToolUserArgsModalProps) {
	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const [form, setForm] = useState<FormState>({ rawJson: '' });
	const [error, setError] = useState<string | null>(null);

	const schemaObj = useMemo(() => getJSONObject(schema), [schema]);
	const properties = useMemo(() => getPropertiesFromJSONSchema(schema) ?? ({} as JSONObject), [schema]);
	const requiredKeys = useMemo(() => getRequiredFromJSONSchema(schema) ?? [], [schema]);

	const allKeys = useMemo(() => Object.keys(properties), [properties]);
	const optionalKeys = useMemo(() => allKeys.filter(k => !requiredKeys.includes(k)), [allKeys, requiredKeys]);

	const exampleInstanceObj = useMemo(() => (schemaObj ? buildExampleFromDraft7Schema(schemaObj) : {}), [schemaObj]);

	const schemaPretty = useMemo(() => (schemaObj ? toPrettyJSON(schemaObj) : null), [schemaObj]);
	const examplePretty = useMemo(() => toPrettyJSON(exampleInstanceObj), [exampleInstanceObj]);

	const schemaMarkdown = useMemo(() => (schemaPretty ? asJsonMarkdownBlock(schemaPretty) : null), [schemaPretty]);
	const exampleMarkdown = useMemo(() => asJsonMarkdownBlock(examplePretty), [examplePretty]);

	// Initialize editor content whenever modal opens
	useEffect(() => {
		if (!isOpen) return;

		let initial = '';

		if (existingInstance && existingInstance.trim() !== '') {
			try {
				const parsed = JSON.parse(existingInstance);
				initial = JSON.stringify(parsed, null, 2);
			} catch {
				initial = existingInstance;
			}
		} else {
			initial = '{}';
		}

		setForm({ rawJson: initial });
		setError(null);
	}, [isOpen, existingInstance, schemaObj, examplePretty]);

	// Manage native <dialog> lifecycle
	useEffect(() => {
		if (!isOpen) return;

		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) dialog.showModal();

		return () => {
			if (dialog.open) dialog.close();
		};
	}, [isOpen]);

	const handleDialogClose = () => {
		onClose();
	};

	const handleFormat = () => {
		const raw = form.rawJson.trim();
		if (!raw) {
			setForm({ rawJson: '{}' });
			setError(null);
			return;
		}

		try {
			const parsed = JSON.parse(raw);
			setForm({ rawJson: JSON.stringify(parsed, null, 2) });
			setError(null);
		} catch (err) {
			setError((err as Error).message || 'Invalid JSON. Please fix it and try again.');
		}
	};

	const handleUseExample = () => {
		setForm({ rawJson: examplePretty });
		setError(null);
	};

	const handleSubmit = (e: FormEvent) => {
		e.preventDefault();
		e.stopPropagation();

		let raw = form.rawJson.trim();
		if (!raw) {
			if (requiredKeys.length > 0) {
				setError('This tool requires options. Provide a JSON object with the required keys.');
				return;
			}
			raw = '{}';
		}

		let parsed: unknown;
		try {
			parsed = JSON.parse(raw);
		} catch (err) {
			setError((err as Error).message || 'Invalid JSON. Please fix it and try again.');
			return;
		}

		if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
			setError('Expected a JSON object (e.g. `{ "key": "value" }`).');
			return;
		}

		const status = computeToolUserArgsStatus(schema, JSON.stringify(parsed));
		if (status.hasSchema && !status.isSatisfied) {
			setError(`Missing required keys: ${status.missingRequired.join(', ')}. Populate them (non-empty) before saving.`);
			return;
		}

		onSave(JSON.stringify(parsed, null, 2));
		dialogRef.current?.close();
	};

	if (!isOpen) return null;

	return createPortal(
		<dialog ref={dialogRef} className="modal" onClose={handleDialogClose}>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-[80vw] min-w-0 overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					{/* header */}
					<div className="mb-4 flex items-center justify-between gap-2">
						<h3 className="flex items-center gap-2 text-lg font-bold">
							<FiTool size={16} />
							<span>Tool options</span>
							<span className="badge badge-neutral">{toolLabel}</span>
						</h3>

						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={() => dialogRef.current?.close()}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>

					{/* stacked content */}
					<form
						noValidate
						onSubmit={handleSubmit}
						className="flex flex-col gap-4"
						onKeyDownCapture={e => {
							e.stopPropagation();
						}}
						onKeyUpCapture={e => {
							e.stopPropagation();
						}}
					>
						{/* small summary (optional but useful) */}
						<div className="space-y-1 text-xs">
							{schemaObj ? (
								<>
									{requiredKeys.length > 0 ? (
										<div>
											<span className="font-semibold">Required keys:</span> {requiredKeys.join(', ')}
										</div>
									) : (
										<div>There are no required keys for this tool.</div>
									)}
									{optionalKeys.length > 0 && (
										<div>
											<span className="font-semibold">Optional keys:</span> {optionalKeys.join(', ')}
										</div>
									)}
								</>
							) : (
								<div>This tool does not define a user configuration schema.</div>
							)}
						</div>

						{/* Editor */}
						<div className="space-y-2">
							<div className="flex items-center justify-between">
								<label className="label p-1">
									<span className="label-text text-sm">Options (JSON)</span>
								</label>

								<div className="flex items-center gap-2">
									<button
										type="button"
										className="btn btn-xs bg-base-300 rounded-lg"
										onClick={handleFormat}
										title="Format JSON"
									>
										<FiRefreshCcw size={12} />
										<span className="ml-1">Format</span>
									</button>

									{schemaObj && (
										<button
											type="button"
											className="btn btn-xs bg-base-300 rounded-lg"
											onClick={handleUseExample}
											title="Replace editor contents with the full schema-derived example"
										>
											Use example
										</button>
									)}
								</div>
							</div>

							<textarea
								className={`textarea textarea-bordered w-full rounded-xl font-mono text-xs ${
									error ? 'textarea-error' : ''
								}`}
								rows={12}
								value={form.rawJson}
								onChange={e => {
									setForm({ rawJson: e.target.value });
									if (error) setError(null);
								}}
								spellCheck={false}
								placeholder={schemaObj ? '{\n  "key": "value"\n}' : 'JSON object; structure depends on this tool.'}
							/>

							<div className="min-h-6 text-xs">
								{error && (
									<span className="text-error flex items-center gap-1">
										<FiAlertCircle size={12} /> {error}
									</span>
								)}
							</div>
						</div>

						{/* Example block (full) */}
						<div className="bg-base-300 rounded-xl p-3">
							<div className="mb-2 text-sm font-semibold">Example options (all keys)</div>
							<MessageContentCard
								messageID={`tool-args:example:${toolLabel}`}
								content={exampleMarkdown}
								streamedText=""
								isStreaming={false}
								isBusy={false}
								align="items-start text-left"
								renderAsMarkdown={true}
							/>
							<div className="mt-2 text-xs opacity-80">
								Full example includes all schema properties. Defaults/enums are respected when present.
							</div>
						</div>

						{/* JSON Schema block */}
						<div className="bg-base-300 rounded-xl p-3">
							<div className="mb-2 text-sm font-semibold">JSON Schema</div>
							{schemaMarkdown ? (
								<MessageContentCard
									messageID={`tool-args:schema:${toolLabel}`}
									content={schemaMarkdown}
									streamedText=""
									isStreaming={false}
									isBusy={false}
									align="items-start text-left"
									renderAsMarkdown={true}
								/>
							) : (
								<div className="text-xs opacity-80">No schema provided for this tool.</div>
							)}
						</div>

						{/* footer */}
						<div className="modal-action">
							<button type="button" className="btn bg-base-300 rounded-xl" onClick={() => dialogRef.current?.close()}>
								Cancel
							</button>
							<button type="submit" className="btn btn-primary rounded-xl">
								Save
							</button>
						</div>
					</form>
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</dialog>,
		document.body
	);
}
