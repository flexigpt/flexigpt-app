import type { ChangeEvent, SubmitEventHandler } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiCopy, FiHelpCircle, FiUpload, FiX } from 'react-icons/fi';

import type { Skill, SkillArgument, SkillInsert } from '@/spec/skill';
import { SkillType } from '@/spec/skill';

import { omitManyKeys } from '@/lib/obj_utils';
import { validateSlug, validateTags } from '@/lib/text_utils';

import { useDialogController } from '@/hooks/use_dialog_controller';

import { skillStoreAPI } from '@/apis/baseapi';

import { Dropdown } from '@/components/dropdown';
import { MANAGEMENT_MODAL_FORM_CLASS } from '@/components/managementui/management_class_consts';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';
import { ReadOnlyValue } from '@/components/read_only_value';

import {
	buildSkillArgumentText,
	buildSkillForkBodyPlaceholder,
	buildSkillMarkdownScaffold,
	formatSkillArgumentList,
	getSkillArgumentCountLabel,
	getSkillInsertDescription,
	getSkillInsertLabel,
	getSkillInsertLongGuidance,
	getSkillResourceCountLabel,
	getSkillResourceTooltip,
	normalizeSkillInsert,
	stringifySkillFrontmatter,
} from '@/skills/lib/skill_artifact_utils';

interface SkillArtifactCreateInput {
	name: string;
	displayName?: string;
	description?: string;
	insert: SkillInsert;
	arguments?: SkillArgument[];
	tags?: string[];
	markdownBody: string;
	isEnabled: boolean;
}

export interface SkillUpsertInput extends Partial<Skill> {
	artifactCreate?: SkillArtifactCreateInput;
}

function buildSkillPreviewArgs(args?: SkillArgument[] | null): Record<string, string> {
	return Object.fromEntries((args ?? []).map(arg => [arg.name, arg.default ?? ''] as const));
}

export interface SkillItem {
	skill: Skill;
	bundleID: string;
	skillSlug: string;
}

type ModalMode = 'add' | 'edit' | 'view' | 'fork';

interface AddEditSkillModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (skillData: SkillUpsertInput) => Promise<void>;
	initialData?: SkillItem; // editing/viewing
	existingSkills: SkillItem[];
	prefillSkills?: SkillItem[];

	mode?: ModalMode;
}

interface ErrorState {
	displayName?: string;
	name?: string;
	slug?: string;
	type?: string;
	location?: string;
	tags?: string;
	markdownBody?: string;
}

interface SkillFormData {
	displayName: string;
	name: string;
	slug: string;
	type: SkillType;
	location: string;
	description: string;
	tags: string;
	isEnabled: boolean;
}

const skillTypeDropdownItems: Record<SkillType, { isEnabled: boolean; displayName: string }> = {
	[SkillType.FS]: { isEnabled: true, displayName: 'Filesystem (fs)' },
	// UI restriction: EmbeddedFS skills are built-in; not allowed to be created/edited.
	[SkillType.EmbeddedFS]: { isEnabled: false, displayName: 'EmbeddedFS (embeddedfs)' },
};

const skillInsertDropdownItems = {
	instructions: { isEnabled: true, displayName: 'Instructions' },
	'user-message': { isEnabled: true, displayName: 'User-message template' },
} as Record<SkillInsert, { isEnabled: boolean; displayName: string }>;

const skillInsertOrderedKeys = ['user-message', 'instructions'] as SkillInsert[];

const ARGUMENT_NAME_RE = /^[A-Za-z_][A-Za-z0-9_]*$/;
const SKILL_ARTIFACT_NAME_RE = /^[a-z0-9][a-z0-9-]{0,63}$/;

function normalizeForUniq(s: string) {
	return s.trim().toLowerCase();
}

function getInitialFormData(
	initialData: SkillItem | undefined,
	existingSkills: SkillItem[],
	mode: ModalMode
): SkillFormData {
	if (initialData) {
		const s = initialData.skill;
		if (mode === 'fork') {
			const baseName = `${s.name || s.slug}-copy`
				.toLowerCase()
				.replaceAll(/[^a-z0-9-]+/g, '-')
				.replaceAll(/-+/g, '-')
				.replaceAll(/^-|-$/g, '')
				.slice(0, 64);
			const name = makeUniqueSkillArtifactName(baseName || 'forked-skill', existingSkills);
			const slug = makeUniqueSlug(
				`${s.slug || name}-copy`,
				existingSkills.map(item => item.skill.slug)
			);

			return {
				displayName: `${s.displayName || s.name || s.slug} Copy`,
				name,
				slug,
				type: SkillType.FS,
				location: '',
				description: s.description ?? '',
				tags: (s.tags ?? []).join(', '),
				isEnabled: true,
			};
		}

		return {
			displayName: s.displayName ?? '',
			name: s.name ?? '',
			slug: s.slug ?? '',
			type: s.type,
			location: s.location ?? '',
			description: s.description ?? '',
			tags: (s.tags ?? []).join(', '),
			isEnabled: s.isEnabled,
		};
	}

	return {
		displayName: '',
		name: '',
		slug: '',
		type: SkillType.FS,
		location: '',
		description: '',
		tags: '',
		isEnabled: true,
	};
}

function makeUniqueSlug(seed: string, existingSlugs: string[]): string {
	const normalized = seed
		.trim()
		.toLowerCase()
		.replaceAll(/[^a-z0-9-]+/g, '-')
		.replaceAll(/-+/g, '-')
		.replaceAll(/^-|-$/g, '');
	const base = normalized || 'skill';
	const existing = new Set(existingSlugs);
	if (!existing.has(base)) {
		return base;
	}

	for (let i = 2; i < 1000; i += 1) {
		const candidate = `${base}-${i}`;
		if (!existing.has(candidate)) {
			return candidate;
		}
	}

	return `${base}-${Date.now().toString(36)}`;
}

function makeUniqueSkillArtifactName(seed: string, existingSkills: SkillItem[]): string {
	const existing = new Set(existingSkills.map(item => item.skill.name));
	let candidate = seed.slice(0, 64);
	for (let i = 2; existing.has(candidate); i += 1) {
		candidate = `${seed.slice(0, Math.max(1, 63 - String(i).length))}-${i}`;
	}
	return candidate;
}

function buildSkillPrefillKey(item: SkillItem): string {
	return `${item.bundleID}:${item.skill.id}`;
}

function parseScaffoldArgumentLines(text: string): SkillArgument[] {
	const out: SkillArgument[] = [];
	const seen = new Set<string>();

	for (const rawLine of text.split('\n')) {
		const line = rawLine.trim();
		if (!line || line.startsWith('#')) {
			continue;
		}

		const [rawName, rawDescription, rawDefault] = line.split('|');
		const name = rawName?.trim() ?? '';
		if (!ARGUMENT_NAME_RE.test(name) || seen.has(name)) {
			continue;
		}

		seen.add(name);
		out.push({
			name,
			description: rawDescription?.trim() || undefined,
			default: rawDefault !== undefined ? rawDefault.trim() : undefined,
		});
	}

	return out;
}

function validateScaffoldArgumentLines(text: string): string | undefined {
	const seen = new Set<string>();

	for (const [idx, rawLine] of text.split('\n').entries()) {
		const line = rawLine.trim();
		if (!line || line.startsWith('#')) {
			continue;
		}

		const columns = line.split('|');
		if (columns.length > 3) {
			return `Argument line ${idx + 1} has too many columns. Use: name | description | default.`;
		}

		const [rawName] = columns;
		const name = rawName?.trim() ?? '';

		if (!name) {
			return `Argument line ${idx + 1} is missing a name. Use: name | description | default.`;
		}
		if (!ARGUMENT_NAME_RE.test(name)) {
			return `Argument "${name}" is invalid. Use letters, numbers, and underscores, starting with a letter or underscore.`;
		}
		if (seen.has(name)) {
			return `Argument "${name}" is declared more than once.`;
		}

		seen.add(name);
	}

	return undefined;
}

function AddEditSkillModalContent({
	onClose,
	onSubmit,
	initialData,
	existingSkills,
	prefillSkills,
	mode,
}: AddEditSkillModalProps) {
	const requestedMode: ModalMode = mode ?? (initialData ? 'edit' : 'add');
	const isForkMode = requestedMode === 'fork';
	// Match the Tool modal pattern: unsupported impls can exist (viewable),
	// but cannot be created/edited in the UI.
	const isLockedSkill =
		!isForkMode && (Boolean(initialData?.skill?.isBuiltIn) || initialData?.skill?.type === SkillType.EmbeddedFS);
	const effectiveMode: ModalMode = isLockedSkill ? 'view' : isForkMode ? 'add' : requestedMode;
	const isViewMode = effectiveMode === 'view';
	const isEditMode = effectiveMode === 'edit';
	const isAddMode = effectiveMode === 'add';

	const [creationMode, setCreationMode] = useState<'create' | 'register'>('create');
	const [formData, setFormData] = useState<SkillFormData>(() =>
		getInitialFormData(initialData, existingSkills, requestedMode)
	);
	const [errors, setErrors] = useState<ErrorState>({});
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);
	const [prefillMode, setPrefillMode] = useState(false);
	const [selectedPrefillKey, setSelectedPrefillKey] = useState<string | null>(null);
	const [previewArgs, setPreviewArgs] = useState<Record<string, string>>(() =>
		buildSkillPreviewArgs(initialData?.skill?.arguments)
	);
	const [previewResult, setPreviewResult] = useState<{
		text: string;
		insert: SkillInsert;
		appliedArguments?: Record<string, string>;
		warnings?: string[];
	} | null>(null);
	const [previewLoading, setPreviewLoading] = useState(false);
	const [previewError, setPreviewError] = useState('');
	const [scaffoldInsert, setScaffoldInsert] = useState<SkillInsert>(() =>
		isForkMode ? normalizeSkillInsert(initialData?.skill?.insert).value : 'user-message'
	);
	const [scaffoldArgumentsText, setScaffoldArgumentsText] = useState(() =>
		isForkMode ? buildSkillArgumentText(initialData?.skill?.arguments) : ''
	);
	const [scaffoldBody, setScaffoldBody] = useState(() =>
		isForkMode && initialData?.skill ? buildSkillForkBodyPlaceholder(initialData.skill) : ''
	);
	const [scaffoldCopied, setScaffoldCopied] = useState(false);
	const [locationCopied, setLocationCopied] = useState(false);

	const artifactSkill = initialData?.skill;
	const artifactArguments = artifactSkill?.arguments ?? [];
	const normalizedArtifactInsert = normalizeSkillInsert(artifactSkill?.insert);
	const artifactArgumentLines = formatSkillArgumentList(artifactSkill?.arguments);
	const artifactFrontmatter = stringifySkillFrontmatter(artifactSkill?.rawFrontmatter);
	const scaffoldArgumentError = validateScaffoldArgumentLines(scaffoldArgumentsText);

	const { dialogRef, requestClose, handleClose, handleCancel, unmountingRef } = useDialogController({
		onClose,
		blockCancel: !isViewMode,
		isBusy: isSubmitting,
	});
	const nameInputRef = useRef<HTMLInputElement | null>(null);

	const scaffoldArguments = useMemo(() => parseScaffoldArgumentLines(scaffoldArgumentsText), [scaffoldArgumentsText]);
	const scaffoldMarkdown = useMemo(
		() =>
			buildSkillMarkdownScaffold({
				name: formData.name,
				displayName: formData.displayName,
				description: formData.description,
				insert: scaffoldInsert,
				arguments: scaffoldArguments,
				body: scaffoldBody,
			}),
		[formData.description, formData.displayName, formData.name, scaffoldArguments, scaffoldBody, scaffoldInsert]
	);

	const validateField = (field: keyof ErrorState, val: string | SkillType, currentErrors: ErrorState): ErrorState => {
		let nextErrors: ErrorState = { ...currentErrors };
		const v = val.trim();

		const requiredFields: Array<keyof ErrorState> = ['name', 'slug', 'type', 'location'];

		if (!v && requiredFields.includes(field)) {
			nextErrors[field] = 'This field is required.';
			return nextErrors;
		}

		if (field === 'slug') {
			const err = validateSlug(v);
			if (err) {
				nextErrors.slug = err;
			} else {
				const clash = existingSkills.some(x => x.skill.slug === v && x.skill.id !== initialData?.skill.id);
				if (clash) {
					nextErrors.slug = 'Slug already in use in this bundle.';
				} else {
					nextErrors = omitManyKeys(nextErrors, ['slug']);
				}
			}
		} else if (field === 'name') {
			// Constraint: within a bundle, skill.name cannot be duplicated
			const norm = normalizeForUniq(v);
			const clash = existingSkills.some(
				x => normalizeForUniq(x.skill.name) === norm && x.skill.id !== initialData?.skill.id
			);
			if (clash) {
				nextErrors.name = 'Skill name must be unique within the bundle.';
			} else if (!SKILL_ARTIFACT_NAME_RE.test(v)) {
				nextErrors.name =
					'Skill name must use lowercase letters, numbers, and hyphens, start with a letter or number, and be at most 64 characters.';
			} else {
				nextErrors = omitManyKeys(nextErrors, ['name']);
			}
		} else if (field === 'tags') {
			if (v === '') {
				nextErrors = omitManyKeys(nextErrors, ['tags']);
			} else {
				const err = validateTags(val);
				if (err) {
					nextErrors.tags = err;
				} else {
					nextErrors = omitManyKeys(nextErrors, ['tags']);
				}
			}
		} else {
			nextErrors = omitManyKeys(nextErrors, [field]);
		}

		return nextErrors;
	};

	const validateForm = (state: SkillFormData): ErrorState => {
		let next: ErrorState = {};
		next = validateField('name', state.name, next);
		next = validateField('slug', state.slug, next);
		next = validateField('type', state.type, next);
		if (!isAddMode || creationMode === 'register') {
			next = validateField('location', state.location, next);
		}
		if (state.tags.trim() !== '') {
			next = validateField('tags', state.tags, next);
		}
		if (isAddMode && creationMode === 'create' && !scaffoldBody.trim()) {
			next.markdownBody = 'SKILL.md body is required.';
		}
		if (isAddMode && creationMode === 'create' && scaffoldArgumentError) {
			next.markdownBody = scaffoldArgumentError;
		}
		return next;
	};

	// oxlint-disable-next-line jsreact-hooks/immutability
	const isAllValid = isViewMode ? true : !isSubmitting && Object.values(validateForm(formData)).every(error => !error);

	useEffect(() => {
		if (!isAddMode) {
			return;
		}

		const focusTimer = window.setTimeout(() => {
			nameInputRef.current?.focus();
		}, 0);

		return () => {
			window.clearTimeout(focusTimer);
		};
	}, [isAddMode]);

	const prefillCandidates = prefillSkills ?? existingSkills;
	const copyableSkills = useMemo(
		() => prefillCandidates.filter(item => item.skill.type === SkillType.FS || creationMode === 'create'),
		[creationMode, prefillCandidates]
	);

	const prefillSourceMap = useMemo<Record<string, SkillItem>>(() => {
		return Object.fromEntries(copyableSkills.map(item => [buildSkillPrefillKey(item), item] as const));
	}, [copyableSkills]);

	const prefillKeys = useMemo(() => Object.keys(prefillSourceMap), [prefillSourceMap]);

	const prefillDropdownItems = useMemo<Record<string, { isEnabled: boolean; displayName: string }>>(
		() =>
			Object.fromEntries(
				Object.entries(prefillSourceMap).map(([key, item]) => [
					key,
					{
						isEnabled: true,
						displayName: `${item.skill.displayName || item.skill.name || item.skill.slug} (${item.skill.slug})`,
					},
				])
			),
		[prefillSourceMap]
	);

	const copyScaffold = useCallback(async () => {
		await navigator.clipboard.writeText(scaffoldMarkdown);
		setScaffoldCopied(true);
		window.setTimeout(() => {
			setScaffoldCopied(false);
		}, 1400);
	}, [scaffoldMarkdown]);

	const copyLocation = useCallback(async () => {
		await navigator.clipboard.writeText(artifactSkill?.location ?? '');
		setLocationCopied(true);
		window.setTimeout(() => {
			setLocationCopied(false);
		}, 1400);
	}, [artifactSkill?.location]);

	const resetPreviewArgs = useCallback(() => {
		setPreviewArgs(buildSkillPreviewArgs(artifactArguments));
		setPreviewResult(null);
		setPreviewError('');
		// oxlint-disable-next-line react-hooks/exhaustive-deps
	}, [artifactArguments]);

	const handleRenderPreview = useCallback(async () => {
		if (!artifactSkill || !initialData) {
			return;
		}

		setPreviewLoading(true);
		setPreviewError('');

		try {
			const resp = await skillStoreAPI.renderSkill(
				{
					bundleID: initialData.bundleID,
					skillSlug: artifactSkill.slug,
					skillID: artifactSkill.id,
				},
				previewArgs
			);

			setPreviewResult({
				text: resp.text,
				insert: resp.insert,
				appliedArguments: resp.appliedArguments ?? {},
				warnings: resp.warnings ?? [],
			});
		} catch (err) {
			setPreviewResult(null);
			setPreviewError(err instanceof Error && err.message.trim() ? err.message : 'Failed to render skill preview.');
		} finally {
			setPreviewLoading(false);
		}
	}, [artifactSkill, initialData, previewArgs]);

	const changeCreationMode = (nextMode: 'create' | 'register') => {
		setCreationMode(nextMode);
		setSubmitError('');
		setErrors(current =>
			nextMode === 'create' ? omitManyKeys(current, ['location']) : omitManyKeys(current, ['markdownBody'])
		);
	};

	const applyPrefill = (key: string) => {
		const source = prefillSourceMap[key];
		if (!source) {
			return;
		}

		const src = source.skill;
		const next: SkillFormData = {
			...formData,
			displayName: src.displayName ?? '',
			name: formData.name,
			slug: formData.slug,
			type: SkillType.FS,
			location: creationMode === 'register' ? (src.location ?? '') : '',
			description: src.description ?? '',
			tags: (src.tags ?? []).join(', '),
			isEnabled: true,
		};

		setFormData(next);
		if (creationMode === 'create') {
			setScaffoldInsert(normalizeSkillInsert(src.insert).value);
			setScaffoldArgumentsText(buildSkillArgumentText(src.arguments));
			setScaffoldBody(buildSkillForkBodyPlaceholder(src));
		}
		setErrors(validateForm(next));
		setSubmitError('');
		setSelectedPrefillKey(key);
		setPrefillMode(false);
	};

	const onSkillTypeChange = (key: SkillType) => {
		// UI restriction: only FS skills can be created/edited
		if (key !== SkillType.FS) {
			return;
		}
		setFormData(prev => ({ ...prev, type: key }));
		setErrors(prev => validateField('type', key, prev));
	};

	const handleInput = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
		const target = e.target as HTMLInputElement;
		const { name, value, type, checked } = target;
		const newVal = type === 'checkbox' ? checked : value;

		setFormData(prev => ({ ...prev, [name]: newVal }));

		if (['displayName', 'name', 'slug', 'type', 'location', 'tags'].includes(name)) {
			setErrors(prev => validateField(name as keyof ErrorState, String(newVal), prev));
		}
	};

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

		const common = {
			displayName: formData.displayName.trim() || undefined,
			name: formData.name.trim(),
			slug: formData.slug.trim(),
			type: formData.type,
			description: formData.description.trim(),
			tags: tagsArr,
			isEnabled: formData.isEnabled,
		};

		const payload: SkillUpsertInput =
			isAddMode && creationMode === 'create'
				? {
						...common,
						artifactCreate: {
							name: formData.name.trim(),
							displayName: common.displayName,
							description: common.description,
							insert: scaffoldInsert,
							arguments: scaffoldArguments,
							tags: tagsArr,
							markdownBody: scaffoldBody,
							isEnabled: formData.isEnabled,
						},
					}
				: {
						...common,
						location: formData.location.trim(),
					};

		setIsSubmitting(true);
		void onSubmit(payload)
			.then(() => {
				requestClose(true);
			})
			.catch((err: unknown) => {
				const msg = err instanceof Error ? err.message : 'Failed to save skill.';
				setSubmitError(msg);
			})
			.finally(() => {
				if (!unmountingRef.current) {
					setIsSubmitting(false);
				}
			});
	};

	const headerTitle = isForkMode
		? 'Fork Skill'
		: effectiveMode === 'view'
			? 'View Skill'
			: effectiveMode === 'edit'
				? 'Edit Skill'
				: 'Add Skill or Template';

	return (
		<dialog ref={dialogRef} className="modal" onClose={handleClose} onCancel={handleCancel}>
			<div className="modal-box bg-base-200 max-h-[85vh] max-w-5xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[85vh] overflow-y-auto p-4 sm:p-6">
					<ModalHeader
						title={headerTitle}
						description="Manage filesystem skill metadata, rendering behavior, resources, and runtime visibility."
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

						{isAddMode && (
							<ModalSection title="Skill behavior">
								<ul className="text-base-content/70 list-disc space-y-1 pl-5 text-xs">
									<li>Instructions provide reusable conversation context.</li>
									<li>User-message templates insert text into the composer.</li>
									<li>Arguments are optional string substitutions.</li>
								</ul>
							</ModalSection>
						)}

						{isForkMode && artifactSkill && (
							<div className="border-warning/30 rounded-xl border p-3 text-sm">
								The fork copies metadata and arguments. Replace the generated placeholder body before saving.
							</div>
						)}

						{isAddMode && (
							<ModalSection title="Source">
								<div className="grid grid-cols-1 gap-2 md:grid-cols-2">
									<label className="border-base-content/10 hover:bg-base-200 flex cursor-pointer items-start gap-3 rounded-2xl border p-3">
										<input
											type="radio"
											className="radio radio-sm mt-1"
											checked={creationMode === 'create'}
											onChange={() => {
												changeCreationMode('create');
											}}
										/>
										<span>
											<span className="block font-medium">Create managed SKILL.md</span>
											<span className="text-base-content/70 block text-xs">
												Create and register a managed skill folder.
											</span>
										</span>
									</label>
									<label className="border-base-content/10 hover:bg-base-200 flex cursor-pointer items-start gap-3 rounded-2xl border p-3">
										<input
											type="radio"
											className="radio radio-sm mt-1"
											checked={creationMode === 'register'}
											onChange={() => {
												changeCreationMode('register');
											}}
										/>
										<span>
											<span className="block font-medium">Register existing folder</span>
											<span className="text-base-content/70 block text-xs">
												Use an existing folder containing SKILL.md.
											</span>
										</span>
									</label>
								</div>
							</ModalSection>
						)}

						{isAddMode && (
							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3">
									<span className="text-sm">Prefill from Existing</span>
								</label>

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
													? 'No compatible source skills are available to copy into this managed skill.'
													: undefined
											}
										>
											<FiUpload size={14} />
											<span className="ml-1">Copy Existing Skill</span>
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
												title="Select skill to copy"
												getDisplayName={key => prefillDropdownItems[key]?.displayName ?? 'Select skill to copy'}
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

						{isAddMode && creationMode === 'create' && (
							<div className="collapse-arrow border-base-content/10 bg-base-100 collapse rounded-2xl border">
								<input type="checkbox" defaultChecked />
								<div className="collapse-title text-sm font-semibold">Managed SKILL.md content</div>
								<div className="collapse-content space-y-4 text-sm">
									<div className="grid grid-cols-12 items-start gap-2">
										<label className="label col-span-12 sm:col-span-3">
											<span className="text-sm">Insert</span>
											<span
												className="tooltip tooltip-right"
												data-tip="instructions becomes session context. user-message becomes a composer template."
											>
												<FiHelpCircle size={12} />
											</span>
										</label>
										<div className="col-span-12 space-y-1 sm:col-span-9">
											<Dropdown<SkillInsert>
												dropdownItems={skillInsertDropdownItems}
												orderedKeys={skillInsertOrderedKeys}
												selectedKey={scaffoldInsert}
												onChange={setScaffoldInsert}
												filterDisabled={false}
												title="Select insert behavior"
												getDisplayName={key => skillInsertDropdownItems[key]?.displayName ?? key}
											/>
											<div className="text-base-content/70 text-xs">{getSkillInsertLongGuidance(scaffoldInsert)}</div>
										</div>
									</div>

									<div className="grid grid-cols-12 items-start gap-2">
										<label className="label col-span-12 sm:col-span-3">
											<span className="text-sm">Arguments</span>
											<span
												className="tooltip tooltip-right"
												data-tip="One per line: name | description | default. Names must match [A-Za-z_][A-Za-z0-9_]*."
											>
												<FiHelpCircle size={12} />
											</span>
										</label>
										<div className="col-span-12 sm:col-span-9">
											<textarea
												className="textarea h-24 w-full rounded-xl font-mono text-xs"
												value={scaffoldArgumentsText}
												onChange={e => {
													setScaffoldArgumentsText(e.target.value);
												}}
												placeholder={'topic | Topic to explain | AI agents\ntext | Text to summarize |'}
												spellCheck="false"
											/>
											<div className="label">
												<span className="text-base-content/70 text-xs">
													Parsed {scaffoldArguments.length} argument{scaffoldArguments.length === 1 ? '' : 's'}. Missing
													values render as defaults, then empty strings.
												</span>
											</div>
											{scaffoldArgumentError && (
												<div className="label">
													<span className="text-error flex items-center gap-1">
														<FiAlertCircle size={12} /> {scaffoldArgumentError}
													</span>
												</div>
											)}
										</div>
									</div>

									<div className="grid grid-cols-12 items-start gap-2">
										<label className="label col-span-12 sm:col-span-3">
											<span className="text-sm">Body</span>
										</label>
										<div className="col-span-12 sm:col-span-9">
											<textarea
												className="textarea h-28 w-full rounded-xl"
												value={scaffoldBody}
												onChange={e => {
													setScaffoldBody(e.target.value);
												}}
												placeholder={
													scaffoldInsert === 'user-message'
														? 'Summarize the following text in a $tone tone:\n\n$text'
														: 'Always follow these instructions when this skill is active...'
												}
												spellCheck="false"
											/>
											{errors.markdownBody && (
												<div className="label">
													<span className="text-error flex items-center gap-1">
														<FiAlertCircle size={12} /> {errors.markdownBody}
													</span>
												</div>
											)}
										</div>
									</div>

									<div className="flex items-center justify-between">
										<div className="text-base-content/70 text-xs">Generated SKILL.md</div>
										<button type="button" className="btn btn-sm btn-ghost rounded-xl" onClick={copyScaffold}>
											<FiCopy size={14} />
											<span className="ml-1">{scaffoldCopied ? 'Copied' : 'Copy'}</span>
										</button>
									</div>
									<pre className="bg-base-200 max-h-72 overflow-auto rounded-2xl p-3 text-xs whitespace-pre-wrap">
										{scaffoldMarkdown}
									</pre>
								</div>
							</div>
						)}

						<ModalSection title="Identity">
							<ModalField
								label="Name"
								htmlFor="skill-name"
								required
								hint="Artifact name from SKILL.md."
								error={errors.name}
							>
								<input
									id="skill-name"
									ref={nameInputRef}
									type="text"
									name="name"
									value={formData.name}
									onChange={handleInput}
									readOnly={isViewMode || isEditMode}
									className={`input w-full rounded-xl ${errors.name ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									aria-invalid={Boolean(errors.name)}
								/>
							</ModalField>

							<ModalField
								label="Slug"
								htmlFor="skill-slug"
								required
								hint="Store identifier within this bundle."
								error={errors.slug}
							>
								<input
									id="skill-slug"
									type="text"
									name="slug"
									value={formData.slug}
									onChange={handleInput}
									readOnly={isViewMode || isEditMode}
									className={`input w-full rounded-xl ${errors.slug ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									aria-invalid={Boolean(errors.slug)}
								/>
							</ModalField>

							<ModalField label="Type" required hint="Built-in EmbeddedFS skills are read only." error={errors.type}>
								{isEditMode || isViewMode ? (
									<ReadOnlyValue value={skillTypeDropdownItems[formData.type].displayName} />
								) : (
									<Dropdown<SkillType>
										dropdownItems={skillTypeDropdownItems}
										selectedKey={formData.type}
										onChange={onSkillTypeChange}
										filterDisabled={true}
										title="Select skill type"
										getDisplayName={k => skillTypeDropdownItems[k].displayName}
									/>
								)}
							</ModalField>

							<ModalField
								label="Location"
								htmlFor="skill-location"
								required={!isAddMode || creationMode === 'register'}
								hint="Folder containing SKILL.md."
								error={errors.location}
							>
								<input
									id="skill-location"
									type="text"
									name="location"
									value={
										isAddMode && creationMode === 'create'
											? 'Managed automatically in the app skill store'
											: formData.location
									}
									onChange={handleInput}
									readOnly={isViewMode || (isAddMode && creationMode === 'create')}
									className={`input w-full rounded-xl ${errors.location ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									aria-invalid={Boolean(errors.location)}
								/>
							</ModalField>
						</ModalSection>

						<ModalSection title="Presentation">
							<ModalField
								label="Display Name"
								htmlFor="skill-display-name"
								hint="Falls back to the skill name when empty."
							>
								<input
									id="skill-display-name"
									type="text"
									name="displayName"
									value={formData.displayName}
									onChange={handleInput}
									readOnly={isViewMode}
									className="input w-full rounded-xl"
									spellCheck="false"
									autoComplete="off"
								/>
							</ModalField>

							<ModalField label="Enabled" htmlFor="skill-enabled">
								<input
									id="skill-enabled"
									type="checkbox"
									name="isEnabled"
									checked={formData.isEnabled}
									onChange={handleInput}
									className="toggle toggle-accent disabled:opacity-80"
									disabled={isViewMode}
								/>
							</ModalField>

							<ModalField label="Description" htmlFor="skill-description" align="start">
								<textarea
									id="skill-description"
									name="description"
									value={formData.description}
									onChange={handleInput}
									readOnly={isViewMode}
									className="textarea h-20 w-full rounded-xl"
									spellCheck="false"
								/>
							</ModalField>

							<ModalField label="Tags" htmlFor="skill-tags" error={errors.tags}>
								<input
									id="skill-tags"
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
							</ModalField>
						</ModalSection>

						{artifactSkill && !isForkMode && (
							<>
								<ModalSection title="Artifact metadata">
									<ManagementInfoGrid>
										<ManagementInfoRow label="Artifact fields" className="sm:grid-cols-1">
											<div className="grid grid-cols-12 gap-2 text-sm">
												<div className="col-span-3 font-semibold">Insert</div>
												<div className="col-span-9 space-y-1">
													<div className="flex items-center gap-2">
														<MetadataPill>{getSkillInsertLabel(artifactSkill.insert)}</MetadataPill>
														{normalizedArtifactInsert.isDefaulted && <MetadataPill>Default behavior</MetadataPill>}
													</div>
													<div className="text-base-content/70 text-xs">
														{getSkillInsertDescription(artifactSkill.insert)}
													</div>
												</div>

												<div className="col-span-3 font-semibold">Arguments</div>
												<div className="col-span-9 space-y-2">
													<div className="text-base-content/70 text-xs">
														{getSkillArgumentCountLabel(artifactSkill.arguments)}
													</div>
													{artifactArgumentLines.length > 0 ? (
														<ul className="space-y-1">
															{artifactArgumentLines.map((line, idx) => (
																<li
																	key={`${line}-${idx}`}
																	className="bg-base-100 rounded-xl px-3 py-2 text-xs whitespace-pre-wrap"
																>
																	{line}
																</li>
															))}
														</ul>
													) : (
														<div className="text-base-content/70 text-xs">No arguments declared.</div>
													)}
												</div>

												<div className="col-span-3 font-semibold">Resources</div>
												<div className="col-span-9 space-y-2">
													<div className="flex flex-wrap items-center gap-2">
														<MetadataPill title={getSkillResourceTooltip(artifactSkill.resources)}>
															{getSkillResourceCountLabel(artifactSkill.resources)}
														</MetadataPill>
														<span className="text-base-content/70 text-xs">
															Resource files are regular files under the skill folder. They are not automatically
															executed or rendered by this page.
														</span>
													</div>
													{artifactSkill.resources?.locations?.length ? (
														<ul className="space-y-1">
															{artifactSkill.resources.locations.map(location => (
																<li
																	key={location}
																	className="bg-base-100 rounded-xl px-3 py-2 font-mono text-xs break-all"
																>
																	{location}
																</li>
															))}
														</ul>
													) : null}
												</div>

												<div className="col-span-3 font-semibold">Digest</div>
												<div className="col-span-9 font-mono text-xs break-all">{artifactSkill.digest || '-'}</div>

												<div className="col-span-3 font-semibold">Runtime warnings</div>
												<div className="col-span-9 space-y-1">
													{artifactSkill.runtimeWarnings?.length ? (
														<ul className="list-disc space-y-1 pl-5 text-xs">
															{artifactSkill.runtimeWarnings.map((warning, idx) => (
																<li key={`${idx}-${warning}`}>{warning}</li>
															))}
														</ul>
													) : (
														<div className="text-base-content/70 text-xs">No runtime warnings recorded.</div>
													)}
												</div>

												<div className="col-span-3 font-semibold">Raw frontmatter</div>
												<div className="col-span-9">
													{artifactFrontmatter ? (
														<pre className="bg-base-100 max-h-56 overflow-auto rounded-2xl p-3 text-xs whitespace-pre-wrap">
															{artifactFrontmatter}
														</pre>
													) : (
														<div className="text-base-content/70 text-xs">No raw frontmatter captured.</div>
													)}
												</div>
											</div>
										</ManagementInfoRow>
									</ManagementInfoGrid>
								</ModalSection>

								<ModalSection title="Store metadata">
									<ManagementInfoGrid>
										<ManagementInfoRow label="ID" mono>
											{artifactSkill.id}
										</ManagementInfoRow>
										<ManagementInfoRow label="Schema version">{artifactSkill.schemaVersion}</ManagementInfoRow>
										<ManagementInfoRow label="Type">{artifactSkill.type}</ManagementInfoRow>
										<ManagementInfoRow label="Location">
											<div className="space-y-2">
												<div className="col-span-3 font-semibold">Type</div>
												<div className="flex flex-wrap items-center gap-2">
													<span className="break-all">{artifactSkill.location || '-'}</span>
													{artifactSkill.location ? (
														<button type="button" className="btn btn-xs btn-ghost rounded-xl" onClick={copyLocation}>
															<FiCopy size={12} />
															<span className="ml-1">{locationCopied ? 'Copied' : 'Copy'}</span>
														</button>
													) : null}
												</div>
												<div className="text-base-content/70 mt-1 text-xs">
													To update resources or edit SKILL.md directly, change files in this folder and then re-enable
													the skill or restart the app to refresh runtime metadata.
												</div>
											</div>
										</ManagementInfoRow>
										<ManagementInfoRow label="Tags">
											{artifactSkill.tags?.length ? (
												<div className="flex flex-wrap gap-1">
													{artifactSkill.tags.map(tag => (
														<MetadataPill key={tag}>{tag}</MetadataPill>
													))}
												</div>
											) : (
												<div className="text-base-content/70 text-xs">No tags.</div>
											)}
										</ManagementInfoRow>
										<ManagementInfoRow label="Built-in">{artifactSkill.isBuiltIn ? 'Yes' : 'No'}</ManagementInfoRow>
										<ManagementInfoRow label="Presence">
											{artifactSkill.presence?.status ?? 'unknown'}
										</ManagementInfoRow>
										<ManagementInfoRow label="Created">{String(artifactSkill.createdAt)}</ManagementInfoRow>
										<ManagementInfoRow label="Modified">{String(artifactSkill.modifiedAt)}</ManagementInfoRow>
									</ManagementInfoGrid>
								</ModalSection>

								<ModalSection
									title="Render preview"
									description="Preview uses the runtime renderer and does not modify source files or stored metadata."
								>
									<div className="alert alert-info rounded-2xl text-sm">
										<div className="space-y-1">
											<div className="font-semibold">Preview the rendered skill body</div>
											<div>
												This uses the runtime render API only. It does not mutate the stored skill record or the source
												files on disk.
											</div>
											<div>Preview requires the skill to be enabled and indexed in the runtime.</div>
										</div>
									</div>

									<div className="space-y-3">
										{artifactArguments.length > 0 ? (
											<div className="grid grid-cols-1 gap-3 md:grid-cols-2">
												{artifactArguments.map(arg => (
													<div key={arg.name} className="border-base-content/10 rounded-2xl border p-3">
														<div className="flex items-start justify-between gap-2">
															<div>
																<div className="font-medium">{arg.name}</div>
																{arg.description ? (
																	<div className="text-base-content/70 mt-1 text-xs">{arg.description}</div>
																) : null}
															</div>
															{arg.default ? <MetadataPill label="Default">{arg.default}</MetadataPill> : null}
														</div>
														<input
															type="text"
															className="input bg-base-100 mt-3 w-full rounded-xl"
															value={previewArgs[arg.name] ?? ''}
															onChange={e => {
																setPreviewArgs(prev => ({
																	...prev,
																	[arg.name]: e.target.value,
																}));
																setPreviewResult(null);
																setPreviewError('');
															}}
															spellCheck="false"
															autoComplete="off"
														/>
													</div>
												))}
											</div>
										) : (
											<div className="text-base-content/70 text-xs">
												This skill declares no arguments. Rendering uses empty string substitutions and frontmatter
												defaults only.
											</div>
										)}

										<div className="flex flex-wrap gap-2">
											<button
												type="button"
												className="btn btn-sm btn-primary rounded-xl"
												onClick={handleRenderPreview}
												disabled={previewLoading}
											>
												{previewLoading ? <span className="loading loading-spinner loading-xs" /> : null}
												<span className={previewLoading ? 'ml-2' : ''}>Render Preview</span>
											</button>
											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												onClick={resetPreviewArgs}
												disabled={previewLoading}
											>
												Reset Arguments
											</button>
										</div>

										{previewError && (
											<div className="alert alert-error rounded-2xl text-sm">
												<div className="flex items-center gap-2">
													<FiAlertCircle size={14} />
													<span>{previewError}</span>
												</div>
											</div>
										)}

										{previewResult && (
											<div className="space-y-3">
												<div className="flex flex-wrap items-center gap-2 text-xs">
													<MetadataPill label="Insert">{previewResult.insert}</MetadataPill>
													{Object.keys(previewResult.appliedArguments ?? {}).length > 0 ? (
														<span className="text-base-content/70">Applied arguments captured from the renderer.</span>
													) : null}
												</div>
												<pre className="bg-base-100 max-h-72 overflow-auto rounded-2xl p-3 text-xs whitespace-pre-wrap">
													{previewResult.text || '(Rendered output is empty.)'}
												</pre>
												{previewResult.warnings?.length ? (
													<div className="space-y-1">
														<div className="text-xs font-semibold">Warnings</div>
														<ul className="list-disc space-y-1 pl-5 text-xs">
															{previewResult.warnings.map((warning, idx) => (
																<li key={`${idx}-${warning}`}>{warning}</li>
															))}
														</ul>
													</div>
												) : null}
											</div>
										)}
									</div>
								</ModalSection>
							</>
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
								<button type="submit" className="btn btn-primary rounded-xl" disabled={!isAllValid}>
									{isSubmitting ? 'Saving…' : 'Save'}
								</button>
							)}
						</ModalActions>
					</form>
				</div>
			</div>

			<ModalBackdrop enabled={isViewMode} />
		</dialog>
	);
}

export function AddEditSkillModal(props: AddEditSkillModalProps) {
	if (!props.isOpen) {
		return null;
	}
	if (typeof document === 'undefined' || !document.body) {
		return null;
	}

	const remountKey = props.initialData
		? `${props.mode ?? 'auto'}:${props.initialData.bundleID}:${props.initialData.skill.id}:${String(
				props.initialData.skill.modifiedAt
			)}:${props.initialData.skill.type}:${props.initialData.skill.isBuiltIn ? '1' : '0'}`
		: `${props.mode ?? 'auto'}:new`;

	return createPortal(<AddEditSkillModalContent key={remountKey} {...props} />, document.body);
}
