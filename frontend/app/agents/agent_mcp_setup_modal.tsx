import { useCallback, useMemo, useState } from 'react';
import { FiAlertCircle, FiKey, FiServer } from 'react-icons/fi';

import type { AgentMCPSetupDescriptor } from '@/spec/agent';
import type { ArtifactRef } from '@/spec/artifact';
import type { MCPServerSetupView, MCPSetupSubmissionValue } from '@/spec/mcp';
import { MCPHTTPAuthMode, MCPInputKind } from '@/spec/mcp';

import { throwIfAborted } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { mcpManagementAPI } from '@/apis/baseapi';

import { Loader } from '@/components/loader';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

interface AgentMCPSetupModalProps {
	isOpen: boolean;
	descriptor: AgentMCPSetupDescriptor | null;
	onClose: () => void;
}

interface InputRow {
	value: string;
	clientID: string;
	clientSecret: string;
}

function emptyRow(): InputRow {
	return {
		value: '',
		clientID: '',
		clientSecret: '',
	};
}

function inputKindLabel(kind: MCPInputKind): string {
	switch (kind) {
		case MCPInputKind.Secret:
			return 'Secret';
		case MCPInputKind.OAuthClientCredentials:
			return 'OAuth client credentials';
		case MCPInputKind.Path:
			return 'Path';
		default:
			return 'Text';
	}
}

function UnresolvedMCPSetupContent({
	descriptor,
	onClose,
}: {
	descriptor: AgentMCPSetupDescriptor;
	onClose: () => void;
}) {
	return (
		<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-xl rounded-2xl p-0">
			<ModalHeader title={`MCP Setup: ${descriptor.name}`} onClose={onClose} />

			<div className="space-y-4 p-4 sm:p-6">
				<div className="alert alert-warning rounded-2xl text-sm">
					<FiAlertCircle size={16} />
					<span>
						This Agent relationship does not currently resolve to an MCP Artifact. Install or repair the MCP dependency
						before configuring it.
					</span>
				</div>

				<ModalActions>
					<button type="button" className="btn bg-base-300 rounded-xl" onClick={onClose}>
						Close
					</button>
				</ModalActions>
			</div>
		</div>
	);
}

function AgentMCPSetupForm({
	descriptor,
	setup,
	onClose,
}: {
	descriptor: AgentMCPSetupDescriptor;
	setup: MCPServerSetupView;
	onClose: () => void;
}) {
	const [rows, setRows] = useState<Record<string, InputRow>>(() =>
		Object.fromEntries(setup.inputs.map(input => [input.name, emptyRow()]))
	);
	const [saveError, setSaveError] = useState('');
	const [isSaving, setIsSaving] = useState(false);

	const updateRow = (name: string, patch: Partial<InputRow>) => {
		setRows(previous => ({
			...previous,
			[name]: {
				...(previous[name] ?? emptyRow()),
				...patch,
			},
		}));
	};

	const missingRequiredInputs = useMemo(() => {
		return setup.inputs.flatMap(input => {
			const row = rows[input.name] ?? emptyRow();
			const label = input.declaration.label || input.name;
			const configured =
				input.declaration.kind === MCPInputKind.Text || input.declaration.kind === MCPInputKind.Path
					? Boolean(input.boundValue?.trim() || input.declaration.default?.trim())
					: input.secretConfigured;

			if (input.declaration.kind === MCPInputKind.OAuthClientCredentials) {
				const hasClientID = Boolean(row.clientID.trim());
				const hasClientSecret = Boolean(row.clientSecret.trim());

				if (hasClientID || hasClientSecret) {
					return hasClientID && (!input.declaration.clientSecretRequired || hasClientSecret) ? [] : [label];
				}

				return input.declaration.required && !configured ? [label] : [];
			}

			if (!input.declaration.required || configured) {
				return [];
			}

			return row.value.trim() ? [] : [label];
		});
	}, [rows, setup.inputs]);

	const submit = async () => {
		if (isSaving) {
			return;
		}

		if (missingRequiredInputs.length > 0) {
			setSaveError(`Configure required MCP inputs: ${missingRequiredInputs.join(', ')}.`);
			return;
		}

		const values: Record<string, MCPSetupSubmissionValue> = Object.fromEntries(
			setup.inputs.map(input => {
				const row = rows[input.name] ?? emptyRow();

				if (input.declaration.kind === MCPInputKind.OAuthClientCredentials) {
					return [
						input.name,
						{
							clientID: row.clientID,
							clientSecret: row.clientSecret,
						},
					];
				}

				return [
					input.name,
					{
						value: row.value,
					},
				];
			})
		);

		setSaveError('');
		setIsSaving(true);

		try {
			await mcpManagementAPI.applyMCPServerSetup(setup.server, values, false);
			onClose();
		} catch (error) {
			setSaveError(getErrorMessage(error, 'Failed to save MCP installation settings.'));
		} finally {
			setIsSaving(false);
		}
	};

	return (
		<div className="modal-box bg-base-200 max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-2xl overflow-hidden rounded-2xl p-0">
			<div className="app-scrollbar-thin max-h-[calc(100dvh-1rem)] overflow-y-auto p-4 sm:p-6">
				<ModalHeader
					title={`MCP Setup: ${descriptor.name}`}
					description={setup.displayName !== descriptor.name ? setup.displayName : undefined}
					onClose={onClose}
					closeDisabled={isSaving}
				/>

				<div className="space-y-5">
					{saveError ? (
						<div className="alert alert-error rounded-2xl text-sm">
							<FiAlertCircle size={16} />
							<span>{saveError}</span>
						</div>
					) : null}

					<ModalSection title="Installation inputs">
						{setup.note ? <div className="bg-base-100 mb-4 rounded-2xl p-3 text-sm">{setup.note}</div> : null}

						{setup.inputs.length === 0 ? (
							<div className="text-base-content/70 text-sm">This MCP server does not declare installation inputs.</div>
						) : (
							<div className="space-y-4">
								{setup.inputs.map(input => {
									const row = rows[input.name] ?? emptyRow();
									const isOAuth = input.declaration.kind === MCPInputKind.OAuthClientCredentials;
									const isSecret = isOAuth || input.declaration.kind === MCPInputKind.Secret;

									return (
										<div key={input.name} className="border-base-content/10 rounded-2xl border p-3">
											<div className="mb-3 flex items-center gap-2">
												{isSecret ? <FiKey size={14} /> : <FiServer size={14} />}
												<div>
													<div className="font-medium">{input.declaration.label || input.name}</div>
													<div className="text-base-content/70 text-xs">
														{[
															inputKindLabel(input.declaration.kind),
															input.declaration.required ? 'Required' : 'Optional',
															input.secretConfigured || input.boundValue ? 'Configured' : undefined,
														]
															.filter(Boolean)
															.join(' · ')}
													</div>
												</div>
											</div>

											{input.declaration.description ? (
												<div className="text-base-content/70 mb-3 text-xs">{input.declaration.description}</div>
											) : null}

											{isOAuth ? (
												<div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
													<input
														type="text"
														className="input w-full rounded-xl"
														value={row.clientID}
														disabled={isSaving}
														autoComplete="off"
														placeholder="Client ID"
														onChange={event => {
															updateRow(input.name, {
																clientID: event.currentTarget.value,
															});
														}}
													/>
													<input
														type="password"
														className="input w-full rounded-xl"
														value={row.clientSecret}
														disabled={isSaving}
														autoComplete="new-password"
														placeholder={
															input.declaration.clientSecretRequired ? 'Client Secret' : 'Client Secret (optional)'
														}
														onChange={event => {
															updateRow(input.name, {
																clientSecret: event.currentTarget.value,
															});
														}}
													/>
												</div>
											) : (
												<ModalField
													label={isSecret ? 'Secret value' : 'Value'}
													htmlFor={`agent-mcp-input-${input.name}`}
												>
													<input
														id={`agent-mcp-input-${input.name}`}
														type={isSecret ? 'password' : 'text'}
														className="input w-full rounded-xl"
														value={row.value}
														disabled={isSaving}
														autoComplete={isSecret ? 'new-password' : 'off'}
														placeholder={input.declaration.placeholder || input.declaration.default}
														onChange={event => {
															updateRow(input.name, {
																value: event.currentTarget.value,
															});
														}}
													/>
												</ModalField>
											)}

											{input.declaration.note ? (
												<div className="text-base-content/60 mt-2 text-xs">{input.declaration.note}</div>
											) : null}
										</div>
									);
								})}
							</div>
						)}
					</ModalSection>

					{descriptor.authMode === MCPHTTPAuthMode.OAuth ? (
						<div className="alert alert-info rounded-2xl text-sm">
							<FiServer size={16} />
							<span>OAuth authorization is initiated from MCP management after setup is saved.</span>
						</div>
					) : null}

					<ModalActions>
						<button type="button" className="btn bg-base-300 rounded-xl" disabled={isSaving} onClick={onClose}>
							Close
						</button>
						<button
							type="button"
							className="btn btn-primary rounded-xl"
							disabled={isSaving}
							onClick={() => {
								void submit();
							}}
						>
							{isSaving ? 'Saving...' : 'Save MCP Setup'}
						</button>
					</ModalActions>
				</div>
			</div>
		</div>
	);
}

function AgentMCPSetupContent({
	descriptor,
	artifact,
	onClose,
}: {
	descriptor: AgentMCPSetupDescriptor;
	artifact: ArtifactRef;
	onClose: () => void;
}) {
	const loadSetup = useCallback(
		async (signal: AbortSignal): Promise<MCPServerSetupView> => {
			const setup = await mcpManagementAPI.getMCPServerSetup(artifact);
			throwIfAborted(signal);
			return setup;
		},
		[artifact]
	);

	const {
		data: setup,
		error,
		isLoading,
		isRefreshing,
		reloadOrThrow,
	} = useAsyncResource(loadSetup, {
		initialData: null as MCPServerSetupView | null,
	});

	if (error) {
		return (
			<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-2xl rounded-2xl p-0">
				<ModalHeader title={`MCP Setup: ${descriptor.name}`} onClose={onClose} />
				<div className="p-4 sm:p-6">
					<ManagementResourceError
						title="MCP setup could not be loaded"
						error={error}
						isRetrying={isRefreshing}
						onRetry={reloadOrThrow}
					/>
				</div>
			</div>
		);
	}

	if (isLoading || !setup) {
		return (
			<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-2xl rounded-2xl p-6">
				<Loader text="Loading MCP installation inputs..." />
			</div>
		);
	}

	return (
		<AgentMCPSetupForm
			key={`${artifact.rootID}:${artifact.artifactID}:${setup.settingsRevision}`}
			descriptor={descriptor}
			setup={setup}
			onClose={onClose}
		/>
	);
}

export function AgentMCPSetupModal({ isOpen, descriptor, onClose }: AgentMCPSetupModalProps) {
	if (!isOpen || !descriptor) {
		return null;
	}

	if (!descriptor.artifact) {
		return (
			<ModalDialog isOpen={isOpen} onClose={onClose}>
				<UnresolvedMCPSetupContent descriptor={descriptor} onClose={onClose} />
			</ModalDialog>
		);
	}

	return (
		<ModalDialog isOpen={isOpen} onClose={onClose}>
			<AgentMCPSetupContent
				key={`${descriptor.artifact.rootID}:${descriptor.artifact.artifactID}`}
				descriptor={descriptor}
				artifact={descriptor.artifact}
				onClose={onClose}
			/>
		</ModalDialog>
	);
}
