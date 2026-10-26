import { useEffect, useMemo, useState } from 'react';
import { FiAlertCircle, FiDownload, FiFileText, FiServer } from 'react-icons/fi';

import type {
	AgentCapabilityOccurrence,
	AgentExportResult,
	AgentMCPSetupDescriptor,
	AgentResolution,
	AgentView,
} from '@/spec/agent';
import { CapabilityResolutionStatus } from '@/spec/artifact';

import { getErrorMessage } from '@/lib/error_utils';

import { agentArtifactRef, agentDisplayName } from '@/apis/agent_management';
import { agentManagementAPI, backendAPI } from '@/apis/baseapi';

import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import { getAgentRelationshipBadgeClass, textToBase64 } from '@/agents/agent_management_utils';
import { AgentMCPSetupModal } from '@/agents/agent_mcp_setup_modal';
import { AgentRecipePreview } from '@/agents/agent_recipe_preview';

interface AgentCapabilityControls {
	isExcluded: (occurrence: AgentCapabilityOccurrence) => boolean;
	onToggle: (occurrence: AgentCapabilityOccurrence) => void;
}

interface AgentDetailsModalProps {
	isOpen: boolean;
	agent: AgentView | null;
	onClose: () => void;
	capabilityControls?: AgentCapabilityControls;
}

interface AgentDetailsModalContentProps {
	agent: AgentView;
	onClose: () => void;
	capabilityControls?: AgentCapabilityControls;
}

function AgentDetailsModalContent({ agent, onClose, capabilityControls }: AgentDetailsModalContentProps) {
	const [resolution, setResolution] = useState<AgentResolution | null>(null);
	const [resolutionError, setResolutionError] = useState('');
	const [exportResult, setExportResult] = useState<AgentExportResult | null>(null);
	const [exportError, setExportError] = useState('');
	const [isSavingYAML, setIsSavingYAML] = useState(false);
	const [yamlSaveError, setYAMLSaveError] = useState('');
	const [setupDescriptor, setSetupDescriptor] = useState<AgentMCPSetupDescriptor | null>(null);

	const agentRef = useMemo(() => agentArtifactRef(agent), [agent]);
	const displayName = agentDisplayName(agent);
	const secondaryName = displayName === agent.name ? undefined : agent.name;

	useEffect(() => {
		let cancelled = false;

		const resolveFallback = async () => {
			if (cancelled) {
				return;
			}
			try {
				const value = await agentManagementAPI.resolveAgent(agentRef);
				if (!cancelled) {
					setResolution(value);
					setResolutionError('');
				}
			} catch (error) {
				if (!cancelled) {
					setResolutionError(getErrorMessage(error, 'Agent declarations could not be resolved.'));
				}
			}
		};

		void (async () => {
			try {
				const value = await agentManagementAPI.exportAgent(agentRef);
				if (!cancelled) {
					setExportResult(value);
				}
				if (cancelled) {
					return;
				}

				await resolveFallback();
			} catch (error) {
				if (!cancelled) {
					setExportError(getErrorMessage(error, 'Managed Agent YAML could not be loaded.'));
				}
				await resolveFallback();
			}
		})();

		return () => {
			cancelled = true;
		};
	}, [agent, agentRef]);

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

	const occurrences = resolution?.capabilities?.occurrences ?? [];
	const setupDescriptors = occurrences
		.filter(occurrence => occurrence.type === 'mcp')
		.map(occurrence => ({
			occurrencePath: occurrence.path,
			name: occurrence.name || 'MCP server',
			artifact: occurrence.target?.artifact,
		}));

	return (
		<>
			<div className="modal-box bg-base-200 flex max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-5xl flex-col overflow-hidden rounded-2xl p-0">
				<ModalHeader title={displayName} description={secondaryName} onClose={onClose} />

				<div className="app-scrollbar-thin min-h-0 flex-1 overflow-x-hidden overflow-y-auto overscroll-contain p-4 sm:p-6">
					<div className="space-y-5 pb-1">
						<ModalSection title="Agent metadata">
							<ManagementInfoGrid>
								<ManagementInfoRow label="Name" mono>
									{agent.name}
								</ManagementInfoRow>
								<ManagementInfoRow label="Managed">{agent.managed ? 'Yes' : 'No'}</ManagementInfoRow>
								<ManagementInfoRow label="Built-in">{agent.builtIn ? 'Yes' : 'No'}</ManagementInfoRow>
								<ManagementInfoRow label="Enabled">{agent.enabled ? 'Yes' : 'No'}</ManagementInfoRow>
								<ManagementInfoRow label="State">{agent.state}</ManagementInfoRow>
								<ManagementInfoRow label="Revision">{agent.revision}</ManagementInfoRow>
								<ManagementInfoRow label="Description">
									<span className="whitespace-pre-wrap">{agent.description || '—'}</span>
								</ManagementInfoRow>
							</ManagementInfoGrid>
						</ModalSection>

						<ModalSection
							title="Composer starter recipe"
							description="This frontend projection resolves mapped Models and Tools, Skill modes, and MCP runtime identities."
						>
							<AgentRecipePreview resolution={resolution} />
						</ModalSection>

						<ModalSection
							title="Agent YAML"
							description="Managed Agent declarations are immutable. Export, edit externally, then import under a new name or after deletion."
							actionsAlign="center"
							actions={
								exportResult ? (
									<button
										type="button"
										className="btn btn-sm btn-ghost inline-flex items-center justify-center gap-2 rounded-xl"
										disabled={isSavingYAML}
										onClick={() => {
											void saveYAML();
										}}
									>
										<FiDownload size={15} className="shrink-0" />
										<span>{isSavingYAML ? 'Saving...' : 'Export YAML'}</span>
									</button>
								) : undefined
							}
						>
							{yamlSaveError ? (
								<div className="alert alert-error rounded-xl text-sm">
									<FiAlertCircle size={16} />
									<span>{yamlSaveError}</span>
								</div>
							) : null}

							{!exportResult && !exportError ? (
								<div className="text-base-content/70 text-sm">
									{agent.managed ? 'Loading normalized YAML...' : 'Loading canonical Agent YAML...'}
								</div>
							) : null}

							{exportError ? (
								<div className="alert alert-warning rounded-xl text-sm">
									<FiAlertCircle size={16} />
									<span>{exportError}</span>
								</div>
							) : null}

							{exportResult ? (
								<pre className="app-scrollbar-thin bg-base-300 max-h-80 overflow-auto rounded-xl p-3 text-xs leading-5 whitespace-pre-wrap">
									{exportResult.content}
								</pre>
							) : null}
						</ModalSection>

						<ModalSection
							title="Capabilities"
							description="Resolved Agent declarations and their current availability."
						>
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

									{occurrences.map(occurrence => {
										const canToggleCapability =
											Boolean(capabilityControls) &&
											occurrence.status === CapabilityResolutionStatus.Available &&
											['tool', 'skill', 'text', 'mcp'].includes(occurrence.type);
										const isExcluded = canToggleCapability && Boolean(capabilityControls?.isExcluded(occurrence));

										return (
											<div key={occurrence.path} className="border-base-content/10 rounded-xl border p-3">
												<div className="flex flex-col items-center gap-3 sm:flex-row sm:justify-between">
													<div className="min-w-0">
														<div className="flex flex-wrap items-center gap-2">
															<span className="badge badge-ghost badge-sm shrink-0 capitalize">{occurrence.type}</span>
															<span className="min-w-0 font-medium break-words">
																{occurrence.name || `Unnamed ${occurrence.type}`}
															</span>
														</div>

														{occurrence.message ? (
															<div className="text-warning mt-2 text-xs leading-5">{occurrence.message}</div>
														) : null}
													</div>

													<div className="flex shrink-0 flex-wrap items-center gap-2 sm:justify-end">
														<span
															className={`badge badge-sm capitalize ${getAgentRelationshipBadgeClass(occurrence.status)}`}
														>
															{occurrence.status}
														</span>
														{occurrence.required ? (
															<span className="badge badge-outline badge-sm">Required</span>
														) : null}
														{isExcluded ? <span className="badge badge-warning badge-sm">Excluded</span> : null}

														{canToggleCapability ? (
															<button
																type="button"
																className="btn btn-ghost btn-xs inline-flex items-center justify-center rounded-lg"
																title={
																	isExcluded
																		? 'Restore this inherited Agent contribution for the next Run.'
																		: 'Exclude this inherited Agent contribution for the next Run.'
																}
																onClick={() => {
																	capabilityControls?.onToggle(occurrence);
																}}
															>
																{isExcluded ? 'Include' : 'Exclude'}
															</button>
														) : null}
													</div>
												</div>

												{occurrence.message ? (
													<div className="text-warning mt-3 text-xs leading-5">{occurrence.message}</div>
												) : null}
											</div>
										);
									})}

									{occurrences.length === 0 ? (
										<div className="text-base-content/70 text-sm">This Agent has no declared capabilities.</div>
									) : null}
								</div>
							) : null}
						</ModalSection>

						<ModalSection title="MCP setup">
							{!resolution && !resolutionError ? (
								<div className="text-base-content/70 text-sm">Loading MCP setup descriptors...</div>
							) : null}

							{resolutionError ? (
								<div className="text-base-content/70 text-sm">
									MCP setup is unavailable until this Agents declarations can be resolved.
								</div>
							) : null}

							{resolution && setupDescriptors.length === 0 ? (
								<div className="text-base-content/70 text-sm">This Agent does not declare any MCP servers.</div>
							) : null}

							{resolution && setupDescriptors.length > 0 ? (
								<div className="space-y-2">
									{setupDescriptors.map(descriptor => (
										<div
											key={descriptor.occurrencePath}
											className="border-base-content/10 bg-base-100/40 flex flex-col gap-3 rounded-xl border p-3 sm:flex-row sm:items-center sm:justify-between"
										>
											<div className="min-w-0">
												<div className="flex items-center gap-2 font-medium">
													<FiServer size={15} className="shrink-0" />
													<span className="truncate">{descriptor.name}</span>
												</div>
											</div>

											<button
												type="button"
												className="btn btn-sm btn-ghost inline-flex shrink-0 items-center justify-center gap-2 rounded-xl"
												onClick={() => {
													setSetupDescriptor(descriptor);
												}}
											>
												<FiFileText size={15} className="shrink-0" />
												<span>{descriptor.artifact ? 'Configure' : 'View setup status'}</span>
											</button>
										</div>
									))}
								</div>
							) : null}
						</ModalSection>
					</div>
				</div>

				<ModalActions>
					<button type="button" className="btn bg-base-300 rounded-xl" onClick={onClose}>
						Close
					</button>
				</ModalActions>
			</div>

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

export function AgentDetailsModal({ isOpen, agent, onClose, capabilityControls }: AgentDetailsModalProps) {
	if (!isOpen || !agent) {
		return null;
	}

	return (
		<ModalDialog isOpen={isOpen} onClose={onClose}>
			<AgentDetailsModalContent
				key={`${agent.ref.rootID}:${agent.ref.artifactID}:${agent.revision}`}
				agent={agent}
				onClose={onClose}
				capabilityControls={capabilityControls}
			/>
			<ModalBackdrop enabled={true} />
		</ModalDialog>
	);
}
