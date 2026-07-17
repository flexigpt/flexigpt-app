import type { SubmitEventHandler } from 'react';
import { useEffect, useMemo, useRef, useState } from 'react';

import { FiAlertCircle, FiRefreshCcw, FiTool } from 'react-icons/fi';

import { focusTextInputAtEnd } from '@/lib/focus_input';
import type { JSONObject, JSONSchema } from '@/lib/jsonschema_utils';
import {
	buildExampleFromDraft7Schema,
	getJSONObject,
	getPropertiesFromJSONSchema,
	getRequiredFromJSONSchema,
} from '@/lib/jsonschema_utils';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

import { MessageContentCard } from '@/chats/messages/message_content_card';
import { computeToolUserArgsStatus } from '@/tools/lib/tool_userargs_utils';

interface ToolUserArgsModalProps {
	isOpen: boolean;
	onClose: () => void;

	toolLabel: string;
	schema: JSONSchema | undefined;
	existingInstance?: string;
	modalIdentity?: string;
	onSave: (newInstance: string) => void;
}

interface FormState {
	rawJson: string;
}

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

function getInitialRawJson(existingInstance?: string): string {
	if (existingInstance && existingInstance.trim() !== '') {
		try {
			const parsed = JSON.parse(existingInstance);
			return JSON.stringify(parsed, null, 2);
		} catch {
			return existingInstance;
		}
	}

	return '{}';
}

function ToolUserArgsModalContent({
	toolLabel,
	schema,
	existingInstance,
	onSave,
}: Omit<ToolUserArgsModalProps, 'isOpen' | 'onClose'>) {
	const { requestClose } = useModalDialogController();
	const textareaRef = useRef<HTMLTextAreaElement | null>(null);

	const [formData, setFormData] = useState<FormState>(() => ({
		rawJson: getInitialRawJson(existingInstance),
	}));
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

	useEffect(() => {
		let raf1 = 0;
		let raf2 = 0;
		raf1 = window.requestAnimationFrame(() => {
			raf2 = window.requestAnimationFrame(() => {
				focusTextInputAtEnd(textareaRef.current);
			});
		});
		return () => {
			window.cancelAnimationFrame(raf1);
			window.cancelAnimationFrame(raf2);
		};
	}, []);

	const handleFormat = () => {
		const raw = formData.rawJson.trim();
		if (!raw) {
			setFormData({ rawJson: '{}' });
			setError(null);
			return;
		}

		try {
			const parsed = JSON.parse(raw);
			setFormData({ rawJson: JSON.stringify(parsed, null, 2) });
			setError(null);
		} catch (err) {
			setError((err as Error).message || 'Invalid JSON. Please fix it and try again.');
		}
	};

	const handleUseExample = () => {
		setFormData({ rawJson: examplePretty });
		setError(null);
	};

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();
		e.stopPropagation();

		let raw = formData.rawJson.trim();
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
		requestClose();
	};

	return (
		<>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-[80vw] min-w-0 overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<ModalHeader
						title={
							<span className="flex items-center gap-2">
								<FiTool size={16} />
								<span>Tool options</span>
								<span className="badge badge-neutral">{toolLabel}</span>
							</span>
						}
						onClose={requestClose}
					/>

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
									<span className="text-sm">Options (JSON)</span>
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
								ref={textareaRef}
								className={`textarea w-full rounded-xl font-mono text-xs ${error ? 'textarea-error' : ''}`}
								rows={12}
								value={formData.rawJson}
								onChange={e => {
									setFormData({ rawJson: e.target.value });
									if (error) {
										setError(null);
									}
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
							<button type="submit" className="btn btn-primary rounded-xl">
								Save
							</button>
						</ModalActions>
					</form>
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</>
	);
}

export function ToolUserArgsModal(props: ToolUserArgsModalProps) {
	if (!props.isOpen) {
		return null;
	}

	const remountKey = JSON.stringify({
		modalIdentity: props.modalIdentity ?? props.toolLabel,
		existingInstance: props.existingInstance ?? null,
	});

	return (
		<ModalDialog isOpen={props.isOpen} onClose={props.onClose}>
			<ToolUserArgsModalContent key={remountKey} {...props} />
		</ModalDialog>
	);
}
