import { useEffect, useMemo, useState } from 'react';
import { FiAlertCircle, FiChevronDown, FiDownload } from 'react-icons/fi';

import type { AgentExportResult, AgentMCPSetupDescriptor, AgentResolution, AgentView } from '@/spec/agent';
import { CapabilityResolutionStatus, CapabilityTargetForm } from '@/spec/artifact';

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

import type { AgentCapabilityControls } from '@/agents/agent_declarations';
import { AgentDeclarations } from '@/agents/agent_declarations';
import { textToBase64 } from '@/agents/agent_management_utils';
import { AgentMCPSetupModal } from '@/agents/agent_mcp_setup_modal';
import { AgentRecipePreview } from '@/agents/agent_recipe_preview';

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

	useEffect(() => {
		let cancelled = false;

		void agentManagementAPI
			.resolveAgent(agentRef)
			.then(value => {
				if (!cancelled) {
					setResolution(value);
					setResolutionError('');
				}
			})
			.catch((error: unknown) => {
				if (!cancelled) {
					setResolutionError(getErrorMessage(error, 'Agent declarations could not be resolved.'));
				}
			});

		void agentManagementAPI
			.exportAgent(agentRef)
			.then(value => {
				if (!cancelled) {
					setExportResult(value);
					setExportError('');
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

	return (
		<>
			<div className="modal-box bg-base-200 flex max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-5xl flex-col overflow-hidden rounded-2xl p-0">
				<ModalHeader title={displayName} onClose={onClose} />

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
							title="Agent contributions"
							description="Declarations grouped by purpose, with their current resolution status. Final Run preparation resolves them again."
						>
							{resolutionError ? (
								<div className="alert alert-warning rounded-2xl text-sm" role="alert">
									<FiAlertCircle size={16} />
									<span>{resolutionError}</span>
								</div>
							) : null}
							{!resolution && !resolutionError ? (
								<div className="text-base-content/70 text-sm">Resolving Agent declarations...</div>
							) : null}
							{resolution ? (
								<div className="space-y-4">
									<AgentRecipePreview resolution={resolution} />
									<p className="text-base-content/70 text-xs">
										{resolution.capabilities.complete
											? 'All required declarations currently resolve.'
											: 'Some declarations are unavailable or ambiguous.'}
									</p>
									<AgentDeclarations
										occurrences={resolution.capabilities.occurrences}
										capabilityControls={capabilityControls}
										onSetup={occurrence => {
											setSetupDescriptor({
												occurrencePath: occurrence.path,
												name: occurrence.name || 'MCP server',
												artifact:
													occurrence.status === CapabilityResolutionStatus.Available &&
													occurrence.target?.form === CapabilityTargetForm.Artifact
														? occurrence.target.artifact
														: undefined,
											});
										}}
									/>
								</div>
							) : null}
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
								<details className="border-base-content/10 group overflow-hidden rounded-xl border">
									<summary className="bg-base-100/40 flex cursor-pointer list-none items-center justify-between gap-3 p-3 text-sm font-medium [&::-webkit-details-marker]:hidden">
										<span>Source YAML</span>
										<FiChevronDown
											size={16}
											className="shrink-0 transition-transform group-open:rotate-180"
											aria-hidden="true"
										/>
									</summary>
									<div className="border-base-content/10 border-t p-3">
										<pre className="app-scrollbar-thin bg-base-300 max-h-80 overflow-auto rounded-xl p-3 text-xs/5 wrap-break-word whitespace-pre-wrap">
											{exportResult.content}
										</pre>
									</div>
								</details>
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
