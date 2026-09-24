import type { SubmitEventHandler } from 'react';
import { useCallback, useMemo, useState } from 'react';
import { FiAlertCircle } from 'react-icons/fi';

import type { ArtifactRef } from '@/spec/artifact';
import type { MCPSetupSubmissionValue, MCPStoreServerInstallationView } from '@/spec/mcp';
import { MCPInputKind } from '@/spec/mcp';

import { throwIfAborted } from '@/lib/async_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { mcpManagementAPI } from '@/apis/baseapi';

import { Loader } from '@/components/loader';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';

interface WorkspaceMCPSetupModalProps {
	isOpen: boolean;
	artifact: ArtifactRef | null;
	displayName: string;
	onClose: () => void;
	onSaved?: () => void;
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

function WorkspaceMCPSetupForm({
	artifact,
	displayName,
	installation,
	onClose,
	onSaved,
}: {
	artifact: ArtifactRef;
	displayName: string;
	installation: MCPStoreServerInstallationView;
	onClose: () => void;
	onSaved?: () => void;
}) {
	const inputs = useMemo(
		() => installation.document.configuration.install.inputs ?? {},
		[installation.document.configuration.install.inputs]
	);
	const [rows, setRows] = useState<Record<string, InputRow>>(() =>
		Object.fromEntries(Object.keys(inputs).map(name => [name, emptyRow()]))
	);
	const [reset, setReset] = useState(false);
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);

	const updateRow = (name: string, patch: Partial<InputRow>) => {
		setRows(previous => ({
			...previous,
			[name]: {
				...(previous[name] ?? emptyRow()),
				...patch,
			},
		}));
	};

	const validate = (): string | undefined => {
		for (const [name, declaration] of Object.entries(inputs)) {
			const row = rows[name] ?? emptyRow();
			const existing = installation.installation.inputs?.[name];
			const label = declaration.label || name;

			const configured =
				!reset &&
				(declaration.kind === MCPInputKind.Text || declaration.kind === MCPInputKind.Path
					? Boolean(existing?.value?.trim() || declaration.default?.trim())
					: Boolean(existing?.secretRef?.trim()));

			if (declaration.kind === MCPInputKind.OAuthClientCredentials) {
				const hasClientID = Boolean(row.clientID.trim());
				const hasClientSecret = Boolean(row.clientSecret.trim());

				if (hasClientID || hasClientSecret) {
					if (!hasClientID) {
						return `"${label}" requires a Client ID.`;
					}
					if (declaration.clientSecretRequired && !hasClientSecret) {
						return `"${label}" requires a Client Secret when replacing its credentials.`;
					}
				} else if (declaration.required && !configured) {
					return `"${label}" requires a Client ID.`;
				}

				continue;
			}

			if (!declaration.required || configured) {
				continue;
			}

			if (!row.value.trim()) {
				return `"${label}" is required.`;
			}
		}

		return undefined;
	};

	const submissionValues = (): Record<string, MCPSetupSubmissionValue> => {
		return Object.fromEntries<MCPSetupSubmissionValue>(
			Object.entries(inputs).map(([name, declaration]): readonly [string, MCPSetupSubmissionValue] => {
				const row = rows[name] ?? emptyRow();

				if (declaration.kind === MCPInputKind.OAuthClientCredentials) {
					return [
						name,
						{
							clientID: row.clientID,
							clientSecret: row.clientSecret,
						},
					];
				}

				return [
					name,
					{
						value: row.value,
					},
				];
			})
		);
	};

	const save: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();

		if (isSubmitting) {
			return;
		}

		setSubmitError('');

		const validationError = validate();
		if (validationError) {
			setSubmitError(validationError);
			return;
		}

		setIsSubmitting(true);

		void mcpManagementAPI.applyMCPServerSetup(artifact, submissionValues(), reset).then(
			() => {
				setIsSubmitting(false);
				onSaved?.();
				onClose();
			},
			(error: unknown) => {
				setSubmitError(error instanceof Error ? error.message : 'MCP setup could not be saved.');
				setIsSubmitting(false);
			}
		);
	};

	return (
		<div className="modal-box bg-base-200 max-h-[85vh] w-[calc(100%-1rem)] max-w-3xl overflow-y-auto rounded-2xl p-0">
			<ModalHeader
				title={`Configure ${displayName}`}
				description="Installation values are stored by MCP management. Secret values are never returned to Workspace or conversation state."
				onClose={onClose}
				closeDisabled={isSubmitting}
			/>

			<form className="space-y-4 p-4 sm:p-6" onSubmit={save}>
				{installation.document.configuration.install.note ? (
					<div className="bg-base-100 rounded-2xl p-3 text-sm">{installation.document.configuration.install.note}</div>
				) : null}

				{submitError ? (
					<div className="alert alert-error rounded-2xl text-sm">
						<FiAlertCircle size={14} />
						<span>{submitError}</span>
					</div>
				) : null}

				{Object.entries(inputs).map(([name, declaration]) => {
					const row = rows[name] ?? emptyRow();
					const binding = installation.installation.inputs?.[name];
					const isOAuth = declaration.kind === MCPInputKind.OAuthClientCredentials;
					const isSecret = isOAuth || declaration.kind === MCPInputKind.Secret;

					return (
						<div key={name} className="bg-base-100 rounded-2xl p-4">
							<div className="mb-3 flex items-start justify-between gap-3">
								<div className="min-w-0">
									<div className="font-semibold">
										{declaration.label || name}
										{declaration.required ? ' *' : ''}
									</div>
									{declaration.label ? null : <div className="text-base-content/60 text-xs">{name}</div>}
								</div>
								<span className="badge badge-ghost badge-xs">{inputKindLabel(declaration.kind)}</span>
							</div>

							{declaration.description ? (
								<p className="text-base-content/70 mb-3 text-xs">{declaration.description}</p>
							) : null}

							{isOAuth ? (
								<div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
									<input
										type="text"
										className="input w-full rounded-xl"
										value={row.clientID}
										disabled={isSubmitting}
										autoComplete="off"
										placeholder="Client ID"
										onChange={event => {
											updateRow(name, {
												clientID: event.currentTarget.value,
											});
										}}
									/>
									<input
										type="password"
										className="input w-full rounded-xl"
										value={row.clientSecret}
										disabled={isSubmitting}
										autoComplete="new-password"
										placeholder={declaration.clientSecretRequired ? 'Client Secret' : 'Client Secret (optional)'}
										onChange={event => {
											updateRow(name, {
												clientSecret: event.currentTarget.value,
											});
										}}
									/>
								</div>
							) : (
								<ModalField label={isSecret ? 'Secret value' : 'Value'} htmlFor={`workspace-mcp-input-${name}`}>
									<input
										id={`workspace-mcp-input-${name}`}
										type={isSecret ? 'password' : 'text'}
										className="input w-full rounded-xl"
										value={row.value}
										disabled={isSubmitting}
										autoComplete={isSecret ? 'new-password' : 'off'}
										placeholder={declaration.placeholder || declaration.default}
										onChange={event => {
											updateRow(name, {
												value: event.currentTarget.value,
											});
										}}
									/>
								</ModalField>
							)}

							{binding?.secretRef || binding?.value ? (
								<div className="text-base-content/60 mt-2 text-xs">
									Configured. Leave this field blank to preserve the existing value.
								</div>
							) : null}

							{declaration.note ? <div className="text-base-content/60 mt-2 text-xs">{declaration.note}</div> : null}
						</div>
					);
				})}

				{Object.keys(inputs).length === 0 ? (
					<div className="text-base-content/60 rounded-2xl border border-dashed p-4 text-sm">
						This MCP server does not declare installation inputs.
					</div>
				) : null}

				{installation.builtIn ? (
					<label className="label cursor-pointer justify-start gap-3">
						<input
							type="checkbox"
							className="checkbox checkbox-sm"
							checked={reset}
							disabled={isSubmitting}
							onChange={event => {
								setReset(event.currentTarget.checked);
							}}
						/>
						<span className="text-sm">Reset saved values that are not supplied here</span>
					</label>
				) : null}

				<ModalActions className="-mx-4 -mb-4 sm:-mx-6 sm:-mb-6">
					<button type="button" className="btn bg-base-300 rounded-xl" disabled={isSubmitting} onClick={onClose}>
						Cancel
					</button>
					<button type="submit" className="btn btn-primary rounded-xl" disabled={isSubmitting}>
						{isSubmitting ? 'Saving...' : 'Save MCP Setup'}
					</button>
				</ModalActions>
			</form>
		</div>
	);
}

function WorkspaceMCPSetupModalSession({
	artifact,
	displayName,
	onClose,
	onSaved,
}: Omit<WorkspaceMCPSetupModalProps, 'isOpen' | 'artifact'> & {
	artifact: ArtifactRef;
}) {
	const loadInstallation = useCallback(
		async (signal: AbortSignal): Promise<MCPStoreServerInstallationView> => {
			const installation = await mcpManagementAPI.getMCPServerInstallation(artifact);
			throwIfAborted(signal);
			return installation;
		},
		[artifact]
	);

	const {
		data: installation,
		error,
		isLoading,
		isRefreshing,
		reloadOrThrow,
	} = useAsyncResource(loadInstallation, {
		initialData: null as MCPStoreServerInstallationView | null,
	});

	return (
		<ModalDialog isOpen onClose={onClose} blockCancel>
			{error ? (
				<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-2xl rounded-2xl p-0">
					<ModalHeader title={`Configure ${displayName}`} onClose={onClose} />
					<div className="p-4 sm:p-6">
						<ManagementResourceError
							title="MCP setup could not be loaded"
							error={error}
							isRetrying={isRefreshing}
							onRetry={reloadOrThrow}
						/>
					</div>
				</div>
			) : isLoading || !installation ? (
				<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-2xl rounded-2xl p-6">
					<Loader text="Loading MCP setup requirements..." />
				</div>
			) : (
				<WorkspaceMCPSetupForm
					key={`${artifact.rootID}:${artifact.artifactID}:${installation.installationRevision}`}
					artifact={artifact}
					displayName={displayName}
					installation={installation}
					onClose={onClose}
					onSaved={onSaved}
				/>
			)}
		</ModalDialog>
	);
}

export function WorkspaceMCPSetupModal({
	isOpen,
	artifact,
	displayName,
	onClose,
	onSaved,
}: WorkspaceMCPSetupModalProps) {
	if (!isOpen || !artifact) {
		return null;
	}

	return (
		<WorkspaceMCPSetupModalSession
			key={`${artifact.rootID}:${artifact.artifactID}`}
			artifact={artifact}
			displayName={displayName}
			onClose={onClose}
			onSaved={onSaved}
		/>
	);
}
