import { useEffect, useMemo, useState } from 'react';
import { FiAlertCircle, FiKey, FiServer } from 'react-icons/fi';

import type { AgentMCPSetupDescriptor } from '@/spec/agent';
import type { ArtifactRef } from '@/spec/artifact';
import type { MCPSetupSubmissionValue, MCPStoreServerInstallationView } from '@/spec/mcp';
import { MCPHTTPAuthMode, MCPInputKind } from '@/spec/mcp';

import { getErrorMessage } from '@/lib/error_utils';

import { mcpManagementAPI } from '@/apis/baseapi';

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

function setupValuesForInstallation(
	installation: MCPStoreServerInstallationView
): Record<string, MCPSetupSubmissionValue> {
	const values: Record<string, MCPSetupSubmissionValue> = {};

	for (const [name, declaration] of Object.entries(installation.document.configuration.install.inputs ?? {})) {
		const binding = installation.installation.inputs?.[name];

		if (declaration.kind === MCPInputKind.Text || declaration.kind === MCPInputKind.Path) {
			values[name] = {
				value: binding?.value ?? '',
			};
		} else {
			values[name] = {};
		}
	}

	return values;
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

				<ModalSection title="Declared MCP relationship">
					<div className="text-sm">
						<span className="font-medium">Dependency:</span> {descriptor.name}
					</div>
				</ModalSection>

				<ModalActions>
					<button type="button" className="btn bg-base-300 rounded-xl" onClick={onClose}>
						Close
					</button>
				</ModalActions>
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
	const [installation, setInstallation] = useState<MCPStoreServerInstallationView | null>(null);
	const [values, setValues] = useState<Record<string, MCPSetupSubmissionValue>>({});
	const [loadError, setLoadError] = useState('');
	const [saveError, setSaveError] = useState('');
	const [isLoading, setIsLoading] = useState(true);
	const [isSaving, setIsSaving] = useState(false);

	useEffect(() => {
		let cancelled = false;

		void mcpManagementAPI
			.getMCPServerInstallation(artifact)
			.then(value => {
				if (cancelled) {
					return;
				}

				setInstallation(value);
				setValues(setupValuesForInstallation(value));
			})
			.catch((error: unknown) => {
				if (!cancelled) {
					setLoadError(getErrorMessage(error, 'MCP installation settings could not be loaded.'));
				}
			})
			.finally(() => {
				if (!cancelled) {
					setIsLoading(false);
				}
			});

		return () => {
			cancelled = true;
		};
	}, [artifact]);

	const inputEntries = useMemo(
		() => Object.entries(installation?.document.configuration.install.inputs ?? {}),
		[installation?.document.configuration.install.inputs]
	);

	const missingRequiredInputs = useMemo(() => {
		if (!installation) {
			return [];
		}

		return inputEntries.flatMap(([name, declaration]) => {
			const submitted = values[name] ?? {};
			const binding = installation.installation.inputs?.[name];
			const label = declaration.label || name;

			if (declaration.kind === MCPInputKind.OAuthClientCredentials) {
				const hasClientID = Boolean(submitted.clientID?.trim());
				const hasClientSecret = Boolean(submitted.clientSecret?.trim());
				const hasSubmission = hasClientID || hasClientSecret;

				if (hasSubmission) {
					return hasClientID && (!declaration.clientSecretRequired || hasClientSecret) ? [] : [label];
				}

				if (!declaration.required) {
					return [];
				}

				return binding?.secretRef?.trim() ? [] : [label];
			}

			if (!declaration.required) {
				return [];
			}

			switch (declaration.kind) {
				case MCPInputKind.Text:
				case MCPInputKind.Path:
					return submitted.value?.trim() || binding?.value?.trim() || declaration.default?.trim() ? [] : [label];

				case MCPInputKind.Secret:
					return submitted.value?.trim() || binding?.secretRef?.trim() ? [] : [label];

				default:
					return [label];
			}
		});
	}, [inputEntries, installation, values]);

	const submit = async () => {
		if (!installation || isSaving) {
			return;
		}

		if (missingRequiredInputs.length > 0) {
			setSaveError(`Configure required MCP inputs: ${missingRequiredInputs.join(', ')}.`);
			return;
		}

		setSaveError('');
		setIsSaving(true);

		try {
			await mcpManagementAPI.applyMCPServerSetup(artifact, values, false);
			const refreshed = await mcpManagementAPI.getMCPServerInstallation(artifact);
			setInstallation(refreshed);
			setValues(setupValuesForInstallation(refreshed));
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
					description={
						installation?.document.displayName && installation.document.displayName !== descriptor.name
							? installation.document.displayName
							: undefined
					}
					onClose={onClose}
					closeDisabled={isSaving}
				/>

				<div className="space-y-5">
					{loadError ? (
						<div className="alert alert-error rounded-2xl text-sm">
							<FiAlertCircle size={16} />
							<span>{loadError}</span>
						</div>
					) : null}

					{saveError ? (
						<div className="alert alert-error rounded-2xl text-sm">
							<FiAlertCircle size={16} />
							<span>{saveError}</span>
						</div>
					) : null}

					{isLoading ? (
						<div className="alert rounded-2xl text-sm">
							<span className="loading loading-spinner loading-sm" />
							<span>Loading MCP installation inputs...</span>
						</div>
					) : null}

					{installation ? (
						<>
							<ModalSection title="Installation inputs">
								{inputEntries.length === 0 ? (
									<div className="text-base-content/70 text-sm">This MCP does not declare installation inputs.</div>
								) : (
									<div className="space-y-4">
										{inputEntries.map(([name, declaration]) => {
											const binding = installation.installation.inputs?.[name];
											const value = values[name] ?? {};
											const label = declaration.label || name;
											const configured =
												declaration.kind === MCPInputKind.Text || declaration.kind === MCPInputKind.Path
													? Boolean(binding?.value?.trim() || declaration.default?.trim())
													: Boolean(binding?.secretRef?.trim());

											if (declaration.kind === MCPInputKind.OAuthClientCredentials) {
												return (
													<div key={name} className="border-base-content/10 rounded-2xl border p-3">
														<div className="mb-3 flex items-center gap-2">
															<FiKey size={14} />
															<div>
																<div className="font-medium">{label}</div>
																<div className="text-base-content/70 text-xs">
																	{[
																		declaration.required ? 'Required' : 'Optional',
																		configured ? 'Configured' : undefined,
																	]
																		.filter(Boolean)
																		.join(' · ')}
																</div>
															</div>
														</div>

														<ModalField label="Client ID" htmlFor={`agent-mcp-client-id-${name}`}>
															<input
																id={`agent-mcp-client-id-${name}`}
																className="input w-full rounded-xl"
																value={value.clientID ?? ''}
																onChange={event => {
																	setValues(previous => ({
																		...previous,
																		[name]: {
																			...previous[name],
																			clientID: event.currentTarget.value,
																		},
																	}));
																}}
																disabled={isSaving}
															/>
														</ModalField>

														<ModalField
															label={declaration.clientSecretRequired ? 'Client Secret' : 'Client Secret (optional)'}
															htmlFor={`agent-mcp-client-secret-${name}`}
														>
															<input
																id={`agent-mcp-client-secret-${name}`}
																type="password"
																className="input w-full rounded-xl"
																value={value.clientSecret ?? ''}
																onChange={event => {
																	setValues(previous => ({
																		...previous,
																		[name]: {
																			...previous[name],
																			clientSecret: event.currentTarget.value,
																		},
																	}));
																}}
																disabled={isSaving}
																autoComplete="off"
															/>
														</ModalField>
													</div>
												);
											}

											return (
												<ModalField
													key={name}
													label={`${label}${declaration.required ? ' *' : ''}`}
													htmlFor={`agent-mcp-input-${name}`}
													hint={[
														declaration.description,
														inputKindLabel(declaration.kind),
														configured ? 'Configured' : undefined,
													]
														.filter(Boolean)
														.join(' · ')}
												>
													<input
														id={`agent-mcp-input-${name}`}
														type={declaration.kind === MCPInputKind.Secret ? 'password' : 'text'}
														className="input w-full rounded-xl"
														value={value.value ?? ''}
														onChange={event => {
															setValues(previous => ({
																...previous,
																[name]: {
																	...previous[name],
																	value: event.currentTarget.value,
																},
															}));
														}}
														disabled={isSaving}
														autoComplete="off"
													/>
												</ModalField>
											);
										})}
									</div>
								)}
							</ModalSection>

							{descriptor.authMode === MCPHTTPAuthMode.OAuth ? (
								<div className="alert alert-info rounded-2xl text-sm">
									<FiServer size={16} />
									<span>
										OAuth authorization is initiated through the MCP management flow after installation inputs are
										saved.
									</span>
								</div>
							) : null}
						</>
					) : null}

					<ModalActions>
						<button type="button" className="btn bg-base-300 rounded-xl" disabled={isSaving} onClick={onClose}>
							Close
						</button>
						<button
							type="button"
							className="btn btn-primary rounded-xl"
							disabled={isLoading || !installation || isSaving}
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

export function AgentMCPSetupModal({ isOpen, descriptor, onClose }: AgentMCPSetupModalProps) {
	if (!isOpen || !descriptor) {
		return null;
	}

	const artifact = descriptor.artifact;

	return (
		<ModalDialog isOpen={isOpen} onClose={onClose}>
			{artifact ? (
				<AgentMCPSetupContent
					key={`${artifact.rootID}:${artifact.artifactID}`}
					descriptor={descriptor}
					artifact={artifact}
					onClose={onClose}
				/>
			) : (
				<UnresolvedMCPSetupContent descriptor={descriptor} onClose={onClose} />
			)}
		</ModalDialog>
	);
}
