import { useCallback, useMemo, useState } from 'react';
import { FiAlertCircle, FiCheckCircle, FiFileText, FiRefreshCw, FiSettings, FiTrash2 } from 'react-icons/fi';

import type { WorkspaceArtifactView, WorkspaceDirectoryView, WorkspaceRuntimePlan } from '@/spec/workspace';
import { WorkspaceDirectoryOrigin } from '@/spec/workspace';

import { throwIfAborted } from '@/lib/async_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { workspaceManagementAPI } from '@/apis/baseapi';

import { Loader } from '@/components/loader';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { StatusBadge } from '@/components/managementui/status_badge';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

import { WorkspaceContents, workspacePathParts } from '@/workspaces/workspace_contents';

interface WorkspaceDirectoryManagementModalProps {
	isOpen: boolean;
	directory: WorkspaceDirectoryView;
	onClose: () => void;
	onChanged: (directory: WorkspaceDirectoryView) => void;
	onRequestRemove: (directory: WorkspaceDirectoryView) => void;
	onShowDefaultPolicy: () => void;
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

function WorkspaceRuntimeContents({ workspace }: { workspace: EffectiveWorkspace }) {
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

				{plan ? <WorkspaceContents plan={plan} /> : null}
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
	onShowDefaultPolicy,
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
		isLoading: isLoadingArtifacts,
		isRefreshing: isRefreshingArtifacts,
		reloadOrThrow: reloadArtifactsOrThrow,
		setData: setArtifacts,
	} = useAsyncResource(loadArtifacts, {
		initialData: null as WorkspaceArtifactView[] | null,
	});

	const isReloadingArtifacts = isLoadingArtifacts || isRefreshingArtifacts;

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

	const setArtifactEnabled = async (artifact: WorkspaceArtifactView) => {
		const key = `artifact:${artifact.artifact.artifactID}`;
		setActionError('');
		setBusy(key);

		try {
			const updatedArtifact = await workspaceManagementAPI.setWorkspaceDirectoryArtifactEnabled(
				directory.ref,
				artifact.artifact,
				artifact.revision,
				!artifact.enabled
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
		<div className="modal-box bg-base-200 flex max-h-[90vh] w-[calc(100%-1rem)] max-w-6xl flex-col overflow-hidden rounded-2xl p-0">
			<ModalHeader
				title={`Manage ${directory.root.displayName}`}
				description="Review the files, Skills, and integrations this repository adds. Advanced discovery controls remain available below."
				onClose={onClose}
				closeDisabled={Boolean(busy)}
			/>

			<div className="app-scrollbar-thin min-h-0 flex-1 space-y-5 overflow-y-auto p-4 sm:p-6">
				{actionError ? (
					<div className="alert alert-error rounded-2xl text-sm" role="alert">
						<FiAlertCircle size={15} />
						<span>{actionError}</span>
					</div>
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
										className={`border-base-content/10 bg-base-100 flex min-w-0 items-center gap-2 rounded-2xl border p-2 ${
											selected ? 'border-secondary/50 bg-secondary/10' : ''
										}`}
									>
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

										{entry.origin === WorkspaceDirectoryOrigin.Default ? (
											<button type="button" className="btn btn-ghost btn-xs rounded-lg" onClick={onShowDefaultPolicy}>
												<FiFileText size={13} />
												View setup
											</button>
										) : null}
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

				<details className="border-base-content/10 bg-base-100 rounded-2xl border">
					<summary className="flex cursor-pointer list-none items-center gap-2 p-4 text-sm font-semibold">
						<FiSettings size={15} />
						<span>Advanced discovery controls</span>
						<span className="text-base-content/50 ml-auto text-xs">{artifacts?.length ?? 0} items</span>
					</summary>

					<div className="border-base-content/10 space-y-3 border-t p-4">
						<div className="flex flex-col gap-3 sm:flex-row sm:items-center">
							<div className="text-base-content/65 min-w-0 grow text-xs">
								Include or exclude individual repository items. Workspace setup files are listed here because changing
								them can affect which Workspace is available.
							</div>
							<button
								type="button"
								className="btn btn-ghost btn-sm rounded-xl"
								disabled={isReloadingArtifacts}
								onClick={() => {
									void reloadArtifacts();
								}}
							>
								<FiRefreshCw size={14} />
								{isReloadingArtifacts ? 'Loading...' : 'Reload items'}
							</button>
						</div>

						{artifactLoadError ? (
							<ManagementResourceError
								title="Repository items could not be loaded"
								error={artifactLoadError}
								isRetrying={isRefreshingArtifacts}
								onRetry={reloadArtifactsOrThrow}
							/>
						) : null}

						{isReloadingArtifacts && artifacts === null ? (
							<div className="text-base-content/60 text-sm">Loading repository items...</div>
						) : null}

						{artifacts?.map(artifact => {
							const key = `artifact:${artifact.artifact.artifactID}`;
							const path = artifactPath(artifact);
							const state = artifact.state.toString();

							return (
								<div
									key={artifact.artifact.artifactID}
									className="border-base-content/10 bg-base-200/40 flex flex-col gap-3 rounded-2xl border p-3 sm:flex-row sm:items-center"
								>
									<div className="min-w-0 grow">
										<div className="flex flex-wrap items-center gap-2">
											<div className="truncate text-sm font-medium">{artifactTitle(artifact)}</div>
											<StatusBadge>{artifactKindLabel(artifact.kind)}</StatusBadge>
											<StatusBadge tone={artifact.enabled ? 'success' : 'warning'}>
												{artifact.enabled ? 'Included' : 'Excluded'}
											</StatusBadge>
											{state !== 'available' ? <StatusBadge tone="warning">{state}</StatusBadge> : null}
										</div>
										{path ? <div className="text-base-content/60 mt-1 truncate font-mono text-xs">{path}</div> : null}
									</div>

									<button
										type="button"
										className="btn btn-sm btn-ghost rounded-xl"
										disabled={Boolean(busy)}
										onClick={() => {
											void setArtifactEnabled(artifact);
										}}
									>
										{busy === key ? 'Saving...' : artifact.enabled ? 'Exclude' : 'Include'}
									</button>
								</div>
							);
						})}

						{artifacts !== null && artifacts.length === 0 ? (
							<ManagementEmptyState>No repository items were discovered.</ManagementEmptyState>
						) : null}

						{artifacts?.some(artifact => artifact.kind === 'mcp') ? (
							<div className="text-base-content/60 text-xs">
								MCP credentials and installation values remain in MCP management and are never exposed through Workspace
								APIs.
							</div>
						) : null}
					</div>
				</details>
			</div>

			<ModalActions className="justify-between">
				<div className="flex flex-wrap items-center gap-2">
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
				</div>

				<button type="button" className="btn btn-primary rounded-xl" disabled={Boolean(busy)} onClick={onClose}>
					Done
				</button>
			</ModalActions>
		</div>
	);
}

export function WorkspaceDirectoryManagementModal(props: WorkspaceDirectoryManagementModalProps) {
	if (!props.isOpen) {
		return null;
	}

	return (
		<ModalDialog isOpen onClose={props.onClose} blockCancel>
			<WorkspaceDirectoryManagementModalContent
				key={props.directory.ref.rootID}
				directory={props.directory}
				onClose={props.onClose}
				onChanged={props.onChanged}
				onRequestRemove={props.onRequestRemove}
				onShowDefaultPolicy={props.onShowDefaultPolicy}
			/>
		</ModalDialog>
	);
}
