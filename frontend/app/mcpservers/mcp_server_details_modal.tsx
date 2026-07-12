import type { ReactNode } from 'react';
import { useCallback } from 'react';

import { FiAlertCircle } from 'react-icons/fi';

import type {
	MCPAuthHealth,
	MCPBundle,
	MCPPromptRef,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPServerConfig,
	MCPServerRuntimeSnapshot,
	MCPToolCapability,
} from '@/spec/mcp';
import { MCPToolRisk } from '@/spec/mcp';

import { redactSensitiveHTTPHeaders } from '@/lib/http_input_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import {
	getAllMCPServerPrompts,
	getAllMCPServerResources,
	getAllMCPServerResourceTemplates,
	getAllMCPServerTools,
} from '@/apis/list_helper';

import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';
import { ModalSection } from '@/components/modal/modal_section';

import {
	getEffectiveMCPServerStatus,
	getMCPApprovalRuleLabel,
	getMCPExecutionModeLabel,
	getMCPServerAuthHealthBadgeClass,
	getMCPServerAuthHealthLabel,
	getMCPStatusBadgeClass,
	getMCPStatusLabel,
	getMCPToolRiskLabel,
	getMCPTransportLabel,
} from '@/mcpservers/lib/mcp_server_utils';

interface MCPServerDetailsModalProps {
	isOpen: boolean;
	onClose: () => void;
	bundle: MCPBundle | null;
	server: MCPServerConfig | null;
	runtime?: MCPServerRuntimeSnapshot;
	authHealth?: MCPAuthHealth;
}

interface DiscoveryData {
	tools: MCPToolCapability[];
	resources: MCPResourceRef[];
	resourceTemplates: MCPResourceTemplateRef[];
	prompts: MCPPromptRef[];
}

function JSONBlock({ value }: { value: unknown }) {
	if (value === undefined || value === null) {
		return <span>-</span>;
	}

	return (
		<pre className="bg-base-300 max-h-60 overflow-auto rounded-2xl p-3 text-xs whitespace-pre-wrap">
			{JSON.stringify(value, null, 2)}
		</pre>
	);
}

function Field({ label, children }: { label: string; children: ReactNode }) {
	return <ManagementInfoRow label={label}>{children}</ManagementInfoRow>;
}

function ArgumentSummary({ args }: { args?: Record<string, { required?: boolean; description?: string } | string> }) {
	if (!args || Object.keys(args).length === 0) {
		return <span>-</span>;
	}

	return (
		<div className="flex flex-wrap gap-1">
			{Object.entries(args).map(([name, def]) => {
				const required = typeof def === 'object' && Boolean(def.required);
				const description = typeof def === 'object' ? def.description : def;
				return (
					<MetadataPill key={name} title={description}>
						{name}
						{required ? '*' : ''}
					</MetadataPill>
				);
			})}
		</div>
	);
}

function getEmptyDiscoveryData(): DiscoveryData {
	return {
		tools: [],
		resources: [],
		resourceTemplates: [],
		prompts: [],
	};
}

interface MCPServerDetailsModalContentProps {
	onClose: () => void;
	bundle: MCPBundle;
	server: MCPServerConfig;
	runtime?: MCPServerRuntimeSnapshot;
	authHealth?: MCPAuthHealth;
}

function MCPServerDetailsModalContent({
	onClose,
	bundle,
	server,
	runtime,
	authHealth,
}: MCPServerDetailsModalContentProps) {
	const loadDiscovery = useCallback(
		async (_signal: AbortSignal) => {
			const results = await Promise.allSettled([
				getAllMCPServerTools(bundle.id, server.id),
				getAllMCPServerResources(bundle.id, server.id),
				getAllMCPServerResourceTemplates(bundle.id, server.id),
				getAllMCPServerPrompts(bundle.id, server.id),
			]);

			const [toolsResult, resourcesResult, resourceTemplatesResult, promptsResult] = results;
			const discoveryError = results
				.filter((result): result is PromiseRejectedResult => result.status === 'rejected')
				.map(result => (result.reason instanceof Error ? result.reason.message : 'Failed to load discovery section.'))
				.find(message => message.length > 0);

			return {
				discovery: {
					tools: toolsResult.status === 'fulfilled' ? toolsResult.value : [],
					resources: resourcesResult.status === 'fulfilled' ? resourcesResult.value : [],
					resourceTemplates: resourceTemplatesResult.status === 'fulfilled' ? resourceTemplatesResult.value : [],
					prompts: promptsResult.status === 'fulfilled' ? promptsResult.value : [],
				} satisfies DiscoveryData,
				discoveryError: discoveryError ?? '',
			};
		},
		[bundle.id, server.id]
	);

	const { data: discoveryResult, isLoading: loadingDiscovery } = useAsyncResource(loadDiscovery, {
		initialData: {
			discovery: getEmptyDiscoveryData(),
			discoveryError: '',
		},
	});

	const { discovery, discoveryError } = discoveryResult;

	const effectiveRuntimeStatus = getEffectiveMCPServerStatus(server.enabled, bundle.isEnabled, runtime);

	return (
		<ManagementDetailsModal
			isOpen={true}
			onClose={onClose}
			title="MCP Server Details"
			description={`${server.displayName} in ${bundle.displayName || bundle.slug}`}
			modalKey={`mcp-server:${bundle.id}:${server.id}:${server.modifiedAt}`}
			width="wide"
			height="tall"
		>
			<ModalSection title="Metadata" description="Identity, runtime state, authorization, and effective server policy.">
				<ManagementInfoGrid>
					<Field label="Display Name">{server.displayName}</Field>
					<Field label="Server ID">{server.id}</Field>
					<Field label="Bundle">{bundle.displayName || bundle.slug}</Field>
					<Field label="Transport">{getMCPTransportLabel(server.transport)}</Field>
					<Field label="Enabled">{server.enabled ? 'Yes' : 'No'}</Field>
					<Field label="Built-in">{server.isBuiltIn ? 'Yes' : 'No'}</Field>
					<Field label="Created">{server.createdAt}</Field>
					<Field label="Modified">{server.modifiedAt}</Field>

					<Field label="Runtime">
						<span className={`badge rounded-xl ${getMCPStatusBadgeClass(effectiveRuntimeStatus)}`}>
							{getMCPStatusLabel(effectiveRuntimeStatus)}
						</span>
						{runtime?.lastError && <div className="text-error mt-1 text-xs">{runtime.lastError}</div>}
					</Field>

					<Field label="Auth">
						<span className={`badge rounded-xl ${getMCPServerAuthHealthBadgeClass(server, authHealth)}`}>
							{getMCPServerAuthHealthLabel(server, authHealth)}
						</span>
						{authHealth?.lastError && <div className="text-error mt-1 text-xs">{authHealth.lastError}</div>}
					</Field>

					<Field label="Server Info">
						<JSONBlock value={runtime?.serverInfo} />
					</Field>

					<Field label="Capabilities">
						<JSONBlock value={runtime?.serverCapabilities} />
					</Field>

					<Field label="Instructions">
						<div className="whitespace-pre-wrap">{runtime?.instructions || '-'}</div>
					</Field>

					<Field label="Config">
						<JSONBlock
							value={{
								transport: server.transport,
								stdio: server.stdio
									? {
											...server.stdio,
											secretEnvRefs: server.stdio.secretEnvRefs
												? Object.fromEntries(Object.keys(server.stdio.secretEnvRefs).map(key => [key, '[configured]']))
												: undefined,
										}
									: undefined,
								streamableHttp: server.streamableHttp
									? {
											...server.streamableHttp,
											headers: redactSensitiveHTTPHeaders(server.streamableHttp.headers),
											clientCredentialRef: server.streamableHttp.clientCredentialRef ? '[configured]' : undefined,
											secretHeaderRefs: server.streamableHttp.secretHeaderRefs
												? Object.fromEntries(
														Object.keys(server.streamableHttp.secretHeaderRefs).map(key => [key, '[configured]'])
													)
												: undefined,
										}
									: undefined,
								trustLevel: server.trustLevel,
								defaultPolicy: server.defaultPolicy,
								toolPolicies: server.toolPolicies,
								appsPolicy: server.appsPolicy,
							}}
						/>
					</Field>
				</ManagementInfoGrid>
			</ModalSection>

			<ModalSection title="Discovery" description="Most recently discovered MCP capabilities for this server.">
				{loadingDiscovery && <div className="text-center text-sm">Loading discovery cache…</div>}

				{discoveryError && (
					<div className="alert alert-warning rounded-2xl text-sm">
						<div className="flex items-center gap-2">
							<FiAlertCircle size={14} />
							<span>{discoveryError}</span>
						</div>
					</div>
				)}

				<div className="space-y-6">
					<div>
						<h4 className="mb-2 text-sm font-semibold">Tools ({discovery.tools.length})</h4>
						<div className="space-y-3">
							{discovery.tools.map(tool => (
								<ManagementItemCard
									key={`${tool.serverID}:${tool.toolName}:${tool.digest}`}
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
													tool.inferredRisk === MCPToolRisk.MCPToolRiskRead
														? 'success'
														: tool.inferredRisk === MCPToolRisk.MCPToolRiskWrite
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
											{tool.app?.resourceUri ? (
												<MetadataPill label="App" title={tool.app.resourceUri}>
													Configured
												</MetadataPill>
											) : null}
											{tool.stale ? <MetadataPill>Stale discovery</MetadataPill> : null}
										</>
									}
								/>
							))}

							{discovery.tools.length === 0 ? (
								<div className="border-base-content/10 rounded-2xl border py-6 text-center text-sm">
									No tools discovered.
								</div>
							) : null}
						</div>
					</div>

					<div>
						<h4 className="mb-2 text-sm font-semibold">Resources ({discovery.resources.length})</h4>
						<div className="space-y-3">
							{discovery.resources.map(resource => (
								<ManagementItemCard
									key={`${resource.serverID}:${resource.uri}:${resource.digest ?? ''}`}
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

							{discovery.resources.length === 0 ? (
								<div className="border-base-content/10 rounded-2xl border py-6 text-center text-sm">
									No resources discovered.
								</div>
							) : null}
						</div>
					</div>

					<div>
						<h4 className="mb-2 text-sm font-semibold">Resource Templates ({discovery.resourceTemplates.length})</h4>
						<div className="space-y-3">
							{discovery.resourceTemplates.map(template => (
								<ManagementItemCard
									key={`${template.serverID}:${template.uriTemplate}:${template.digest ?? ''}`}
									title={template.displayName || template.name || template.uriTemplate}
									subtitle={template.uriTemplate}
									description={template.description}
									metadata={
										<>
											<MetadataPill label="MIME">{template.mimeType || 'Unknown'}</MetadataPill>
											<ArgumentSummary args={template.arguments} />
										</>
									}
								/>
							))}

							{discovery.resourceTemplates.length === 0 ? (
								<div className="border-base-content/10 rounded-2xl border py-6 text-center text-sm">
									No resource templates discovered.
								</div>
							) : null}
						</div>
					</div>

					<div>
						<h4 className="mb-2 text-sm font-semibold">Prompts ({discovery.prompts.length})</h4>
						<div className="space-y-3">
							{discovery.prompts.map(prompt => (
								<ManagementItemCard
									key={`${prompt.serverID}:${prompt.promptName}:${prompt.digest ?? ''}`}
									title={prompt.displayName || prompt.promptName}
									subtitle={prompt.promptName}
									description={prompt.description}
									metadata={<ArgumentSummary args={prompt.arguments} />}
								/>
							))}

							{discovery.prompts.length === 0 ? (
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

export function MCPServerDetailsModal({
	isOpen,
	onClose,
	bundle,
	server,
	runtime,
	authHealth,
}: MCPServerDetailsModalProps) {
	if (!isOpen || !bundle || !server) {
		return null;
	}
	if (typeof document === 'undefined' || !document.body) {
		return null;
	}

	return (
		<MCPServerDetailsModalContent
			key={`${bundle.id}:${server.id}:${server.modifiedAt}`}
			onClose={onClose}
			bundle={bundle}
			server={server}
			runtime={runtime}
			authHealth={authHealth}
		/>
	);
}
