import type { SubmitEventHandler } from 'react';
import { useEffect, useMemo, useState } from 'react';
import { FiAlertCircle, FiFileText, FiPlay, FiRefreshCw } from 'react-icons/fi';

import type { ArtifactRef, StoreArtifact } from '@/spec/artifact';
import type { PluginListItem } from '@/spec/plugin';
import type { SkillArgument, SkillDocumentInput, StoreManagedSkillDocument, StoreSkillListItem } from '@/spec/skill';
import { SkillInsert } from '@/spec/skill';

import { sessionManagementAPI, skillManagementAPI } from '@/apis/baseapi';

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

const SKILL_NAME_PATTERN = /^[a-z0-9][a-z0-9-]{0,63}$/;
const ARGUMENT_NAME_PATTERN = /^[A-Za-z_][A-Za-z0-9_]*$/;

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
	onSaved: (artifact: ArtifactRef) => Promise<void> | void;
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
		argumentsText: argumentsToText(document.arguments),
		tagsText: (document.tags ?? []).join(', '),
		markdownBody: document.markdownBody,
		enabled,
	};
}

function argumentsToText(argumentsList: SkillArgument[] | undefined): string {
	return (argumentsList ?? [])
		.map(argument => [argument.name, argument.description ?? '', argument.default ?? ''].join(' | '))
		.join('\n');
}

function parseArguments(raw: string): { values: SkillArgument[]; error?: string } {
	const seen = new Set<string>();
	const values: SkillArgument[] = [];

	for (const [index, rawLine] of raw.split('\n').entries()) {
		const line = rawLine.trim();
		if (!line) {
			continue;
		}

		const columns = line.split('|');
		if (columns.length > 3) {
			return {
				values: [],
				error: `Argument line ${index + 1} has too many columns. Use: name | description | default.`,
			};
		}

		const name = columns[0]?.trim() ?? '';
		if (!ARGUMENT_NAME_PATTERN.test(name)) {
			return {
				values: [],
				error: `Argument "${name || `on line ${index + 1}`}" is invalid.`,
			};
		}
		if (seen.has(name)) {
			return {
				values: [],
				error: `Argument "${name}" appears more than once.`,
			};
		}

		seen.add(name);
		values.push({
			name,
			description: columns[1]?.trim() || undefined,
			default: columns.length === 3 ? (columns[2]?.trim() ?? '') : undefined,
		});
	}

	return { values };
}

function yamlString(value: string): string {
	return JSON.stringify(value);
}

function buildSkillMarkdown(document: SkillDocumentInput): string {
	const lines = ['---', `name: ${yamlString(document.name)}`, `description: ${yamlString(document.description)}`];

	if (document.displayName?.trim()) {
		lines.push(`displayName: ${yamlString(document.displayName.trim())}`);
	}

	lines.push(`insert: ${document.insert}`);

	if ((document.tags ?? []).length > 0) {
		lines.push('tags:');
		for (const tag of document.tags ?? []) {
			lines.push(`  - ${yamlString(tag)}`);
		}
	}

	if ((document.arguments ?? []).length > 0) {
		lines.push('arguments:');
		for (const argument of document.arguments ?? []) {
			lines.push(`  - name: ${yamlString(argument.name)}`);
			if (argument.description?.trim()) {
				lines.push(`    description: ${yamlString(argument.description.trim())}`);
			}
			if (argument.default !== undefined) {
				lines.push(`    default: ${yamlString(argument.default)}`);
			}
		}
	}

	lines.push('---', '', document.markdownBody.trim(), '');
	return lines.join('\n');
}

function makeForkName(seed: string): string {
	const normalized = seed
		.trim()
		.toLowerCase()
		.replaceAll(/[^a-z0-9-]+/g, '-')
		.replaceAll(/-+/g, '-')
		.replaceAll(/^-|-$/g, '')
		.slice(0, 58);

	return normalized ? `${normalized}-copy` : 'skill-copy';
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
	onSaved,
}: Omit<ManagedSkillEditorModalProps, 'isOpen'>) {
	const [loadingDocument, setLoadingDocument] = useState(mode === 'edit' && Boolean(skill));
	const [form, setForm] = useState<ManagedSkillForm>(() => {
		const initialDocument = seed?.document ?? emptyDocument();
		const initialEnabled = seed?.enabled ?? true;

		if (mode === 'fork') {
			return {
				...formFromDocument(initialDocument, initialEnabled),
				name: makeForkName(initialDocument.name),
				displayName: `${initialDocument.displayName || initialDocument.name || 'Skill'} Copy`,
			};
		}

		return formFromDocument(initialDocument, initialEnabled);
	});
	const [error, setError] = useState('');
	const [saving, setSaving] = useState(false);

	useEffect(() => {
		if (mode !== 'edit' || !skill) {
			// oxlint-disable-next-line react/set-state-in-effect
			setLoadingDocument(false);
			return;
		}

		let cancelled = false;
		setLoadingDocument(true);
		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setError('');

		void skillManagementAPI
			.getManagedSkillDocument(skill.ref)
			.then((value: StoreManagedSkillDocument) => {
				if (!cancelled) {
					setForm(formFromDocument(value.document, value.artifact.enabled));
				}
			})
			.catch((cause: unknown) => {
				if (!cancelled) {
					setError(cause instanceof Error ? cause.message : 'Managed Skill document could not be loaded.');
				}
			})
			.finally(() => {
				if (!cancelled) {
					setLoadingDocument(false);
				}
			});

		return () => {
			cancelled = true;
		};
	}, [mode, skill]);

	const parsedArguments = useMemo(() => parseArguments(form.argumentsText), [form.argumentsText]);

	const update = <K extends keyof ManagedSkillForm>(key: K, value: ManagedSkillForm[K]) => {
		setForm(previous => ({
			...previous,
			[key]: value,
		}));
		setError('');
	};

	const submit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();

		const name = form.name.trim();
		const description = form.description.trim();
		const markdownBody = form.markdownBody.trim();

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

		const document: SkillDocumentInput = {
			name,
			displayName: form.displayName.trim() || undefined,
			description,
			insert: form.insert,
			arguments: parsedArguments.values,
			tags: form.tagsText
				.split(',')
				.map(tag => tag.trim())
				.filter(Boolean),
			markdownBody,
		};
		const skillMD = [...new TextEncoder().encode(buildSkillMarkdown(document))];

		setSaving(true);
		setError('');

		void (async () => {
			if (mode === 'edit' && skill) {
				const result = await skillManagementAPI.replaceManagedSkill({
					plugin: plugin.ref,
					expectedPluginRevision: plugin.revision,
					artifact: skill.ref,
					expectedArtifactRevision: skill.revision,
					skillName: name,
					skillMD,
					enabled: form.enabled,
				});
				await onSaved(artifactRefFromStoreArtifact(result.artifact));
				onClose();
				return;
			}

			const result = await skillManagementAPI.createManagedSkill({
				plugin: plugin.ref,
				expectedPluginRevision: plugin.revision,
				skillName: name,
				skillMD,
				enabled: form.enabled,
			});
			await onSaved(artifactRefFromStoreArtifact(result.artifact));
			onClose();
		})()
			.catch((cause: unknown) => {
				setError(cause instanceof Error ? cause.message : 'Skill could not be saved.');
			})
			.finally(() => {
				setSaving(false);
			});
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
							The managed copy keeps the selected insert behavior and editable body. Review the copied content before
							saving.
						</div>
					) : null}

					{loadingDocument ? (
						<Loader text="Loading managed SKILL.md..." />
					) : (
						<>
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
									<select
										className="select w-full rounded-xl"
										value={form.insert}
										disabled={saving}
										onChange={event => {
											update('insert', event.currentTarget.value as SkillInsert);
										}}
									>
										<option value={SkillInsert.Instructions}>Instructions</option>
										<option value={SkillInsert.UserMessage}>User-message template</option>
									</select>
								</ModalField>

								<ModalField
									label="Arguments"
									htmlFor="managed-skill-arguments"
									hint="One argument per line: name | description | default."
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
						</>
					)}

					<ModalActions>
						<button type="button" className="btn bg-base-300 rounded-xl" disabled={saving} onClick={onClose}>
							Cancel
						</button>
						<button type="submit" className="btn btn-primary rounded-xl" disabled={saving || loadingDocument}>
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
	const [managedDocument, setManagedDocument] = useState<StoreManagedSkillDocument | null>(null);
	const [summary, setSummary] = useState<{
		Artifact: ArtifactRef;
		IsEnabled: boolean;
		Insert: SkillInsert;
		HasArguments: boolean;
		HasResources: boolean;
	} | null>(null);
	const [error, setError] = useState('');
	const [templateArguments, setTemplateArguments] = useState('{}');
	const [templatePreview, setTemplatePreview] = useState('');
	const [templatePreviewError, setTemplatePreviewError] = useState('');
	const [templatePreviewLoading, setTemplatePreviewLoading] = useState(true);

	useEffect(() => {
		let cancelled = false;

		void (async () => {
			const [artifactResult, summaryResult, documentResult] = await Promise.allSettled([
				skillManagementAPI.getSkill(skill.ref),
				skillManagementAPI.describeArtifactSkill(skill.ref),
				skill.managed ? skillManagementAPI.getManagedSkillDocument(skill.ref) : Promise.resolve(null),
			]);

			if (cancelled) {
				return;
			}

			if (artifactResult.status !== 'fulfilled') {
				setError(
					artifactResult.reason instanceof Error ? artifactResult.reason.message : 'Skill details could not be loaded.'
				);
			}

			if (summaryResult.status === 'fulfilled') {
				setSummary(summaryResult.value);
			}
			if (documentResult.status === 'fulfilled' && documentResult.value) {
				setManagedDocument(documentResult.value);
			}
		})();

		return () => {
			cancelled = true;
		};
	}, [skill]);

	const renderTemplatePreview = () => {
		if (!summary || summary.Insert !== SkillInsert.UserMessage) {
			return;
		}

		setTemplatePreviewLoading(true);
		setTemplatePreviewError('');

		void Promise.resolve()
			.then(async () => {
				const argumentsValue = parseTemplateArguments(templateArguments);
				const response = await sessionManagementAPI.prepareUserMessageSkillTemplate({
					skill: {
						rootID: skill.ref.rootID,
						artifactID: skill.ref.artifactID,
						expectedRevision: skill.revision,
						...(skill.definitionDigest ? { expectedDigest: skill.definitionDigest } : {}),
					},
					arguments: argumentsValue,
				});
				setTemplatePreview(response.text);
			})
			.catch((cause: unknown) => {
				setTemplatePreviewError(cause instanceof Error ? cause.message : 'Template rendering was rejected.');
			})
			.finally(() => {
				setTemplatePreviewLoading(false);
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
				</ManagementInfoGrid>
			</ModalSection>

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

			{managedDocument ? (
				<ModalSection title="Managed SKILL.md">
					<div className="mb-3 flex flex-wrap gap-2">
						{(managedDocument.document.tags ?? []).map(tag => (
							<MetadataPill key={tag} label="Tag">
								{tag}
							</MetadataPill>
						))}
					</div>

					{(managedDocument.document.arguments ?? []).length > 0 ? (
						<div className="mb-3 space-y-1">
							<div className="text-sm font-semibold">Arguments</div>
							{managedDocument.document.arguments?.map(argument => (
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
						content={managedDocument.document.markdownBody}
						isBusy={false}
						align="items-start text-left"
						renderAsMarkdown={true}
					/>
				</ModalSection>
			) : null}

			{summary?.Insert === SkillInsert.UserMessage ? (
				<ModalSection title="Session template preview">
					<div className="text-base-content/70 mb-3 text-sm">
						This preview uses the Session Runtime template preparation path. It does not create or mutate a Run.
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

					{templatePreview ? (
						<div className="mt-3">
							<MessageContentCard
								messageID={`skill-template-preview:${skill.ref.rootID}:${skill.ref.artifactID}`}
								content={templatePreview}
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
