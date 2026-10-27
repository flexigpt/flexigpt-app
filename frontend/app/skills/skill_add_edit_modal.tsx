import type { SubmitEventHandler } from 'react';
import { useEffect, useMemo, useState } from 'react';
import { FiAlertCircle, FiFileText } from 'react-icons/fi';

import type { ArtifactRef, StoreArtifact } from '@/spec/artifact';
import type { PluginListItem } from '@/spec/plugin';
import type {
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
import { ManagementArtifactSource } from '@/components/managementui/management_artifact_source';
import {
	ManagementModalBody,
	ManagementModalDialog,
	ManagementModalPanel,
} from '@/components/managementui/management_modal_layout';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import { canonicalSkillSource } from '@/skills/skill_artifact_source';
import {
	encodeSkillDocument,
	formatSkillArguments,
	makeUniqueSkillCopyName,
	parseSkillArguments,
} from '@/skills/skill_management_utils';

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
	const generatedSkillSource = useMemo(() => {
		if (parsedArguments.error) {
			return undefined;
		}

		return canonicalSkillSource(documentFromForm(form, parsedArguments.values));
	}, [form, parsedArguments]);
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
		<ManagementModalPanel>
			<ModalHeader title={title} onClose={onClose} closeDisabled={saving} />

			<form className="flex min-h-0 flex-1 flex-col" onSubmit={submit}>
				<ManagementModalBody>
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

							{generatedSkillSource ? (
								<ManagementArtifactSource
									key={generatedSkillSource}
									label="generated SKILL.md"
									prefetched={{
										data: generatedSkillSource,
										text: generatedSkillSource,
										fileName: 'SKILL.md',
										extension: 'md',
									}}
									load={async () => ({
										data: generatedSkillSource,
										text: generatedSkillSource,
										fileName: 'SKILL.md',
										extension: 'md',
									})}
									note="This is the complete source that will be saved, including frontmatter."
								/>
							) : (
								<p className="text-base-content/60 text-sm">Fix the arguments before generating SKILL.md.</p>
							)}
						</>
					)}
				</ManagementModalBody>

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
		</ManagementModalPanel>
	);
}

export function ManagedSkillEditorModal(props: ManagedSkillEditorModalProps) {
	if (!props.isOpen) {
		return null;
	}

	return (
		<ManagementModalDialog isOpen onClose={props.onClose} blockCancel>
			<ManagedSkillEditorModalContent
				key={`${props.plugin.ref.rootID}:${props.plugin.ref.artifactID}:${props.skill?.ref.artifactID ?? 'new'}:${
					props.skill?.revision ?? props.plugin.revision
				}:${props.mode ?? 'create'}`}
				{...props}
			/>
		</ManagementModalDialog>
	);
}
