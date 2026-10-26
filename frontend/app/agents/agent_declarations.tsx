import { useCallback, useState } from 'react';
import { FiChevronDown, FiFileText } from 'react-icons/fi';

import type { AgentCapabilityOccurrence } from '@/spec/agent';
import { isAgentSkillUseAsInstructions } from '@/spec/agent';
import { CapabilityResolutionStatus, CapabilityTargetForm } from '@/spec/artifact';
import { TextInsert } from '@/spec/text';

import { throwIfAborted } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { skillManagementAPI, textStoreAPI } from '@/apis/baseapi';

import { getAgentRelationshipBadgeClass } from '@/agents/agent_management_utils';

export interface AgentCapabilityControls {
	isExcluded: (occurrence: AgentCapabilityOccurrence) => boolean;
	onToggle: (occurrence: AgentCapabilityOccurrence) => void;
}

interface ContributionPreview {
	title: string;
	body: string;
	isDocument?: boolean;
	note?: string;
}

function ContributionContent({ occurrence }: { occurrence: AgentCapabilityOccurrence }) {
	const load = useCallback(
		async (signal: AbortSignal): Promise<ContributionPreview> => {
			const ref = occurrence.target?.artifact;
			if (!ref || occurrence.target?.form !== CapabilityTargetForm.Artifact) {
				throw new Error('This declaration has no Artifact content to preview.');
			}

			if (occurrence.type === 'text') {
				const text = await textStoreAPI.materializeText(ref);
				throwIfAborted(signal);
				return {
					title:
						text.insert === TextInsert.Instructions
							? 'Instruction text'
							: text.insert === TextInsert.UserMessage
								? 'User-message text'
								: 'Text content',
					body: text.content || '(Empty text)',
					note: `Materialized from ${text.name}; Run preparation resolves it again.`,
				};
			}

			if (isAgentSkillUseAsInstructions(occurrence.skillUseMode)) {
				try {
					const result = await skillManagementAPI.previewInstructionSkill({
						skill: ref,
						arguments: {},
					});
					throwIfAborted(signal);
					if (!result.text.trim()) {
						throw new Error('This Skill rendered no instruction text.');
					}
					return {
						title: 'Rendered instructions (empty arguments)',
						body: result.text,
						note: 'Run preparation may render this Skill with different arguments.',
					};
				} catch (error) {
					throwIfAborted(signal);
					const skill = await skillManagementAPI.getSkill(ref);
					throwIfAborted(signal);
					const details = await skillManagementAPI.getArtifactSkillDetails(ref, skill.revision);
					throwIfAborted(signal);
					return {
						title: 'Skill document',
						body: JSON.stringify(details.document, null, 2) || '(Empty document)',
						isDocument: true,
						note: `Rendered instructions unavailable: ${getErrorMessage(error, 'Preview failed.')}`,
					};
				}
			}

			const skill = await skillManagementAPI.getSkill(ref);
			throwIfAborted(signal);
			const details = await skillManagementAPI.getArtifactSkillDetails(ref, skill.revision);
			throwIfAborted(signal);
			return {
				title: 'Skill document',
				body: JSON.stringify(details.document, null, 2) || '(Empty document)',
				isDocument: true,
				note: 'This document may reference resources or render differently during Run preparation.',
			};
		},
		[occurrence]
	);
	const { data, error } = useAsyncResource(load, {
		initialData: null as ContributionPreview | null,
	});

	if (error) {
		return (
			<p className="text-error text-xs" role="alert">
				{getErrorMessage(error, 'Content could not be loaded.')}
			</p>
		);
	}
	if (!data) {
		return <p className="text-base-content/70 text-xs">Loading content...</p>;
	}

	return (
		<div className="space-y-2">
			<div className="text-base-content/70 text-xs font-medium">{data.title}</div>
			<pre
				className={`app-scrollbar-thin bg-base-300 max-h-80 overflow-auto rounded-xl p-3 wrap-break-word whitespace-pre-wrap ${
					data.isDocument ? 'font-mono text-xs' : 'font-sans text-sm'
				}`}
			>
				{data.body}
			</pre>
			{data.note ? <p className="text-base-content/70 text-xs">{data.note}</p> : null}
		</div>
	);
}

function ContributionRow({
	occurrence,
	capabilityControls,
	onSetup,
}: {
	occurrence: AgentCapabilityOccurrence;
	capabilityControls?: AgentCapabilityControls;
	onSetup: (occurrence: AgentCapabilityOccurrence) => void;
}) {
	const [expanded, setExpanded] = useState(false);
	const canToggle =
		Boolean(capabilityControls) &&
		occurrence.status === CapabilityResolutionStatus.Available &&
		(occurrence.type === 'tool' ||
			occurrence.type === 'skill' ||
			occurrence.type === 'text' ||
			occurrence.type === 'mcp');
	const isExcluded = canToggle && Boolean(capabilityControls?.isExcluded(occurrence));
	const hasDetails =
		occurrence.type === 'skill' ||
		occurrence.type === 'text' ||
		occurrence.type === 'mcp' ||
		canToggle ||
		(occurrence.type === 'model' && occurrence.includeSystemPrompt !== undefined);
	const heading = (
		<div className="flex min-w-0 flex-1 flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
			<span className="min-w-0 text-sm font-semibold wrap-break-word">
				{occurrence.name || `Unnamed ${occurrence.type}`}
			</span>
			<span className="flex shrink-0 flex-wrap items-center gap-2">
				<span className={`badge badge-sm capitalize ${getAgentRelationshipBadgeClass(occurrence.status)}`}>
					{occurrence.status}
				</span>
				{occurrence.required ? <span className="badge badge-outline badge-sm">Required</span> : null}
				{isExcluded ? <span className="badge badge-warning badge-sm">Excluded</span> : null}
			</span>
		</div>
	);

	return (
		<div className="border-base-content/10 rounded-xl border">
			{hasDetails ? (
				<details
					className="group"
					onToggle={event => {
						setExpanded(event.currentTarget.open);
					}}
				>
					<summary className="flex cursor-pointer list-none items-start gap-2 p-2 [&::-webkit-details-marker]:hidden">
						{heading}
						<FiChevronDown size={16} className="mt-0.5 shrink-0 transition-transform group-open:rotate-180" />
					</summary>
					<div className="border-base-content/10 space-y-2 border-t p-2">
						{occurrence.type === 'model' && occurrence.includeSystemPrompt !== undefined ? (
							<p className="text-base-content/70 text-xs">
								Include model system prompt: {occurrence.includeSystemPrompt ? 'Yes' : 'No'}
							</p>
						) : null}
						{occurrence.type === 'skill' && occurrence.skillUseMode ? (
							<p className="text-base-content/70 text-xs">Use mode: {occurrence.skillUseMode}</p>
						) : null}
						{expanded && (occurrence.type === 'skill' || occurrence.type === 'text') ? (
							occurrence.status === CapabilityResolutionStatus.Available ? (
								<ContributionContent occurrence={occurrence} />
							) : (
								<p className="text-base-content/70 text-xs">Content is unavailable until this declaration resolves.</p>
							)
						) : null}
						{occurrence.type === 'mcp' ? (
							<p className="text-base-content/70 text-xs">
								Setup configures installation inputs. Connection and policy approval are still required at Run time.
							</p>
						) : null}
						{occurrence.type === 'mcp' || canToggle ? (
							<div className="flex flex-wrap items-center gap-2">
								{occurrence.type === 'mcp' ? (
									<button
										type="button"
										className="btn btn-ghost btn-sm rounded-xl"
										onClick={() => {
											onSetup(occurrence);
										}}
									>
										<FiFileText size={15} />
										{occurrence.status === CapabilityResolutionStatus.Available &&
										occurrence.target?.form === CapabilityTargetForm.Artifact &&
										occurrence.target.artifact
											? 'Configure MCP'
											: 'View setup status'}
									</button>
								) : null}
								{canToggle ? (
									<button
										type="button"
										className="btn btn-ghost btn-sm rounded-xl"
										onClick={() => {
											capabilityControls?.onToggle(occurrence);
										}}
									>
										{isExcluded ? 'Include for next Run' : 'Exclude for next Run'}
									</button>
								) : null}
							</div>
						) : null}
					</div>
				</details>
			) : (
				<div className="p-3">{heading}</div>
			)}
			{occurrence.message ? <p className="text-warning px-3 pb-3 text-xs/5">{occurrence.message}</p> : null}
		</div>
	);
}

const GROUPS = [
	{ type: 'model', title: 'Model' },
	{ type: 'tool', title: 'Tools' },
	{ type: 'skill', title: 'Skills' },
	{ type: 'text', title: 'Text and instructions' },
	{ type: 'mcp', title: 'MCP servers' },
] as const;

export function AgentDeclarations({
	occurrences,
	capabilityControls,
	onSetup,
}: {
	occurrences: AgentCapabilityOccurrence[];
	capabilityControls?: AgentCapabilityControls;
	onSetup: (occurrence: AgentCapabilityOccurrence) => void;
}) {
	if (occurrences.length === 0) {
		return <p className="text-base-content/70 text-sm">This Agent has no declared contributions.</p>;
	}

	const groups = [
		...GROUPS.map(group => ({
			title: group.title,
			entries: occurrences.filter(occurrence => occurrence.type === group.type),
		})),
		{
			title: 'Other declarations',
			entries: occurrences.filter(occurrence => !GROUPS.some(group => group.type === occurrence.type)),
		},
	].filter(group => group.entries.length > 0);

	return (
		<div className="space-y-5">
			{groups.map(group => (
				<div key={group.title} className="space-y-2">
					<h5 className="text-sm font-semibold">{group.title}</h5>
					{group.entries.map(occurrence => (
						<ContributionRow
							key={occurrence.path}
							occurrence={occurrence}
							capabilityControls={capabilityControls}
							onSetup={onSetup}
						/>
					))}
				</div>
			))}
		</div>
	);
}
