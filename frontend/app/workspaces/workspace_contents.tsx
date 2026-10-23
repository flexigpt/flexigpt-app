// oxlint-disable react/only-export-components
import type { ReactNode } from 'react';
import { FiBookOpen, FiFileText, FiServer, FiZap } from 'react-icons/fi';

import type {
	WorkspaceArtifactView,
	WorkspaceMCPServer,
	WorkspacePromptContribution,
	WorkspaceRuntimePlan,
	WorkspaceSkill,
} from '@/spec/workspace';
import { artifactRefKey, ArtifactState } from '@/spec/artifact';
import { WorkspaceInsertTarget } from '@/spec/workspace';

import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { StatusBadge } from '@/components/managementui/status_badge';

export interface WorkspaceContentSelectionState {
	selectedContextIDs: ReadonlySet<string>;
	selectedSkillIDs: ReadonlySet<string>;
	selectedMCPIDs: ReadonlySet<string>;
	activeSkillIDs?: ReadonlySet<string>;
}

interface WorkspaceContentsProps {
	plan: WorkspaceRuntimePlan;
	selection?: WorkspaceContentSelectionState;
	artifactViews?: ReadonlyMap<string, WorkspaceArtifactView>;
	disabled?: boolean;
	onToggleContext?: (context: WorkspacePromptContribution, selected: boolean) => void;
	onToggleSkill?: (skill: WorkspaceSkill, selected: boolean) => void;
	onSetSkillActive?: (skill: WorkspaceSkill, active: boolean) => void;
	onToggleMCP?: (server: WorkspaceMCPServer, selected: boolean) => void;
	onSetArtifactEnabled?: (artifact: WorkspaceArtifactView, enabled: boolean) => void | Promise<void>;
	renderMCPAction?: (server: WorkspaceMCPServer) => ReactNode;
}

export interface WorkspaceContentCounts {
	instructionFiles: number;
	referenceFiles: number;
	skills: number;
	mcpServers: number;
}

export interface WorkspacePathParts {
	name: string;
	directory: string;
	path: string;
}

function normalizeWorkspacePath(locator: string | undefined): string {
	if (!locator) {
		return '';
	}

	const normalized = locator.replaceAll('\\', '/').replace(/^\.\/+/, '');
	return normalized === '.' ? '' : normalized;
}

function isWorkspaceManifestPath(locator: string): boolean {
	const segments = locator.split('/');
	const filename = (segments.at(-1) ?? '').toLowerCase();

	return (
		filename === 'workspace.yaml' ||
		filename === 'workspace.yml' ||
		filename === 'workspace.json' ||
		filename.endsWith('.workspace.yaml') ||
		filename.endsWith('.workspace.yml') ||
		filename.endsWith('.workspace.json')
	);
}

export function workspacePathParts(locator: string | undefined): WorkspacePathParts {
	const normalized = normalizeWorkspacePath(locator);
	if (!normalized) {
		return {
			name: '',
			directory: '',
			path: '',
		};
	}

	const segments = normalized.split('/').filter(Boolean);
	const name = segments.pop() ?? normalized;

	return {
		name,
		directory: segments.join('/'),
		path: normalized,
	};
}

function contributionSourceLocators(context: WorkspacePromptContribution): string[] {
	const values = context.sourceLocators ?? [];
	const unique = new Set<string>();

	for (const value of values) {
		const normalized = normalizeWorkspacePath(value);
		if (normalized) {
			unique.add(normalized);
		}
	}

	if (unique.size === 0) {
		const fallback = normalizeWorkspacePath(context.locator);
		if (fallback && !isWorkspaceManifestPath(fallback)) {
			unique.add(fallback);
		}
	}

	return [...unique];
}

function includedContextCount(context: WorkspacePromptContribution): number {
	const locators = contributionSourceLocators(context);
	if (locators.length > 0) {
		return locators.length;
	}

	return context.includedBytes > 0 ? 1 : 0;
}

function hasVisibleContext(context: WorkspacePromptContribution): boolean {
	return contributionSourceLocators(context).length > 0 || context.includedBytes > 0;
}

export function getWorkspaceContentCounts(
	plan: WorkspaceRuntimePlan | undefined,
	selection?: Partial<WorkspaceContentSelectionState>
): WorkspaceContentCounts {
	if (!plan) {
		return {
			instructionFiles: 0,
			referenceFiles: 0,
			skills: 0,
			mcpServers: 0,
		};
	}

	let instructionFiles = 0;
	let referenceFiles = 0;

	for (const context of plan.prompt.contributions) {
		if (!hasVisibleContext(context)) {
			continue;
		}

		if (selection?.selectedContextIDs && !selection.selectedContextIDs.has(artifactRefKey(context.artifact))) {
			continue;
		}

		if (context.insert === WorkspaceInsertTarget.Instructions) {
			instructionFiles += includedContextCount(context);
		} else {
			referenceFiles += includedContextCount(context);
		}
	}

	const skills = plan.skills.skills.filter(skill => {
		return !selection?.selectedSkillIDs || selection.selectedSkillIDs.has(artifactRefKey(skill.artifact));
	}).length;

	const mcpServers = plan.mcpServers.servers.filter(server => {
		return !selection?.selectedMCPIDs || selection.selectedMCPIDs.has(artifactRefKey(server.artifact));
	}).length;

	return {
		instructionFiles,
		referenceFiles,
		skills,
		mcpServers,
	};
}

function WorkspacePathList({ locators }: { locators: string[] }) {
	if (locators.length === 0) {
		return null;
	}

	return (
		<div className="mt-3 max-h-48 space-y-1 overflow-y-auto">
			{locators.map((locator, index) => {
				const path = workspacePathParts(locator);

				return (
					<div
						key={`${locator}:${index}`}
						className="border-base-content/10 bg-base-200/50 flex min-w-0 items-center gap-2 rounded-xl border px-3 py-2"
						title={path.path}
					>
						<FiFileText size={14} className="text-base-content/60 shrink-0" />
						<div className="min-w-0">
							<div className="truncate text-xs font-medium">{path.name || locator}</div>
							<div className="text-base-content/60 truncate font-mono text-[11px]">
								{path.directory || 'Repository root'}
							</div>
						</div>
					</div>
				);
			})}
		</div>
	);
}

function WorkspaceContentCard({
	icon,
	title,
	description,
	typeLabel,
	selected,
	selectionMode,
	canSelect,
	disabled,
	onSelectedChange,
	actions,
	children,
	statusLabel,
}: {
	icon: ReactNode;
	title: string;
	description?: string;
	typeLabel: string;
	selected: boolean;
	selectionMode: boolean;
	canSelect: boolean;
	disabled: boolean;
	onSelectedChange?: (selected: boolean) => void;
	actions?: ReactNode;
	children?: ReactNode;
	statusLabel?: string;
}) {
	return (
		<div
			className={`border-base-content/10 bg-base-100 rounded-2xl border p-3 ${
				selectionMode && selected ? 'border-secondary/50 bg-secondary/10' : ''
			}`}
		>
			<div className="flex min-w-0 items-start gap-3">
				{canSelect ? (
					<input
						type="checkbox"
						className="checkbox checkbox-sm mt-0.5 shrink-0"
						checked={selected}
						disabled={disabled}
						aria-label={`${selected ? 'Remove' : 'Add'} ${title}`}
						onChange={event => {
							onSelectedChange?.(event.currentTarget.checked);
						}}
					/>
				) : (
					<span className="text-base-content/60 mt-0.5 shrink-0">{icon}</span>
				)}

				<div className="min-w-0 grow">
					<div className="flex flex-wrap items-center gap-2">
						<div className="min-w-0 grow truncate text-sm font-medium">{title}</div>
						<StatusBadge>{typeLabel}</StatusBadge>
						<StatusBadge tone={selected ? 'success' : 'neutral'}>
							{statusLabel ?? (selectionMode ? (selected ? 'Added' : 'Not added') : 'Included')}
						</StatusBadge>
					</div>

					{description ? <div className="text-base-content/65 mt-1 text-xs">{description}</div> : null}

					{children}
				</div>
			</div>

			{actions ? (
				<div className="border-base-content/10 mt-3 flex flex-wrap items-center justify-end gap-2 border-t pt-3">
					{actions}
				</div>
			) : null}
		</div>
	);
}

function WorkspaceContentSection({
	title,
	description,
	count,
	children,
}: {
	title: string;
	description: string;
	count: number;
	children: ReactNode;
}) {
	return (
		<section className="space-y-2">
			<div className="flex items-end justify-between gap-3">
				<div>
					<div className="text-sm font-semibold">{title}</div>
					<div className="text-base-content/60 mt-0.5 text-xs">{description}</div>
				</div>
				<div className="text-base-content/50 shrink-0 text-xs">{count}</div>
			</div>
			<div className="space-y-2">{children}</div>
		</section>
	);
}

export function WorkspaceContents({
	plan,
	selection,
	artifactViews,
	disabled = false,
	onToggleContext,
	onToggleSkill,
	onSetSkillActive,
	onToggleMCP,
	onSetArtifactEnabled,
	renderMCPAction,
}: WorkspaceContentsProps) {
	const renderArtifactControl = (artifactRef: WorkspaceArtifactView['artifact'], label: string): ReactNode => {
		const artifact = artifactViews?.get(artifactRefKey(artifactRef));
		if (!artifact || !onSetArtifactEnabled) {
			return undefined;
		}

		return (
			<button
				type="button"
				className="btn btn-sm btn-ghost rounded-xl"
				disabled={disabled || artifact.state !== ArtifactState.Available}
				onClick={() => {
					void onSetArtifactEnabled(artifact, !artifact.enabled);
				}}
			>
				{artifact.enabled ? `Exclude ${label}` : `Include ${label}`}
			</button>
		);
	};

	const instructionContexts = plan.prompt.contributions.filter(
		context => context.insert === WorkspaceInsertTarget.Instructions && hasVisibleContext(context)
	);
	const referenceContexts = plan.prompt.contributions.filter(
		context => context.insert !== WorkspaceInsertTarget.Instructions && hasVisibleContext(context)
	);
	const selectionMode = selection !== undefined;
	const hasContents =
		instructionContexts.length > 0 ||
		referenceContexts.length > 0 ||
		plan.skills.skills.length > 0 ||
		plan.mcpServers.servers.length > 0;

	if (!hasContents) {
		return (
			<ManagementEmptyState>
				This Workspace does not currently add repository files, Skills, or MCP servers.
			</ManagementEmptyState>
		);
	}

	const renderContext = (context: WorkspacePromptContribution, instruction: boolean) => {
		const key = artifactRefKey(context.artifact);
		const selected = selection ? selection.selectedContextIDs.has(key) : true;
		const locators = contributionSourceLocators(context);
		const title =
			locators.length > 0
				? instruction
					? 'Repository instruction files'
					: 'Repository reference files'
				: instruction
					? 'Repository instruction files'
					: 'Repository reference files';
		const description =
			locators.length > 0
				? `${locators.length} ${locators.length === 1 ? 'file is' : 'files are'} added together.`
				: context.locator
					? 'No matching repository files are currently available.'
					: 'Inline content supplied by this Workspace.';

		return (
			<WorkspaceContentCard
				key={key}
				icon={instruction ? <FiBookOpen size={15} /> : <FiFileText size={15} />}
				title={title}
				description={description}
				typeLabel={instruction ? 'Instructions' : 'Reference'}
				selected={selected}
				selectionMode={selectionMode}
				canSelect={Boolean(onToggleContext)}
				disabled={disabled}
				onSelectedChange={next => {
					onToggleContext?.(context, next);
				}}
				actions={renderArtifactControl(context.artifact, locators.length === 1 ? 'file' : 'files')}
			>
				<WorkspacePathList locators={locators} />
				{context.truncated ? (
					<div className="text-warning mt-2 text-xs">
						Only part of this content fits within the Workspace prompt limit.
					</div>
				) : null}
			</WorkspaceContentCard>
		);
	};

	const renderSkill = (skill: WorkspaceSkill) => {
		const key = artifactRefKey(skill.artifact);
		const instruction = skill.insert === WorkspaceInsertTarget.Instructions;
		const selected = selection ? (instruction ? selection.selectedSkillIDs.has(key) : true) : true;
		const active = selection?.activeSkillIDs?.has(key) ?? false;
		const path = normalizeWorkspacePath(skill.locator);
		const artifactControl = renderArtifactControl(skill.artifact, 'Skill');
		const activeControl =
			instruction && selected && onSetSkillActive ? (
				<label className="flex items-center gap-2 text-xs">
					<span>Active</span>
					<input
						type="checkbox"
						className="toggle toggle-accent toggle-sm"
						checked={active}
						disabled={disabled}
						onChange={event => {
							onSetSkillActive(skill, event.currentTarget.checked);
						}}
					/>
				</label>
			) : undefined;

		return (
			<WorkspaceContentCard
				key={key}
				icon={<FiZap size={15} />}
				title={skill.displayName || skill.name}
				description={
					instruction
						? 'Adds reusable instructions to the conversation Skill session.'
						: 'Available to the Workspace as a runtime Skill.'
				}
				typeLabel="Skill"
				selected={selected}
				selectionMode={selectionMode}
				canSelect={instruction && Boolean(onToggleSkill)}
				disabled={disabled}
				statusLabel={instruction ? undefined : 'Runtime'}
				onSelectedChange={next => {
					onToggleSkill?.(skill, next);
				}}
				actions={
					artifactControl || activeControl ? (
						<>
							{artifactControl}
							{activeControl}
						</>
					) : undefined
				}
			>
				<WorkspacePathList locators={path ? [path] : []} />
			</WorkspaceContentCard>
		);
	};

	const renderServer = (server: WorkspaceMCPServer) => {
		const key = artifactRefKey(server.artifact);
		const selected = selection ? selection.selectedMCPIDs.has(key) : true;
		const locator = normalizeWorkspacePath(server.locator);
		const entry = server.subresourceLocator?.trim();
		const artifactControl = renderArtifactControl(server.artifact, 'MCP server');

		return (
			<WorkspaceContentCard
				key={key}
				icon={<FiServer size={15} />}
				title={server.displayName || server.name}
				description={
					entry
						? `MCP server from ${entry} in the repository configuration file.`
						: server.builtIn
							? 'Application-provided MCP server included by this Workspace.'
							: 'MCP server included by this Workspace.'
				}
				typeLabel="MCP"
				selected={selected}
				selectionMode={selectionMode}
				canSelect={Boolean(onToggleMCP)}
				disabled={disabled}
				onSelectedChange={next => {
					onToggleMCP?.(server, next);
				}}
				actions={
					artifactControl || renderMCPAction?.(server) ? (
						<>
							{artifactControl}
							{renderMCPAction?.(server)}
						</>
					) : undefined
				}
			>
				<WorkspacePathList locators={locator ? [locator] : []} />
			</WorkspaceContentCard>
		);
	};

	return (
		<div className="space-y-5">
			{instructionContexts.length > 0 ? (
				<WorkspaceContentSection
					title="Instructions"
					description="Guidance automatically supplied to the model."
					count={instructionContexts.reduce((total, value) => total + includedContextCount(value), 0)}
				>
					{instructionContexts.map(context => renderContext(context, true))}
				</WorkspaceContentSection>
			) : null}

			{referenceContexts.length > 0 ? (
				<WorkspaceContentSection
					title="Reference files"
					description="README, documentation, and other repository context."
					count={referenceContexts.reduce((total, value) => total + includedContextCount(value), 0)}
				>
					{referenceContexts.map(context => renderContext(context, false))}
				</WorkspaceContentSection>
			) : null}

			{plan.skills.skills.length > 0 ? (
				<WorkspaceContentSection
					title="Skills"
					description="Repository Skills available to the conversation."
					count={plan.skills.skills.length}
				>
					{plan.skills.skills.map(renderSkill)}
				</WorkspaceContentSection>
			) : null}

			{plan.mcpServers.servers.length > 0 ? (
				<WorkspaceContentSection
					title="MCP servers"
					description="Repository integrations made available by this Workspace."
					count={plan.mcpServers.servers.length}
				>
					{plan.mcpServers.servers.map(renderServer)}
				</WorkspaceContentSection>
			) : null}
		</div>
	);
}
