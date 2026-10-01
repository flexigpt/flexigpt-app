import type { SubmitEventHandler } from 'react';
import { useCallback, useState } from 'react';
import { FiAlertCircle } from 'react-icons/fi';

import type { ArtifactRef } from '@/spec/artifact';
import type { MCPServerSetupView, MCPSetupSubmissionValue } from '@/spec/mcp';
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
	setup,
	displayName,
	onClose,
	onSaved,
}: {
	setup: MCPServerSetupView;
	displayName: string;
	onClose: () => void;
	onSaved?: () => void;
}) {
	const [rows, setRows] = useState<Record<string, InputRow>>(() =>
		Object.fromEntries(setup.inputs.map(input => [input.name, emptyRow()]))
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
		for (const input of setup.inputs) {
			const row = rows[input.name] ?? emptyRow();
			const label = input.declaration.label || input.name;
			const configured =
				!reset &&
				(input.declaration.kind === MCPInputKind.Text || input.declaration.kind === MCPInputKind.Path
					? Boolean(input.boundValue?.trim() || input.declaration.default?.trim())
					: input.secretConfigured);

			if (input.declaration.kind === MCPInputKind.OAuthClientCredentials) {
				const hasClientID = Boolean(row.clientID.trim());
				const hasClientSecret = Boolean(row.clientSecret.trim());

				if (hasClientID || hasClientSecret) {
					if (!hasClientID) {
						return `"${label}" requires a Client ID.`;
					}
					if (input.declaration.clientSecretRequired && !hasClientSecret) {
						return `"${label}" requires a Client Secret when replacing its credentials.`;
					}
				} else if (input.declaration.required && !configured) {
					return `"${label}" requires a Client ID.`;
				}

				continue;
			}

			if (!input.declaration.required || configured) {
				continue;
			}

			if (!row.value.trim()) {
				return `"${label}" is required.`;
			}
		}

		return undefined;
	};

	const buildSubmission = (): Record<string, MCPSetupSubmissionValue> =>
		Object.fromEntries(
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
		) as Record<string, MCPSetupSubmissionValue>;

	const submit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();

		if (isSubmitting) {
			return;
		}

		const validationError = validate();
		if (validationError) {
			setSubmitError(validationError);
			return;
		}

		setSubmitError('');
		setIsSubmitting(true);

		void mcpManagementAPI
			.applyMCPServerSetup(setup.server, buildSubmission(), reset)
			.then(
				() => {
					onSaved?.();
					onClose();
				},
				(error: unknown) => {
					setSubmitError(error instanceof Error ? error.message : 'MCP setup could not be saved.');
				}
			)
			.finally(() => {
				setIsSubmitting(false);
			});
	};

	return (
		<div className="modal-box bg-base-200 max-h-[85vh] w-[calc(100%-1rem)] max-w-3xl overflow-y-auto rounded-2xl p-0">
			<ModalHeader
				title={`Configure ${displayName}`}
				description="MCP settings and credentials are stored locally. Existing secret values are never displayed."
				onClose={onClose}
				closeDisabled={isSubmitting}
			/>

			<form className="space-y-4 p-4 sm:p-6" onSubmit={submit} aria-busy={isSubmitting}>
				{setup.note ? <div className="bg-base-100 rounded-2xl p-3 text-sm">{setup.note}</div> : null}

				{submitError ? (
					<div className="alert alert-error rounded-2xl text-sm">
						<FiAlertCircle size={14} />
						<span>{submitError}</span>
					</div>
				) : null}

				{setup.inputs.map(input => {
					const row = rows[input.name] ?? emptyRow();
					const isOAuth = input.declaration.kind === MCPInputKind.OAuthClientCredentials;
					const isSecret = isOAuth || input.declaration.kind === MCPInputKind.Secret;

					return (
						<div key={input.name} className="bg-base-100 rounded-2xl p-4">
							<div className="mb-3 flex items-start justify-between gap-3">
								<div className="min-w-0">
									<div className="font-semibold">
										{input.declaration.label || input.name}
										{input.declaration.required ? ' *' : ''}
									</div>
									{input.declaration.label ? null : <div className="text-base-content/60 text-xs">{input.name}</div>}
								</div>
								<span className="badge badge-ghost badge-xs">{inputKindLabel(input.declaration.kind)}</span>
							</div>

							{input.declaration.description ? (
								<p className="text-base-content/70 mb-3 text-xs">{input.declaration.description}</p>
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
											updateRow(input.name, {
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
										placeholder={input.declaration.clientSecretRequired ? 'Client Secret' : 'Client Secret (optional)'}
										onChange={event => {
											updateRow(input.name, {
												clientSecret: event.currentTarget.value,
											});
										}}
									/>
								</div>
							) : (
								<ModalField label={isSecret ? 'Secret value' : 'Value'} htmlFor={`workspace-mcp-input-${input.name}`}>
									<input
										id={`workspace-mcp-input-${input.name}`}
										type={isSecret ? 'password' : 'text'}
										className="input w-full rounded-xl"
										value={row.value}
										disabled={isSubmitting}
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

							{input.secretConfigured || input.boundValue ? (
								<div className="text-base-content/60 mt-2 text-xs">
									Configured. Leave this field blank to preserve the existing value.
								</div>
							) : null}

							{input.declaration.note ? (
								<div className="text-base-content/60 mt-2 text-xs">{input.declaration.note}</div>
							) : null}
						</div>
					);
				})}

				{setup.inputs.length === 0 ? (
					<div className="text-base-content/60 rounded-2xl border border-dashed p-4 text-sm">
						This MCP server does not declare installation inputs.
					</div>
				) : null}

				{setup.builtIn ? (
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
			) : isLoading || !setup ? (
				<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-2xl rounded-2xl p-6">
					<Loader text="Loading MCP setup requirements..." />
				</div>
			) : (
				<WorkspaceMCPSetupForm
					key={`${artifact.rootID}:${artifact.artifactID}:${setup.settingsRevision}`}
					setup={setup}
					displayName={displayName}
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
