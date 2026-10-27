import { useCallback, useMemo, useState } from 'react';
import { FiAlertCircle, FiCheckCircle, FiRefreshCw, FiTrash2 } from 'react-icons/fi';

import type { WorkspaceArtifactView, WorkspaceDirectoryView, WorkspaceRuntimePlan } from '@/spec/workspace';
import { artifactRefKey, ArtifactState } from '@/spec/artifact';
import { WorkspaceDirectoryOrigin } from '@/spec/workspace';

import { throwIfAborted } from '@/lib/async_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { workspaceManagementAPI } from '@/apis/baseapi';

import { Loader } from '@/components/loader';
import {
	ManagementModalBody,
	ManagementModalDialog,
	ManagementModalPanel,
} from '@/components/managementui/management_modal_layout';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { StatusBadge } from '@/components/managementui/status_badge';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalHeader } from '@/components/modal/modal_header';

import { WorkspaceContents, workspacePathParts } from '@/workspaces/workspace_contents';
import { WorkspaceDefaultPolicyDisclosure } from '@/workspaces/workspace_default_policy_disclosure';

interface WorkspaceDirectoryManagementModalProps {
	isOpen: boolean;
	directory: WorkspaceDirectoryView;
	onClose: () => void;
	onChanged: (directory: WorkspaceDirectoryView) => void;
	onRequestRemove: (directory: WorkspaceDirectoryView) => void;
}

type EffectiveWorkspace = WorkspaceDirectoryView['workspaces'][number];

function effectiveWorkspaceKey(entry: EffectiveWorkspace): string {
	const artifact = entry.workspace.artifact;
	return `${artifact.rootID}:${artifact.id}`;
}

function effectiveWorkspaceTitle(entry: EffectiveWorkspace): string {
	return entry.workspace.artifact.displayName || entry.workspace.artifact.logicalName;
}

function effectiveWorkspaceLocator(entry: EffectiveWorkspace): string {
	if (entry.origin === WorkspaceDirectoryOrigin.Default) {
		return '';
	}

	return entry.manifestLocator ?? entry.workspace.artifact.binding.locator;
}

function artifactKindLabel(kind: string): string {
	switch (kind) {
		case 'workspace':
			return 'Workspace setup';
		case 'text':
			return 'Repository text';
		case 'skill':
			return 'Skill';
		case 'mcp':
			return 'MCP server';
		case 'mcp.policy':
			return 'MCP policy';
		case 'agent':
			return 'Agent';
		case 'plugin':
			return 'Plugin';
		case 'model':
			return 'Model';
		case 'team':
			return 'Team';
		case 'loop':
			return 'Loop';
		case 'workflow':
			return 'Workflow';
		default:
			return kind || 'Repository item';
	}
}

function artifactPath(artifact: WorkspaceArtifactView): string {
	const locator = artifact.locator ?? '';
	const subresource = artifact.subresourceLocator?.toString() ?? '';

	return [locator, subresource].filter(Boolean).join(' / ');
}

function artifactTitle(artifact: WorkspaceArtifactView): string {
	if (artifact.displayName) {
		return artifact.displayName;
	}

	const path = workspacePathParts(artifact.locator);
	return path.name || artifact.logicalName || artifactKindLabel(artifact.kind);
}

function diagnosticTone(severity: string): 'error' | 'warning' | 'info' | 'neutral' {
	switch (severity) {
		case 'error':
			return 'error';
		case 'warning':
			return 'warning';
		case 'info':
			return 'info';
		default:
			return 'neutral';
	}
}

function capabilityTypeLabel(type: string): string {
	switch (type) {
		case 'text':
			return 'Repository file';
		case 'skill':
			return 'Skill';
		case 'mcp':
			return 'MCP server';
		case 'agent':
			return 'Agent';
		case 'model':
			return 'Model';
		case 'tool':
			return 'Tool';
		default:
			return type;
	}
}

function WorkspaceRuntimeContents({
	workspace,
	artifactViews,
	onSetArtifactEnabled,
	disabled,
}: {
	workspace: EffectiveWorkspace;
	artifactViews: ReadonlyMap<string, WorkspaceArtifactView>;
	onSetArtifactEnabled: (artifact: WorkspaceArtifactView, enabled: boolean) => void | Promise<void>;
	disabled: boolean;
}) {
	const workspaceRootID = workspace.workspace.artifact.rootID;
	const workspaceArtifactID = workspace.workspace.artifact.id;

	const loadPlan = useCallback(
		async (signal: AbortSignal): Promise<WorkspaceRuntimePlan> => {
			const plan = await workspaceManagementAPI.resolveWorkspaceRuntimePlan(
				{
					rootID: workspaceRootID,
					artifactID: workspaceArtifactID,
				},
				{
					requireComplete: false,
				}
			);
			throwIfAborted(signal);
			return plan;
		},
		[workspaceArtifactID, workspaceRootID]
	);

	const {
		data: plan,
		error,
		isLoading,
		isRefreshing,
		reloadOrThrow,
	} = useAsyncResource(loadPlan, {
		initialData: null as WorkspaceRuntimePlan | null,
	});

	const capabilityIssues =
		plan?.capabilities.occurrences.filter(
			occurrence => occurrence.required && occurrence.status.toString() !== 'available'
		) ?? [];

	return (
		<>
			<section className="space-y-3">
				<div>
					<div className="text-sm font-semibold">What this Workspace adds</div>
					<div className="text-base-content/60 mt-0.5 text-xs">Only resolved, usable content is shown here.</div>
				</div>

				{error ? (
					<ManagementResourceError
						title="Workspace contents could not be resolved"
						error={error}
						isRetrying={isRefreshing}
						onRetry={reloadOrThrow}
					/>
				) : null}

				{(isLoading || isRefreshing) && !plan ? <Loader text="Resolving repository contents..." /> : null}

				{plan ? (
					<WorkspaceContents
						plan={plan}
						artifactViews={artifactViews}
						onSetArtifactEnabled={onSetArtifactEnabled}
						disabled={disabled}
					/>
				) : null}
			</section>

			{capabilityIssues.length > 0 ? (
				<section className="space-y-2">
					<div className="text-sm font-semibold">Items that need attention</div>
					{capabilityIssues.map((issue, index) => (
						<div
							key={`${issue.path}:${issue.type}:${issue.name || ''}:${index}`}
							className="border-warning/40 bg-warning/10 rounded-2xl border p-3 text-sm"
						>
							<div className="flex flex-wrap items-center gap-2">
								<StatusBadge tone="warning">{capabilityTypeLabel(issue.type)}</StatusBadge>
								<div className="font-medium">{issue.name || `Required ${capabilityTypeLabel(issue.type)}`}</div>
							</div>
							<div className="text-base-content/70 mt-1 text-xs">
								{issue.message || `This item is currently ${issue.status}.`}
							</div>
						</div>
					))}
				</section>
			) : null}
		</>
	);
}

function WorkspaceDirectoryManagementModalContent({
	directory,
	onClose,
	onChanged,
	onRequestRemove,
}: Omit<WorkspaceDirectoryManagementModalProps, 'isOpen'>) {
	const [selectedWorkspaceKey, setSelectedWorkspaceKey] = useState(
		directory.workspaces[0] ? effectiveWorkspaceKey(directory.workspaces[0]) : ''
	);
	const [runtimePlanVersion, setRuntimePlanVersion] = useState(0);
	const [busy, setBusy] = useState('');
	const [actionError, setActionError] = useState('');

	const selectedWorkspace = useMemo(
		() =>
			directory.workspaces.find(entry => effectiveWorkspaceKey(entry) === selectedWorkspaceKey) ??
			directory.workspaces[0],
		[directory.workspaces, selectedWorkspaceKey]
	);

	const directoryRootID = directory.ref.rootID;

	const loadArtifacts = useCallback(
		async (signal: AbortSignal): Promise<WorkspaceArtifactView[]> => {
			const values = await workspaceManagementAPI.listWorkspaceDirectoryArtifacts({
				rootID: directoryRootID,
			});
			throwIfAborted(signal);
			return values;
		},
		[directoryRootID]
	);

	const {
		data: artifacts,
		error: artifactLoadError,
		isRefreshing: isRefreshingArtifacts,
		reloadOrThrow: reloadArtifactsOrThrow,
		setData: setArtifacts,
	} = useAsyncResource(loadArtifacts, {
		initialData: null as WorkspaceArtifactView[] | null,
	});

	const artifactsByKey = useMemo(
		() => new Map((artifacts ?? []).map(artifact => [artifactRefKey(artifact.artifact), artifact] as const)),
		[artifacts]
	);

	const reloadArtifacts = useCallback(async () => {
		await reloadArtifactsOrThrow().catch(() => undefined);
	}, [reloadArtifactsOrThrow]);

	const refreshDirectory = async () => {
		setActionError('');
		setBusy('refresh');

		try {
			const updated = await workspaceManagementAPI.refreshWorkspaceDirectory(directory.ref);
			onChanged(updated);
			setRuntimePlanVersion(previous => previous + 1);
			await reloadArtifacts();
		} catch (cause) {
			setActionError(cause instanceof Error ? cause.message : 'The repository could not be refreshed.');
		} finally {
			setBusy('');
		}
	};

	const setDirectoryEnabled = async () => {
		setActionError('');
		setBusy('enabled');

		try {
			const updated = await workspaceManagementAPI.setWorkspaceDirectoryEnabled(
				directory.ref,
				directory.directorySource.revision,
				!directory.enabled
			);
			onChanged(updated);
			setRuntimePlanVersion(previous => previous + 1);
		} catch (cause) {
			setActionError(cause instanceof Error ? cause.message : 'The repository status could not be changed.');
		} finally {
			setBusy('');
		}
	};

	const setArtifactEnabled = async (artifact: WorkspaceArtifactView, enabled: boolean) => {
		const key = `artifact:${artifact.artifact.artifactID}`;
		setActionError('');
		setBusy(key);

		try {
			const updatedArtifact = await workspaceManagementAPI.setWorkspaceDirectoryArtifactEnabled(
				directory.ref,
				artifact.artifact,
				artifact.revision,
				enabled
			);

			setArtifacts(
				previous =>
					previous?.map(value =>
						value.artifact.artifactID === updatedArtifact.artifact.artifactID ? updatedArtifact : value
					) ?? null
			);

			const updatedDirectory = await workspaceManagementAPI.getWorkspaceDirectory(directory.ref);
			onChanged(updatedDirectory);
			setRuntimePlanVersion(previous => previous + 1);
		} catch (cause) {
			setActionError(cause instanceof Error ? cause.message : 'The repository item could not be changed.');
		} finally {
			setBusy('');
		}
	};

	return (
		<ManagementModalPanel width="wide">
			<ModalHeader
				title={`Manage ${directory.root.displayName}`}
				description="Review the files, Skills, and integrations this repository adds. Advanced discovery controls remain available below."
				onClose={onClose}
				closeDisabled={Boolean(busy)}
			/>

			<ManagementModalBody>
				{actionError ? (
					<div className="alert alert-error rounded-2xl text-sm" role="alert">
						<FiAlertCircle size={15} />
						<span>{actionError}</span>
					</div>
				) : null}

				{artifactLoadError ? (
					<ManagementResourceError
						title="Workspace item controls could not be loaded"
						error={artifactLoadError}
						isRetrying={isRefreshingArtifacts}
						onRetry={reloadArtifactsOrThrow}
					/>
				) : null}

				<div className="border-base-content/10 bg-base-100 flex flex-col gap-3 rounded-2xl border p-4 sm:flex-row sm:items-center">
					<div className="min-w-0 grow">
						<div className="flex flex-wrap items-center gap-2">
							<div className="font-semibold">{directory.root.displayName}</div>
							<StatusBadge tone={directory.enabled ? 'success' : 'warning'}>
								{directory.enabled ? 'Enabled' : 'Paused'}
							</StatusBadge>
							<StatusBadge>
								{directory.workspaces.length} Workspace
								{directory.workspaces.length === 1 ? '' : 's'}
							</StatusBadge>
						</div>
						<div className="text-base-content/60 mt-1 text-xs">
							Repository paths shown below are relative to this directory.
						</div>
					</div>

					{directory.enabled ? (
						<div className="text-success flex items-center gap-2 text-xs">
							<FiCheckCircle size={14} />
							Repository content is available to conversations
						</div>
					) : (
						<div className="text-warning text-xs">
							Enable this repository to resolve and use its Workspace contents.
						</div>
					)}
				</div>

				{directory.workspaces.length > 0 ? (
					<section className="space-y-2">
						<div>
							<div className="text-sm font-semibold">Workspace setup</div>
							<div className="text-base-content/60 mt-0.5 text-xs">
								Choose a setup to inspect when this repository exposes more than one.
							</div>
						</div>

						<div>
							{directory.workspaces.map(entry => {
								const key = effectiveWorkspaceKey(entry);
								const selected = key === effectiveWorkspaceKey(selectedWorkspace ?? entry);
								const locator = effectiveWorkspaceLocator(entry);
								const path = workspacePathParts(locator);

								return (
									<div
										key={key}
										className={`border-base-content/10 bg-base-100 flex min-w-0 flex-col gap-2 rounded-2xl border p-2 ${
											selected ? 'border-secondary/50 bg-secondary/10' : ''
										}`}
									>
										<div className="flex min-w-0 items-center gap-2">
											<button
												type="button"
												className="min-w-0 grow rounded-xl p-2 text-left"
												onClick={() => {
													setSelectedWorkspaceKey(key);
												}}
											>
												<div className="truncate text-sm font-medium">{effectiveWorkspaceTitle(entry)}</div>
												<div className="text-base-content/60 mt-1 truncate font-mono text-xs">
													{entry.origin === WorkspaceDirectoryOrigin.Default
														? 'Automatic repository setup'
														: path.path || 'Repository workspace file'}
												</div>
											</button>

											<StatusBadge tone={entry.origin === WorkspaceDirectoryOrigin.Default ? 'info' : 'success'}>
												{entry.origin === WorkspaceDirectoryOrigin.Default ? 'Automatic' : 'Repository'}
											</StatusBadge>
										</div>

										{entry.origin === WorkspaceDirectoryOrigin.Default ? <WorkspaceDefaultPolicyDisclosure /> : null}
									</div>
								);
							})}
						</div>
					</section>
				) : (
					<div className="alert alert-warning rounded-2xl text-sm">
						<FiAlertCircle size={15} />
						<span>
							No usable Workspace setup is available. Review the repository files that need attention below, then
							refresh.
						</span>
					</div>
				)}

				{selectedWorkspace && directory.enabled ? (
					<WorkspaceRuntimeContents
						key={`${effectiveWorkspaceKey(selectedWorkspace)}:${selectedWorkspace.workspace.artifact.revision}:${runtimePlanVersion}`}
						workspace={selectedWorkspace}
						artifactViews={artifactsByKey}
						onSetArtifactEnabled={setArtifactEnabled}
						disabled={Boolean(busy)}
					/>
				) : null}

				{directory.diagnostics?.length ? (
					<section className="space-y-2">
						<div>
							<div className="text-sm font-semibold">Repository files that need attention</div>
							<div className="text-base-content/60 mt-0.5 text-xs">Fix these files and refresh the repository.</div>
						</div>

						{directory.diagnostics.map((diagnostic, index) => (
							<div
								key={`${diagnostic.code}:${diagnostic.message}:${index}`}
								className="border-base-content/10 bg-base-100 rounded-2xl border p-3 text-sm"
							>
								<div className="flex flex-wrap items-center gap-2">
									<StatusBadge tone={diagnosticTone(diagnostic.severity)}>{diagnostic.severity}</StatusBadge>
									<div>{diagnostic.message}</div>
								</div>
								{diagnostic.location?.locator ? (
									<div className="text-base-content/60 mt-2 font-mono text-xs">{diagnostic.location.locator}</div>
								) : null}
							</div>
						))}
					</section>
				) : null}

				{artifacts?.some(artifact => artifact.state === ArtifactState.Available && !artifact.enabled) ? (
					<section className="space-y-2">
						<div>
							<div className="text-sm font-semibold">Excluded items</div>
							<div className="text-base-content/60 mt-0.5 text-xs">
								These repository items are available but are not currently added by this Workspace.
							</div>
						</div>

						{artifacts
							.filter(artifact => artifact.state === ArtifactState.Available && !artifact.enabled)
							.map(artifact => {
								const isText = artifact.kind === 'text';
								const path = artifactPath(artifact);

								return (
									<div
										key={artifact.artifact.artifactID}
										className="border-base-content/10 bg-base-100 flex flex-col gap-3 rounded-2xl border p-3 sm:flex-row sm:items-center"
									>
										<div className="min-w-0 grow">
											<div className="flex flex-wrap items-center gap-2">
												<div className="truncate text-sm font-medium">
													{isText ? 'Repository file group' : artifactTitle(artifact)}
												</div>
												<StatusBadge tone="warning">Excluded</StatusBadge>
												<StatusBadge>{artifactKindLabel(artifact.kind)}</StatusBadge>
											</div>
											<div className="text-base-content/60 mt-1 truncate font-mono text-xs">
												{isText ? 'Source files selected by this Workspace' : path || 'Repository Workspace setup'}
											</div>
										</div>

										<button
											type="button"
											className="btn btn-sm btn-ghost rounded-xl"
											disabled={Boolean(busy)}
											onClick={() => {
												void setArtifactEnabled(artifact, true);
											}}
										>
											Include
										</button>
									</div>
								);
							})}
					</section>
				) : null}
			</ManagementModalBody>

			<ModalActions
				leading={
					<button
						type="button"
						className="btn btn-ghost text-error rounded-xl"
						disabled={Boolean(busy)}
						onClick={() => {
							onRequestRemove(directory);
						}}
					>
						<FiTrash2 size={14} />
						Remove
					</button>
				}
			>
				<button
					type="button"
					className="btn btn-ghost rounded-xl"
					disabled={Boolean(busy)}
					onClick={() => {
						void refreshDirectory();
					}}
				>
					<FiRefreshCw size={14} />
					{busy === 'refresh' ? 'Refreshing...' : 'Refresh repository'}
				</button>

				<button
					type="button"
					className="btn btn-ghost rounded-xl"
					disabled={Boolean(busy)}
					onClick={() => {
						void setDirectoryEnabled();
					}}
				>
					{busy === 'enabled' ? 'Saving...' : directory.enabled ? 'Pause repository' : 'Enable repository'}
				</button>

				<button type="button" className="btn btn-primary rounded-xl" disabled={Boolean(busy)} onClick={onClose}>
					Done
				</button>
			</ModalActions>
		</ManagementModalPanel>
	);
}

export function WorkspaceDirectoryManagementModal(props: WorkspaceDirectoryManagementModalProps) {
	if (!props.isOpen) {
		return null;
	}

	return (
		<ManagementModalDialog isOpen onClose={props.onClose} blockCancel>
			<WorkspaceDirectoryManagementModalContent
				key={props.directory.ref.rootID}
				directory={props.directory}
				onClose={props.onClose}
				onChanged={props.onChanged}
				onRequestRemove={props.onRequestRemove}
			/>
		</ManagementModalDialog>
	);
}
