// oxlint-disable react/only-export-components
import { useCallback, useMemo, useRef, useState } from 'react';
import { FiAlertCircle, FiDownload, FiFileText, FiPlay, FiRefreshCw } from 'react-icons/fi';

import type { ArtifactRef } from '@/spec/artifact';
import type { ArtifactSkillDetails, SkillArgument, SkillDocumentInput, StoreSkillListItem } from '@/spec/skill';
import { ArtifactState } from '@/spec/artifact';
import { SkillInsert } from '@/spec/skill';

import { utf8TextToBase64 } from '@/lib/base64_utils';

import type { ComposerSkillCatalogItem } from '@/apis/skill_management';
import { backendAPI, skillManagementAPI } from '@/apis/baseapi';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import { MessageContentCard } from '@/chats/messages/message_content_card';
import { encodeSkillDocument } from '@/skills/skill_management_utils';
import { getSkillDisplayName, getSkillSourceLabel } from '@/skills/skill_presentation';

export interface SkillDetailsSubject {
	ref: ArtifactRef;
	revision?: number;
	displayName?: string;
	name?: string;
	description?: string;
	state?: ArtifactState;
	enabled?: boolean;
	builtIn?: boolean;
	managed?: boolean;
	runtimeResolved?: boolean;
	insert?: SkillInsert;
	hasArguments?: boolean;
	hasResources?: boolean;
	sourceLabel?: string;
}

interface StoreSkillDetailsOptions {
	sourceLabel?: string;
}

interface SkillDetailsReferenceOptions {
	displayName?: string;
	name?: string;
	description?: string;
	enabled?: boolean;
	state?: ArtifactState;
	builtIn?: boolean;
	managed?: boolean;
	runtimeResolved?: boolean;
	sourceLabel?: string;
}

interface SkillDetailsModalProps {
	isOpen: boolean;
	skill: SkillDetailsSubject | null;
	onClose: () => void;
	showPreview?: boolean;
}

interface LoadedSkillSource {
	details: ArtifactSkillDetails;
	revision: number;
}

function resourceFileName(location: string): string {
	const parts = location.replaceAll('\\', '/').split('/').filter(Boolean);

	return parts.at(-1) || 'Resource file';
}

function skillAvailabilityLabel(skill: SkillDetailsSubject): string {
	if (skill.enabled === false) {
		return 'Disabled';
	}

	if (skill.runtimeResolved === false) {
		return 'Needs attention';
	}

	if (skill.state && skill.state !== ArtifactState.Available) {
		return 'Unavailable';
	}

	return 'Available';
}

function skillBehaviorLabel(insert?: SkillInsert): string {
	switch (insert) {
		case SkillInsert.UserMessage:
			return 'User-message template';

		case SkillInsert.Instructions:
			return 'Instruction Skill';

		default:
			return 'Skill';
	}
}

function canonicalSkillSource(document: SkillDocumentInput): string {
	return new TextDecoder().decode(new Uint8Array(encodeSkillDocument(document)));
}

function parsePreviewArguments(raw: string): Record<string, string> {
	const trimmed = raw.trim();
	if (!trimmed) {
		return {};
	}

	let parsed: unknown;
	try {
		parsed = JSON.parse(trimmed);
	} catch {
		throw new Error('Arguments must be valid JSON.');
	}

	if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
		throw new Error('Arguments must be a JSON object.');
	}

	const output: Record<string, string> = {};

	for (const [key, value] of Object.entries(parsed)) {
		if (!key.trim()) {
			throw new Error('Argument names must not be empty.');
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

		throw new Error(`Argument "${key}" must be a string, number, boolean, or null.`);
	}

	return output;
}

function previewArgumentsFromDocument(document: SkillDocumentInput): string {
	return JSON.stringify(
		Object.fromEntries((document.arguments ?? []).map(argument => [argument.name, argument.default ?? ''])),
		null,
		2
	);
}

async function loadSkillSource(skill: SkillDetailsSubject): Promise<LoadedSkillSource> {
	const loadAtRevision = async (revision: number): Promise<LoadedSkillSource> => ({
		details: await skillManagementAPI.getArtifactSkillDetails(skill.ref, revision),
		revision,
	});

	if (skill.revision !== undefined) {
		try {
			return await loadAtRevision(skill.revision);
		} catch {
			// A restored selection can be stale. Resolve current metadata only
			// after the user explicitly asks to inspect the source.
		}
	}

	const artifact = await skillManagementAPI.getSkill(skill.ref);
	return loadAtRevision(artifact.revision);
}

function SkillArgumentList({ argumentsList }: { argumentsList: SkillArgument[] }) {
	if (argumentsList.length === 0) {
		return <div className="text-base-content/60 text-sm">This Skill has no declared arguments.</div>;
	}

	return (
		<div className="space-y-2">
			{argumentsList.map(argument => (
				<div key={argument.name} className="border-base-300 bg-base-100 rounded-xl border px-3 py-2 text-sm">
					<div className="font-medium">{argument.name}</div>
					{argument.description ? (
						<div className="text-base-content/70 mt-1 text-xs">{argument.description}</div>
					) : null}
					{argument.default !== undefined ? (
						<div className="text-base-content/60 mt-1 text-xs">Default: {argument.default || '(empty value)'}</div>
					) : null}
				</div>
			))}
		</div>
	);
}

function SkillSourceDisclosure({
	source,
	loading,
	loadError,
	saving,
	saveError,
	onEnsureSource,
	onSaveSource,
}: {
	source: LoadedSkillSource | null;
	loading: boolean;
	loadError: string;
	saving: boolean;
	saveError: string;
	onEnsureSource: () => Promise<LoadedSkillSource | null>;
	onSaveSource: () => Promise<void>;
}) {
	const sourceText = useMemo(() => (source ? canonicalSkillSource(source.details.document) : ''), [source]);

	return (
		<div className="space-y-3">
			<div className="flex flex-wrap items-center justify-between gap-2">
				<div className="text-sm font-medium">SKILL.md source</div>

				<button
					type="button"
					className="btn btn-ghost btn-sm rounded-xl"
					disabled={saving}
					onClick={() => {
						void onSaveSource();
					}}
				>
					<FiDownload size={14} />
					<span>{saving ? 'Saving...' : loading ? 'Loading...' : 'Export SKILL.md'}</span>
				</button>
			</div>

			{saveError ? (
				<div className="alert alert-error rounded-xl text-sm">
					<FiAlertCircle size={14} />
					<span>{saveError}</span>
				</div>
			) : null}

			<details
				className="border-base-300 bg-base-100 rounded-xl border"
				onToggle={event => {
					if (event.currentTarget.open) {
						void onEnsureSource();
					}
				}}
			>
				<summary className="hover:bg-base-200 flex cursor-pointer list-none items-center gap-2 rounded-xl px-3 py-2 text-sm font-medium">
					<FiFileText size={14} />
					View SKILL.md source
				</summary>

				<div className="border-base-300 space-y-3 border-t p-3">
					{loading ? <div className="text-base-content/60 text-sm">Loading SKILL.md source...</div> : null}

					{loadError ? (
						<div className="alert alert-warning rounded-xl text-sm">
							<FiAlertCircle size={14} />
							<div className="min-w-0 flex-1">
								<div>{loadError}</div>
								<button
									type="button"
									className="btn btn-ghost btn-xs mt-2 rounded-lg"
									disabled={loading}
									onClick={() => {
										void onEnsureSource();
									}}
								>
									Retry
								</button>
							</div>
						</div>
					) : null}

					{source ? (
						<>
							<p className="text-base-content/60 text-xs">
								Canonical SKILL.md reconstructed from the verified Skill document. Original formatting and comments are
								not available through the current API.
							</p>

							<pre className="app-scrollbar-thin bg-base-300 max-h-96 overflow-auto rounded-xl p-3 text-xs/5 wrap-break-word whitespace-pre-wrap">
								{sourceText}
							</pre>
						</>
					) : null}
				</div>
			</details>
		</div>
	);
}

function SkillPreviewSection({ skill, source }: { skill: SkillDetailsSubject; source: LoadedSkillSource }) {
	const [argumentsJSON, setArgumentsJSON] = useState(() => previewArgumentsFromDocument(source.details.document));
	const [preview, setPreview] = useState<string | null>(null);
	const [previewError, setPreviewError] = useState(false);
	const [previewLoading, setPreviewLoading] = useState(false);

	const renderPreview = () => {
		if (previewLoading) {
			return;
		}

		setPreviewLoading(true);
		setPreviewError(false);

		void Promise.resolve()
			.then(async () => {
				const argumentsValue = parsePreviewArguments(argumentsJSON);
				const selection = {
					rootID: skill.ref.rootID,
					artifactID: skill.ref.artifactID,
					expectedRevision: source.revision,
				};

				const result =
					source.details.document.insert === SkillInsert.UserMessage
						? await skillManagementAPI.prepareUserMessageSkillTemplate({
								skill: selection,
								arguments: argumentsValue,
							})
						: await skillManagementAPI.previewInstructionSkill({
								skill: selection,
								arguments: argumentsValue,
							});

				setPreview(result.text || '(Rendered output is empty.)');
			})
			.catch(() => {
				setPreview(null);
				setPreviewError(true);
			})
			.finally(() => {
				setPreviewLoading(false);
			});
	};

	return (
		<ModalSection
			title={source.details.document.insert === SkillInsert.UserMessage ? 'Template preview' : 'Instruction preview'}
			description="Preview is available only after SKILL.md source was explicitly loaded."
		>
			<label className="block">
				<span className="mb-1 block text-sm">Arguments JSON</span>
				<textarea
					className="textarea min-h-24 w-full rounded-xl font-mono text-xs"
					value={argumentsJSON}
					onChange={event => {
						setArgumentsJSON(event.currentTarget.value);
						setPreviewError(false);
					}}
					spellCheck={false}
				/>
			</label>

			<div className="mt-3">
				<button
					type="button"
					className="btn btn-sm btn-primary rounded-xl"
					disabled={previewLoading}
					onClick={renderPreview}
				>
					{previewLoading ? <FiRefreshCw className="animate-spin" size={14} /> : <FiPlay size={14} />}
					<span>Render preview</span>
				</button>
			</div>

			{previewError ? (
				<div className="alert alert-warning mt-3 rounded-xl text-sm">
					<FiAlertCircle size={14} />
					<span>The Skill preview could not be rendered. Check the arguments and Skill availability.</span>
				</div>
			) : null}

			{preview !== null ? (
				<div className="mt-3">
					<MessageContentCard
						messageID={`skill-preview:${skill.ref.rootID}:${skill.ref.artifactID}:${source.revision}`}
						content={preview}
						isBusy={false}
						align="items-start text-left"
						renderAsMarkdown={true}
					/>
				</div>
			) : null}
		</ModalSection>
	);
}

function SkillDetailsModalContent({
	skill,
	onClose,
	showPreview = false,
}: Omit<SkillDetailsModalProps, 'isOpen'> & {
	skill: SkillDetailsSubject;
}) {
	const [source, setSource] = useState<LoadedSkillSource | null>(null);
	const [sourceLoading, setSourceLoading] = useState(false);
	const [sourceError, setSourceError] = useState('');
	const [sourceSaving, setSourceSaving] = useState(false);
	const [sourceSaveError, setSourceSaveError] = useState('');
	const sourcePromiseRef = useRef<Promise<LoadedSkillSource | null> | null>(null);

	const ensureSource = useCallback(async (): Promise<LoadedSkillSource | null> => {
		if (source) {
			return source;
		}

		if (sourcePromiseRef.current) {
			return sourcePromiseRef.current;
		}

		setSourceLoading(true);
		setSourceError('');

		const request = (async () => {
			try {
				const loaded = await loadSkillSource(skill);
				setSource(loaded);
				return loaded;
			} catch {
				setSourceError('SKILL.md source could not be loaded.');
				return null;
			} finally {
				setSourceLoading(false);
			}
		})();

		sourcePromiseRef.current = request;

		void request.finally(() => {
			if (sourcePromiseRef.current === request) {
				sourcePromiseRef.current = null;
			}
		});

		return request;
	}, [skill, source]);

	const saveSource = useCallback(async () => {
		if (sourceSaving) {
			return;
		}

		const loaded = await ensureSource();
		if (!loaded) {
			return;
		}

		setSourceSaving(true);
		setSourceSaveError('');

		try {
			await backendAPI.saveFile('SKILL.md', utf8TextToBase64(canonicalSkillSource(loaded.details.document)), [
				{
					DisplayName: 'Skill Markdown',
					Extensions: ['md'],
				},
			]);
		} catch {
			setSourceSaveError('SKILL.md could not be exported.');
		} finally {
			setSourceSaving(false);
		}
	}, [ensureSource, sourceSaving]);

	const sourceLabel = skill.sourceLabel ?? getSkillSourceLabel(skill);
	const sourceDetails = source?.details;
	const behavior = skillBehaviorLabel(sourceDetails?.document.insert ?? skill.insert);
	const sourceTags = sourceDetails?.document.tags ?? [];
	const hasArguments = sourceDetails
		? (sourceDetails.document.arguments?.length ?? 0) > 0
		: skill.hasArguments === true;
	const hasResources = sourceDetails ? sourceDetails.resources.hasResources : skill.hasResources === true;
	const resourceNames = useMemo(
		() => [
			...new Set(
				(sourceDetails?.resources.locations ?? [])
					.map(r => {
						return resourceFileName(r);
					})
					.filter(Boolean)
			),
		],
		[sourceDetails?.resources.locations]
	);
	const diagnosticCount = sourceDetails?.artifact.diagnostics?.length ?? 0;

	return (
		<div className="modal-box bg-base-200 flex max-h-[85vh] w-[calc(100%-1rem)] max-w-3xl flex-col overflow-hidden rounded-2xl p-0">
			<ModalHeader
				title={
					<span className="flex items-center gap-2">
						<FiFileText size={16} />
						<span>{getSkillDisplayName(skill)}</span>
					</span>
				}
				description={skill.description || behavior}
				onClose={onClose}
			/>

			<div className="app-scrollbar-thin min-h-0 flex-1 space-y-5 overflow-y-auto p-4 sm:p-6">
				<ModalSection title="About">
					<div className="border-base-300 bg-base-100 grid gap-3 rounded-2xl border p-3 text-sm sm:grid-cols-3">
						<div>
							<div className="text-base-content/60 text-xs">Behavior</div>
							<div className="mt-1 font-medium">{behavior}</div>
						</div>
						<div>
							<div className="text-base-content/60 text-xs">Availability</div>
							<div className="mt-1 font-medium">{skillAvailabilityLabel(skill)}</div>
						</div>
						<div>
							<div className="text-base-content/60 text-xs">Source</div>
							<span className="badge badge-ghost badge-sm mt-1">{sourceLabel}</span>
						</div>
					</div>

					{sourceTags.length > 0 ? (
						<div className="mt-3 flex flex-wrap gap-1">
							{sourceTags.map(tag => (
								<span key={tag} className="badge badge-ghost badge-sm">
									{tag}
								</span>
							))}
						</div>
					) : null}
				</ModalSection>

				<ModalSection title="Arguments">
					{sourceDetails ? (
						<SkillArgumentList argumentsList={sourceDetails.document.arguments ?? []} />
					) : hasArguments ? (
						<div className="text-base-content/60 text-sm">
							This Skill declares arguments. Open SKILL.md source to inspect them.
						</div>
					) : (
						<div className="text-base-content/60 text-sm">No declared arguments are currently known.</div>
					)}
				</ModalSection>

				<ModalSection title="Resources">
					{sourceDetails && hasResources ? (
						<>
							{resourceNames.length > 0 ? (
								<div className="flex flex-wrap gap-1">
									{resourceNames.map(name => (
										<span key={name} className="badge badge-ghost badge-sm">
											{name}
										</span>
									))}
								</div>
							) : (
								<div className="text-base-content/60 text-sm">This Skill includes resource files.</div>
							)}

							{sourceDetails.resources.moreLocations ? (
								<div className="text-base-content/60 mt-2 text-xs">Additional resource files are available.</div>
							) : null}
						</>
					) : hasResources ? (
						<div className="text-base-content/60 text-sm">
							This Skill includes resource files. Open SKILL.md source to inspect available details.
						</div>
					) : (
						<div className="text-base-content/60 text-sm">No resources are currently known.</div>
					)}
				</ModalSection>

				<ModalSection title="Source">
					<SkillSourceDisclosure
						source={source}
						loading={sourceLoading}
						loadError={sourceError}
						saving={sourceSaving}
						saveError={sourceSaveError}
						onEnsureSource={ensureSource}
						onSaveSource={saveSource}
					/>
				</ModalSection>

				{diagnosticCount > 0 ? (
					<div className="alert alert-warning rounded-xl text-sm">
						<FiAlertCircle size={14} />
						<span>
							This Skill has {diagnosticCount} source diagnostic{diagnosticCount === 1 ? '' : 's'} that need attention.
						</span>
					</div>
				) : null}

				{showPreview && source ? (
					<SkillPreviewSection
						key={`${skill.ref.rootID}:${skill.ref.artifactID}:${source.revision}`}
						skill={skill}
						source={source}
					/>
				) : null}
			</div>

			<ModalActions>
				<button type="button" className="btn bg-base-300 rounded-xl" onClick={onClose}>
					Close
				</button>
			</ModalActions>
		</div>
	);
}

export function skillDetailsSubjectFromComposerSkill(skill: ComposerSkillCatalogItem): SkillDetailsSubject {
	return {
		ref: {
			rootID: skill.ref.rootID,
			artifactID: skill.ref.artifactID,
		},
		revision: skill.revision,
		displayName: skill.displayName,
		name: skill.name,
		description: skill.description,
		state: skill.state,
		enabled: skill.enabled,
		builtIn: skill.builtIn,
		managed: skill.managed,
		runtimeResolved: skill.runtimeResolved,
		insert: skill.insert,
		hasArguments: skill.hasArguments,
		hasResources: skill.hasResources,
		sourceLabel: getSkillSourceLabel(skill),
	};
}

export function skillDetailsSubjectFromStoreSkill(
	skill: StoreSkillListItem,
	options: StoreSkillDetailsOptions = {}
): SkillDetailsSubject {
	return {
		ref: {
			rootID: skill.ref.rootID,
			artifactID: skill.ref.artifactID,
		},
		revision: skill.revision,
		displayName: skill.displayName,
		name: skill.name,
		description: skill.description,
		state: skill.state,
		enabled: skill.enabled,
		builtIn: skill.builtIn,
		managed: skill.managed,
		sourceLabel: options.sourceLabel ?? getSkillSourceLabel(skill),
	};
}

export function skillDetailsSubjectFromReference(
	ref: ArtifactRef,
	options: SkillDetailsReferenceOptions = {}
): SkillDetailsSubject {
	return {
		ref: {
			rootID: ref.rootID,
			artifactID: ref.artifactID,
		},
		displayName: options.displayName,
		name: options.name,
		description: options.description,
		state: options.state,
		enabled: options.enabled,
		builtIn: options.builtIn,
		managed: options.managed,
		runtimeResolved: options.runtimeResolved,
		sourceLabel: options.sourceLabel,
	};
}

export function SkillDetailsModal({ isOpen, skill, onClose, showPreview = false }: SkillDetailsModalProps) {
	if (!isOpen || !skill) {
		return null;
	}

	return (
		<ModalDialog isOpen={isOpen} onClose={onClose}>
			<SkillDetailsModalContent
				key={`${skill.ref.rootID}:${skill.ref.artifactID}:${skill.revision ?? 'current'}:${
					showPreview ? 'preview' : 'readonly'
				}`}
				skill={skill}
				onClose={onClose}
				showPreview={showPreview}
			/>
		</ModalDialog>
	);
}
