import { useEffect, useMemo, useState } from 'react';
import { FiAlertCircle } from 'react-icons/fi';

import type { AgentMCPSetupDescriptor, AgentResolution, AgentView } from '@/spec/agent';
import { CapabilityResolutionStatus, CapabilityTargetForm } from '@/spec/artifact';

import { getErrorMessage } from '@/lib/error_utils';

import { agentArtifactRef, agentDisplayName } from '@/apis/agent_management';
import { agentManagementAPI } from '@/apis/baseapi';

import { ManagementArtifactSource } from '@/components/managementui/management_artifact_source';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import {
	ManagementModalBody,
	ManagementModalDialog,
	ManagementModalPanel,
} from '@/components/managementui/management_modal_layout';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import type { AgentCapabilityControls } from '@/agents/agent_declarations';
import { loadAgentArtifactSource } from '@/agents/agent_artifact_source';
import { AgentDeclarations } from '@/agents/agent_declarations';
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

function AgentYAMLDisclosure({ agentRef, managed }: { agentRef: AgentView['ref']; managed: boolean }) {
	return (
		<ModalSection
			title="Agent YAML"
			description={
				managed
					? 'Managed Agent declarations are immutable. Export, edit externally, then import a replacement.'
					: 'Canonical Agent YAML is available for inspection.'
			}
		>
			<ManagementArtifactSource label="Agent YAML" load={() => loadAgentArtifactSource(agentRef)} />
		</ModalSection>
	);
}

function AgentDetailsModalContent({ agent, onClose, capabilityControls }: AgentDetailsModalContentProps) {
	const [resolution, setResolution] = useState<AgentResolution | null>(null);
	const [resolutionError, setResolutionError] = useState('');
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

		return () => {
			cancelled = true;
		};
	}, [agent, agentRef]);

	return (
		<>
			<ManagementModalPanel width="wide">
				<ModalHeader title={displayName} onClose={onClose} />

				<ManagementModalBody>
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

						<AgentYAMLDisclosure agentRef={agentRef} managed={agent.managed} />
					</div>
				</ManagementModalBody>

				<ModalActions>
					<button type="button" className="btn bg-base-300 rounded-xl" onClick={onClose}>
						Close
					</button>
				</ModalActions>
			</ManagementModalPanel>

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
		<ManagementModalDialog isOpen={isOpen} onClose={onClose}>
			<AgentDetailsModalContent
				key={`${agent.ref.rootID}:${agent.ref.artifactID}:${agent.revision}`}
				agent={agent}
				onClose={onClose}
				capabilityControls={capabilityControls}
			/>
		</ManagementModalDialog>
	);
}
