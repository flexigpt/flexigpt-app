import type { ReactNode } from 'react';
import { useCallback } from 'react';

import { FiAlertCircle } from 'react-icons/fi';

import type {
	MCPAuthHealth,
	MCPPromptRef,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPServerRuntimeSnapshot,
	MCPToolCapability,
} from '@/spec/mcp_artifact';
import { MCPToolRisk } from '@/spec/mcp_artifact';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { mcpAPI } from '@/apis/baseapi';

import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';
import { ModalSection } from '@/components/modal/modal_section';

import type { MCPBundleView, MCPServerView } from '@/mcpservers/lib/mcp_management';
import { serverRefLabel } from '@/mcpservers/lib/mcp_management';
import {
	getEffectiveMCPServerStatus,
	getMCPApprovalRuleLabel,
	getMCPExecutionModeLabel,
	getMCPServerAuthHealthBadgeClass,
	getMCPServerAuthHealthLabel,
	getMCPStatusBadgeClass,
	getMCPStatusLabel,
	getMCPToolRiskLabel,
} from '@/mcpservers/lib/mcp_server_utils';

interface MCPServerDetailsModalProps {
	isOpen: boolean;
	onClose: () => void;
	bundle: MCPBundleView | null;
	server: MCPServerView | null;
	runtime?: MCPServerRuntimeSnapshot;
	authHealth?: MCPAuthHealth;
}

interface DiscoveryData {
	tools: MCPToolCapability[];
	resources: MCPResourceRef[];
	resourceTemplates: MCPResourceTemplateRef[];
	prompts: MCPPromptRef[];
	error?: string;
}

function JSONBlock({ value }: { value: unknown }) {
	if (value === undefined || value === null) {
		return <span>—</span>;
	}

	return (
		<pre className="bg-base-300 max-h-72 overflow-auto rounded-2xl p-3 text-xs whitespace-pre-wrap">
			{JSON.stringify(value, null, 2)}
		</pre>
	);
}

function Field({ label, children }: { label: string; children: ReactNode }) {
	return <ManagementInfoRow label={label}>{children}</ManagementInfoRow>;
}

function ArgumentSummary({
	arguments: values,
}: {
	arguments?: Record<string, { required?: boolean; description?: string }>;
}) {
	if (!values || Object.keys(values).length === 0) {
		return <span>—</span>;
	}

	return (
		<div className="flex flex-wrap gap-1">
			{Object.entries(values).map(([name, definition]) => (
				<MetadataPill key={name} title={definition.description}>
					{name}
					{definition.required ? '*' : ''}
				</MetadataPill>
			))}
		</div>
	);
}

function MCPServerDetailsModalContent({
	onClose,
	bundle,
	server,
	runtime,
	authHealth,
}: {
	onClose: () => void;
	bundle: MCPBundleView;
	server: MCPServerView;
	runtime?: MCPServerRuntimeSnapshot;
	authHealth?: MCPAuthHealth;
}) {
	const hasDiscoverySnapshot = Boolean(runtime?.snapshotDigest);

	const loadDiscovery = useCallback(
		async (_signal: AbortSignal): Promise<DiscoveryData> => {
			if (!hasDiscoverySnapshot) {
				return {
					tools: [],
					resources: [],
					resourceTemplates: [],
					prompts: [],
				};
			}

			const results = await Promise.allSettled([
				mcpAPI.listMCPServerTools(server.ref),
				mcpAPI.listMCPServerResources(server.ref),
				mcpAPI.listMCPServerResourceTemplates(server.ref),
				mcpAPI.listMCPServerPrompts(server.ref),
			]);

			const [tools, resources, templates, prompts] = results;
			const failure = results.find((result): result is PromiseRejectedResult => result.status === 'rejected');

			return {
				tools: tools.status === 'fulfilled' ? tools.value : [],
				resources: resources.status === 'fulfilled' ? resources.value : [],
				resourceTemplates: templates.status === 'fulfilled' ? templates.value : [],
				prompts: prompts.status === 'fulfilled' ? prompts.value : [],
				error:
					failure?.reason instanceof Error
						? failure.reason.message
						: failure
							? 'Some discovery data could not be loaded.'
							: undefined,
			};
		},
		[hasDiscoverySnapshot, server.ref]
	);

	const { data, isLoading } = useAsyncResource(loadDiscovery, {
		initialData: {
			tools: [],
			resources: [],
			resourceTemplates: [],
			prompts: [],
		},
	});

	const status = getEffectiveMCPServerStatus(server, runtime?.status);

	return (
		<ManagementDetailsModal
			isOpen={true}
			onClose={onClose}
			title="MCP Server Details"
			description={`${server.displayName} in ${bundle.displayName}`}
			modalKey={`mcp-server:${server.ref.rootID}:${server.ref.artifactID}:${server.artifact.revision}`}
			width="wide"
			height="tall"
		>
			{server.loadError ? (
				<div className="alert alert-error rounded-2xl text-sm">
					<FiAlertCircle size={14} />
					<span>{server.loadError}</span>
				</div>
			) : null}

			<ModalSection title="Artifact and installation">
				<ManagementInfoGrid>
					<Field label="Display Name">{server.displayName}</Field>
					<Field label="Logical Name">{server.logicalName}</Field>
					<Field label="Artifact Ref">
						<span className="font-mono text-xs">{serverRefLabel(server)}</span>
					</Field>
					<Field label="Artifact Revision">{server.artifact.revision}</Field>
					<Field label="Built-in">{server.builtIn ? 'Yes' : 'No'}</Field>
					<Field label="Runtime Enabled">{server.runtimeEnabled ? 'Yes' : 'No'}</Field>
					<Field label="Created">{server.artifact.createdAt.toLocaleString()}</Field>
					<Field label="Modified">{server.artifact.modifiedAt.toLocaleString()}</Field>

					<Field label="Runtime">
						<span className={`badge rounded-xl ${getMCPStatusBadgeClass(status)}`}>{getMCPStatusLabel(status)}</span>
						{runtime?.lastError ? <div className="text-error mt-1 text-xs">{runtime.lastError}</div> : null}
					</Field>

					<Field label="Authorization">
						<span className={`badge rounded-xl ${getMCPServerAuthHealthBadgeClass(server, authHealth)}`}>
							{getMCPServerAuthHealthLabel(server, authHealth)}
						</span>
						{authHealth?.lastError ? <div className="text-error mt-1 text-xs">{authHealth.lastError}</div> : null}
					</Field>

					<Field label="Runtime Config">
						<JSONBlock
							value={{
								core: server.document?.mcpServer,
								auth: server.document?.extension.auth,
								install: server.document?.extension.install,
								connectionProfiles: server.document?.extension.connectionProfiles,
							}}
						/>
					</Field>

					<Field label="Installation Data">
						<JSONBlock
							value={{
								schemaVersion: server.installation?.schemaVersion,
								selectedConnectionProfile: server.installation?.selectedConnectionProfile,
								inputs: Object.fromEntries(
									Object.entries(server.installation?.inputs ?? {}).map(([name, binding]) => [
										name,
										binding.secretRef ? { secretRef: '[configured]' } : binding,
									])
								),
								additionalPolicies: server.installation?.additionalPolicies,
							}}
						/>
					</Field>

					<Field label="Effective Policy">
						<JSONBlock value={server.policy} />
					</Field>

					<Field label="Server Info">
						<JSONBlock value={runtime?.serverInfo} />
					</Field>

					<Field label="Capabilities">
						<JSONBlock value={runtime?.serverCapabilities} />
					</Field>

					<Field label="Instructions">
						<div className="whitespace-pre-wrap">{runtime?.instructions || '—'}</div>
					</Field>
				</ManagementInfoGrid>
			</ModalSection>

			<ModalSection title="Discovery" description="Current process-local discovery snapshot.">
				{isLoading ? <div className="text-center text-sm">Loading discovery cache…</div> : null}

				{data.error ? (
					<div className="alert alert-warning mb-4 rounded-2xl text-sm">
						<FiAlertCircle size={14} />
						<span>{data.error}</span>
					</div>
				) : null}

				<div className="space-y-6">
					<div>
						<h4 className="mb-2 text-sm font-semibold">Tools ({data.tools.length})</h4>
						<div className="space-y-3">
							{data.tools.map(tool => (
								<ManagementItemCard
									key={`${tool.server.artifactID}:${tool.toolName}:${tool.digest}`}
									title={tool.displayName || tool.title || tool.toolName}
									subtitle={tool.toolName}
									description={tool.description}
									status={
										<>
											<StatusBadge tone={tool.enabled ? 'success' : 'neutral'}>
												{tool.enabled ? 'Enabled' : 'Disabled'}
											</StatusBadge>
											<StatusBadge
												tone={
													tool.inferredRisk === MCPToolRisk.Read
														? 'success'
														: tool.inferredRisk === MCPToolRisk.Write
															? 'warning'
															: 'error'
												}
											>
												{getMCPToolRiskLabel(tool.inferredRisk)}
											</StatusBadge>
										</>
									}
									metadata={
										<>
											<MetadataPill label="Approval">{getMCPApprovalRuleLabel(tool.approvalRule)}</MetadataPill>
											<MetadataPill label="Execution">{getMCPExecutionModeLabel(tool.executionMode)}</MetadataPill>
											{tool.app?.resourceUri ? <MetadataPill label="App">Configured</MetadataPill> : null}
										</>
									}
								/>
							))}
							{data.tools.length === 0 ? (
								<div className="border-base-content/10 rounded-2xl border py-6 text-center text-sm">
									No tools discovered.
								</div>
							) : null}
						</div>
					</div>

					<div>
						<h4 className="mb-2 text-sm font-semibold">Resources ({data.resources.length})</h4>
						<div className="space-y-3">
							{data.resources.map(resource => (
								<ManagementItemCard
									key={`${resource.server.artifactID}:${resource.uri}:${resource.digest ?? ''}`}
									title={resource.displayName || resource.name || resource.uri}
									subtitle={resource.uri}
									description={resource.description}
									metadata={
										<>
											<MetadataPill label="MIME">{resource.mimeType || 'Unknown'}</MetadataPill>
											{resource.size !== undefined ? <MetadataPill label="Size">{resource.size}</MetadataPill> : null}
										</>
									}
								/>
							))}
							{data.resources.length === 0 ? (
								<div className="border-base-content/10 rounded-2xl border py-6 text-center text-sm">
									No resources discovered.
								</div>
							) : null}
						</div>
					</div>

					<div>
						<h4 className="mb-2 text-sm font-semibold">Resource Templates ({data.resourceTemplates.length})</h4>
						<div className="space-y-3">
							{data.resourceTemplates.map(template => (
								<ManagementItemCard
									key={`${template.server.artifactID}:${template.uriTemplate}:${template.digest ?? ''}`}
									title={template.displayName || template.name || template.uriTemplate}
									subtitle={template.uriTemplate}
									description={template.description}
									metadata={
										<>
											<MetadataPill label="MIME">{template.mimeType || 'Unknown'}</MetadataPill>
											<ArgumentSummary arguments={template.arguments} />
										</>
									}
								/>
							))}
							{data.resourceTemplates.length === 0 ? (
								<div className="border-base-content/10 rounded-2xl border py-6 text-center text-sm">
									No resource templates discovered.
								</div>
							) : null}
						</div>
					</div>

					<div>
						<h4 className="mb-2 text-sm font-semibold">Prompts ({data.prompts.length})</h4>
						<div className="space-y-3">
							{data.prompts.map(prompt => (
								<ManagementItemCard
									key={`${prompt.server.artifactID}:${prompt.promptName}:${prompt.digest ?? ''}`}
									title={prompt.displayName || prompt.promptName}
									subtitle={prompt.promptName}
									description={prompt.description}
									metadata={<ArgumentSummary arguments={prompt.arguments} />}
								/>
							))}
							{data.prompts.length === 0 ? (
								<div className="border-base-content/10 rounded-2xl border py-6 text-center text-sm">
									No prompts discovered.
								</div>
							) : null}
						</div>
					</div>
				</div>
			</ModalSection>
		</ManagementDetailsModal>
	);
}

export function MCPServerDetailsModal(props: MCPServerDetailsModalProps) {
	if (!props.isOpen || !props.bundle || !props.server) {
		return null;
	}

	return (
		<MCPServerDetailsModalContent
			key={`${props.server.ref.rootID}:${props.server.ref.artifactID}:${props.server.artifact.revision}`}
			onClose={props.onClose}
			bundle={props.bundle}
			server={props.server}
			runtime={props.runtime}
			authHealth={props.authHealth}
		/>
	);
}
