import { useState } from 'react';
import { FiEye, FiFileText } from 'react-icons/fi';

import type { AgentCapabilityOccurrence } from '@/spec/agent';
import { CapabilityResolutionStatus, CapabilityTargetForm } from '@/spec/artifact';

import { AgentContributionDetailsModal } from '@/agents/agent_contribution_details_modal';
import { getAgentRelationshipBadgeClass } from '@/agents/agent_management_utils';

export interface AgentCapabilityControls {
	isExcluded: (occurrence: AgentCapabilityOccurrence) => boolean;
	onToggle: (occurrence: AgentCapabilityOccurrence) => void;
}

function ContributionRow({
	occurrence,
	capabilityControls,
	onSetup,
	onView,
}: {
	occurrence: AgentCapabilityOccurrence;
	capabilityControls?: AgentCapabilityControls;
	onSetup: (occurrence: AgentCapabilityOccurrence) => void;
	onView: (occurrence: AgentCapabilityOccurrence) => void;
}) {
	const sourceArtifact =
		occurrence.status === CapabilityResolutionStatus.Available &&
		occurrence.target?.form === CapabilityTargetForm.Artifact
			? occurrence.target.artifact
			: undefined;
	const canView =
		Boolean(sourceArtifact) &&
		(occurrence.type === 'tool' ||
			occurrence.type === 'skill' ||
			occurrence.type === 'text' ||
			occurrence.type === 'mcp' ||
			occurrence.type === 'model' ||
			occurrence.type === 'agent');
	const canToggle =
		Boolean(capabilityControls) &&
		occurrence.status === CapabilityResolutionStatus.Available &&
		(occurrence.type === 'tool' ||
			occurrence.type === 'skill' ||
			occurrence.type === 'text' ||
			occurrence.type === 'mcp');
	const isExcluded = canToggle && Boolean(capabilityControls?.isExcluded(occurrence));
	const detailSummary =
		occurrence.type === 'model' && occurrence.includeSystemPrompt !== undefined
			? `System prompt: ${occurrence.includeSystemPrompt ? 'included' : 'not included'}`
			: occurrence.type === 'skill' && occurrence.skillUseMode
				? `Use mode: ${occurrence.skillUseMode}`
				: occurrence.type === 'mcp'
					? 'Setup, connection, and policy approval are required before Run use.'
					: undefined;

	const actions =
		canView || occurrence.type === 'mcp' || canToggle ? (
			<div className="flex shrink-0 flex-wrap items-center justify-end gap-2">
				{canView ? (
					<button
						type="button"
						className="btn btn-ghost btn-sm rounded-xl"
						onClick={() => {
							onView(occurrence);
						}}
					>
						<FiEye size={15} />
						View
					</button>
				) : null}

				{occurrence.type === 'mcp' ? (
					<button
						type="button"
						className="btn btn-ghost btn-sm rounded-xl"
						onClick={() => {
							onSetup(occurrence);
						}}
					>
						<FiFileText size={15} />
						{occurrence.status === CapabilityResolutionStatus.Available && sourceArtifact
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
		) : null;

	return (
		<div className="border-base-content/10 rounded-xl border p-3">
			<div className="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
				<div className="min-w-0">
					<div className="flex flex-wrap items-center gap-2">
						<span className="min-w-0 text-sm font-semibold wrap-break-word">
							{occurrence.name || `Unnamed ${occurrence.type}`}
						</span>
						<span className={`badge badge-sm capitalize ${getAgentRelationshipBadgeClass(occurrence.status)}`}>
							{occurrence.status}
						</span>
						{occurrence.required ? <span className="badge badge-outline badge-sm">Required</span> : null}
						{isExcluded ? <span className="badge badge-warning badge-sm">Excluded</span> : null}
					</div>

					{detailSummary ? <p className="text-base-content/60 mt-1 text-xs">{detailSummary}</p> : null}
					{occurrence.message ? (
						<p className="text-warning mt-2 text-xs/5 wrap-break-word">{occurrence.message}</p>
					) : null}
				</div>

				{actions}
			</div>
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
	const [viewedOccurrence, setViewedOccurrence] = useState<AgentCapabilityOccurrence | null>(null);

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
		<>
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
								onView={setViewedOccurrence}
							/>
						))}
					</div>
				))}
			</div>

			<AgentContributionDetailsModal
				isOpen={viewedOccurrence !== null}
				occurrence={viewedOccurrence}
				onClose={() => {
					setViewedOccurrence(null);
				}}
			/>
		</>
	);
}
