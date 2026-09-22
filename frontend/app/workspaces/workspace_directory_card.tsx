import { useCallback, useEffect, useState } from 'react';
import { FiAlertCircle, FiChevronDown, FiChevronUp, FiFileText, FiRefreshCw, FiTrash2 } from 'react-icons/fi';

import type { WorkspaceArtifactView, WorkspaceDirectoryView } from '@/spec/workspace';
import { WorkspaceDirectoryOrigin } from '@/spec/workspace';

import { workspaceManagementAPI } from '@/apis/baseapi';

import { ActionRow } from '@/components/managementui/action_row';
import { ManagementBundleCard } from '@/components/managementui/management_bundle_card';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';

interface WorkspaceDirectoryCardProps {
	directory: WorkspaceDirectoryView;
	onChanged: (directory: WorkspaceDirectoryView) => void;
	onRequestRemove: (directory: WorkspaceDirectoryView) => void;
	onShowDefaultPolicy: () => void;
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

function originLabel(origin: WorkspaceDirectoryOrigin): string {
	return origin === WorkspaceDirectoryOrigin.Default ? 'Default policy' : 'Repository manifest';
}

export function WorkspaceDirectoryCard({
	directory,
	onChanged,
	onRequestRemove,
	onShowDefaultPolicy,
}: WorkspaceDirectoryCardProps) {
	const [expanded, setExpanded] = useState(false);
	const [artifacts, setArtifacts] = useState<WorkspaceArtifactView[] | null>(null);
	const [artifactError, setArtifactError] = useState('');
	const [busy, setBusy] = useState('');
	const [actionError, setActionError] = useState('');

	const loadArtifacts = useCallback(async () => {
		setArtifactError('');

		try {
			setArtifacts(await workspaceManagementAPI.listWorkspaceDirectoryArtifacts(directory.ref));
		} catch (cause) {
			setArtifactError(cause instanceof Error ? cause.message : 'Discovered declarations could not be loaded.');
		}
	}, [directory.ref]);

	useEffect(() => {
		if (expanded && artifacts === null && !artifactError) {
			// oxlint-disable-next-line react/set-state-in-effect
			void loadArtifacts();
		}
	}, [artifactError, artifacts, expanded, loadArtifacts]);

	const mutateDirectory = async (key: string, action: () => Promise<WorkspaceDirectoryView>) => {
		setActionError('');
		setBusy(key);

		try {
			onChanged(await action());
			setArtifacts(null);
		} catch (cause) {
			setActionError(cause instanceof Error ? cause.message : 'Workspace directory operation failed.');
		} finally {
			setBusy('');
		}
	};

	const setArtifactEnabled = async (artifact: WorkspaceArtifactView, enabled: boolean) => {
		const key = `artifact:${artifact.artifact.artifactID}`;
		setActionError('');
		setBusy(key);

		try {
			const updated = await workspaceManagementAPI.setWorkspaceDirectoryArtifactEnabled(
				directory.ref,
				artifact.artifact,
				artifact.revision,
				enabled
			);

			setArtifacts(
				previous =>
					previous?.map(value => (value.artifact.artifactID === updated.artifact.artifactID ? updated : value)) ?? null
			);
			onChanged(await workspaceManagementAPI.getWorkspaceDirectory(directory.ref));
		} catch (cause) {
			setActionError(cause instanceof Error ? cause.message : 'Artifact enablement could not be changed.');
		} finally {
			setBusy('');
		}
	};

	const manifestCount = directory.workspaces.filter(value => value.origin === WorkspaceDirectoryOrigin.Manifest).length;
	const hasManifestDiagnostic = directory.diagnostics?.some(
		diagnostic => diagnostic.code === 'workspace.manifest-invalid'
	);

	return (
		<ManagementBundleCard
			title={directory.root.displayName}
			identity={<span className="font-mono text-xs">{directory.root.id}</span>}
			description="Repository-oriented Workspace directory"
			status={
				<>
					<StatusBadge tone={directory.enabled ? 'success' : 'warning'}>
						{directory.enabled ? 'Enabled' : 'Disabled'}
					</StatusBadge>
					<StatusBadge>
						{directory.workspaces.length} effective Workspace{directory.workspaces.length === 1 ? '' : 's'}
					</StatusBadge>
				</>
			}
			disclosure={
				<button
					type="button"
					className="btn btn-sm btn-ghost rounded-xl"
					onClick={() => {
						setExpanded(value => !value);
					}}
				>
					{expanded ? 'Hide details' : 'Manage'}
					{expanded ? <FiChevronUp size={15} /> : <FiChevronDown size={15} />}
				</button>
			}
			metadata={
				<>
					<MetadataPill label="Policy">{directory.policyID}</MetadataPill>
					<MetadataPill label="Policy version">{directory.policyVersion}</MetadataPill>
					<MetadataPill label="Directory Source">{directory.directorySource.displayName}</MetadataPill>
				</>
			}
			actions={
				<>
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						disabled={Boolean(busy)}
						onClick={() => {
							void mutateDirectory('refresh', () => workspaceManagementAPI.refreshWorkspaceDirectory(directory.ref));
						}}
					>
						<FiRefreshCw size={14} />
						{busy === 'refresh' ? 'Refreshing...' : 'Refresh'}
					</button>
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						disabled={Boolean(busy)}
						onClick={() => {
							void mutateDirectory('enabled', () =>
								workspaceManagementAPI.setWorkspaceDirectoryEnabled(
									directory.ref,
									directory.directorySource.revision,
									!directory.enabled
								)
							);
						}}
					>
						{directory.enabled ? 'Disable directory' : 'Enable directory'}
					</button>
					<button
						type="button"
						className="btn btn-sm btn-ghost text-error rounded-xl"
						disabled={Boolean(busy)}
						onClick={() => {
							onRequestRemove(directory);
						}}
					>
						<FiTrash2 size={14} />
						Remove
					</button>
				</>
			}
		>
			{actionError ? (
				<div className="alert alert-error mt-4 rounded-2xl text-sm">
					<FiAlertCircle size={14} />
					<span>{actionError}</span>
				</div>
			) : null}

			{expanded ? (
				<div className="mt-5 space-y-4">
					<div className="border-base-content/10 bg-base-100 rounded-2xl border p-4 text-sm">
						{directory.workspaces.length === 0 && hasManifestDiagnostic ? (
							<>
								<div className="font-semibold">Workspace manifest needs attention</div>
								<div className="text-base-content/70 mt-1 text-xs">
									A known Workspace manifest exists but did not produce an available Workspace declaration. Fix the
									manifest and refresh. The default policy intentionally remains inactive while a manifest is present.
								</div>
							</>
						) : directory.workspaces.length === 0 ? (
							<>
								<div className="font-semibold">No effective Workspace is currently available</div>
								<div className="text-base-content/70 mt-1 text-xs">
									Enable the directory and refresh it. If the repository contains a Workspace manifest, fix any
									declaration diagnostics shown below.
								</div>
							</>
						) : manifestCount === 0 ? (
							<>
								<div className="font-semibold">Default Workspace policy is active</div>
								<div className="text-base-content/70 mt-1 text-xs">
									This repository has no Workspace manifest. Copy the default policy into `workspace.yaml` when
									repository-specific composition is needed.
								</div>
								<button type="button" className="btn btn-sm btn-ghost mt-3 rounded-xl" onClick={onShowDefaultPolicy}>
									<FiFileText size={14} />
									View default workspace.yaml
								</button>
							</>
						) : manifestCount === 1 ? (
							<>
								<div className="font-semibold">Repository Workspace manifest is active</div>
								<div className="text-base-content/70 mt-1 text-xs">
									This directory has one effective Workspace declaration. Add additional `*.workspace.yaml`,
									`*.workspace.yml`, or `*.workspace.json` files to expose peer Workspace views.
								</div>
							</>
						) : (
							<>
								<div className="font-semibold">Multiple Workspace manifests are active</div>
								<div className="text-base-content/70 mt-1 text-xs">
									Each declaration is a peer Workspace. Conversations select one effective Workspace at a time;
									Workspaces do not import one another.
								</div>
							</>
						)}
					</div>

					<div className="space-y-3">
						<div className="text-sm font-semibold">Effective Workspace declarations</div>

						{directory.workspaces.map(entry => {
							const artifact = entry.workspace.artifact;

							return (
								<ManagementItemCard
									key={artifact.id}
									title={artifact.displayName || artifact.logicalName}
									description={entry.workspace.description}
									subtitle={
										<span className="font-mono text-xs">{entry.manifestLocator ?? artifact.binding.locator}</span>
									}
									status={
										<>
											<StatusBadge tone={entry.origin === WorkspaceDirectoryOrigin.Default ? 'info' : 'success'}>
												{originLabel(entry.origin)}
											</StatusBadge>
											<StatusBadge tone={artifact.enabled ? 'success' : 'warning'}>
												{artifact.enabled ? 'Enabled' : 'Disabled'}
											</StatusBadge>
										</>
									}
									metadata={
										<>
											<MetadataPill label="Declaration">{artifact.logicalName}</MetadataPill>
											<MetadataPill label="State">{artifact.state}</MetadataPill>
											<MetadataPill label="Revision">{artifact.revision}</MetadataPill>
										</>
									}
								/>
							);
						})}

						{directory.workspaces.length === 0 ? (
							<ManagementEmptyState>No effective Workspace declarations are available.</ManagementEmptyState>
						) : null}
					</div>

					<div className="space-y-3">
						<div className="flex items-center justify-between gap-3">
							<div>
								<div className="text-sm font-semibold">Discovered declarations</div>
								<div className="text-base-content/60 text-xs">
									Artifact enablement is the only individual Workspace runtime enablement control.
								</div>
							</div>
							<button
								type="button"
								className="btn btn-sm btn-ghost rounded-xl"
								onClick={() => {
									void loadArtifacts();
								}}
							>
								<FiRefreshCw size={14} />
								Reload declarations
							</button>
						</div>

						{artifactError ? <div className="alert alert-warning rounded-2xl text-sm">{artifactError}</div> : null}

						{artifacts?.map(artifact => (
							<ManagementItemCard
								key={artifact.artifact.artifactID}
								title={artifact.displayName || artifact.logicalName}
								subtitle={
									<span className="font-mono text-xs">
										{artifact.locator}
										{artifact.subresourceLocator ? ` / ${artifact.subresourceLocator}` : ''}
									</span>
								}
								status={
									<>
										<StatusBadge tone={artifact.enabled ? 'success' : 'warning'}>
											{artifact.enabled ? 'Enabled' : 'Disabled'}
										</StatusBadge>
										<StatusBadge>{artifact.state}</StatusBadge>
									</>
								}
								metadata={
									<>
										<MetadataPill label="Type">{artifact.kind}</MetadataPill>
										<MetadataPill label="Name">{artifact.logicalName}</MetadataPill>
										<MetadataPill label="Source">{artifact.sourceID}</MetadataPill>
									</>
								}
							>
								<ActionRow>
									<button
										type="button"
										className="btn btn-sm btn-ghost rounded-xl"
										disabled={busy === `artifact:${artifact.artifact.artifactID}`}
										onClick={() => {
											void setArtifactEnabled(artifact, !artifact.enabled);
										}}
									>
										{artifact.enabled ? 'Disable Artifact' : 'Enable Artifact'}
									</button>
									{artifact.kind === 'mcp' ? (
										<span className="text-base-content/60 text-xs">
											MCP secrets and installation inputs are managed in MCP management and are never exposed by
											Workspace APIs.
										</span>
									) : null}
								</ActionRow>
							</ManagementItemCard>
						))}

						{artifacts !== null && artifacts.length === 0 ? (
							<ManagementEmptyState>No Workspace declarations were discovered.</ManagementEmptyState>
						) : null}
					</div>

					{directory.diagnostics?.length ? (
						<div className="space-y-2">
							<div className="text-sm font-semibold">Directory diagnostics</div>
							{directory.diagnostics.map((diagnostic, index) => (
								<div
									key={`${diagnostic.code}:${diagnostic.message}:${index}`}
									className="border-base-content/10 rounded-2xl border p-3 text-sm"
								>
									<div className="flex flex-wrap gap-2">
										<StatusBadge tone={diagnosticTone(diagnostic.severity)}>{diagnostic.severity}</StatusBadge>
										<MetadataPill label="Code">{diagnostic.code}</MetadataPill>
									</div>
									<div className="mt-2">{diagnostic.message}</div>
									{diagnostic.location?.locator ? (
										<div className="text-base-content/60 mt-2 font-mono text-xs">{diagnostic.location.locator}</div>
									) : null}
								</div>
							))}
						</div>
					) : null}
				</div>
			) : null}
		</ManagementBundleCard>
	);
}
