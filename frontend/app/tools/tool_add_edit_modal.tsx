import { type ChangeEvent, type FormEvent, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiX } from 'react-icons/fi';

import { HTTPBodyOutputMode, type Tool, ToolImplType } from '@/spec/tool';

import type { JSONSchema } from '@/lib/jsonschema_utils';
import { omitManyKeys } from '@/lib/obj_utils';
import { validateSlug, validateTags } from '@/lib/text_utils';
import { MessageEnterValidURL, validateUrlForInput } from '@/lib/url_utils';
import { DEFAULT_SEMVER, suggestNextMinorVersion } from '@/lib/version_utils';

import { Dropdown } from '@/components/dropdown';
import { ModalBackdrop } from '@/components/modal_backdrop';
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
	initialData?: ToolItem; // when editing/viewing
	existingTools: ToolItem[];
	mode?: ModalMode;
}

const TOOL_TYPE_LABEL_GO = 'Go';
const TOOL_TYPE_LABEL_HTTP = 'HTTP';
const TOOL_TYPE_LABEL_SDK = 'SDK';

type ErrorState = {
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
	tags?: string;
};

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

export function AddEditToolModal({
	isOpen,
	onClose,
	onSubmit,
	initialData,
	existingTools,
	mode,
}: AddEditToolModalProps) {
	const effectiveMode: ModalMode = mode ?? (initialData ? 'edit' : 'add');
	const isViewMode = effectiveMode === 'view';
	const isEditMode = effectiveMode === 'edit';

	const [formData, setFormData] = useState({
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
		httpResponseBodyOutputMode: HTTPBodyOutputMode.Auto,
		httpTimeoutMS: '',
	});

	const [errors, setErrors] = useState<ErrorState>({});
	const [submitError, setSubmitError] = useState<string>('');

	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const displayNameInputRef = useRef<HTMLInputElement | null>(null);
	const httpUrlInputRef = useRef<HTMLInputElement | null>(null);

	useEffect(() => {
		if (!isOpen) return;

		if (initialData) {
			const t = initialData.tool;
			const existingVersionsForSlug = existingTools.filter(x => x.tool.slug === t.slug).map(x => x.tool.version);

			const nextV = isEditMode ? suggestNextMinorVersion(t.version, existingVersionsForSlug).suggested : t.version;

			setFormData({
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
				httpHeaders: JSON.stringify(t.httpImpl?.request.headers ?? {}, null, 2),
				httpQuery: JSON.stringify(t.httpImpl?.request.query ?? {}, null, 2),
				httpBody: t.httpImpl?.request.body ?? '',
				httpAuthType: t.httpImpl?.request.auth?.type ?? '',
				httpAuthIn: t.httpImpl?.request.auth?.in ?? '',

				httpAuthName: t.httpImpl?.request.auth?.name ?? '',
				httpAuthValueTemplate: t.httpImpl?.request.auth?.valueTemplate ?? '',
				httpResponseCodes: (t.httpImpl?.response.successCodes ?? []).join(','),
				httpResponseErrorMode: t.httpImpl?.response.errorMode ?? '',
				httpResponseBodyOutputMode: t.httpImpl?.response.bodyOutputMode ?? HTTPBodyOutputMode.Auto,
				httpTimeoutMS: t.httpImpl?.request.timeoutMS !== undefined ? String(t.httpImpl.request.timeoutMS) : '',
			});
		} else {
			setFormData({
				displayName: '',
				slug: '',
				version: DEFAULT_SEMVER,

				description: '',
				tags: '',
				isEnabled: true,

				userCallable: true,
				llmCallable: true,
				autoExecReco: false,

				type: ToolImplType.HTTP,
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
				httpResponseBodyOutputMode: HTTPBodyOutputMode.Auto,
				httpTimeoutMS: '',
			});
		}
		setErrors({});
		setSubmitError('');
	}, [isOpen, initialData, isEditMode]);

	useEffect(() => {
		if (!isOpen) return;
		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) {
			dialog.showModal();
		}

		window.setTimeout(() => {
			if (!isViewMode) displayNameInputRef.current?.focus();
		}, 0);

		return () => {
			if (dialog.open) dialog.close();
		};
	}, [isOpen, isViewMode]);

	const handleDialogClose = () => {
		onClose();
	};

	const validateField = (field: keyof ErrorState, val: string, currentErrors: ErrorState): ErrorState => {
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
				const clash = existingTools.some(t => t.tool.slug === v && t.tool.id !== initialData?.tool.id);
				if (clash) newErrs.slug = 'Slug already in use.';
				else newErrs = omitManyKeys(newErrs, ['slug']);
			}
		} else if (field === 'version') {
			if (!v) newErrs.version = 'Version is required.';
			else if (isEditMode && initialData?.tool && v === initialData.tool.version)
				newErrs.version = 'New version must be different from the current version.';
			else {
				const slugToCheck = initialData?.tool.slug ?? formData.slug.trim();
				const versionClash = existingTools.some(t => t.tool.slug === slugToCheck && t.tool.version === v);
				if (versionClash) newErrs.version = 'That version already exists for this slug.';
				else newErrs = omitManyKeys(newErrs, ['version']);
			}
		} else if (field === 'tags') {
			if (v === '') {
				newErrs = omitManyKeys(newErrs, ['tags']);
			} else {
				const err = validateTags(val);
				if (err) newErrs.tags = err;
				else newErrs = omitManyKeys(newErrs, ['tags']);
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
		} else if (field === 'httpUrl' && formData.type === ToolImplType.HTTP) {
			const { error } = validateUrlForInput(v, httpUrlInputRef.current, {
				required: true,
				requiredMessage: 'HTTP URL is required.',
			});

			if (error) newErrs.httpUrl = error;
			else newErrs = omitManyKeys(newErrs, ['httpUrl']);
		} else if (field === 'httpHeaders') {
			if (v === '') {
				newErrs = omitManyKeys(newErrs, ['httpHeaders']);
			} else {
				try {
					const parsed = JSON.parse(val);
					if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
						newErrs.httpHeaders = 'Headers must be a JSON object';
					} else {
						newErrs = omitManyKeys(newErrs, ['httpHeaders']);
					}
				} catch {
					newErrs.httpHeaders = 'Invalid JSON';
				}
			}
		} else if (field === 'httpQuery') {
			if (v === '') {
				newErrs = omitManyKeys(newErrs, ['httpQuery']);
			} else {
				try {
					const parsed = JSON.parse(val);
					if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
						newErrs.httpQuery = 'Query must be a JSON object';
					} else {
						newErrs = omitManyKeys(newErrs, ['httpQuery']);
					}
				} catch {
					newErrs.httpQuery = 'Invalid JSON';
				}
			}
		} else if (field === 'httpResponseCodes') {
			if (v === '') {
				newErrs = omitManyKeys(newErrs, ['httpResponseCodes']);
			} else {
				const parts = v
					.split(',')
					.map(s => s.trim())
					.filter(Boolean);
				const nums = parts.map(p => Number(p));
				const bad = nums.some(n => !Number.isFinite(n) || n <= 0);
				if (bad) newErrs.httpResponseCodes = 'Success codes must be comma-separated numbers (e.g. 200,201)';
				else newErrs = omitManyKeys(newErrs, ['httpResponseCodes']);
			}
		} else if (field === 'httpTimeoutMS') {
			if (v === '') newErrs = omitManyKeys(newErrs, ['httpTimeoutMS']);
			else {
				const n = Number(v);
				if (!Number.isFinite(n) || n < 1) newErrs.httpTimeoutMS = 'Timeout must be a positive number (ms).';
				else newErrs = omitManyKeys(newErrs, ['httpTimeoutMS']);
			}
		} else {
			newErrs = omitManyKeys(newErrs, [field]);
		}

		return newErrs;
	};

	const validateForm = (state: typeof formData): ErrorState => {
		let newErrs: ErrorState = {};
		newErrs = validateField('displayName', state.displayName, newErrs);
		newErrs = validateField('slug', state.slug, newErrs);
		newErrs = validateField('version', state.version, newErrs);

		newErrs = validateField('type', state.type, newErrs);
		newErrs = validateField('argSchema', state.argSchema, newErrs);

		if (state.tags.trim() !== '') {
			newErrs = validateField('tags', state.tags, newErrs);
		}

		if (state.type === ToolImplType.HTTP) {
			newErrs = validateField('httpUrl', state.httpUrl, newErrs);
			newErrs = validateField('httpHeaders', state.httpHeaders, newErrs);
			newErrs = validateField('httpQuery', state.httpQuery, newErrs);
			newErrs = validateField('httpResponseCodes', state.httpResponseCodes, newErrs);
			newErrs = validateField('httpTimeoutMS', state.httpTimeoutMS, newErrs);
		}

		return newErrs;
	};

	const handleInput = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
		const { name, value, type, checked } = e.target as HTMLInputElement;
		const newVal = type === 'checkbox' ? checked : value;

		setFormData(prev => ({ ...prev, [name]: newVal }));

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
			setErrors(prev => validateField(name as keyof ErrorState, String(newVal), prev));
		}
	};

	const isAllValid = useMemo(() => {
		if (isViewMode) return true;
		const hasErrs = Object.values(errors).some(Boolean);
		const required =
			formData.displayName.trim() &&
			formData.slug.trim() &&
			formData.version.trim() &&
			formData.argSchema.trim() &&
			(formData.type === ToolImplType.HTTP ? formData.httpUrl.trim() : true);
		return Boolean(required) && !hasErrs;
	}, [errors, formData, isViewMode]);

	const handleSubmit = (e: FormEvent) => {
		e.preventDefault();
		e.stopPropagation();
		if (isViewMode) return;
		setSubmitError('');

		const nextErrors = validateForm(formData);
		setErrors(nextErrors);
		if (Object.values(nextErrors).some(Boolean)) return;

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

		let httpImpl: Tool['httpImpl'] | undefined = undefined;

		if (formData.type === ToolImplType.HTTP) {
			const httpUrlInput = httpUrlInputRef.current;
			const { normalized: normalizedHttpUrl, error: httpUrlError } = validateUrlForInput(
				formData.httpUrl,
				httpUrlInput,
				{
					required: true,
					requiredMessage: 'HTTP URL is required.',
				}
			);

			if (!normalizedHttpUrl || httpUrlError) {
				setErrors(prev => ({ ...prev, httpUrl: httpUrlError ?? MessageEnterValidURL }));
				httpUrlInput?.focus();
				return;
			}

			let headers: Record<string, string> | undefined;
			let query: Record<string, string> | undefined;

			try {
				headers = formData.httpHeaders.trim()
					? (JSON.parse(formData.httpHeaders) as Record<string, string>)
					: undefined;
			} catch {
				setErrors(prev => ({ ...prev, httpHeaders: 'Invalid JSON' }));
				return;
			}

			try {
				query = formData.httpQuery.trim() ? (JSON.parse(formData.httpQuery) as Record<string, string>) : undefined;
			} catch {
				setErrors(prev => ({ ...prev, httpQuery: 'Invalid JSON' }));
				return;
			}

			const successCodes = formData.httpResponseCodes.trim()
				? formData.httpResponseCodes
						.split(',')
						.map(s => Number(s.trim()))
						.filter(n => Number.isFinite(n) && n > 0)
				: undefined;

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
								valueTemplate: formData.httpAuthValueTemplate,
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

		onSubmit({
			displayName: formData.displayName.trim(),
			slug: formData.slug.trim(),
			description: formData.description.trim() || undefined,
			isEnabled: formData.isEnabled,
			userCallable: formData.userCallable,
			llmCallable: formData.llmCallable,
			autoExecReco: formData.autoExecReco,
			tags: tagsArr.length ? tagsArr : undefined,
			type: formData.type,
			argSchema: parsedArgSchema,
			httpImpl,
			version: formData.version,
		})
			.then(() => {
				dialogRef.current?.close();
			})
			.catch((err: unknown) => {
				const msg = err instanceof Error ? err.message : 'Failed to save tool.';
				setSubmitError(msg);
			});
	};

	const onToolTypeChange = (key: ToolImplType) => {
		// UI restriction: only HTTP tools can be created/edited in current implementation
		if (key !== ToolImplType.HTTP) return;
		setFormData(prev => ({ ...prev, type: key }));
		setErrors(prev => validateField('type', key, prev));
	};

	if (!isOpen) return null;

	const headerTitle = effectiveMode === 'view' ? 'View Tool' : effectiveMode === 'edit' ? 'Edit Tool' : 'Add Tool';

	return createPortal(
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleDialogClose}
			onCancel={e => {
				// Form mode (add/edit): block Esc close. View mode: allow.
				if (!isViewMode) e.preventDefault();
			}}
		>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-3xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<div className="mb-4 flex items-center justify-between">
						<h3 className="text-lg font-bold">{headerTitle}</h3>
						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={() => dialogRef.current?.close()}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>

					<form noValidate onSubmit={handleSubmit} className="space-y-4">
						{submitError && (
							<div className="alert alert-error rounded-2xl text-sm">
								<div className="flex items-center gap-2">
									<FiAlertCircle size={14} />
									<span>{submitError}</span>
								</div>
							</div>
						)}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Display Name*</span>
							</label>
							<div className="col-span-9">
								<input
									ref={displayNameInputRef}
									type="text"
									name="displayName"
									value={formData.displayName}
									onChange={handleInput}
									readOnly={isViewMode}
									className={`input input-bordered w-full rounded-xl ${errors.displayName ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									aria-invalid={Boolean(errors.displayName)}
								/>
								{errors.displayName && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.displayName}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Slug*</span>
								<span className="label-text-alt tooltip tooltip-right" data-tip="Lower-case, URL-friendly.">
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									name="slug"
									value={formData.slug}
									onChange={handleInput}
									className={`input input-bordered w-full rounded-xl ${errors.slug ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									readOnly={isViewMode || isEditMode}
									aria-invalid={Boolean(errors.slug)}
								/>
								{errors.slug && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.slug}
										</span>
									</div>
								)}
							</div>
						</div>
						{/* Version */}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Version*</span>
								<span
									className="label-text-alt tooltip tooltip-right"
									data-tip="Once created, existing versions are not edited. Edit creates a new version."
								>
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									name="version"
									value={formData.version}
									onChange={handleInput}
									readOnly={isViewMode}
									className={`input input-bordered w-full rounded-xl ${errors.version ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									aria-invalid={Boolean(errors.version)}
								/>
								{isEditMode && initialData?.tool && (
									<div className="label">
										<span className="label-text-alt text-base-content/70 text-xs">
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
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.version}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3 cursor-pointer">
								<span className="label-text text-sm">Enabled</span>
							</label>
							<div className="col-span-9">
								<input
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
							<label className="label col-span-3 cursor-pointer">
								<span className="label-text text-sm">User Callable</span>
							</label>
							<div className="col-span-9">
								<input
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
							<label className="label col-span-3 cursor-pointer">
								<span className="label-text text-sm">LLM Callable</span>
							</label>
							<div className="col-span-9">
								<input
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
							<label className="label col-span-3 cursor-pointer">
								<span className="label-text text-sm">AutoExecute Recommendation</span>
							</label>
							<div className="col-span-9">
								<input
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
							<label className="label col-span-3">
								<span className="label-text text-sm">Type*</span>
							</label>
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
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.type}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Description</span>
							</label>
							<div className="col-span-9">
								<textarea
									name="description"
									value={formData.description}
									onChange={handleInput}
									readOnly={isViewMode}
									className="textarea textarea-bordered h-20 w-full rounded-xl"
									spellCheck="false"
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-start gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Arg JSONSchema*</span>
								<span className="label-text-alt tooltip tooltip-right" data-tip="JSON Schema for arguments">
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<textarea
									name="argSchema"
									value={formData.argSchema}
									onChange={handleInput}
									readOnly={isViewMode}
									className={`textarea textarea-bordered h-24 w-full rounded-xl ${errors.argSchema ? 'textarea-error' : ''}`}
									spellCheck="false"
									aria-invalid={Boolean(errors.argSchema)}
								/>
								{errors.argSchema && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.argSchema}
										</span>
									</div>
								)}
							</div>
						</div>

						{formData.type === ToolImplType.HTTP && (
							<>
								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">HTTP URL*</span>
									</label>
									<div className="col-span-9">
										<input
											ref={httpUrlInputRef}
											type="url"
											name="httpUrl"
											value={formData.httpUrl}
											onChange={handleInput}
											readOnly={isViewMode}
											className={`input input-bordered w-full rounded-xl ${errors.httpUrl ? 'input-error' : ''}`}
											spellCheck="false"
											autoComplete="off"
											aria-invalid={Boolean(errors.httpUrl)}
											placeholder="https://api.example.com/endpoint OR api.example.com/endpoint"
										/>
										{errors.httpUrl && (
											<div className="label">
												<span className="label-text-alt text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.httpUrl}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">HTTP Method</span>
									</label>
									<div className="col-span-9">
										<input
											type="text"
											name="httpMethod"
											value={formData.httpMethod}
											onChange={handleInput}
											readOnly={isViewMode}
											className="input input-bordered w-full rounded-xl"
											spellCheck="false"
											autoComplete="off"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-start gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Headers (JSON)</span>
									</label>
									<div className="col-span-9">
										<textarea
											name="httpHeaders"
											value={formData.httpHeaders}
											onChange={handleInput}
											readOnly={isViewMode}
											className={`textarea textarea-bordered h-16 w-full rounded-xl ${errors.httpHeaders ? 'textarea-error' : ''}`}
											spellCheck="false"
										/>
										{errors.httpHeaders && (
											<div className="label">
												<span className="label-text-alt text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.httpHeaders}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-start gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Query (JSON)</span>
									</label>
									<div className="col-span-9">
										<textarea
											name="httpQuery"
											value={formData.httpQuery}
											onChange={handleInput}
											readOnly={isViewMode}
											className={`textarea textarea-bordered h-16 w-full rounded-xl ${errors.httpQuery ? 'textarea-error' : ''}`}
											spellCheck="false"
										/>
										{errors.httpQuery && (
											<div className="label">
												<span className="label-text-alt text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.httpQuery}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-start gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Body</span>
									</label>
									<div className="col-span-9">
										<textarea
											name="httpBody"
											value={formData.httpBody}
											onChange={handleInput}
											readOnly={isViewMode}
											className="textarea textarea-bordered h-16 w-full rounded-xl"
											spellCheck="false"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Timeout (ms)</span>
									</label>
									<div className="col-span-9">
										<input
											type="text"
											name="httpTimeoutMS"
											value={formData.httpTimeoutMS}
											onChange={handleInput}
											readOnly={isViewMode}
											className={`input input-bordered w-full rounded-xl ${errors.httpTimeoutMS ? 'input-error' : ''}`}
											spellCheck="false"
											autoComplete="off"
										/>
										{errors.httpTimeoutMS && (
											<div className="label">
												<span className="label-text-alt text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.httpTimeoutMS}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Auth Type</span>
									</label>
									<div className="col-span-9">
										<input
											type="text"
											name="httpAuthType"
											value={formData.httpAuthType}
											onChange={handleInput}
											readOnly={isViewMode}
											className="input input-bordered w-full rounded-xl"
											spellCheck="false"
											autoComplete="off"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Auth In</span>
									</label>
									<div className="col-span-9">
										<input
											type="text"
											name="httpAuthIn"
											value={formData.httpAuthIn}
											onChange={handleInput}
											readOnly={isViewMode}
											className="input input-bordered w-full rounded-xl"
											spellCheck="false"
											autoComplete="off"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Auth Name</span>
									</label>
									<div className="col-span-9">
										<input
											type="text"
											name="httpAuthName"
											value={formData.httpAuthName}
											onChange={handleInput}
											readOnly={isViewMode}
											className="input input-bordered w-full rounded-xl"
											spellCheck="false"
											autoComplete="off"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Auth Value Template</span>
									</label>
									<div className="col-span-9">
										<input
											type="text"
											name="httpAuthValueTemplate"
											value={formData.httpAuthValueTemplate}
											onChange={handleInput}
											readOnly={isViewMode}
											className="input input-bordered w-full rounded-xl"
											spellCheck="false"
											autoComplete="off"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Success Codes (comma)</span>
									</label>
									<div className="col-span-9">
										<input
											type="text"
											name="httpResponseCodes"
											value={formData.httpResponseCodes}
											onChange={handleInput}
											readOnly={isViewMode}
											className={`input input-bordered w-full rounded-xl ${errors.httpResponseCodes ? 'input-error' : ''}`}
											spellCheck="false"
											autoComplete="off"
										/>
										{errors.httpResponseCodes && (
											<div className="label">
												<span className="label-text-alt text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.httpResponseCodes}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Error Mode</span>
									</label>
									<div className="col-span-9">
										<input
											type="text"
											name="httpResponseErrorMode"
											value={formData.httpResponseErrorMode}
											onChange={handleInput}
											readOnly={isViewMode}
											className="input input-bordered w-full rounded-xl"
											spellCheck="false"
											autoComplete="off"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Body Output Mode</span>
									</label>
									<div className="col-span-9">
										{isViewMode ? (
											<ReadOnlyValue value={bodyOutputModeItems[formData.httpResponseBodyOutputMode].displayName} />
										) : (
											<Dropdown<HTTPBodyOutputMode>
												dropdownItems={bodyOutputModeItems}
												selectedKey={formData.httpResponseBodyOutputMode}
												onChange={m => {
													setFormData(p => ({ ...p, httpResponseBodyOutputMode: m }));
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
							<label className="label col-span-3">
								<span className="label-text text-sm">Tags</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									name="tags"
									value={formData.tags}
									onChange={handleInput}
									readOnly={isViewMode}
									className={`input input-bordered w-full rounded-xl ${errors.tags ? 'input-error' : ''}`}
									placeholder="comma, separated, tags"
									spellCheck="false"
									aria-invalid={Boolean(errors.tags)}
								/>
								{errors.tags && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.tags}
										</span>
									</div>
								)}
							</div>
						</div>

						{isViewMode && initialData?.tool && (
							<>
								<div className="divider">Metadata</div>
								<div className="grid grid-cols-12 gap-2 text-sm">
									<div className="col-span-3 font-semibold">ID</div>
									<div className="col-span-9">{initialData.tool.id}</div>
									<div className="col-span-3 font-semibold">Schema</div>
									<div className="col-span-9">{initialData.tool.schemaVersion}</div>
									<div className="col-span-3 font-semibold">LLM Tool Type</div>
									<div className="col-span-9">{initialData.tool.llmToolType}</div>
									<div className="col-span-3 font-semibold">Built-in</div>
									<div className="col-span-9">{initialData.tool.isBuiltIn ? 'Yes' : 'No'}</div>
									<div className="col-span-3 font-semibold">Created</div>
									<div className="col-span-9">{initialData.tool.createdAt}</div>
									<div className="col-span-3 font-semibold">Modified</div>
									<div className="col-span-9">{initialData.tool.modifiedAt}</div>
								</div>
							</>
						)}

						<div className="modal-action">
							<button type="button" className="btn bg-base-300 rounded-xl" onClick={() => dialogRef.current?.close()}>
								{isViewMode ? 'Close' : 'Cancel'}
							</button>
							{!isViewMode && (
								<button type="submit" className="btn btn-primary rounded-xl" disabled={!isAllValid}>
									Save
								</button>
							)}
						</div>
					</form>
				</div>
			</div>
			<ModalBackdrop enabled={isViewMode} />
		</dialog>,
		document.body
	);
}
