import type { ChangeEvent, SubmitEventHandler } from 'react';
import { useEffect, useMemo, useRef, useState } from 'react';

import { FiAlertCircle, FiAlertTriangle, FiHelpCircle, FiUpload, FiX } from 'react-icons/fi';

import type { Tool } from '@/spec/tool';
import { HTTPBodyOutputMode, ToolImplType } from '@/spec/tool';

import {
	omitSensitiveHTTPHeaders,
	parseHTTPHeadersJSON,
	parseHTTPStatusCodes,
	parseStringRecordJSON,
	REDACTED_HTTP_VALUE,
	redactSensitiveHTTPHeaders,
	restoreRedactedHTTPHeaders,
	validateHTTPURLTemplateSecurity,
} from '@/lib/http_input_utils';
import type { JSONSchema } from '@/lib/jsonschema_utils';
import { omitManyKeys } from '@/lib/obj_utils';
import { validateSlug, validateTags } from '@/lib/text_utils';
import { MessageEnterValidURL, validateUrlForInput } from '@/lib/url_utils';
import { DEFAULT_SEMVER, isSemverVersion, suggestNextMinorVersion } from '@/lib/version_utils';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { Dropdown } from '@/components/dropdown';
import { MANAGEMENT_MODAL_FORM_CLASS } from '@/components/managementui/management_class_consts';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';
import { ReadOnlyValue } from '@/components/read_only_value';

interface ToolItem {
	tool: Tool;
	bundleID: string;
	toolSlug: string;
}

type ModalMode = 'add' | 'edit' | 'view';

interface AddEditToolModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (toolData: Partial<Tool>) => Promise<void>;
	initialData?: ToolItem;
	existingTools: ToolItem[];
	mode?: ModalMode;
}

type AddEditToolModalContentProps = Omit<AddEditToolModalProps, 'isOpen'>;

const TOOL_TYPE_LABEL_GO = 'Go';
const TOOL_TYPE_LABEL_HTTP = 'HTTP';
const TOOL_TYPE_LABEL_SDK = 'SDK';

interface ErrorState {
	displayName?: string;
	slug?: string;
	version?: string;
	type?: string;
	argSchema?: string;
	goFunc?: string;
	httpUrl?: string;
	httpHeaders?: string;
	httpQuery?: string;
	httpResponseCodes?: string;
	httpTimeoutMS?: string;
	httpAuth?: string;
	tags?: string;
}

const bodyOutputModeItems: Record<HTTPBodyOutputMode, { isEnabled: boolean; displayName: string }> = {
	[HTTPBodyOutputMode.Auto]: { isEnabled: true, displayName: 'Auto' },
	[HTTPBodyOutputMode.Text]: { isEnabled: true, displayName: 'Text' },
	[HTTPBodyOutputMode.File]: { isEnabled: true, displayName: 'File' },
	[HTTPBodyOutputMode.Image]: { isEnabled: true, displayName: 'Image' },
};

const toolTypeDropdownItems: Record<ToolImplType, { isEnabled: boolean; displayName: string }> = {
	[ToolImplType.Go]: { isEnabled: false, displayName: TOOL_TYPE_LABEL_GO },
	[ToolImplType.HTTP]: { isEnabled: true, displayName: TOOL_TYPE_LABEL_HTTP },
	[ToolImplType.SDK]: { isEnabled: false, displayName: TOOL_TYPE_LABEL_SDK },
};

const EMPTY_FORM_DATA = {
	displayName: '',
	slug: '',
	version: DEFAULT_SEMVER,
	description: '',
	tags: '',
	isEnabled: true,

	userCallable: true,
	llmCallable: true,
	autoExecReco: false,

	type: ToolImplType.HTTP as ToolImplType,
	argSchema: '{}',

	goFunc: '',

	httpUrl: '',
	httpMethod: 'GET',
	httpHeaders: '{}',
	httpQuery: '{}',
	httpBody: '',
	httpAuthType: '',
	httpAuthIn: '',
	httpAuthName: '',
	httpAuthValueTemplate: '',
	httpResponseCodes: '',
	httpResponseErrorMode: '',
	httpResponseBodyOutputMode: HTTPBodyOutputMode.Auto as HTTPBodyOutputMode,
	httpTimeoutMS: '',
};

type ToolFormData = typeof EMPTY_FORM_DATA;

function validateHTTPToolURL(raw: string, input: HTMLInputElement | null) {
	const result = validateUrlForInput(raw, input, {
		required: true,
		requiredMessage: 'HTTP URL is required.',
	});

	return {
		...result,
		error:
			result.error ??
			(result.normalized ? validateHTTPURLTemplateSecurity(result.normalized, 'HTTP tool URL') : MessageEnterValidURL),
	};
}

function buildInitialFormData(
	initialData: ToolItem | undefined,
	existingTools: ToolItem[],
	isEditMode: boolean,
	isViewMode = false
): ToolFormData {
	if (!initialData) {
		return { ...EMPTY_FORM_DATA };
	}

	const t = initialData.tool;
	const existingVersionsForSlug = existingTools.filter(x => x.tool.slug === t.slug).map(x => x.tool.version);
	const nextV = isEditMode ? suggestNextMinorVersion(t.version, existingVersionsForSlug).suggested : t.version;
	const protectSensitiveValues = isEditMode || isViewMode;

	return {
		displayName: t.displayName,
		slug: t.slug,
		version: nextV,

		description: t.description ?? '',
		tags: (t.tags ?? []).join(', '),
		isEnabled: t.isEnabled,

		userCallable: t.userCallable,
		llmCallable: t.llmCallable,
		autoExecReco: t.autoExecReco,

		type: t.type,
		argSchema: JSON.stringify(t.argSchema ?? {}, null, 2),

		goFunc: t.goImpl?.func ?? '',

		httpUrl: t.httpImpl?.request.urlTemplate ?? '',
		httpMethod: t.httpImpl?.request.method ?? 'GET',
		httpHeaders: JSON.stringify(
			protectSensitiveValues
				? (redactSensitiveHTTPHeaders(t.httpImpl?.request.headers) ?? {})
				: (t.httpImpl?.request.headers ?? {}),
			null,
			2
		),
		httpQuery: JSON.stringify(t.httpImpl?.request.query ?? {}, null, 2),
		httpBody: t.httpImpl?.request.body ?? '',
		httpAuthType: t.httpImpl?.request.auth?.type ?? '',
		httpAuthIn: t.httpImpl?.request.auth?.in ?? '',
		httpAuthName: t.httpImpl?.request.auth?.name ?? '',
		httpAuthValueTemplate:
			protectSensitiveValues && t.httpImpl?.request.auth?.valueTemplate
				? '[configured]'
				: (t.httpImpl?.request.auth?.valueTemplate ?? ''),
		httpResponseCodes: (t.httpImpl?.response.successCodes ?? []).join(','),
		httpResponseErrorMode: t.httpImpl?.response.errorMode ?? '',
		httpResponseBodyOutputMode: t.httpImpl?.response.bodyOutputMode ?? HTTPBodyOutputMode.Auto,
		httpTimeoutMS: t.httpImpl?.request.timeoutMS !== undefined ? String(t.httpImpl.request.timeoutMS) : '',
	};
}

function buildToolPrefillKey(item: ToolItem): string {
	return `${item.bundleID}:${item.tool.id}:${item.tool.version}`;
}

function AddEditToolModalContent({
	onSubmit,
	initialData,
	existingTools,
	mode,
}: Omit<AddEditToolModalContentProps, 'isOpen' | 'onClose'>) {
	const effectiveMode: ModalMode = mode ?? (initialData ? 'edit' : 'add');
	const isViewMode = effectiveMode === 'view';
	const isEditMode = effectiveMode === 'edit';

	const [formData, setFormData] = useState<ToolFormData>(() =>
		buildInitialFormData(initialData, existingTools, isEditMode, isViewMode)
	);
	const [errors, setErrors] = useState<ErrorState>({});
	const [prefillMode, setPrefillMode] = useState(false);
	const [selectedPrefillKey, setSelectedPrefillKey] = useState<string | null>(null);
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);

	const { requestClose, unmountingRef } = useModalDialogController();
	const displayNameInputRef = useRef<HTMLInputElement | null>(null);
	const httpUrlInputRef = useRef<HTMLInputElement | null>(null);

	const copyableTools = useMemo(
		() => existingTools.filter(item => item.tool.type === ToolImplType.HTTP),
		[existingTools]
	);

	const prefillSourceMap = useMemo<Record<string, ToolItem>>(() => {
		return Object.fromEntries(copyableTools.map(item => [buildToolPrefillKey(item), item] as const));
	}, [copyableTools]);

	const prefillKeys = useMemo(() => Object.keys(prefillSourceMap), [prefillSourceMap]);

	const prefillDropdownItems = useMemo<Record<string, { isEnabled: boolean; displayName: string }>>(
		() =>
			Object.fromEntries(
				Object.entries(prefillSourceMap).map(([key, item]) => [
					key,
					{
						isEnabled: true,
						displayName: `${item.tool.displayName || item.tool.slug} (${item.tool.slug}@${item.tool.version})`,
					},
				])
			),
		[prefillSourceMap]
	);

	useEffect(() => {
		const focusTimer = window.setTimeout(() => {
			if (!isViewMode) {
				displayNameInputRef.current?.focus();
			}
		}, 0);

		return () => {
			window.clearTimeout(focusTimer);
		};
	}, [isViewMode]);

	const validateField = (
		field: keyof ErrorState,
		val: string,
		currentErrors: ErrorState,
		state: ToolFormData = formData
	): ErrorState => {
		let newErrs: ErrorState = { ...currentErrors };
		const v = val.trim();

		if (!v && ['displayName', 'slug', 'version', 'type', 'argSchema'].includes(field)) {
			newErrs[field] = 'This field is required.';
			return newErrs;
		}

		if (field === 'slug') {
			const err = validateSlug(v);
			if (err) {
				newErrs.slug = err;
			} else {
				const clash = !isEditMode && existingTools.some(t => t.tool.slug === v && t.tool.id !== initialData?.tool.id);
				if (clash) {
					newErrs.slug = 'Slug already in use.';
				} else {
					newErrs = omitManyKeys(newErrs, ['slug']);
				}
			}
		} else if (field === 'version') {
			if (!v) {
				newErrs.version = 'Version is required.';
			} else if (!isSemverVersion(v)) {
				newErrs.version = `Version must use semantic version format, for example ${DEFAULT_SEMVER}.`;
			} else if (isEditMode && initialData?.tool && v === initialData.tool.version) {
				newErrs.version = 'New version must be different from the current version.';
			} else {
				const slugToCheck = initialData?.tool.slug ?? state.slug.trim();
				const versionClash = existingTools.some(t => t.tool.slug === slugToCheck && t.tool.version === v);
				if (versionClash) {
					newErrs.version = 'That version already exists for this slug.';
				} else {
					newErrs = omitManyKeys(newErrs, ['version']);
				}
			}
		} else if (field === 'tags') {
			if (v === '') {
				newErrs = omitManyKeys(newErrs, ['tags']);
			} else {
				const err = validateTags(val);
				if (err) {
					newErrs.tags = err;
				} else {
					newErrs = omitManyKeys(newErrs, ['tags']);
				}
			}
		} else if (field === 'argSchema') {
			try {
				const parsed = JSON.parse(val);
				if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
					newErrs.argSchema = 'Arg schema must be a JSON object';
				} else {
					newErrs = omitManyKeys(newErrs, ['argSchema']);
				}
			} catch {
				newErrs.argSchema = 'Invalid JSON';
			}
		} else if (field === 'httpUrl' && state.type === ToolImplType.HTTP) {
			const { error } = validateHTTPToolURL(v, httpUrlInputRef.current);

			if (error) {
				newErrs.httpUrl = error;
			} else {
				newErrs = omitManyKeys(newErrs, ['httpUrl']);
			}
		} else if (field === 'httpHeaders') {
			if (v === '') {
				newErrs = omitManyKeys(newErrs, ['httpHeaders']);
			} else {
				try {
					restoreRedactedHTTPHeaders(
						parseHTTPHeadersJSON(val, 'Headers'),
						isEditMode ? initialData?.tool.httpImpl?.request.headers : undefined
					);
					newErrs = omitManyKeys(newErrs, ['httpHeaders']);
				} catch (error) {
					newErrs.httpHeaders = error instanceof Error ? error.message : 'Headers must be valid JSON.';
				}
			}
		} else if (field === 'httpQuery') {
			if (v === '') {
				newErrs = omitManyKeys(newErrs, ['httpQuery']);
			} else {
				try {
					parseStringRecordJSON(val, 'Query');
					newErrs = omitManyKeys(newErrs, ['httpQuery']);
				} catch (error) {
					newErrs.httpQuery = error instanceof Error ? error.message : 'Query must be valid JSON.';
				}
			}
		} else if (field === 'httpResponseCodes') {
			if (v === '') {
				newErrs = omitManyKeys(newErrs, ['httpResponseCodes']);
			} else {
				try {
					parseHTTPStatusCodes(v);
					newErrs = omitManyKeys(newErrs, ['httpResponseCodes']);
				} catch (error) {
					newErrs.httpResponseCodes = error instanceof Error ? error.message : 'Invalid HTTP status codes.';
				}
			}
		} else if (field === 'httpTimeoutMS') {
			if (v === '') {
				newErrs = omitManyKeys(newErrs, ['httpTimeoutMS']);
			} else {
				const n = Number(v);
				if (!Number.isInteger(n) || n < 1) {
					newErrs.httpTimeoutMS = 'Timeout must be a positive integer in milliseconds.';
				} else {
					newErrs = omitManyKeys(newErrs, ['httpTimeoutMS']);
				}
			}
		} else if (field === 'httpAuth') {
			if (!state.httpAuthType.trim()) {
				newErrs = omitManyKeys(newErrs, ['httpAuth']);
			} else if (state.httpAuthType !== 'apiKey') {
				newErrs.httpAuth = 'Only apiKey authentication is supported by the HTTP tool schema.';
			} else if (state.httpAuthIn !== 'header' && state.httpAuthIn !== 'query') {
				newErrs.httpAuth = 'Auth In must be either "header" or "query".';
			} else if (!state.httpAuthName.trim()) {
				newErrs.httpAuth = 'Auth Name is required when authentication is configured.';
			} else if (!state.httpAuthValueTemplate) {
				newErrs.httpAuth = 'Auth Value Template is required when authentication is configured.';
			} else if (!isEditMode && state.httpAuthValueTemplate === REDACTED_HTTP_VALUE) {
				newErrs.httpAuth = 'Enter an authentication value instead of using the redaction marker.';
			} else if (/[\r\n\u0000]/.test(state.httpAuthValueTemplate)) {
				newErrs.httpAuth = 'Auth Value Template must not contain CR, LF, or NUL.';
			} else {
				newErrs = omitManyKeys(newErrs, ['httpAuth']);
			}
		} else {
			newErrs = omitManyKeys(newErrs, [field]);
		}

		return newErrs;
	};

	const validateForm = (state: ToolFormData): ErrorState => {
		let newErrs: ErrorState = {};
		newErrs = validateField('displayName', state.displayName, newErrs, state);
		newErrs = validateField('slug', state.slug, newErrs, state);
		newErrs = validateField('version', state.version, newErrs, state);

		newErrs = validateField('type', state.type, newErrs, state);
		newErrs = validateField('argSchema', state.argSchema, newErrs, state);

		if (state.tags.trim() !== '') {
			newErrs = validateField('tags', state.tags, newErrs, state);
		}

		if (state.type === ToolImplType.HTTP) {
			newErrs = validateField('httpUrl', state.httpUrl, newErrs, state);
			newErrs = validateField('httpHeaders', state.httpHeaders, newErrs, state);
			newErrs = validateField('httpQuery', state.httpQuery, newErrs, state);
			newErrs = validateField('httpResponseCodes', state.httpResponseCodes, newErrs, state);
			newErrs = validateField('httpTimeoutMS', state.httpTimeoutMS, newErrs, state);
			newErrs = validateField('httpAuth', state.httpAuthType, newErrs, state);
		}

		return newErrs;
	};

	const applyPrefill = (key: string) => {
		const source = prefillSourceMap[key];
		if (!source) {
			return;
		}

		const copied = buildInitialFormData(source, existingTools, false, false);
		const next: ToolFormData = {
			...copied,
			slug: formData.slug,
			version: formData.version,
			isEnabled: true,
			httpHeaders: JSON.stringify(omitSensitiveHTTPHeaders(source.tool.httpImpl?.request.headers) ?? {}, null, 2),
			httpAuthValueTemplate: '',
		};

		setFormData(next);
		setErrors(validateForm(next));
		setSubmitError('');
		setSelectedPrefillKey(key);
		setPrefillMode(false);
	};

	const handleInput = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
		if (isSubmitting) {
			return;
		}

		const { name, value, type, checked } = e.target as HTMLInputElement;
		const newVal = type === 'checkbox' ? checked : value;
		const nextFormData = { ...formData, [name]: newVal } as ToolFormData;

		setFormData(nextFormData);

		if (
			[
				'displayName',
				'slug',
				'version',
				'type',
				'argSchema',
				'httpUrl',
				'httpHeaders',
				'httpQuery',
				'httpResponseCodes',
				'httpTimeoutMS',
				'tags',
			].includes(name)
		) {
			setErrors(previousErrors => {
				if (['slug', 'version', 'type'].includes(name)) {
					return validateForm(nextFormData);
				}

				return validateField(name as keyof ErrorState, String(newVal), previousErrors, nextFormData);
			});
		}
	};

	const formIsValid = Object.values(validateForm(formData)).every(error => !error);
	const requiredFieldsPresent =
		formData.displayName.trim() &&
		formData.slug.trim() &&
		formData.version.trim() &&
		formData.argSchema.trim() &&
		(formData.type === ToolImplType.HTTP ? formData.httpUrl.trim() : true);

	const isAllValid = isViewMode || (!isSubmitting && formIsValid && Boolean(requiredFieldsPresent));

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();
		e.stopPropagation();

		if (isViewMode || isSubmitting) {
			return;
		}

		setSubmitError('');

		const nextErrors = validateForm(formData);
		setErrors(nextErrors);
		if (Object.values(nextErrors).some(Boolean)) {
			return;
		}

		const tagsArr = formData.tags
			.split(',')
			.map(t => t.trim())
			.filter(Boolean);

		let parsedArgSchema: JSONSchema;
		try {
			parsedArgSchema = JSON.parse(formData.argSchema) as JSONSchema;
		} catch {
			setErrors(prev => ({ ...prev, argSchema: 'Invalid JSON' }));
			return;
		}

		let httpImpl: Tool['httpImpl'] | undefined;

		if (formData.type === ToolImplType.HTTP) {
			const httpUrlInput = httpUrlInputRef.current;
			const { normalized: normalizedHttpUrl, error: httpUrlError } = validateHTTPToolURL(
				formData.httpUrl,
				httpUrlInput
			);

			if (!normalizedHttpUrl || httpUrlError) {
				setErrors(prev => ({ ...prev, httpUrl: httpUrlError ?? MessageEnterValidURL }));
				httpUrlInput?.focus();
				return;
			}

			let headers: Record<string, string> | undefined;
			let query: Record<string, string> | undefined;

			try {
				const parsed = restoreRedactedHTTPHeaders(
					parseHTTPHeadersJSON(formData.httpHeaders, 'Headers'),
					isEditMode ? initialData?.tool.httpImpl?.request.headers : undefined
				);
				headers = Object.keys(parsed).length > 0 ? parsed : undefined;
			} catch (error) {
				setErrors(prev => ({
					...prev,
					httpHeaders: error instanceof Error ? error.message : 'Headers could not be resolved.',
				}));
				return;
			}

			try {
				const parsed = parseStringRecordJSON(formData.httpQuery, 'Query');
				query = Object.keys(parsed).length > 0 ? parsed : undefined;
			} catch {
				setErrors(prev => ({ ...prev, httpQuery: 'Query must be a JSON object containing string values.' }));
				return;
			}

			const successCodes = parseHTTPStatusCodes(formData.httpResponseCodes);
			const authValueTemplate =
				isEditMode && formData.httpAuthValueTemplate === REDACTED_HTTP_VALUE
					? (initialData?.tool.httpImpl?.request.auth?.valueTemplate ?? '')
					: formData.httpAuthValueTemplate;

			httpImpl = {
				request: {
					method: formData.httpMethod || 'GET',
					urlTemplate: normalizedHttpUrl,
					headers,
					query,
					body: formData.httpBody || undefined,
					timeoutMS: formData.httpTimeoutMS.trim() ? Number(formData.httpTimeoutMS.trim()) : undefined,
					auth: formData.httpAuthType
						? {
								type: formData.httpAuthType,
								in: formData.httpAuthIn || undefined,
								name: formData.httpAuthName || undefined,
								valueTemplate: authValueTemplate,
							}
						: undefined,
				},
				response: {
					successCodes,
					errorMode: formData.httpResponseErrorMode || undefined,
					bodyOutputMode: formData.httpResponseBodyOutputMode || undefined,
				},
			};
		}

		setIsSubmitting(true);
		void onSubmit({
			displayName: formData.displayName.trim(),
			slug: formData.slug.trim(),
			description: formData.description.trim() || undefined,
			isEnabled: formData.isEnabled,
			userCallable: formData.userCallable,
			llmCallable: formData.llmCallable,
			autoExecReco: formData.autoExecReco,
			tags: tagsArr.length > 0 ? tagsArr : undefined,
			type: formData.type,
			argSchema: parsedArgSchema,
			httpImpl,
			version: formData.version,
		})
			.then(() => {
				if (!unmountingRef.current) {
					requestClose(true);
				}
			})
			.catch((err: unknown) => {
				if (!unmountingRef.current) {
					const msg = err instanceof Error ? err.message : 'Failed to save tool.';
					setSubmitError(msg);
				}
			})
			.finally(() => {
				if (!unmountingRef.current) {
					setIsSubmitting(false);
				}
			});
	};

	const onToolTypeChange = (key: ToolImplType) => {
		if (key !== ToolImplType.HTTP) {
			return;
		}
		setFormData(prev => ({ ...prev, type: key }));
		setErrors(prev => validateField('type', key, prev));
	};

	const headerTitle = effectiveMode === 'view' ? 'View Tool' : effectiveMode === 'edit' ? 'Edit Tool' : 'Add Tool';

	return (
		<>
			<div className="modal-box bg-base-200 max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-3xl overflow-hidden rounded-2xl p-0">
				<div className="app-scrollbar-thin max-h-[calc(100dvh-1rem)] overflow-y-auto p-4 sm:p-6">
					<ModalHeader
						title={headerTitle}
						description="Configure immutable tool versions and their HTTP runtime behavior."
						onClose={() => {
							requestClose();
						}}
						closeDisabled={isSubmitting}
					/>

					<form noValidate onSubmit={handleSubmit} className={MANAGEMENT_MODAL_FORM_CLASS}>
						{submitError && (
							<div className="alert alert-error rounded-2xl text-sm">
								<div className="flex items-center gap-2">
									<FiAlertCircle size={14} />
									<span>{submitError}</span>
								</div>
							</div>
						)}

						{formData.autoExecReco && !isViewMode && (
							<div className="border-warning/40 bg-warning/10 rounded-2xl border p-2 text-sm">
								<div className="flex items-start gap-2">
									<FiAlertTriangle className="mt-0.5 shrink-0" size={16} />
									<p>
										Only recommend auto-execute for trusted, low-risk tools. Avoid it for tools that write files, call
										network endpoints, run shell/script commands, or handle sensitive data unless the workflow is
										explicitly designed for that risk.
									</p>
								</div>
							</div>
						)}

						{effectiveMode === 'add' && (
							<div className="grid grid-cols-12 items-center gap-2">
								<div className="label col-span-3">
									<span className="text-sm">Prefill from Existing</span>
								</div>

								<div className="col-span-9 flex items-center gap-2">
									{!prefillMode && (
										<button
											type="button"
											className="btn btn-sm btn-ghost flex items-center rounded-xl"
											onClick={() => {
												setPrefillMode(true);
											}}
											disabled={prefillKeys.length === 0}
											title={
												prefillKeys.length === 0
													? 'No HTTP tools are available to copy. Only HTTP tools can be created here.'
													: undefined
											}
										>
											<FiUpload size={14} />
											<span className="ml-1">Copy Existing Tool</span>
										</button>
									)}

									{prefillMode && (
										<>
											<Dropdown<string>
												dropdownItems={prefillDropdownItems}
												orderedKeys={prefillKeys}
												selectedKey={selectedPrefillKey ?? ''}
												onChange={applyPrefill}
												disabled={prefillKeys.length === 0}
												filterDisabled={false}
												title="Select tool to copy"
												getDisplayName={key => prefillDropdownItems[key]?.displayName ?? 'Select tool to copy'}
											/>
											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												onClick={() => {
													setPrefillMode(false);
													setSelectedPrefillKey(null);
												}}
												title="Cancel prefill"
											>
												<FiX size={12} />
											</button>
										</>
									)}
								</div>
							</div>
						)}

						<ModalField label="Display Name" htmlFor="tool-display-name" required error={errors.displayName}>
							<input
								id="tool-display-name"
								ref={displayNameInputRef}
								type="text"
								name="displayName"
								value={formData.displayName}
								onChange={handleInput}
								readOnly={isViewMode}
								className={`input w-full rounded-xl ${errors.displayName ? 'input-error' : ''}`}
								spellCheck="false"
								autoComplete="off"
								aria-invalid={Boolean(errors.displayName)}
							/>
						</ModalField>

						<div className="grid grid-cols-12 items-center gap-2">
							<label htmlFor="tool-slug" className="label col-span-3">
								<span className="text-sm">Slug*</span>
								<span className="tooltip tooltip-right" data-tip="Lower-case, URL-friendly.">
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<input
									id="tool-slug"
									type="text"
									name="slug"
									value={formData.slug}
									onChange={handleInput}
									className={`input w-full rounded-xl ${errors.slug ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									readOnly={isViewMode || isEditMode}
									aria-invalid={Boolean(errors.slug)}
								/>
								{errors.slug && (
									<div className="label">
										<span className="text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.slug}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label htmlFor="tool-version" className="label col-span-3">
								<span className="text-sm">Version*</span>
								<span
									className="tooltip tooltip-right"
									data-tip="Once created, existing versions are not edited. Edit creates a new version."
								>
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<input
									id="tool-version"
									type="text"
									name="version"
									value={formData.version}
									onChange={handleInput}
									readOnly={isViewMode}
									className={`input w-full rounded-xl ${errors.version ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									aria-invalid={Boolean(errors.version)}
								/>
								{isEditMode && initialData?.tool && (
									<div className="label">
										<span className="text-base-content/70 text-xs">
											Current: {initialData.tool.version} · Suggested next:{' '}
											{
												suggestNextMinorVersion(
													initialData.tool.version,
													existingTools.filter(x => x.tool.slug === initialData.tool.slug).map(x => x.tool.version)
												).suggested
											}
										</span>
									</div>
								)}
								{errors.version && (
									<div className="label">
										<span className="text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.version}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label htmlFor="tool-enabled" className="label col-span-3 cursor-pointer">
								<span className="text-sm">Enabled</span>
							</label>
							<div className="col-span-9">
								<input
									id="tool-enabled"
									type="checkbox"
									name="isEnabled"
									checked={formData.isEnabled}
									onChange={handleInput}
									className="toggle toggle-accent disabled:opacity-80"
									disabled={isViewMode}
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label htmlFor="tool-user-callable" className="label col-span-3 cursor-pointer">
								<span className="text-sm">User Callable</span>
							</label>
							<div className="col-span-9">
								<input
									id="tool-user-callable"
									type="checkbox"
									name="userCallable"
									checked={formData.userCallable}
									onChange={handleInput}
									className="toggle toggle-accent disabled:opacity-80"
									disabled={isViewMode}
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label htmlFor="tool-llm-callable" className="label col-span-3 cursor-pointer">
								<span className="text-sm">LLM Callable</span>
							</label>
							<div className="col-span-9">
								<input
									id="tool-llm-callable"
									type="checkbox"
									name="llmCallable"
									checked={formData.llmCallable}
									onChange={handleInput}
									className="toggle toggle-accent disabled:opacity-80"
									disabled={isViewMode}
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label htmlFor="tool-auto-execute" className="label col-span-3 cursor-pointer">
								<span className="text-sm">AutoExecute</span>
							</label>
							<div className="col-span-9">
								<input
									id="tool-auto-execute"
									type="checkbox"
									name="autoExecReco"
									checked={formData.autoExecReco}
									onChange={handleInput}
									className="toggle toggle-accent disabled:opacity-80"
									disabled={isViewMode}
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<div className="label col-span-3">
								<span className="text-sm">Type*</span>
							</div>
							<div className="col-span-9">
								{isEditMode || isViewMode ? (
									<ReadOnlyValue value={toolTypeDropdownItems[formData.type].displayName} />
								) : (
									<Dropdown<ToolImplType>
										dropdownItems={toolTypeDropdownItems}
										selectedKey={formData.type}
										onChange={onToolTypeChange}
										filterDisabled={true}
										title="Select tool type"
										getDisplayName={k => toolTypeDropdownItems[k].displayName}
									/>
								)}
								{errors.type && (
									<div className="label">
										<span className="text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.type}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label htmlFor="tool-description" className="label col-span-3">
								<span className="text-sm">Description</span>
							</label>
							<div className="col-span-9">
								<textarea
									id="tool-description"
									name="description"
									value={formData.description}
									onChange={handleInput}
									readOnly={isViewMode}
									className="textarea h-20 w-full rounded-xl"
									spellCheck="false"
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-start gap-2">
							<label htmlFor="tool-arg-schema" className="label col-span-3">
								<span className="text-sm">Arg JSONSchema*</span>
								<span className="tooltip tooltip-right" data-tip="JSON Schema for arguments">
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<textarea
									id="tool-arg-schema"
									name="argSchema"
									value={formData.argSchema}
									onChange={handleInput}
									readOnly={isViewMode}
									className={`textarea h-24 w-full rounded-xl ${errors.argSchema ? 'textarea-error' : ''}`}
									spellCheck="false"
									aria-invalid={Boolean(errors.argSchema)}
								/>
								{errors.argSchema && (
									<div className="label">
										<span className="text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.argSchema}
										</span>
									</div>
								)}
							</div>
						</div>

						{formData.type === ToolImplType.HTTP && (
							<>
								<div className="border-info/30 bg-info/10 rounded-2xl border p-3 text-sm">
									<div className="flex items-start gap-2">
										<FiAlertTriangle className="mt-0.5 shrink-0" size={16} />
										<p>
											HTTP tools can call network endpoints and may include model-provided arguments. Review the URL,
											headers, authentication template, and auto-execute setting before using the tool in a workflow.
										</p>
									</div>
								</div>
								<div className="grid grid-cols-12 items-center gap-2">
									<label htmlFor="tool-http-url" className="label col-span-3">
										<span className="text-sm">HTTP URL*</span>
									</label>
									<div className="col-span-9">
										<input
											id="tool-http-url"
											ref={httpUrlInputRef}
											type="url"
											name="httpUrl"
											value={formData.httpUrl}
											onChange={handleInput}
											readOnly={isViewMode}
											className={`input w-full rounded-xl ${errors.httpUrl ? 'input-error' : ''}`}
											spellCheck="false"
											autoComplete="off"
											aria-invalid={Boolean(errors.httpUrl)}
											placeholder="https://api.example.com/endpoint OR api.example.com/endpoint"
										/>
										{errors.httpUrl && (
											<div className="label">
												<span className="text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.httpUrl}
												</span>
											</div>
										)}
										<div className="label">
											<span className="text-base-content/70 text-xs">
												Remote tool endpoints require HTTPS. Plain HTTP is limited to localhost and loopback addresses.
												Embedded URL credentials are not allowed.
											</span>
										</div>
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label htmlFor="tool-http-method" className="label col-span-3">
										<span className="text-sm">HTTP Method</span>
									</label>
									<div className="col-span-9">
										<input
											id="tool-http-method"
											type="text"
											name="httpMethod"
											value={formData.httpMethod}
											onChange={handleInput}
											readOnly={isViewMode}
											className="input w-full rounded-xl"
											spellCheck="false"
											autoComplete="off"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-start gap-2">
									<label htmlFor="tool-http-headers" className="label col-span-3">
										<span className="text-sm">Headers (JSON)</span>
									</label>
									<div className="col-span-9">
										<textarea
											id="tool-http-headers"
											name="httpHeaders"
											value={formData.httpHeaders}
											onChange={handleInput}
											readOnly={isViewMode}
											className={`textarea h-16 w-full rounded-xl ${errors.httpHeaders ? 'textarea-error' : ''}`}
											spellCheck="false"
										/>
										{errors.httpHeaders && (
											<div className="label">
												<span className="text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.httpHeaders}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-start gap-2">
									<label htmlFor="tool-http-query" className="label col-span-3">
										<span className="text-sm">Query (JSON)</span>
									</label>
									<div className="col-span-9">
										<textarea
											id="tool-http-query"
											name="httpQuery"
											value={formData.httpQuery}
											onChange={handleInput}
											readOnly={isViewMode}
											className={`textarea h-16 w-full rounded-xl ${errors.httpQuery ? 'textarea-error' : ''}`}
											spellCheck="false"
										/>
										{errors.httpQuery && (
											<div className="label">
												<span className="text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.httpQuery}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-start gap-2">
									<label htmlFor="tool-http-body" className="label col-span-3">
										<span className="text-sm">Body</span>
									</label>
									<div className="col-span-9">
										<textarea
											id="tool-http-body"
											name="httpBody"
											value={formData.httpBody}
											onChange={handleInput}
											readOnly={isViewMode}
											className="textarea h-16 w-full rounded-xl"
											spellCheck="false"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label htmlFor="tool-http-timeout" className="label col-span-3">
										<span className="text-sm">Timeout (ms)</span>
									</label>
									<div className="col-span-9">
										<input
											id="tool-http-timeout"
											type="text"
											name="httpTimeoutMS"
											value={formData.httpTimeoutMS}
											onChange={handleInput}
											readOnly={isViewMode}
											className={`input w-full rounded-xl ${errors.httpTimeoutMS ? 'input-error' : ''}`}
											spellCheck="false"
											autoComplete="off"
										/>
										{errors.httpTimeoutMS && (
											<div className="label">
												<span className="text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.httpTimeoutMS}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label htmlFor="tool-http-auth-type" className="label col-span-3">
										<span className="text-sm">Auth Type</span>
									</label>
									<div className="col-span-9">
										<input
											id="tool-http-auth-type"
											type="text"
											name="httpAuthType"
											value={formData.httpAuthType}
											onChange={handleInput}
											readOnly={isViewMode}
											className="input w-full rounded-xl"
											spellCheck="false"
											autoComplete="off"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label htmlFor="tool-http-auth-in" className="label col-span-3">
										<span className="text-sm">Auth In</span>
									</label>
									<div className="col-span-9">
										<input
											id="tool-http-auth-in"
											type="text"
											name="httpAuthIn"
											value={formData.httpAuthIn}
											onChange={handleInput}
											readOnly={isViewMode}
											className="input w-full rounded-xl"
											spellCheck="false"
											autoComplete="off"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label htmlFor="tool-http-auth-name" className="label col-span-3">
										<span className="text-sm">Auth Name</span>
									</label>
									<div className="col-span-9">
										<input
											id="tool-http-auth-name"
											type="text"
											name="httpAuthName"
											value={formData.httpAuthName}
											onChange={handleInput}
											readOnly={isViewMode}
											className="input w-full rounded-xl"
											spellCheck="false"
											autoComplete="off"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label htmlFor="tool-http-auth-value-template" className="label col-span-3">
										<span className="text-sm">Auth Value Template</span>
									</label>
									<div className="col-span-9">
										<input
											id="tool-http-auth-value-template"
											type="text"
											name="httpAuthValueTemplate"
											value={formData.httpAuthValueTemplate}
											onChange={handleInput}
											readOnly={isViewMode}
											className={`input w-full rounded-xl ${errors.httpAuth ? 'input-error' : ''}`}
											spellCheck="false"
											autoComplete="off"
											aria-invalid={Boolean(errors.httpAuth)}
										/>
										{errors.httpAuth ? (
											<div className="label">
												<span className="text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.httpAuth}
												</span>
											</div>
										) : null}
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label htmlFor="tool-http-response-codes" className="label col-span-3">
										<span className="text-sm">Success Codes (comma)</span>
									</label>
									<div className="col-span-9">
										<input
											id="tool-http-response-codes"
											type="text"
											name="httpResponseCodes"
											value={formData.httpResponseCodes}
											onChange={handleInput}
											readOnly={isViewMode}
											className={`input w-full rounded-xl ${errors.httpResponseCodes ? 'input-error' : ''}`}
											spellCheck="false"
											autoComplete="off"
										/>
										{errors.httpResponseCodes && (
											<div className="label">
												<span className="text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.httpResponseCodes}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label htmlFor="tool-http-response-error-mode" className="label col-span-3">
										<span className="text-sm">Error Mode</span>
									</label>
									<div className="col-span-9">
										<input
											id="tool-http-response-error-mode"
											type="text"
											name="httpResponseErrorMode"
											value={formData.httpResponseErrorMode}
											onChange={handleInput}
											readOnly={isViewMode}
											className="input w-full rounded-xl"
											spellCheck="false"
											autoComplete="off"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<div className="label col-span-3">
										<span className="text-sm">Body Output Mode</span>
									</div>
									<div className="col-span-9">
										{isViewMode ? (
											<ReadOnlyValue value={bodyOutputModeItems[formData.httpResponseBodyOutputMode].displayName} />
										) : (
											<Dropdown<HTTPBodyOutputMode>
												dropdownItems={bodyOutputModeItems}
												selectedKey={formData.httpResponseBodyOutputMode}
												onChange={m => {
													setFormData(prev => ({ ...prev, httpResponseBodyOutputMode: m }));
												}}
												filterDisabled={false}
												title="Select output mode"
												getDisplayName={k => bodyOutputModeItems[k].displayName}
											/>
										)}
									</div>
								</div>
							</>
						)}

						<div className="grid grid-cols-12 items-center gap-2">
							<label htmlFor="tool-tags" className="label col-span-3">
								<span className="text-sm">Tags</span>
							</label>
							<div className="col-span-9">
								<input
									id="tool-tags"
									type="text"
									name="tags"
									value={formData.tags}
									onChange={handleInput}
									readOnly={isViewMode}
									className={`input w-full rounded-xl ${errors.tags ? 'input-error' : ''}`}
									placeholder="comma, separated, tags"
									spellCheck="false"
									aria-invalid={Boolean(errors.tags)}
								/>
								{errors.tags && (
									<div className="label">
										<span className="text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.tags}
										</span>
									</div>
								)}
							</div>
						</div>

						{isViewMode && initialData?.tool && (
							<ModalSection title="Metadata">
								<ManagementInfoGrid>
									<ManagementInfoRow label="ID" mono>
										{initialData.tool.id}
									</ManagementInfoRow>
									<ManagementInfoRow label="Schema version">{initialData.tool.schemaVersion}</ManagementInfoRow>
									<ManagementInfoRow label="LLM tool type">{initialData.tool.llmToolType}</ManagementInfoRow>
									<ManagementInfoRow label="Built-in">{initialData.tool.isBuiltIn ? 'Yes' : 'No'}</ManagementInfoRow>
									<ManagementInfoRow label="Created">{initialData.tool.createdAt}</ManagementInfoRow>
									<ManagementInfoRow label="Modified">{initialData.tool.modifiedAt}</ManagementInfoRow>
								</ManagementInfoGrid>
							</ModalSection>
						)}

						<ModalActions>
							<button
								type="button"
								className="btn bg-base-300 rounded-xl"
								onClick={() => {
									requestClose();
								}}
								disabled={isSubmitting}
							>
								{isViewMode ? 'Close' : 'Cancel'}
							</button>
							{!isViewMode && (
								<button type="submit" className="btn btn-primary rounded-xl" disabled={!isAllValid || isSubmitting}>
									{isSubmitting ? 'Saving...' : 'Save'}
								</button>
							)}
						</ModalActions>
					</form>
				</div>
			</div>
			<ModalBackdrop enabled={isViewMode} />
		</>
	);
}

export function AddEditToolModal({ isOpen, initialData, mode, ...rest }: AddEditToolModalProps) {
	if (!isOpen) {
		return null;
	}

	const effectiveMode: ModalMode = mode ?? (initialData ? 'edit' : 'add');
	const modalKey = `${effectiveMode}:${initialData?.tool.id ?? 'new'}:${
		initialData?.tool.version ?? DEFAULT_SEMVER
	}:${initialData?.tool.modifiedAt ?? 'unknown-modified'}`;

	return (
		<ModalDialog isOpen={isOpen} onClose={rest.onClose} blockCancel={effectiveMode !== 'view'}>
			<AddEditToolModalContent key={modalKey} initialData={initialData} mode={mode} {...rest} />
		</ModalDialog>
	);
}
