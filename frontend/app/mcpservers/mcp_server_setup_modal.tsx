import type { SubmitEventHandler } from 'react';
import { useMemo, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle } from 'react-icons/fi';

import type { MCPServerConfig, MCPServerSetupInput, MCPServerSetupInputValue } from '@/spec/mcp';
import { MCPServerSetupInputKind } from '@/spec/mcp';

import { validateHTTPHeaderName, validateHTTPURLSecurity } from '@/lib/http_input_utils';

import { useDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalHeader } from '@/components/modal/modal_header';

import { getMCPSetupInputKindLabel, isMCPSetupInputConfigured } from '@/mcpservers/lib/mcp_server_utils';

interface MCPServerSetupModalProps {
	isOpen: boolean;
	server: MCPServerConfig | null;
	onClose: () => void;
	onSubmit: (inputValues: Record<string, MCPServerSetupInputValue>, reset: boolean) => Promise<void>;
}

interface RowState {
	value: string;
	clientID: string;
	clientSecret: string;
}

function emptyRow(): RowState {
	return { value: '', clientID: '', clientSecret: '' };
}

const ENV_NAME_RE = /^[A-Za-z_][A-Za-z0-9_]*$/;

function validateSetupRow(input: MCPServerSetupInput, row: RowState): string | undefined {
	const label = input.label || input.id;

	switch (input.kind) {
		case MCPServerSetupInputKind.OAuthClientCredentials:
			if (row.clientID.trim() || row.clientSecret) {
				if (!row.clientID.trim()) {
					return `"${label}" requires a Client ID when credentials are supplied.`;
				}
				if (row.clientID !== row.clientID.trim()) {
					return `"${label}" Client ID must not have leading or trailing whitespace.`;
				}
				if (row.clientSecret && !row.clientSecret.trim()) {
					return `"${label}" Client Secret must not contain only whitespace.`;
				}
			}
			return undefined;

		case MCPServerSetupInputKind.StreamableHTTPURL:
			if (!row.value.trim()) {
				return undefined;
			}
			return validateHTTPURLSecurity(row.value.trim(), `"${label}"`);

		case MCPServerSetupInputKind.ClientIDMetadataDocumentURL:
			if (!row.value.trim()) {
				return undefined;
			}
			try {
				const url = new URL(row.value.trim());
				if (url.protocol !== 'https:') {
					return `"${label}" must use https.`;
				}
				if (url.username || url.password) {
					return `"${label}" must not include embedded credentials.`;
				}
				if (!url.pathname || url.pathname === '/') {
					return `"${label}" must include a document path.`;
				}
				if (url.hash) {
					return `"${label}" must not include a fragment.`;
				}
			} catch {
				return `"${label}" must be a valid HTTPS URL.`;
			}
			return undefined;

		case MCPServerSetupInputKind.HTTPHeader: {
			const headerName = input.httpHeader?.headerName ?? '';
			const headerNameError = validateHTTPHeaderName(headerName, `"${label}" header name`);
			if (headerNameError) {
				return headerNameError;
			}
			const value = `${input.httpHeader?.valuePrefix ?? ''}${row.value}${input.httpHeader?.valueSuffix ?? ''}`;
			return /[\r\n\u0000]/.test(value) ? `"${label}" must not contain CR, LF, or NUL.` : undefined;
		}

		case MCPServerSetupInputKind.StdioEnv: {
			const envName = input.stdioEnv?.envName ?? '';
			if (!ENV_NAME_RE.test(envName)) {
				return `"${label}" has an invalid environment variable name.`;
			}
			const value = `${input.stdioEnv?.valuePrefix ?? ''}${row.value}${input.stdioEnv?.valueSuffix ?? ''}`;
			return value.includes('\u0000') ? `"${label}" must not contain NUL.` : undefined;
		}

		default:
			return `"${label}" uses an unsupported setup input kind.`;
	}
}

function MCPServerSetupModalContent({
	server,
	onClose,
	onSubmit,
}: MCPServerSetupModalProps & { server: MCPServerConfig }) {
	const inputs = useMemo(() => server.setup?.inputs ?? [], [server.setup]);

	const [rows, setRows] = useState<Record<string, RowState>>(() => {
		const init: Record<string, RowState> = {};
		for (const input of inputs) {
			init[input.id] = emptyRow();
		}
		return init;
	});
	const [reset, setReset] = useState(false);
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);

	const { dialogRef, requestClose, handleClose, handleCancel, unmountingRef } = useDialogController({
		onClose,
		blockCancel: true,
		isBusy: isSubmitting,
	});

	const updateRow = (id: string, patch: Partial<RowState>) => {
		setRows(prev => ({ ...prev, [id]: { ...prev[id], ...patch } }));
	};

	const effectiveConfigured = (input: MCPServerSetupInput): boolean =>
		reset ? false : isMCPSetupInputConfigured(server, input);

	const validate = (): string => {
		if (new Set(inputs.map(input => input.id)).size !== inputs.length) {
			return 'Server setup metadata contains duplicate input IDs.';
		}

		for (const input of inputs) {
			const row = rows[input.id] ?? emptyRow();

			if (!input.required) {
				const rowError = validateSetupRow(input, row);
				if (rowError) {
					return rowError;
				}
				continue;
			}
			if (effectiveConfigured(input)) {
				const rowError = validateSetupRow(input, row);
				if (rowError) {
					return rowError;
				}
				continue;
			}

			if (input.kind === MCPServerSetupInputKind.OAuthClientCredentials) {
				if (!row.clientID.trim()) {
					return `"${input.label || input.id}" requires a Client ID.`;
				}
				if (input.oauthClientCredentials?.clientSecretRequired && !row.clientSecret.trim()) {
					return `"${input.label || input.id}" requires a Client Secret.`;
				}
			} else if (!row.value.trim()) {
				return `"${input.label || input.id}" is required.`;
			}

			const rowError = validateSetupRow(input, row);
			if (rowError) {
				return rowError;
			}
		}
		return '';
	};

	const buildInputValues = (): Record<string, MCPServerSetupInputValue> => {
		const out: Record<string, MCPServerSetupInputValue> = {};
		for (const input of inputs) {
			const row = rows[input.id] ?? emptyRow();
			if (input.kind === MCPServerSetupInputKind.OAuthClientCredentials) {
				if (row.clientID.trim() || row.clientSecret.trim()) {
					out[input.id] = { clientID: row.clientID.trim(), clientSecret: row.clientSecret };
				}
			} else if (row.value.trim()) {
				out[input.id] = { value: row.value };
			}
		}
		return out;
	};

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = async e => {
		e.preventDefault();
		e.stopPropagation();

		if (isSubmitting) {
			return;
		}

		setSubmitError('');

		const err = validate();
		if (err) {
			setSubmitError(err);
			return;
		}

		const values = buildInputValues();
		if (Object.keys(values).length === 0 && !reset) {
			setSubmitError('No changes to apply.');
			return;
		}

		setIsSubmitting(true);
		try {
			await onSubmit(values, reset);
			if (!unmountingRef.current) {
				requestClose(true);
			}
		} catch (error) {
			if (!unmountingRef.current) {
				setSubmitError(error instanceof Error ? error.message : 'Failed to apply setup.');
			}
		} finally {
			if (!unmountingRef.current) {
				setIsSubmitting(false);
			}
		}
	};

	return (
		<dialog ref={dialogRef} className="modal" onClose={handleClose} onCancel={handleCancel}>
			<div className="modal-box bg-base-200 max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-3xl overflow-hidden rounded-2xl p-0">
				<div className="app-scrollbar-thin max-h-[calc(100dvh-1rem)] overflow-y-auto p-4 sm:p-6">
					<ModalHeader
						title={`Configure ${server.displayName}`}
						description="Provide the values this MCP server needs. Stored secrets are never displayed."
						onClose={() => {
							requestClose();
						}}
						closeDisabled={isSubmitting}
					/>

					<form noValidate onSubmit={handleSubmit} className="space-y-4" aria-busy={isSubmitting}>
						{server.setup?.note && <div className="bg-base-100 rounded-2xl p-3 text-sm">{server.setup.note}</div>}

						{submitError && (
							<div className="alert alert-error rounded-2xl text-sm">
								<div className="flex items-center gap-2">
									<FiAlertCircle size={14} />
									<span>{submitError}</span>
								</div>
							</div>
						)}

						{reset ? (
							<div className="alert alert-warning rounded-2xl text-sm">
								<div className="flex items-center gap-2">
									<FiAlertCircle size={14} />
									<span>Reset clears existing setup values that are not supplied in this save.</span>
								</div>
							</div>
						) : null}

						{inputs.map(input => {
							const row = rows[input.id] ?? emptyRow();
							const configured = effectiveConfigured(input);
							const isOAuth = input.kind === MCPServerSetupInputKind.OAuthClientCredentials;
							const isSecret =
								isOAuth ||
								(input.kind === MCPServerSetupInputKind.HTTPHeader && Boolean(input.httpHeader?.secret)) ||
								(input.kind === MCPServerSetupInputKind.StdioEnv && Boolean(input.stdioEnv?.secret));

							return (
								<div key={input.id} className="bg-base-100 rounded-2xl p-4">
									<div className="mb-2 flex items-center justify-between gap-2">
										<div className="font-semibold">
											{input.label || input.id}
											{input.required ? ' *' : ''}
										</div>
										<span className="badge badge-xs rounded-xl">{getMCPSetupInputKindLabel(input.kind)}</span>
									</div>

									{input.description && <p className="text-base-content/70 mb-2 text-xs">{input.description}</p>}

									{isOAuth ? (
										<div className="grid grid-cols-1 gap-2 md:grid-cols-2">
											<input
												className="input w-full rounded-xl"
												placeholder="Client ID"
												value={row.clientID}
												autoComplete="off"
												spellCheck="false"
												disabled={isSubmitting}
												onChange={e => {
													updateRow(input.id, { clientID: e.target.value });
												}}
											/>
											<input
												type="password"
												className="input w-full rounded-xl"
												placeholder={
													input.oauthClientCredentials?.clientSecretRequired
														? 'Client Secret'
														: 'Client Secret (optional)'
												}
												value={row.clientSecret}
												autoComplete="new-password"
												disabled={isSubmitting}
												onChange={e => {
													updateRow(input.id, { clientSecret: e.target.value });
												}}
											/>
										</div>
									) : (
										<input
											type={isSecret ? 'password' : 'text'}
											className="input w-full rounded-xl"
											placeholder={input.placeholder}
											value={row.value}
											autoComplete={isSecret ? 'new-password' : 'off'}
											spellCheck="false"
											disabled={isSubmitting}
											onChange={e => {
												updateRow(input.id, { value: e.target.value });
											}}
										/>
									)}

									<div className="mt-1 flex items-center justify-between">
										{input.note && <span className="text-base-content/60 text-xs">{input.note}</span>}
										{configured && (
											<span className="text-base-content/60 text-xs">
												{isSecret ? 'Already configured. Leave blank to keep it.' : 'Already configured.'}
											</span>
										)}
									</div>
								</div>
							);
						})}

						{server.isBuiltIn && (
							<label className="label cursor-pointer justify-start gap-3">
								<input
									type="checkbox"
									className="checkbox checkbox-sm"
									checked={reset}
									onChange={e => {
										setReset(e.target.checked);
									}}
									disabled={isSubmitting}
								/>
								<span className="text-sm">Reset existing setup before applying</span>
							</label>
						)}

						<ModalActions className="-mx-4 mt-6 -mb-4 sm:-mx-6 sm:-mb-6">
							<button
								type="button"
								className="btn bg-base-300 rounded-xl"
								onClick={() => {
									requestClose();
								}}
								disabled={isSubmitting}
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
		</dialog>
	);
}

export function MCPServerSetupModal(props: MCPServerSetupModalProps) {
	if (!props.isOpen || !props.server) {
		return null;
	}
	if (typeof document === 'undefined' || !document.body) {
		return null;
	}

	const remountKey = `${props.server.bundleID}:${props.server.id}:${props.server.modifiedAt}`;
	return createPortal(<MCPServerSetupModalContent key={remountKey} {...props} server={props.server} />, document.body);
}
