import { useEffect, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiCheck, FiX } from 'react-icons/fi';

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

import {
	getAllMCPServerPrompts,
	getAllMCPServerResources,
	getAllMCPServerResourceTemplates,
	getAllMCPServerTools,
} from '@/apis/list_helper';

import { ModalBackdrop } from '@/components/modal_backdrop';

import {
	getMCPApprovalRuleLabel,
	getMCPAuthHealthBadgeClass,
	getMCPAuthHealthLabel,
	getMCPExecutionModeLabel,
	getMCPStatusBadgeClass,
	getMCPStatusLabel,
	getMCPToolRiskBadgeClass,
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

type DiscoveryData = {
	tools: MCPToolCapability[];
	resources: MCPResourceRef[];
	resourceTemplates: MCPResourceTemplateRef[];
	prompts: MCPPromptRef[];
};

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

function Field({ label, children }: { label: string; children: React.ReactNode }) {
	return (
		<div className="grid grid-cols-12 gap-2 text-sm">
			<div className="col-span-3 font-semibold">{label}</div>
			<div className="col-span-9 wrap-break-word">{children}</div>
		</div>
	);
}

function ArgumentSummary({ args }: { args?: Record<string, { required?: boolean; description?: string } | string> }) {
	if (!args || Object.keys(args).length === 0) return <span>-</span>;

	return (
		<div className="flex flex-wrap justify-center gap-1">
			{Object.entries(args).map(([name, def]) => {
				const required = typeof def === 'object' && Boolean(def.required);
				const description = typeof def === 'object' ? def.description : def;
				return (
					<span
						key={name}
						className={`badge badge-xs rounded-xl ${required ? 'badge-warning' : 'badge-ghost'}`}
						title={description}
					>
						{name}
						{required ? '*' : ''}
					</span>
				);
			})}
		</div>
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
	const dialogRef = useRef<HTMLDialogElement | null>(null);

	const [discovery, setDiscovery] = useState<DiscoveryData>({
		tools: [],
		resources: [],
		resourceTemplates: [],
		prompts: [],
	});
	const [loadingDiscovery, setLoadingDiscovery] = useState(false);
	const [discoveryError, setDiscoveryError] = useState('');

	useEffect(() => {
		if (!isOpen) return;

		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) dialog.showModal();

		return () => {
			if (dialog.open) dialog.close();
		};
	}, [isOpen]);

	useEffect(() => {
		if (!isOpen || !bundle || !server) return;

		let cancelled = false;

		// eslint-disable-next-line react-hooks/set-state-in-effect, react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setLoadingDiscovery(true);
		// eslint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setDiscoveryError('');

		Promise.allSettled([
			getAllMCPServerTools(bundle.id, server.id),
			getAllMCPServerResources(bundle.id, server.id),
			getAllMCPServerResourceTemplates(bundle.id, server.id),
			getAllMCPServerPrompts(bundle.id, server.id),
		])
			.then(results => {
				if (cancelled) return;
				const [toolsResult, resourcesResult, resourceTemplatesResult, promptsResult] = results;
				const errors = results
					.filter((result): result is PromiseRejectedResult => result.status === 'rejected')
					.map(result => (result.reason instanceof Error ? result.reason.message : 'Failed to load discovery section.'))
					.filter(Boolean);
				setDiscovery({
					tools: toolsResult.status === 'fulfilled' ? toolsResult.value : [],
					resources: resourcesResult.status === 'fulfilled' ? resourcesResult.value : [],
					resourceTemplates: resourceTemplatesResult.status === 'fulfilled' ? resourceTemplatesResult.value : [],
					prompts: promptsResult.status === 'fulfilled' ? promptsResult.value : [],
				});
				if (errors.length > 0) {
					setDiscoveryError(errors[0]);
				}
			})
			.finally(() => {
				if (cancelled) return;
				setLoadingDiscovery(false);
			});

		return () => {
			cancelled = true;
		};
	}, [bundle, isOpen, server]);

	const handleDialogClose = () => {
		onClose();
	};

	if (!isOpen || !bundle || !server) return null;
	if (typeof document === 'undefined' || !document.body) return null;

	return createPortal(
		<dialog ref={dialogRef} className="modal" onClose={handleDialogClose}>
			<div className="modal-box bg-base-200 max-h-[85vh] max-w-6xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[85vh] overflow-y-auto p-6">
					<div className="mb-4 flex items-center justify-between">
						<h3 className="text-lg font-bold">MCP Server Details</h3>
						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={() => dialogRef.current?.close()}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>

					<div className="space-y-3">
						<Field label="Display Name">{server.displayName}</Field>
						<Field label="Server ID">{server.id}</Field>
						<Field label="Bundle">{bundle.displayName || bundle.slug}</Field>
						<Field label="Transport">{getMCPTransportLabel(server.transport)}</Field>
						<Field label="Enabled">{server.enabled ? 'Yes' : 'No'}</Field>
						<Field label="Built-in">{server.isBuiltIn ? 'Yes' : 'No'}</Field>
						<Field label="Created">{server.createdAt}</Field>
						<Field label="Modified">{server.modifiedAt}</Field>

						<Field label="Runtime">
							<span className={`badge rounded-xl ${getMCPStatusBadgeClass(runtime?.status)}`}>
								{getMCPStatusLabel(runtime?.status)}
							</span>
							{runtime?.lastError && <div className="text-error mt-1 text-xs">{runtime.lastError}</div>}
						</Field>

						<Field label="Auth">
							<span className={`badge rounded-xl ${getMCPAuthHealthBadgeClass(authHealth?.state)}`}>
								{getMCPAuthHealthLabel(authHealth?.state)}
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
													? Object.fromEntries(
															Object.keys(server.stdio.secretEnvRefs).map(key => [key, '[configured]'])
														)
													: undefined,
											}
										: undefined,
									streamableHttp: server.streamableHttp
										? {
												...server.streamableHttp,
												clientCredentialRef: server.streamableHttp.clientCredentialRef ? '[configured]' : undefined,
											}
										: undefined,
									trustLevel: server.trustLevel,
									defaultPolicy: server.defaultPolicy,
									toolPolicies: server.toolPolicies,
									appsPolicy: server.appsPolicy,
								}}
							/>
						</Field>
					</div>

					<div className="divider">Discovery</div>

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
							<div className="border-base-content/10 overflow-x-auto rounded-2xl border">
								<table className="table-zebra table w-full">
									<thead>
										<tr className="bg-base-300 text-sm font-semibold">
											<th>Display Name</th>
											<th className="text-center">Name</th>
											<th className="text-center">Risk</th>
											<th className="text-center">Approval</th>
											<th className="text-center">Execution</th>
											<th className="text-center">Enabled</th>
											<th className="text-center">Stale</th>
										</tr>
									</thead>
									<tbody>
										{discovery.tools.map(tool => (
											<tr key={`${tool.serverID}:${tool.toolName}:${tool.digest}`}>
												<td>
													<div>{tool.displayName || tool.title || tool.toolName}</div>
													{tool.description && (
														<div className="text-base-content/70 max-w-xl text-xs">{tool.description}</div>
													)}
												</td>
												<td className="text-center">{tool.toolName}</td>
												<td className="text-center">
													<span className={`badge rounded-xl ${getMCPToolRiskBadgeClass(tool.inferredRisk)}`}>
														{getMCPToolRiskLabel(tool.inferredRisk)}
													</span>
												</td>
												<td className="text-center">{getMCPApprovalRuleLabel(tool.approvalRule)}</td>
												<td className="text-center">{getMCPExecutionModeLabel(tool.executionMode)}</td>
												<td className="text-center">
													{tool.enabled ? <FiCheck className="mx-auto" /> : <FiX className="mx-auto" />}
												</td>
												<td className="text-center">
													{tool.stale ? <FiCheck className="mx-auto" /> : <FiX className="mx-auto" />}
												</td>
											</tr>
										))}

										{discovery.tools.length === 0 && (
											<tr>
												<td colSpan={7} className="py-3 text-center text-sm">
													No tools discovered.
												</td>
											</tr>
										)}
									</tbody>
								</table>
							</div>
						</div>

						<div>
							<h4 className="mb-2 text-sm font-semibold">Resources ({discovery.resources.length})</h4>
							<div className="border-base-content/10 overflow-x-auto rounded-2xl border">
								<table className="table-zebra table w-full">
									<thead>
										<tr className="bg-base-300 text-sm font-semibold">
											<th>Display Name</th>
											<th>URI</th>
											<th className="text-center">MIME</th>
										</tr>
									</thead>
									<tbody>
										{discovery.resources.map(resource => (
											<tr key={`${resource.serverID}:${resource.uri}:${resource.digest ?? ''}`}>
												<td>
													<div>{resource.displayName}</div>
													{resource.description && (
														<div className="text-base-content/70 max-w-xl text-xs">{resource.description}</div>
													)}
												</td>
												<td className="max-w-lg break-all">{resource.uri}</td>
												<td className="text-center">{resource.mimeType || '-'}</td>
											</tr>
										))}

										{discovery.resources.length === 0 && (
											<tr>
												<td colSpan={3} className="py-3 text-center text-sm">
													No resources discovered.
												</td>
											</tr>
										)}
									</tbody>
								</table>
							</div>
						</div>

						<div>
							<h4 className="mb-2 text-sm font-semibold">Resource Templates ({discovery.resourceTemplates.length})</h4>
							<div className="border-base-content/10 overflow-x-auto rounded-2xl border">
								<table className="table-zebra table w-full">
									<thead>
										<tr className="bg-base-300 text-sm font-semibold">
											<th>Display Name</th>
											<th>URI Template</th>
											<th className="text-center">MIME</th>
											<th className="text-center">Arguments</th>
										</tr>
									</thead>
									<tbody>
										{discovery.resourceTemplates.map(template => (
											<tr key={`${template.serverID}:${template.uriTemplate}:${template.digest ?? ''}`}>
												<td>
													<div>{template.displayName}</div>
													{template.description && (
														<div className="text-base-content/70 max-w-xl text-xs">{template.description}</div>
													)}
												</td>
												<td className="max-w-lg break-all">{template.uriTemplate}</td>
												<td className="text-center">{template.mimeType || '-'}</td>
												<td className="text-center">
													<ArgumentSummary args={template.arguments} />
												</td>
											</tr>
										))}

										{discovery.resourceTemplates.length === 0 && (
											<tr>
												<td colSpan={4} className="py-3 text-center text-sm">
													No resource templates discovered.
												</td>
											</tr>
										)}
									</tbody>
								</table>
							</div>
						</div>

						<div>
							<h4 className="mb-2 text-sm font-semibold">Prompts ({discovery.prompts.length})</h4>
							<div className="border-base-content/10 overflow-x-auto rounded-2xl border">
								<table className="table-zebra table w-full">
									<thead>
										<tr className="bg-base-300 text-sm font-semibold">
											<th>Display Name</th>
											<th className="text-center">Name</th>
											<th className="text-center">Arguments</th>
										</tr>
									</thead>
									<tbody>
										{discovery.prompts.map(prompt => (
											<tr key={`${prompt.serverID}:${prompt.promptName}:${prompt.digest ?? ''}`}>
												<td>
													<div>{prompt.displayName}</div>
													{prompt.description && (
														<div className="text-base-content/70 max-w-xl text-xs">{prompt.description}</div>
													)}
												</td>
												<td className="text-center">{prompt.promptName}</td>
												<td className="text-center">
													<ArgumentSummary args={prompt.arguments} />
												</td>
											</tr>
										))}

										{discovery.prompts.length === 0 && (
											<tr>
												<td colSpan={3} className="py-3 text-center text-sm">
													No prompts discovered.
												</td>
											</tr>
										)}
									</tbody>
								</table>
							</div>
						</div>
					</div>

					<div className="modal-action">
						<button type="button" className="btn bg-base-300 rounded-xl" onClick={() => dialogRef.current?.close()}>
							Close
						</button>
					</div>
				</div>
			</div>
			<ModalBackdrop enabled={true} />
		</dialog>,
		document.body
	);
}
