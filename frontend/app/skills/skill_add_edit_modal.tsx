import type { SubmitEventHandler } from 'react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { FiAlertCircle, FiCopy, FiFileText, FiPlay, FiRefreshCw } from 'react-icons/fi';

import type { ArtifactRef, StoreArtifact } from '@/spec/artifact';
import type { PluginListItem } from '@/spec/plugin';
import type {
	ArtifactSkillDetails,
	ManagedSkillReplaceResult,
	SkillArgument,
	SkillDocumentInput,
	StoreManagedSkillDocument,
	StoreSkillListItem,
} from '@/spec/skill';
import { SkillInsert } from '@/spec/skill';

import { skillManagementAPI } from '@/apis/baseapi';

import type { DropdownItem } from '@/components/dropdown';
import { Dropdown } from '@/components/dropdown';
import { Loader } from '@/components/loader';
import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import { MessageContentCard } from '@/chats/messages/message_content_card';
import {
	encodeSkillDocument,
	formatSkillArguments,
	makeUniqueSkillCopyName,
	parseSkillArguments,
} from '@/skills/lib/skill_management_utils';

const SKILL_NAME_PATTERN = /^[a-z0-9][a-z0-9-]{0,63}$/;

const SKILL_INSERT_DROPDOWN_KEYS = [SkillInsert.Instructions, SkillInsert.UserMessage] as const;

const SKILL_INSERT_DROPDOWN_ITEMS: Record<SkillInsert, DropdownItem> = {
	[SkillInsert.Instructions]: { isEnabled: true },
	[SkillInsert.UserMessage]: { isEnabled: true },
};

const SKILL_INSERT_DROPDOWN_LABELS: Record<SkillInsert, string> = {
	[SkillInsert.Instructions]: 'Instructions',
	[SkillInsert.UserMessage]: 'User-message template',
};

export type ManagedSkillEditorMode = 'create' | 'edit' | 'fork';

export interface ManagedSkillEditorSeed {
	document: SkillDocumentInput;
	enabled?: boolean;
}

export interface ManagedSkillEditorModalProps {
	isOpen: boolean;
	plugin: PluginListItem;
	skill?: StoreSkillListItem;
	mode?: ManagedSkillEditorMode;
	seed?: ManagedSkillEditorSeed;
	onClose: () => void;
	existingNames?: readonly string[];
	prefillDocuments?: ReadonlyArray<{
		key: string;
		label: string;
		document: SkillDocumentInput;
	}>;
	onSaved: (artifact: ArtifactRef, document: SkillDocumentInput, enabled: boolean) => Promise<void> | void;
}

interface ManagedSkillForm {
	name: string;
	displayName: string;
	description: string;
	insert: SkillInsert;
	argumentsText: string;
	tagsText: string;
	markdownBody: string;
	enabled: boolean;
	rawFrontmatter?: Record<string, unknown>;
}

interface SkillArtifactDetailsModalProps {
	isOpen: boolean;
	skill: StoreSkillListItem | null;
	onClose: () => void;
}

function emptyDocument(): SkillDocumentInput {
	return {
		name: '',
		description: '',
		insert: SkillInsert.Instructions,
		arguments: [],
		tags: [],
		markdownBody: '',
	};
}

function artifactRefFromStoreArtifact(value: StoreArtifact): ArtifactRef {
	return {
		rootID: value.rootID,
		artifactID: value.id,
	};
}

function formFromDocument(document: SkillDocumentInput, enabled: boolean): ManagedSkillForm {
	return {
		name: document.name,
		displayName: document.displayName ?? '',
		description: document.description,
		insert: document.insert,
		argumentsText: formatSkillArguments(document.arguments),
		tagsText: (document.tags ?? []).join(', '),
		markdownBody: document.markdownBody,
		enabled,
		rawFrontmatter: document.rawFrontmatter,
	};
}

function documentFromForm(form: ManagedSkillForm, argumentsList: SkillArgument[]): SkillDocumentInput {
	return {
		name: form.name.trim(),
		displayName: form.displayName.trim() || undefined,
		description: form.description.trim(),
		insert: form.insert,
		arguments: argumentsList,
		tags: form.tagsText
			.split(',')
			.map(tag => tag.trim())
			.filter(Boolean),
		markdownBody: form.markdownBody.trim(),
		rawFrontmatter: form.rawFrontmatter,
	};
}

function parseTemplateArguments(raw: string): Record<string, string> {
	const trimmed = raw.trim();
	if (!trimmed) {
		return {};
	}

	let parsed: unknown;
	try {
		parsed = JSON.parse(trimmed);
	} catch {
		throw new Error('Template arguments must be valid JSON.');
	}

	if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
		throw new Error('Template arguments must be a JSON object.');
	}

	const output: Record<string, string> = {};

	for (const [key, value] of Object.entries(parsed)) {
		if (!key.trim()) {
			throw new Error('Template argument names must not be empty.');
		}

		if (typeof value === 'string') {
			output[key] = value;
			continue;
		}
		if (typeof value === 'number' || typeof value === 'boolean') {
			output[key] = String(value);
			continue;
		}
		if (value === null) {
			output[key] = '';
			continue;
		}

		throw new Error(`Template argument "${key}" must be a string, number, boolean, or null.`);
	}

	return output;
}

function ManagedSkillEditorModalContent({
	plugin,
	skill,
	mode = 'create',
	seed,
	onClose,
	existingNames = [],
	prefillDocuments = [],
	onSaved,
}: Omit<ManagedSkillEditorModalProps, 'isOpen'>) {
	const [loadingDocument, setLoadingDocument] = useState(mode === 'edit' && Boolean(skill));
	const [documentLoadFailed, setDocumentLoadFailed] = useState(false);
	const [selectedPrefillKey, setSelectedPrefillKey] = useState('');
	const [form, setForm] = useState<ManagedSkillForm>(() => {
		const initialDocument = seed?.document ?? emptyDocument();
		const initialEnabled = seed?.enabled ?? true;

		if (mode === 'fork') {
			return {
				...formFromDocument(initialDocument, initialEnabled),
				name: makeUniqueSkillCopyName(initialDocument.name, existingNames),
				displayName: `${initialDocument.displayName || initialDocument.name || 'Skill'} Copy`,
			};
		}

		return formFromDocument(initialDocument, initialEnabled);
	});
	const [error, setError] = useState('');
	const [saving, setSaving] = useState(false);

	useEffect(() => {
		if (mode !== 'edit' || !skill) {
			return;
		}

		let cancelled = false;

		const loadDocument = async () => {
			try {
				const value: StoreManagedSkillDocument = await skillManagementAPI.getManagedSkillDocument(skill.ref);
				if (!cancelled) {
					if (value.artifact.revision !== skill.revision) {
						setDocumentLoadFailed(true);
						setError('The Skill changed while its document was loading. Close, refresh, and reopen it.');
						return;
					}
					setForm(formFromDocument(value.document, value.artifact.enabled));
				}
			} catch (cause) {
				if (!cancelled) {
					setDocumentLoadFailed(true);
					setError(cause instanceof Error ? cause.message : 'Managed Skill document could not be loaded.');
				}
			} finally {
				if (!cancelled) {
					setLoadingDocument(false);
				}
			}
		};

		void loadDocument();

		return () => {
			cancelled = true;
		};
	}, [mode, skill]);

	const parsedArguments = useMemo(() => parseSkillArguments(form.argumentsText), [form.argumentsText]);
	const existingNameKeys = useMemo(() => {
		const keys = new Set<string>();
		for (const existingName of existingNames) {
			keys.add(existingName.trim().toLowerCase());
		}
		return keys;
	}, [existingNames]);
	const prefillDropdownItems = useMemo(
		() =>
			Object.fromEntries(prefillDocuments.map(source => [source.key, { isEnabled: true }])) as Record<
				string,
				DropdownItem
			>,
		[prefillDocuments]
	);

	const update = <K extends keyof ManagedSkillForm>(key: K, value: ManagedSkillForm[K]) => {
		setForm(previous => ({
			...previous,
			[key]: value,
		}));
		if (!documentLoadFailed) {
			setError('');
		}
	};

	const applyPrefill = (key: string) => {
		const source = prefillDocuments.find(value => value.key === key);
		if (!source) {
			return;
		}

		setForm(previous => {
			const nextDocument: SkillDocumentInput = {
				...source.document,
				name: previous.name,
				displayName: previous.displayName || source.document.displayName,
			};
			return formFromDocument(nextDocument, previous.enabled);
		});
		setError('');
	};

	const copyGeneratedDocument = async () => {
		if (parsedArguments.error) {
			setError(parsedArguments.error);
			return;
		}
		if (typeof navigator === 'undefined' || !navigator.clipboard) {
			setError('Clipboard access is unavailable.');
			return;
		}

		try {
			const document = documentFromForm(form, parsedArguments.values);
			const markdown = new TextDecoder().decode(new Uint8Array(encodeSkillDocument(document)));
			await navigator.clipboard.writeText(markdown);
		} catch (cause) {
			setError(cause instanceof Error ? cause.message : 'Generated SKILL.md could not be copied.');
		}
	};

	const submit: SubmitEventHandler<HTMLFormElement> = async event => {
		event.preventDefault();
		event.stopPropagation();

		const name = form.name.trim();
		const description = form.description.trim();
		const markdownBody = form.markdownBody.trim();

		if (saving || documentLoadFailed) {
			return;
		}
		if (!SKILL_NAME_PATTERN.test(name)) {
			setError(
				'Skill name must use lowercase letters, numbers, and hyphens, start with a letter or number, and be at most 64 characters.'
			);
			return;
		}
		if (!description) {
			setError('Skill description is required.');
			return;
		}
		if (!markdownBody) {
			setError('SKILL.md body is required.');
			return;
		}
		if (parsedArguments.error) {
			setError(parsedArguments.error);
			return;
		}
		if (mode !== 'edit' && existingNameKeys.has(name.toLowerCase())) {
			setError('A Skill with this name already exists in the destination Root.');
			return;
		}

		const document = documentFromForm(form, parsedArguments.values);
		const skillMD = encodeSkillDocument(document);

		setSaving(true);
		setError('');

		let saved = false;
		let result: ManagedSkillReplaceResult;

		try {
			if (mode === 'edit' && skill) {
				result = await skillManagementAPI.replaceManagedSkill({
					plugin: plugin.ref,
					expectedPluginRevision: plugin.revision,
					artifact: skill.ref,
					expectedArtifactRevision: skill.revision,
					skillName: name,
					skillMD,
					enabled: form.enabled,
				});
			} else {
				result = await skillManagementAPI.createManagedSkill({
					plugin: plugin.ref,
					expectedPluginRevision: plugin.revision,
					skillName: name,
					skillMD,
					enabled: form.enabled,
				});
			}
			await onSaved(artifactRefFromStoreArtifact(result.artifact), document, form.enabled);
			saved = true;
			onClose();
		} catch (cause) {
			setError(cause instanceof Error ? cause.message : 'Skill could not be saved.');
		} finally {
			if (!saved) {
				setSaving(false);
			}
		}
	};

	const title =
		mode === 'edit' ? 'Edit Managed Skill' : mode === 'fork' ? 'Fork into Managed Skill' : 'Create Managed Skill';

	return (
		<div className="modal-box bg-base-200 max-h-[85vh] w-[calc(100%-1rem)] max-w-3xl overflow-hidden rounded-2xl p-0">
			<div className="app-scrollbar-thin max-h-[85vh] overflow-y-auto p-4 sm:p-6">
				<ModalHeader title={title} onClose={onClose} closeDisabled={saving} />

				<form className="space-y-4" onSubmit={submit}>
					{error ? (
						<div className="alert alert-error rounded-2xl text-sm">
							<FiAlertCircle size={15} />
							<span>{error}</span>
						</div>
					) : null}

					{mode === 'fork' ? (
						<div className="alert alert-info rounded-2xl text-sm">
							This copies the managed document, including its behavior, arguments, tags, and body. Package resource
							files are not copied.
						</div>
					) : null}

					{loadingDocument ? (
						<Loader text="Loading managed SKILL.md..." />
					) : documentLoadFailed ? (
						<div className="alert alert-warning rounded-2xl text-sm">
							The managed document was not loaded. Close this editor, refresh the Skill Plugin, and reopen it before
							saving.
						</div>
					) : (
						<>
							{mode === 'create' && prefillDocuments.length > 0 ? (
								<ModalSection title="Prefill">
									<ModalField label="Copy another managed document">
										<Dropdown<string>
											dropdownItems={prefillDropdownItems}
											orderedKeys={prefillDocuments.map(source => source.key)}
											selectedKey={selectedPrefillKey}
											onChange={key => {
												setSelectedPrefillKey(key);
												applyPrefill(key);
											}}
											filterDisabled={false}
											title="Copy another managed document"
											placeholderLabel="Choose a managed Skill"
											getDisplayName={key => prefillDocuments.find(source => source.key === key)?.label ?? key}
											disabled={saving}
											inlineMenu
											maxMenuHeight={240}
										/>
									</ModalField>
								</ModalSection>
							) : null}

							<ModalSection title="Identity">
								<ModalField label="Skill name" htmlFor="managed-skill-name" required>
									<input
										id="managed-skill-name"
										className="input w-full rounded-xl font-mono"
										value={form.name}
										readOnly={mode === 'edit'}
										disabled={saving}
										onChange={event => {
											update('name', event.currentTarget.value);
										}}
										spellCheck={false}
									/>
								</ModalField>

								<ModalField label="Display name" htmlFor="managed-skill-display-name">
									<input
										id="managed-skill-display-name"
										className="input w-full rounded-xl"
										value={form.displayName}
										disabled={saving}
										onChange={event => {
											update('displayName', event.currentTarget.value);
										}}
									/>
								</ModalField>

								<ModalField label="Description" htmlFor="managed-skill-description" required>
									<textarea
										id="managed-skill-description"
										className="textarea min-h-20 w-full rounded-xl"
										value={form.description}
										disabled={saving}
										onChange={event => {
											update('description', event.currentTarget.value);
										}}
									/>
								</ModalField>
							</ModalSection>

							<ModalSection title="Skill behavior">
								<ModalField label="Insert behavior">
									<Dropdown<SkillInsert>
										dropdownItems={SKILL_INSERT_DROPDOWN_ITEMS}
										orderedKeys={SKILL_INSERT_DROPDOWN_KEYS}
										selectedKey={form.insert}
										onChange={insert => {
											update('insert', insert);
										}}
										filterDisabled={false}
										title="Select Skill insert behavior"
										getDisplayName={insert => SKILL_INSERT_DROPDOWN_LABELS[insert]}
										disabled={saving}
										inlineMenu
										maxMenuHeight={180}
									/>
								</ModalField>

								<ModalField
									label="Arguments"
									htmlFor="managed-skill-arguments"
									hint={
										'One argument per line: name | description | default. Escape pipes as \\| and line breaks as \\n. # comments are allowed.'
									}
								>
									<textarea
										id="managed-skill-arguments"
										className="textarea min-h-24 w-full rounded-xl font-mono text-xs"
										value={form.argumentsText}
										disabled={saving}
										onChange={event => {
											update('argumentsText', event.currentTarget.value);
										}}
										placeholder={'topic | Topic to explain | AI agents\ntext | Text to summarize |'}
										spellCheck={false}
									/>
									{parsedArguments.error ? (
										<div className="text-error mt-1 text-xs">{parsedArguments.error}</div>
									) : (
										<div className="text-base-content/60 mt-1 text-xs">
											{parsedArguments.values.length} parsed argument
											{parsedArguments.values.length === 1 ? '' : 's'}.
										</div>
									)}
								</ModalField>

								<ModalField label="Tags" htmlFor="managed-skill-tags" hint="Comma-separated.">
									<input
										id="managed-skill-tags"
										className="input w-full rounded-xl"
										value={form.tagsText}
										disabled={saving}
										onChange={event => {
											update('tagsText', event.currentTarget.value);
										}}
										placeholder="research, writing"
									/>
								</ModalField>

								<label className="flex items-center gap-2 text-sm">
									<input
										type="checkbox"
										className="toggle toggle-sm"
										checked={form.enabled}
										disabled={saving}
										onChange={event => {
											update('enabled', event.currentTarget.checked);
										}}
									/>
									Enable this Skill after saving
								</label>
							</ModalSection>

							<ModalSection title="SKILL.md body">
								<ModalField label="Markdown body" htmlFor="managed-skill-body" required>
									<textarea
										id="managed-skill-body"
										className="textarea min-h-52 w-full rounded-xl"
										value={form.markdownBody}
										disabled={saving}
										onChange={event => {
											update('markdownBody', event.currentTarget.value);
										}}
										placeholder="Write reusable instructions or a user-message template here."
										spellCheck={false}
									/>
								</ModalField>
							</ModalSection>

							<ModalSection title="Generated SKILL.md">
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									disabled={saving || Boolean(parsedArguments.error)}
									onClick={copyGeneratedDocument}
								>
									<FiCopy size={14} />
									Copy complete document
								</button>
							</ModalSection>
						</>
					)}

					<ModalActions>
						<button type="button" className="btn bg-base-300 rounded-xl" disabled={saving} onClick={onClose}>
							Cancel
						</button>
						<button
							type="submit"
							className="btn btn-primary rounded-xl"
							disabled={saving || loadingDocument || documentLoadFailed}
						>
							<FiFileText size={14} />
							<span>{saving ? 'Saving...' : 'Save Skill'}</span>
						</button>
					</ModalActions>
				</form>
			</div>
		</div>
	);
}

export function ManagedSkillEditorModal(props: ManagedSkillEditorModalProps) {
	if (!props.isOpen) {
		return null;
	}

	return (
		<ModalDialog isOpen onClose={props.onClose} blockCancel>
			<ManagedSkillEditorModalContent
				key={`${props.plugin.ref.rootID}:${props.plugin.ref.artifactID}:${props.skill?.ref.artifactID ?? 'new'}:${
					props.skill?.revision ?? props.plugin.revision
				}:${props.mode ?? 'create'}`}
				{...props}
			/>
		</ModalDialog>
	);
}

function SkillArtifactDetailsModalContent({ skill, onClose }: { skill: StoreSkillListItem; onClose: () => void }) {
	const [artifact, setArtifact] = useState<StoreArtifact | null>(null);
	const [details, setDetails] = useState<ArtifactSkillDetails | null>(null);
	const [summary, setSummary] = useState<{
		Artifact: ArtifactRef;
		IsEnabled: boolean;
		Insert: SkillInsert;
		HasArguments: boolean;
		HasResources: boolean;
	} | null>(null);
	const [error, setError] = useState('');
	const [templateArguments, setTemplateArguments] = useState('{}');
	const [templatePreview, setTemplatePreview] = useState<string | null>(null);
	const [templatePreviewError, setTemplatePreviewError] = useState('');
	const [templatePreviewLoading, setTemplatePreviewLoading] = useState(false);
	const [locationCopyError, setLocationCopyError] = useState('');
	const previewRequestRef = useRef(0);

	useEffect(() => {
		let cancelled = false;

		void (async () => {
			const [artifactResult, summaryResult, documentResult] = await Promise.allSettled([
				skillManagementAPI.getSkill(skill.ref),
				skillManagementAPI.describeArtifactSkill(skill.ref),
				skillManagementAPI.getArtifactSkillDetails(skill.ref, skill.revision),
			]);

			if (cancelled) {
				return;
			}

			if (artifactResult.status !== 'fulfilled') {
				setError(
					artifactResult.reason instanceof Error ? artifactResult.reason.message : 'Skill details could not be loaded.'
				);
			} else {
				setArtifact(artifactResult.value);
			}

			if (summaryResult.status === 'fulfilled') {
				setSummary(summaryResult.value);
			}
			if (documentResult.status === 'fulfilled') {
				setDetails(documentResult.value);
				setTemplateArguments(
					JSON.stringify(
						Object.fromEntries(
							(documentResult.value.document.arguments ?? []).map(argument => [argument.name, argument.default ?? ''])
						),
						null,
						2
					)
				);
			} else if (documentResult.status === 'rejected') {
				setError(
					documentResult.reason instanceof Error
						? documentResult.reason.message
						: 'Verified Skill document could not be loaded.'
				);
			}
			if (summaryResult.status === 'rejected') {
				setError(previous =>
					[
						previous,
						summaryResult.reason instanceof Error
							? summaryResult.reason.message
							: 'Skill interpretation could not be loaded.',
					]
						.filter(Boolean)
						.join('\n')
				);
			}
		})();

		return () => {
			cancelled = true;
			previewRequestRef.current += 1;
		};
	}, [skill]);

	const renderTemplatePreview = () => {
		if (!summary || templatePreviewLoading) {
			return;
		}
		const request = ++previewRequestRef.current;

		setTemplatePreviewLoading(true);
		setTemplatePreviewError('');

		void Promise.resolve()
			.then(async () => {
				const argumentsValue = parseTemplateArguments(templateArguments);
				const command = {
					skill: {
						rootID: skill.ref.rootID,
						artifactID: skill.ref.artifactID,
						expectedRevision: skill.revision,
						...(skill.definitionDigest ? { expectedDigest: skill.definitionDigest } : {}),
					},
					arguments: argumentsValue,
				};
				const response =
					summary.Insert === SkillInsert.UserMessage
						? await skillManagementAPI.prepareUserMessageSkillTemplate(command)
						: await skillManagementAPI.previewInstructionSkill(command);
				if (request === previewRequestRef.current) {
					setTemplatePreview(response.text);
				}
			})
			.catch((cause: unknown) => {
				if (request === previewRequestRef.current) {
					setTemplatePreview(null);
					setTemplatePreviewError(cause instanceof Error ? cause.message : 'Template rendering was rejected.');
				}
			})
			.finally(() => {
				if (request === previewRequestRef.current) {
					setTemplatePreviewLoading(false);
				}
			});
	};

	return (
		<ManagementDetailsModal
			isOpen
			onClose={onClose}
			title="Skill Details"
			modalKey={`skill:${skill.ref.rootID}:${skill.ref.artifactID}:${skill.revision}`}
		>
			{error ? <div className="alert alert-warning rounded-2xl text-sm">{error}</div> : null}

			<ModalSection title="Artifact">
				<ManagementInfoGrid>
					<ManagementInfoRow label="Display name">{skill.displayName || skill.name}</ManagementInfoRow>
					<ManagementInfoRow label="Logical name" mono>
						{skill.name}
					</ManagementInfoRow>
					<ManagementInfoRow label="Artifact" mono>
						{skill.ref.rootID}/{skill.ref.artifactID}
					</ManagementInfoRow>
					<ManagementInfoRow label="State">{skill.state}</ManagementInfoRow>
					<ManagementInfoRow label="Enabled">{skill.enabled ? 'Yes' : 'No'}</ManagementInfoRow>
					<ManagementInfoRow label="Managed">{skill.managed ? 'Yes' : 'No'}</ManagementInfoRow>
					<ManagementInfoRow label="Built-in">{skill.builtIn ? 'Yes' : 'No'}</ManagementInfoRow>
					<ManagementInfoRow label="Revision">{skill.revision}</ManagementInfoRow>
					<ManagementInfoRow label="Definition digest">{skill.definitionDigest || '—'}</ManagementInfoRow>
					<ManagementInfoRow label="Description">{skill.description || '—'}</ManagementInfoRow>
					<ManagementInfoRow label="Source" mono>
						{artifact?.binding.sourceID || '—'}
					</ManagementInfoRow>
					<ManagementInfoRow label="Source locator" mono>
						{artifact?.binding.locator || '—'}
					</ManagementInfoRow>
					<ManagementInfoRow label="Runtime location" mono>
						<span className="break-all">{details?.location || '—'}</span>
						<button
							type="button"
							className="btn btn-xs btn-ghost ml-2 rounded-lg"
							disabled={!details?.location}
							onClick={() => {
								if (!details?.location || !navigator.clipboard) {
									setLocationCopyError('Clipboard access is unavailable.');
									return;
								}
								void navigator.clipboard.writeText(details.location).catch(() => {
									setLocationCopyError('Runtime location could not be copied.');
								});
							}}
						>
							<FiCopy size={12} /> Copy
						</button>
					</ManagementInfoRow>
					<ManagementInfoRow label="Subresource locator" mono>
						{artifact?.binding.subresourceLocator || '—'}
					</ManagementInfoRow>
					<ManagementInfoRow label="Created">{artifact?.createdAt || '—'}</ManagementInfoRow>
					<ManagementInfoRow label="Modified">{artifact?.modifiedAt || '—'}</ManagementInfoRow>
				</ManagementInfoGrid>
			</ModalSection>
			{locationCopyError ? <div className="alert alert-warning rounded-xl text-xs">{locationCopyError}</div> : null}
			{artifact?.diagnostics?.length ? (
				<ModalSection title="Source diagnostics">
					<ul className="space-y-2 text-sm">
						{artifact.diagnostics.map((diagnostic, index) => (
							<li key={`${diagnostic.code}:${index}`}>
								<strong>{diagnostic.code}</strong>: {diagnostic.message}
								{diagnostic.location?.locator ? ` (${diagnostic.location.locator})` : ''}
							</li>
						))}
					</ul>
				</ModalSection>
			) : null}

			{summary ? (
				<ModalSection title="Runtime interpretation">
					<div className="flex flex-wrap gap-2">
						<MetadataPill label="Insert">{summary.Insert}</MetadataPill>
						<MetadataPill label="Runtime enabled">{summary.IsEnabled ? 'Yes' : 'No'}</MetadataPill>
						<MetadataPill label="Arguments">{summary.HasArguments ? 'Declared' : 'None'}</MetadataPill>
						<MetadataPill label="Resources">{summary.HasResources ? 'Present' : 'None'}</MetadataPill>
					</div>
					<div className="text-base-content/70 mt-3 text-sm">
						{summary.Insert === SkillInsert.UserMessage
							? 'This is a user-message template. Its rendered text is inserted through Session Runtime before a Run is submitted.'
							: 'This is an instruction Skill. Enabling and activating it is Session-owned; it is not converted into frontend-owned runtime state.'}
					</div>
				</ModalSection>
			) : null}

			{details ? (
				<ModalSection title="Verified SKILL.md">
					<div className="mb-3 flex flex-wrap gap-2">
						{(details.document.tags ?? []).map(tag => (
							<MetadataPill key={tag} label="Tag">
								{tag}
							</MetadataPill>
						))}
					</div>

					{(details.document.arguments ?? []).length > 0 ? (
						<div className="mb-3 space-y-1">
							<div className="text-sm font-semibold">Arguments</div>
							{details.document.arguments?.map(argument => (
								<div key={argument.name} className="bg-base-100 rounded-xl px-3 py-2 text-xs">
									<span className="font-mono">{argument.name}</span>
									{argument.description ? ` · ${argument.description}` : ''}
									{argument.default !== undefined ? ` · default: ${argument.default}` : ''}
								</div>
							))}
						</div>
					) : null}

					<MessageContentCard
						messageID={`skill-document:${skill.ref.rootID}:${skill.ref.artifactID}`}
						content={details.document.markdownBody}
						isBusy={false}
						align="items-start text-left"
						renderAsMarkdown={true}
					/>
				</ModalSection>
			) : null}
			{details?.runtimeWarnings?.length ? (
				<ModalSection title="Runtime warnings">
					<ul className="list-disc space-y-1 pl-5 text-sm">
						{details.runtimeWarnings.map((warning, index) => (
							<li key={`${index}:${warning}`}>{warning}</li>
						))}
					</ul>
				</ModalSection>
			) : null}
			{details?.resources.hasResources ? (
				<ModalSection title={`Resources (${details.resources.totalCount})`}>
					<ul className="space-y-1 text-xs">
						{details.resources.locations?.map(location => (
							<li key={location} className="font-mono break-all">
								{location}
							</li>
						))}
					</ul>
					{details.resources.moreLocations ? (
						<p className="mt-2 text-xs">Additional locations were omitted from this bounded listing.</p>
					) : null}
				</ModalSection>
			) : null}

			{summary ? (
				<ModalSection
					title={summary.Insert === SkillInsert.UserMessage ? 'Session template preview' : 'Instruction Skill preview'}
				>
					<div className="text-base-content/70 mb-3 text-sm">
						This read-only Session Runtime preview does not create or mutate a Run or Skill session.
					</div>

					<label className="block">
						<span className="mb-1 block text-sm">Arguments JSON</span>
						<textarea
							className="textarea min-h-24 w-full rounded-xl font-mono text-xs"
							value={templateArguments}
							onChange={event => {
								setTemplateArguments(event.currentTarget.value);
								setTemplatePreviewError('');
							}}
							spellCheck={false}
						/>
					</label>

					<div className="mt-3 flex gap-2">
						<button
							type="button"
							className="btn btn-sm btn-primary rounded-xl"
							disabled={templatePreviewLoading}
							onClick={renderTemplatePreview}
						>
							{templatePreviewLoading ? <FiRefreshCw className="animate-spin" size={14} /> : <FiPlay size={14} />}
							<span>Render Preview</span>
						</button>
					</div>

					{templatePreviewError ? (
						<div className="alert alert-error mt-3 rounded-2xl text-sm">
							<FiAlertCircle size={14} />
							<span>{templatePreviewError}</span>
						</div>
					) : null}

					{templatePreview !== null ? (
						<div className="mt-3">
							<MessageContentCard
								messageID={`skill-template-preview:${skill.ref.rootID}:${skill.ref.artifactID}`}
								content={templatePreview || '(Rendered output is empty.)'}
								isBusy={false}
								align="items-start text-left"
								renderAsMarkdown={true}
							/>
						</div>
					) : null}
				</ModalSection>
			) : null}

			<ModalActions>
				<button type="button" className="btn bg-base-300 rounded-xl" onClick={onClose}>
					Close
				</button>
			</ModalActions>
		</ManagementDetailsModal>
	);
}

export function SkillArtifactDetailsModal({ isOpen, skill, onClose }: SkillArtifactDetailsModalProps) {
	if (!isOpen || !skill) {
		return null;
	}

	return (
		<SkillArtifactDetailsModalContent
			key={`${skill.ref.rootID}:${skill.ref.artifactID}:${skill.revision}`}
			skill={skill}
			onClose={onClose}
		/>
	);
}
