import { type SubmitEventHandler, useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiX } from 'react-icons/fi';

import { type PromptBundle, PromptRoleEnum } from '@/spec/prompt';

import { focusTextInputAtEnd } from '@/lib/focus_input';
import { validateSlug } from '@/lib/text_utils';
import { DEFAULT_SEMVER, suggestNextMinorVersion } from '@/lib/version_utils';

import { Dropdown } from '@/components/dropdown';

import type { SystemPromptDraft, SystemPromptRole } from '@/prompts/lib/use_system_prompts';

type SystemPromptAddModalProps = {
	isOpen: boolean;
	mode: 'add' | 'fork';
	initialDraft: SystemPromptDraft | null;
	bundles: PromptBundle[];
	getExistingVersions: (bundleID: string, slug: string) => string[];
	onClose: () => void;
	onSave: (draft: SystemPromptDraft) => Promise<void> | void;
};

type SystemPromptAddModalInnerProps = Omit<SystemPromptAddModalProps, 'isOpen'>;

type ErrorState = {
	bundleID?: string;
	displayName?: string;
	slug?: string;
	version?: string;
	content?: string;
};

const SYSTEM_PROMPT_PLACEHOLDER_RE = /\{\{([a-zA-Z_][a-zA-Z0-9_-]*)\}\}/g;

function getSystemPromptContentError(content: string): string | undefined {
	if (!content.trim()) {
		return 'Prompt content is required.';
	}

	const names = new Set<string>();
	for (const match of content.matchAll(SYSTEM_PROMPT_PLACEHOLDER_RE)) {
		if (match[1]) {
			names.add(match[1]);
		}
	}

	if (names.size === 0) {
		return undefined;
	}

	const placeholders = [...names].map(name => `{{${name}}}`);
	return placeholders.length === 1
		? `Unresolved placeholder ${placeholders[0]} is not allowed here. Replace it with final text first.`
		: `Unresolved placeholders are not allowed here: ${placeholders.join(', ')}. Replace them with final text first.`;
}

function buildBundleLabel(bundle: PromptBundle): string {
	return `${bundle.displayName || bundle.slug} (${bundle.slug})`;
}

function pickFallbackDraft(initialDraft: SystemPromptDraft | null): SystemPromptDraft {
	return (
		initialDraft ?? {
			bundleID: '',
			displayName: 'System Prompt',
			slug: 'system-prompt',
			version: DEFAULT_SEMVER,
			role: PromptRoleEnum.System,
			content: '',
		}
	);
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

function SystemPromptAddModalInner({
	mode,
	initialDraft,
	bundles,
	getExistingVersions,
	onClose,
	onSave,
}: SystemPromptAddModalInnerProps) {
	const [formData, setFormData] = useState<SystemPromptDraft>(() => pickFallbackDraft(initialDraft));
	const [errors, setErrors] = useState<ErrorState>({});
	const [submitError, setSubmitError] = useState('');
	const [isSaving, setIsSaving] = useState(false);
	const [versionTouched, setVersionTouched] = useState(false);
	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const displayNameInputRef = useRef<HTMLInputElement | null>(null);
	useEffect(() => {
		const dialog = dialogRef.current;
		if (!dialog) return;
		let raf1 = 0;
		let raf2 = 0;
		try {
			if (!dialog.open) {
				dialog.showModal();
			}
		} catch {
			// ignore showModal errors
		}
		raf1 = window.requestAnimationFrame(() => {
			raf2 = window.requestAnimationFrame(() => {
				focusTextInputAtEnd(displayNameInputRef.current);
			});
		});

		return () => {
			window.cancelAnimationFrame(raf1);
			window.cancelAnimationFrame(raf2);
		};
	}, []);

	const bundleDropdownItems = useMemo(() => {
		const map: Record<string, { isEnabled: boolean }> = {};
		for (const bundle of bundles) {
			if (bundle.isBuiltIn) continue;
			map[bundle.id] = { isEnabled: bundle.isEnabled };
		}
		return map;
	}, [bundles]);

	const selectedBundle = useMemo(
		() => bundles.find(bundle => bundle.id === formData.bundleID) ?? null,
		[bundles, formData.bundleID]
	);

	const roleDropdownItems = useMemo<Record<SystemPromptRole, { isEnabled: boolean }>>(
		() => ({
			[PromptRoleEnum.System]: { isEnabled: true },
			[PromptRoleEnum.Developer]: { isEnabled: true },
		}),
		[]
	);

	const existingVersions = useMemo(() => {
		const slug = formData.slug.trim();
		if (!formData.bundleID || !slug) return [];
		return getExistingVersions(formData.bundleID, slug);
	}, [formData.bundleID, formData.slug, getExistingVersions]);

	const suggestedVersion = useMemo(() => {
		const seedVersion = initialDraft?.version?.trim() || DEFAULT_SEMVER;
		return suggestNextMinorVersion(seedVersion, existingVersions).suggested;
	}, [existingVersions, initialDraft?.version]);

	// effectiveVersion is used in the form and submit — no need for an effect to sync it.
	const effectiveVersion = versionTouched ? formData.version : suggestedVersion;

	const handleDialogClose = useCallback(() => {
		onClose();
	}, [onClose]);

	const requestClose = useCallback(() => {
		if (!closeDialogSafely(dialogRef.current)) {
			onClose();
		}
	}, [onClose]);

	const validateForm = (state: SystemPromptDraft): ErrorState => {
		const nextErrors: ErrorState = {};

		const selected = bundles.find(bundle => bundle.id === state.bundleID) ?? null;
		if (!state.bundleID) {
			nextErrors.bundleID = 'Bundle is required.';
		} else if (!selected) {
			nextErrors.bundleID = 'Selected bundle was not found.';
		} else if (selected.isBuiltIn) {
			nextErrors.bundleID = 'Built-in bundles cannot be used here.';
		} else if (!selected.isEnabled) {
			nextErrors.bundleID = 'Selected bundle is disabled. Enable it from Prompt Bundles first.';
		}

		if (!state.displayName.trim()) {
			nextErrors.displayName = 'Display name is required.';
		}

		const slug = state.slug.trim();
		if (!slug) {
			nextErrors.slug = 'Slug is required.';
		} else {
			const slugError = validateSlug(slug);
			if (slugError) {
				nextErrors.slug = slugError;
			}
		}

		const version = state.version.trim();
		if (!version) {
			nextErrors.version = 'Version is required.';
		} else if (slug && state.bundleID && existingVersions.includes(version)) {
			nextErrors.version = 'That version already exists for this slug in the selected bundle.';
		}

		const contentError = getSystemPromptContentError(state.content);
		if (contentError) nextErrors.content = contentError;
		return nextErrors;
	};

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();
		e.stopPropagation();

		if (isSaving) return;

		const normalized: SystemPromptDraft = {
			bundleID: formData.bundleID,
			displayName: formData.displayName.trim(),
			slug: formData.slug.trim(),
			version: effectiveVersion.trim(),
			role: formData.role,
			content: formData.content.trim(),
		};

		const nextErrors = validateForm(normalized);
		setErrors(nextErrors);

		if (Object.keys(nextErrors).length > 0) {
			return;
		}

		setIsSaving(true);
		setSubmitError('');

		void Promise.resolve(onSave(normalized))
			.then(() => {
				requestClose();
			})
			.catch((error: unknown) => {
				setSubmitError(error instanceof Error ? error.message : 'Failed to save prompt.');
			})
			.finally(() => {
				setIsSaving(false);
			});
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
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-xl overflow-auto rounded-2xl">
				<div className="mb-4 flex items-center justify-between">
					<h3 className="text-lg font-bold">{mode === 'fork' ? 'Fork System Prompt' : 'Add System Prompt'}</h3>
					<button type="button" className="btn btn-sm btn-circle bg-base-300" onClick={requestClose} aria-label="Close">
						<FiX size={12} />
					</button>
				</div>

				<form onSubmit={handleSubmit} className="space-y-4">
					{submitError ? (
						<div className="alert alert-error rounded-2xl text-sm">
							<div className="flex items-center gap-2">
								<FiAlertCircle size={14} />
								<span>{submitError}</span>
							</div>
						</div>
					) : null}
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="col-span-3 text-sm opacity-70">
							Bundle
							<HelpHint content="Use an existing custom bundle. Bundles cannot be created here." />
						</label>
						<div className="col-span-9">
							<Dropdown<string>
								dropdownItems={bundleDropdownItems}
								selectedKey={formData.bundleID}
								onChange={bundleID => {
									setFormData(prev => ({ ...prev, bundleID }));
									setErrors(prev => ({ ...prev, bundleID: undefined }));
								}}
								filterDisabled={false}
								title="Select bundle"
								getDisplayName={bundleID => {
									const bundle = bundles.find(item => item.id === bundleID);
									return bundle ? buildBundleLabel(bundle) : bundleID;
								}}
							/>
							{errors.bundleID ? <div className="text-error mt-1 text-xs">{errors.bundleID}</div> : null}
							<div className="mt-1 text-xs opacity-70">Only enabled custom bundles can accept new prompt versions.</div>
						</div>
					</div>
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="col-span-3 text-sm opacity-70">Display Name</label>
						<div className="col-span-9">
							<input
								ref={displayNameInputRef}
								type="text"
								className={`input input-bordered w-full rounded-xl ${errors.displayName ? 'input-error' : ''}`}
								value={formData.displayName}
								onChange={e => {
									setFormData(prev => ({ ...prev, displayName: e.target.value }));
									setErrors(prev => ({ ...prev, displayName: undefined }));
								}}
								spellCheck="false"
							/>
							{errors.displayName ? <div className="text-error mt-1 text-xs">{errors.displayName}</div> : null}
						</div>
					</div>
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="col-span-3 text-sm opacity-70">Slug</label>
						<div className="col-span-9">
							<input
								type="text"
								className={`input input-bordered w-full rounded-xl ${errors.slug ? 'input-error' : ''}`}
								value={formData.slug}
								onChange={e => {
									setFormData(prev => ({ ...prev, slug: e.target.value }));
									setErrors(prev => ({ ...prev, slug: undefined }));
								}}
								spellCheck="false"
							/>
							{errors.slug ? <div className="text-error mt-1 text-xs">{errors.slug}</div> : null}
						</div>
					</div>
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="col-span-3 text-sm opacity-70">
							Version
							<HelpHint content="Fork usually keeps the same slug and creates a new version." />
						</label>
						<div className="col-span-9">
							<input
								type="text"
								className={`input input-bordered w-full rounded-xl ${errors.version ? 'input-error' : ''}`}
								value={effectiveVersion}
								onChange={e => {
									setVersionTouched(true);
									setFormData(prev => ({ ...prev, version: e.target.value }));
									setErrors(prev => ({ ...prev, version: undefined }));
								}}
								spellCheck="false"
							/>
							<div className="mt-1 text-xs opacity-70">
								Suggested next version: {suggestedVersion}
								{selectedBundle ? ` in ${buildBundleLabel(selectedBundle)}` : ''}
							</div>
							{errors.version ? <div className="text-error mt-1 text-xs">{errors.version}</div> : null}
						</div>
					</div>
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="col-span-3 text-sm opacity-70">Role</label>
						<div className="col-span-9">
							<Dropdown<SystemPromptRole>
								dropdownItems={roleDropdownItems}
								selectedKey={formData.role}
								onChange={role => {
									setFormData(prev => ({ ...prev, role }));
								}}
								filterDisabled={false}
								title="Select role"
								getDisplayName={role => (role === PromptRoleEnum.Developer ? 'Developer' : 'System')}
							/>
						</div>
					</div>
					<div>
						<textarea
							className="textarea textarea-bordered h-40 w-full rounded-xl"
							value={formData.content}
							onChange={e => {
								const nextContent = e.target.value;
								setFormData(prev => ({ ...prev, content: nextContent }));
								setErrors(prev => ({
									...prev,
									content: getSystemPromptContentError(nextContent),
								}));
							}}
							placeholder="Enter system/developer instructions here..."
							spellCheck="false"
						/>
						{errors.content ? <div className="text-error mt-1 text-xs">{errors.content}</div> : null}
					</div>
					<div className="modal-action">
						<button type="button" className="btn bg-base-300 rounded-xl" onClick={requestClose}>
							Cancel
						</button>
						<button type="submit" className="btn btn-primary rounded-xl" disabled={isSaving}>
							{isSaving ? 'Saving…' : mode === 'fork' ? 'Fork' : 'Save'}
						</button>
					</div>
				</form>
			</div>
		</dialog>
	);
}

export function SystemPromptAddModal({
	isOpen,
	mode,
	initialDraft,
	bundles,
	getExistingVersions,
	onClose,
	onSave,
}: SystemPromptAddModalProps) {
	if (!isOpen || typeof document === 'undefined') return null;
	const key = initialDraft
		? `${mode}:${initialDraft.bundleID}:${initialDraft.slug}:${initialDraft.version}`
		: `${mode}:new`;

	return createPortal(
		<SystemPromptAddModalInner
			key={key}
			mode={mode}
			initialDraft={initialDraft}
			bundles={bundles}
			getExistingVersions={getExistingVersions}
			onClose={onClose}
			onSave={onSave}
		/>,
		document.body
	);
}
