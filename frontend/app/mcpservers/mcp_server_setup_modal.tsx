import type { SubmitEventHandler } from 'react';
import { useMemo, useState } from 'react';

import { FiAlertCircle } from 'react-icons/fi';

import { MCPInputKind } from '@/spec/mcp_artifact';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

import type { MCPServerView, MCPSetupSubmissionValue } from '@/mcpservers/lib/mcp_management';
import { serverSetupInputs } from '@/mcpservers/lib/mcp_management';

interface MCPServerSetupModalProps {
	isOpen: boolean;
	server: MCPServerView | null;
	onClose: () => void;
	onSubmit: (server: MCPServerView, values: Record<string, MCPSetupSubmissionValue>, reset: boolean) => Promise<void>;
}

interface RowState {
	value: string;
	clientID: string;
	clientSecret: string;
}

function emptyRow(): RowState {
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

function MCPServerSetupModalContent({
	server,
	onSubmit,
}: Omit<MCPServerSetupModalProps, 'isOpen' | 'onClose'> & { server: MCPServerView }) {
	const inputs = useMemo(() => serverSetupInputs(server), [server]);
	const [rows, setRows] = useState<Record<string, RowState>>(() =>
		Object.fromEntries(inputs.map(input => [input.name, emptyRow()]))
	);
	const [reset, setReset] = useState(false);
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);
	const { requestClose, unmountingRef } = useModalDialogController();

	const updateRow = (name: string, patch: Partial<RowState>) => {
		setRows(previous => ({
			...previous,
			[name]: {
				...(previous[name] ?? emptyRow()),
				...patch,
			},
		}));
	};

	const validate = (): string | undefined => {
		for (const input of inputs) {
			const row = rows[input.name] ?? emptyRow();
			const existingConfigured =
				input.declaration.kind === MCPInputKind.Text || input.declaration.kind === MCPInputKind.Path
					? Boolean(input.boundValue?.trim() || input.declaration.default?.trim())
					: Boolean(input.boundSecretRef?.trim());

			if (!input.declaration.required || existingConfigured) {
				continue;
			}

			if (input.declaration.kind === MCPInputKind.OAuthClientCredentials) {
				if (!row.clientID.trim()) {
					return `"${input.declaration.label || input.name}" requires a Client ID.`;
				}

				if (input.declaration.clientSecretRequired && !row.clientSecret.trim()) {
					return `"${input.declaration.label || input.name}" requires a Client Secret.`;
				}

				continue;
			}

			if (!row.value.trim()) {
				return `"${input.declaration.label || input.name}" is required.`;
			}
		}

		return undefined;
	};

	const buildSubmission = (): Record<string, MCPSetupSubmissionValue> =>
		Object.fromEntries(
			inputs.map(input => {
				const row = rows[input.name] ?? emptyRow();

				if (input.declaration.kind === MCPInputKind.OAuthClientCredentials) {
					return [
						input.name,
						{
							clientID: row.clientID,
							clientSecret: row.clientSecret,
						} as MCPSetupSubmissionValue,
					];
				}

				return [
					input.name,
					{
						value: row.value,
					} as MCPSetupSubmissionValue,
				];
			})
		);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = async event => {
		event.preventDefault();
		event.stopPropagation();

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

		try {
			await onSubmit(server, buildSubmission(), reset);

			if (!unmountingRef.current) {
				requestClose(true);
			}
		} catch (error) {
			if (!unmountingRef.current) {
				setSubmitError(error instanceof Error ? error.message : 'Failed to save MCP setup.');
			}
		} finally {
			if (!unmountingRef.current) {
				setIsSubmitting(false);
			}
		}
	};

	return (
		<>
			<div className="modal-box bg-base-200 max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-3xl overflow-hidden rounded-2xl p-0">
				<div className="app-scrollbar-thin max-h-[calc(100dvh-1rem)] overflow-y-auto p-4 sm:p-6">
					<ModalHeader
						title={`Configure ${server.displayName}`}
						description="Values are stored in Artifact-local installation data. Stored secret values are never displayed."
						onClose={() => {
							requestClose();
						}}
						closeDisabled={isSubmitting}
					/>

					<form noValidate onSubmit={handleSubmit} className="space-y-4" aria-busy={isSubmitting}>
						{server.document?.extension.install.note ? (
							<div className="bg-base-100 rounded-2xl p-3 text-sm">{server.document.extension.install.note}</div>
						) : null}

						{submitError ? (
							<div className="alert alert-error rounded-2xl text-sm">
								<FiAlertCircle size={14} />
								<span>{submitError}</span>
							</div>
						) : null}

						{reset ? (
							<div className="alert alert-warning rounded-2xl text-sm">
								<FiAlertCircle size={14} />
								<span>Reset removes existing installation bindings that are not supplied in this save.</span>
							</div>
						) : null}

						{inputs.map(input => {
							const row = rows[input.name] ?? emptyRow();
							const label = input.declaration.label || input.name;
							const isOAuth = input.declaration.kind === MCPInputKind.OAuthClientCredentials;
							const isSecret = isOAuth || input.declaration.kind === MCPInputKind.Secret;

							return (
								<div key={input.name} className="bg-base-100 rounded-2xl p-4">
									<div className="mb-2 flex items-center justify-between gap-3">
										<div className="min-w-0">
											<div className="truncate font-semibold">
												{label}
												{input.declaration.required ? ' *' : ''}
											</div>
											<div className="text-base-content/60 font-mono text-xs">{input.name}</div>
										</div>
										<span className="badge badge-xs rounded-xl">{inputKindLabel(input.declaration.kind)}</span>
									</div>

									{input.declaration.description ? (
										<p className="text-base-content/70 mb-3 text-xs">{input.declaration.description}</p>
									) : null}

									{isOAuth ? (
										<div className="grid grid-cols-1 gap-3 md:grid-cols-2">
											<input
												type="text"
												className="input w-full rounded-xl"
												placeholder="Client ID"
												value={row.clientID}
												autoComplete="off"
												spellCheck="false"
												disabled={isSubmitting}
												onChange={event => {
													updateRow(input.name, {
														clientID: event.target.value,
													});
												}}
											/>
											<input
												type="password"
												className="input w-full rounded-xl"
												placeholder={
													input.declaration.clientSecretRequired ? 'Client Secret' : 'Client Secret (optional)'
												}
												value={row.clientSecret}
												autoComplete="new-password"
												disabled={isSubmitting}
												onChange={event => {
													updateRow(input.name, {
														clientSecret: event.target.value,
													});
												}}
											/>
										</div>
									) : (
										<input
											type={isSecret ? 'password' : 'text'}
											className="input w-full rounded-xl"
											placeholder={input.declaration.placeholder}
											value={row.value}
											autoComplete={isSecret ? 'new-password' : 'off'}
											spellCheck="false"
											disabled={isSubmitting}
											onChange={event => {
												updateRow(input.name, {
													value: event.target.value,
												});
											}}
										/>
									)}

									<div className="mt-2 flex flex-wrap items-center justify-between gap-2">
										{input.declaration.note ? (
											<span className="text-base-content/60 text-xs">{input.declaration.note}</span>
										) : (
											<span />
										)}
										{input.boundSecretRef || input.boundValue ? (
											<span className="text-base-content/60 text-xs">
												{isSecret ? 'Configured. Leave blank to keep it.' : 'Configured. Leave blank to keep it.'}
											</span>
										) : null}
									</div>
								</div>
							);
						})}

						{server.builtIn ? (
							<label className="label cursor-pointer justify-start gap-3">
								<input
									type="checkbox"
									className="checkbox checkbox-sm"
									checked={reset}
									disabled={isSubmitting}
									onChange={event => {
										setReset(event.target.checked);
									}}
								/>
								<span className="text-sm">Reset existing built-in installation bindings</span>
							</label>
						) : null}

						<ModalActions className="-mx-4 mt-6 -mb-4 sm:-mx-6 sm:-mb-6">
							<button
								type="button"
								className="btn bg-base-300 rounded-xl"
								disabled={isSubmitting}
								onClick={() => {
									requestClose();
								}}
							>
								Cancel
							</button>
							<button type="submit" className="btn btn-primary rounded-xl" disabled={isSubmitting}>
								{isSubmitting ? 'Saving...' : 'Save'}
							</button>
						</ModalActions>
					</form>
				</div>
			</div>
			<ModalBackdrop enabled={false} />
		</>
	);
}

export function MCPServerSetupModal(props: MCPServerSetupModalProps) {
	if (!props.isOpen || !props.server) {
		return null;
	}

	return (
		<ModalDialog isOpen={props.isOpen} onClose={props.onClose} blockCancel>
			<MCPServerSetupModalContent
				key={`${props.server.ref.rootID}:${props.server.ref.artifactID}:${props.server.artifact.revision}`}
				{...props}
				server={props.server}
			/>
		</ModalDialog>
	);
}
