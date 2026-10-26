// oxlint-disable react/only-export-components
import { useEffect, useMemo, useState } from 'react';
import { FiAlertCircle, FiFileText, FiPlay, FiRefreshCw } from 'react-icons/fi';

import type { ArtifactRef, ArtifactState } from '@/spec/artifact';
import type {
	ArtifactSkillDetails,
	SkillArgument,
	SkillDocumentInput,
	SkillResourceInfo,
	StoreSkillListItem,
} from '@/spec/skill';
import { SkillInsert } from '@/spec/skill';

import type { ComposerSkillCatalogItem } from '@/apis/skill_management';
import { skillManagementAPI } from '@/apis/baseapi';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import { MessageContentCard } from '@/chats/messages/message_content_card';
import { getSkillDisplayName, getSkillSourceLabel } from '@/skills/skill_presentation';

export interface SkillDetailsSubject {
	ref: ArtifactRef;
	revision: number;
	displayName: string;
	name: string;
	description?: string;
	state?: ArtifactState;
	enabled: boolean;
	builtIn: boolean;
	managed: boolean;
	runtimeResolved?: boolean;
	insert?: SkillInsert;
	arguments?: SkillArgument[];
	tags?: string[];
	document?: SkillDocumentInput;
	resources?: SkillResourceInfo;
	sourceLabel?: string;
}

interface StoreSkillDetailsOptions {
	sourceLabel?: string;
	insert?: SkillInsert;
	arguments?: SkillArgument[];
	tags?: string[];
	document?: SkillDocumentInput;
	resources?: SkillResourceInfo;
}

interface SkillDetailsModalProps {
	isOpen: boolean;
	skill: SkillDetailsSubject | null;
	onClose: () => void;
	showPreview?: boolean;
}

function resourceFileName(location: string): string {
	const parts = location.replaceAll('\\', '/').split('/').filter(Boolean);

	return parts.at(-1) || 'Resource file';
}

function skillAvailabilityLabel(skill: SkillDetailsSubject): string {
	if (!skill.enabled) {
		return 'Disabled';
	}

	if (skill.runtimeResolved === false) {
		return 'Needs attention';
	}

	if (skill.state && skill.state.toString() !== 'available') {
		return 'Unavailable';
	}

	return 'Available';
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

	const values: Record<string, string> = {};

	for (const [key, value] of Object.entries(parsed)) {
		if (!key.trim()) {
			throw new Error('Argument names must not be empty.');
		}

		if (typeof value === 'string') {
			values[key] = value;
			continue;
		}

		if (typeof value === 'number' || typeof value === 'boolean') {
			values[key] = String(value);
			continue;
		}

		if (value === null) {
			values[key] = '';
			continue;
		}

		throw new Error(`Argument "${key}" must be a string, number, boolean, or null.`);
	}

	return values;
}

function previewArgumentsFromDocument(document: SkillDocumentInput): string {
	return JSON.stringify(
		Object.fromEntries((document.arguments ?? []).map(argument => [argument.name, argument.default ?? ''])),
		null,
		2
	);
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

function SkillPreviewSection({ skill, document }: { skill: SkillDetailsSubject; document: SkillDocumentInput }) {
	const [argumentsJSON, setArgumentsJSON] = useState(() => previewArgumentsFromDocument(document));
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
					expectedRevision: skill.revision,
				};

				const result =
					document.insert === SkillInsert.UserMessage
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
			title={document.insert === SkillInsert.UserMessage ? 'Template preview' : 'Instruction preview'}
			description="This preview is read-only and does not change the current conversation."
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
				<div className="alert alert-warning mt-3 rounded-2xl text-sm">
					<FiAlertCircle size={14} />
					<span>The Skill preview could not be rendered. Check the arguments and Skill availability.</span>
				</div>
			) : null}

			{preview !== null ? (
				<div className="mt-3">
					<MessageContentCard
						messageID={`skill-preview:${skill.ref.rootID}:${skill.ref.artifactID}:${skill.revision}`}
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
	const [details, setDetails] = useState<ArtifactSkillDetails | null>(null);
	const [loading, setLoading] = useState(true);
	const [loadFailed, setLoadFailed] = useState(false);

	useEffect(() => {
		let cancelled = false;

		void skillManagementAPI
			.getArtifactSkillDetails(skill.ref, skill.revision)
			.then(value => {
				if (!cancelled) {
					setDetails(value);
				}
			})
			.catch(() => {
				if (!cancelled) {
					setLoadFailed(true);
				}
			})
			.finally(() => {
				if (!cancelled) {
					setLoading(false);
				}
			});

		return () => {
			cancelled = true;
		};
	}, [skill.ref, skill.revision]);

	const skillDocument = details?.document ?? skill.document;
	const argumentsList = skillDocument?.arguments ?? skill.arguments ?? [];
	const tags = skillDocument?.tags ?? skill.tags ?? [];
	const resources = details?.resources ?? skill.resources;
	const sourceLabel = skill.sourceLabel ?? getSkillSourceLabel(skill);
	const insert = skillDocument?.insert ?? skill.insert;
	const behavior = insert === SkillInsert.UserMessage ? 'User-message template' : 'Instruction Skill';
	const resourceNames = useMemo(
		() => [
			...new Set(
				(resources?.locations ?? [])
					.map(r => {
						return resourceFileName(r);
					})
					.filter(Boolean)
			),
		],
		[resources?.locations]
	);
	const diagnosticCount = details?.artifact.diagnostics?.length ?? 0;

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
				{loading ? <div className="text-base-content/60 text-sm">Loading Skill details...</div> : null}

				{loadFailed ? (
					<div className="alert alert-warning rounded-2xl text-sm">
						<FiAlertCircle size={15} />
						<span>Some current Skill details could not be loaded. Available catalog information is shown below.</span>
					</div>
				) : null}

				{diagnosticCount > 0 ? (
					<div className="alert alert-warning rounded-2xl text-sm">
						<FiAlertCircle size={15} />
						<span>
							This Skill has {diagnosticCount} source diagnostic{diagnosticCount === 1 ? '' : 's'} that need attention.
						</span>
					</div>
				) : null}

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

					{tags.length > 0 ? (
						<div className="mt-3 flex flex-wrap gap-1">
							{tags.map(tag => (
								<span key={tag} className="badge badge-ghost badge-sm">
									{tag}
								</span>
							))}
						</div>
					) : null}
				</ModalSection>

				<ModalSection title="Arguments">
					<SkillArgumentList argumentsList={argumentsList} />
				</ModalSection>

				{resources?.hasResources ? (
					<ModalSection title={`Resources (${resources.totalCount})`}>
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

						{resources.moreLocations ? (
							<div className="text-base-content/60 mt-2 text-xs">Additional resource files are available.</div>
						) : null}
					</ModalSection>
				) : null}

				<ModalSection title="Skill instructions">
					{skillDocument?.markdownBody.trim() ? (
						<MessageContentCard
							messageID={`skill-document:${skill.ref.rootID}:${skill.ref.artifactID}:${skill.revision}`}
							content={skillDocument.markdownBody}
							isBusy={false}
							align="items-start text-left"
							renderAsMarkdown={true}
						/>
					) : (
						<div className="text-base-content/60 text-sm">No readable Skill instructions are available.</div>
					)}
				</ModalSection>

				{showPreview && skillDocument ? (
					<SkillPreviewSection
						key={`${skill.ref.rootID}:${skill.ref.artifactID}:${skill.revision}:${skillDocument.insert}`}
						skill={skill}
						document={skillDocument}
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
		arguments: skill.arguments,
		tags: skill.tags,
		document: skill.document,
		resources: skill.resources,
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
		insert: options.insert,
		arguments: options.arguments,
		tags: options.tags,
		document: options.document,
		resources: options.resources,
		sourceLabel: options.sourceLabel ?? getSkillSourceLabel(skill),
	};
}

export function SkillDetailsModal({ isOpen, skill, onClose, showPreview = false }: SkillDetailsModalProps) {
	if (!isOpen || !skill) {
		return null;
	}

	return (
		<ModalDialog isOpen={isOpen} onClose={onClose}>
			<SkillDetailsModalContent
				key={`${skill.ref.rootID}:${skill.ref.artifactID}:${skill.revision}:${showPreview ? 'preview' : 'readonly'}`}
				skill={skill}
				onClose={onClose}
				showPreview={showPreview}
			/>
		</ModalDialog>
	);
}
