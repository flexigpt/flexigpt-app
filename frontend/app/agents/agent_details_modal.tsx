import { useEffect, useState } from 'react';
import { FiAlertCircle, FiDownload, FiFileText, FiServer } from 'react-icons/fi';

import type { AgentExportResult, AgentMCPSetupDescriptor, AgentResolution, AgentView } from '@/spec/agent';
import { AgentImportRelationshipStatus } from '@/spec/agent';

import { agentStoreAPI, backendAPI } from '@/apis/baseapi';

import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import { AgentMCPSetupModal } from '@/agents/agent_mcp_setup_modal';
import { AgentRecipePreview } from '@/agents/agent_recipe_preview';
import {
	agentArtifactRef,
	agentDisplayName,
	formatArtifactRef,
	formatDateish,
	getErrorMessage,
	textToBase64,
} from '@/agents/lib/agent_management';

interface AgentDetailsModalProps {
	isOpen: boolean;
	agent: AgentView | null;
	onClose: () => void;
}

function getOccurrenceBadgeClass(status: string): string {
	switch (status) {
		case AgentImportRelationshipStatus.Available.toString():
			return 'badge-success';
		case AgentImportRelationshipStatus.Ambiguous.toString():
			return 'badge-warning';
		default:
			return 'badge-error';
	}
}

export function AgentDetailsModal({ isOpen, agent, onClose }: AgentDetailsModalProps) {
	const [resolution, setResolution] = useState<AgentResolution | null>(null);
	const [resolutionError, setResolutionError] = useState('');
	const [exportResult, setExportResult] = useState<AgentExportResult | null>(null);
	const [exportError, setExportError] = useState('');
	const [isSavingYAML, setIsSavingYAML] = useState(false);
	const [yamlSaveError, setYAMLSaveError] = useState('');
	const [setupDescriptor, setSetupDescriptor] = useState<AgentMCPSetupDescriptor | null>(null);

	const agentRef = agent ? agentArtifactRef(agent) : null;

	useEffect(() => {
		if (!isOpen || !agent || !agentRef) {
			return;
		}

		let cancelled = false;

		// oxlint-disable-next-line react/set-state-in-effect react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setResolution(null);
		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setResolutionError('');
		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setSetupDescriptor(null);

		void agentStoreAPI
			.resolveAgent(agentRef)
			.then(value => {
				if (!cancelled) {
					setResolution(value);
				}
			})
			.catch((error: unknown) => {
				if (!cancelled) {
					setResolutionError(getErrorMessage(error, 'Agent declarations could not be resolved.'));
				}
			});

		return () => {
			cancelled = true;
		};
	}, [agent, agentRef, isOpen]);

	useEffect(() => {
		if (!isOpen || !agent || !agentRef) {
			return;
		}

		let cancelled = false;

		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change react/set-state-in-effect
		setExportResult(null);
		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setExportError('');
		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setYAMLSaveError('');

		void agentStoreAPI
			.exportAgent(agentRef)
			.then(value => {
				if (!cancelled) {
					setExportResult(value);
				}
			})
			.catch((error: unknown) => {
				if (!cancelled) {
					setExportError(getErrorMessage(error, 'Managed Agent YAML could not be loaded.'));
				}
			});

		return () => {
			cancelled = true;
		};
	}, [agent, agentRef, isOpen]);

	if (!isOpen || !agent) {
		return null;
	}

	const saveYAML = async () => {
		if (!exportResult || isSavingYAML) {
			return;
		}

		setYAMLSaveError('');
		setIsSavingYAML(true);

		try {
			await backendAPI.saveFile(exportResult.suggestedFileName, textToBase64(exportResult.content), [
				{
					DisplayName: 'YAML',
					Extensions: ['yaml', 'yml'],
				},
			]);
		} catch (error) {
			setYAMLSaveError(getErrorMessage(error, 'Failed to export Agent YAML.'));
		} finally {
			setIsSavingYAML(false);
		}
	};

	const setupDescriptors = exportResult?.mcpSetupDescriptors ?? [];

	return (
		<>
			<ModalDialog isOpen={isOpen} onClose={onClose}>
				<div className="modal-box bg-base-200 max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-5xl overflow-hidden rounded-2xl p-0">
					<div className="app-scrollbar-thin max-h-[calc(100dvh-1rem)] overflow-y-auto p-4 sm:p-6">
						<ModalHeader
							title={agentDisplayName(agent)}
							description={`${agent.name} · ${formatArtifactRef(agentArtifactRef(agent))}`}
							onClose={onClose}
						/>

						<div className="space-y-5">
							<ModalSection title="Agent metadata">
								<ManagementInfoGrid>
									<ManagementInfoRow label="Name" mono>
										{agent.name}
									</ManagementInfoRow>
									<ManagementInfoRow label="Managed">{agent.managed ? 'Yes' : 'No'}</ManagementInfoRow>
									<ManagementInfoRow label="Built-in">{agent.builtIn ? 'Yes' : 'No'}</ManagementInfoRow>
									<ManagementInfoRow label="Enabled">{agent.artifact.enabled ? 'Yes' : 'No'}</ManagementInfoRow>
									<ManagementInfoRow label="State">{agent.artifact.state}</ManagementInfoRow>
									<ManagementInfoRow label="Created">{formatDateish(agent.artifact.createdAt)}</ManagementInfoRow>
									<ManagementInfoRow label="Modified">{formatDateish(agent.artifact.modifiedAt)}</ManagementInfoRow>
									<ManagementInfoRow label="Description">
										<span className="whitespace-pre-wrap">{agent.description || '—'}</span>
									</ManagementInfoRow>
								</ManagementInfoGrid>
							</ModalSection>

							<ModalSection
								title="Composer starter recipe"
								description="This frontend projection resolves mapped Models and Tools, Skill modes, and MCP runtime identities."
							>
								<AgentRecipePreview agent={agent} />
							</ModalSection>

							<ModalSection
								title="Agent YAML"
								description="Managed Agent declarations are immutable. Export, edit externally, then import under a new name or after deletion."
							>
								{yamlSaveError ? (
									<div className="alert alert-error mb-3 rounded-2xl text-sm">
										<FiAlertCircle size={16} />
										<span>{yamlSaveError}</span>
									</div>
								) : null}

								{agent.managed && !exportResult && !exportError ? (
									<div className="text-base-content/70 text-sm">Loading normalized YAML...</div>
								) : null}

								{exportError ? (
									<div className="alert alert-warning rounded-2xl text-sm">
										<FiAlertCircle size={16} />
										<span>{exportError}</span>
									</div>
								) : null}

								{exportResult ? (
									<div className="space-y-3">
										<div className="flex flex-wrap items-center justify-between gap-2">
											<div className="text-base-content/70 text-xs">
												Definition: <span className="font-mono">{exportResult.definitionDigest}</span>
											</div>
											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												disabled={isSavingYAML}
												onClick={() => {
													void saveYAML();
												}}
											>
												<FiDownload size={15} />
												<span>{isSavingYAML ? 'Saving...' : 'Export YAML'}</span>
											</button>
										</div>

										<pre className="bg-base-300 max-h-96 overflow-auto rounded-2xl p-3 text-xs whitespace-pre-wrap">
											{exportResult.content}
										</pre>

										{exportResult.resolutionIssue ? (
											<div className="alert alert-warning rounded-2xl text-sm">
												<FiAlertCircle size={16} />
												<span>{exportResult.resolutionIssue.message}</span>
											</div>
										) : null}
									</div>
								) : null}

								{!exportResult && !exportError ? (
									<div className="text-base-content/70 text-sm">Loading canonical Agent YAML...</div>
								) : null}
							</ModalSection>

							<ModalSection title="Resolved declarations">
								{resolutionError ? (
									<div className="alert alert-warning rounded-2xl text-sm">
										<FiAlertCircle size={16} />
										<span>{resolutionError}</span>
									</div>
								) : null}

								{!resolution && !resolutionError ? (
									<div className="text-base-content/70 text-sm">Resolving Agent declarations...</div>
								) : null}

								{resolution ? (
									<div className="space-y-3">
										<div className="text-base-content/70 text-xs">
											{resolution.capabilities.complete
												? 'All required Agent declarations currently resolve.'
												: 'Some Agent declarations are unavailable or ambiguous.'}
										</div>

										{resolution.capabilities.occurrences.map(occurrence => (
											<div key={occurrence.path} className="border-base-content/10 rounded-2xl border p-3">
												<div className="flex flex-wrap items-center gap-2">
													<span className="font-medium">
														{occurrence.type}
														{occurrence.name ? `: ${occurrence.name}` : ''}
													</span>
													<span className={`badge badge-sm ${getOccurrenceBadgeClass(occurrence.status)}`}>
														{occurrence.status}
													</span>
													{occurrence.required ? <span className="badge badge-outline badge-sm">Required</span> : null}
												</div>

												<div className="text-base-content/70 mt-1 font-mono text-xs">{occurrence.path}</div>

												{occurrence.scope ? (
													<div className="text-base-content/70 mt-1 text-xs">Scope: {occurrence.scope}</div>
												) : null}

												{occurrence.artifact ? (
													<div className="text-base-content/70 mt-1 text-xs">
														Artifact: {formatArtifactRef(occurrence.artifact)}
													</div>
												) : null}

												{occurrence.mapped ? (
													<div className="text-base-content/70 mt-1 text-xs">
														Mapped target: {occurrence.mapped.provider}/{occurrence.mapped.identifier}
													</div>
												) : null}

												{occurrence.code || occurrence.message ? (
													<div className="text-warning mt-2 text-xs">
														{[occurrence.code, occurrence.message].filter(Boolean).join(': ')}
													</div>
												) : null}
											</div>
										))}

										{resolution.capabilities.occurrences.length === 0 ? (
											<div className="text-base-content/70 text-sm">This Agent has no declared capabilities.</div>
										) : null}
									</div>
								) : null}
							</ModalSection>

							<ModalSection title="MCP setup">
								{setupDescriptors.length === 0 ? (
									<div className="text-base-content/70 text-sm">
										No managed MCP setup descriptors are available for this Agent.
									</div>
								) : (
									<div className="space-y-3">
										{setupDescriptors.map(descriptor => (
											<div
												key={descriptor.occurrencePath}
												className="border-base-content/10 flex flex-col gap-3 rounded-2xl border p-3 sm:flex-row sm:items-center sm:justify-between"
											>
												<div className="min-w-0">
													<div className="flex items-center gap-2 font-medium">
														<FiServer size={15} />
														<span>{descriptor.name}</span>
													</div>
													<div className="text-base-content/70 mt-1 font-mono text-xs">{descriptor.occurrencePath}</div>
													<div className="text-base-content/70 mt-1 text-xs">
														{descriptor.transport || 'named MCP relationship'}
														{descriptor.authMode ? ` · ${descriptor.authMode}` : ''}
														{descriptor.inputs?.length ? ` · ${descriptor.inputs.length} input(s)` : ''}
													</div>
												</div>

												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													onClick={() => {
														setSetupDescriptor(descriptor);
													}}
												>
													<FiFileText size={15} />
													<span>{descriptor.artifact ? 'Configure' : 'View setup status'}</span>
												</button>
											</div>
										))}
									</div>
								)}
							</ModalSection>
						</div>

						<ModalActions>
							<button type="button" className="btn bg-base-300 rounded-xl" onClick={onClose}>
								Close
							</button>
						</ModalActions>
					</div>
				</div>
			</ModalDialog>

			<AgentMCPSetupModal
				isOpen={setupDescriptor !== null}
				descriptor={setupDescriptor}
				onClose={() => {
					setSetupDescriptor(null);
				}}
			/>
		</>
	);
}
