import { type SubmitEventHandler, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiX } from 'react-icons/fi';

import {
	type MCPServerConfig,
	type MCPServerSetupInput,
	MCPServerSetupInputKind,
	type MCPServerSetupInputValue,
} from '@/spec/mcp';

import { ModalBackdrop } from '@/components/modal_backdrop';

import { getMCPSetupInputKindLabel, isMCPSetupInputConfigured } from '@/mcpservers/lib/mcp_server_utils';

interface MCPServerSetupModalProps {
	isOpen: boolean;
	server: MCPServerConfig | null;
	onClose: () => void;
	onSubmit: (inputValues: Record<string, MCPServerSetupInputValue>, reset: boolean) => Promise<void>;
}

type RowState = {
	value: string;
	clientID: string;
	clientSecret: string;
};

function emptyRow(): RowState {
	return { value: '', clientID: '', clientSecret: '' };
}

function MCPServerSetupModalContent({
	server,
	onClose,
	onSubmit,
}: MCPServerSetupModalProps & { server: MCPServerConfig }) {
	const inputs = useMemo(() => server.setup?.inputs ?? [], [server.setup]);

	const [rows, setRows] = useState<Record<string, RowState>>(() => {
		const init: Record<string, RowState> = {};
		for (const input of inputs) init[input.id] = emptyRow();
		return init;
	});
	const [reset, setReset] = useState(false);
	const [submitError, setSubmitError] = useState('');

	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const isUnmountingRef = useRef(false);

	useEffect(() => {
		const dialog = dialogRef.current;
		if (!dialog) return;
		if (!dialog.open) {
			try {
				dialog.showModal();
			} catch {
				// keep rendering safely
			}
		}
		return () => {
			isUnmountingRef.current = true;
			if (dialog.open) dialog.close();
		};
	}, []);

	const requestClose = () => {
		const dialog = dialogRef.current;
		if (dialog?.open) {
			dialog.close();
			return;
		}
		onClose();
	};

	const handleDialogClose = () => {
		if (isUnmountingRef.current) return;
		onClose();
	};

	const updateRow = (id: string, patch: Partial<RowState>) => {
		setRows(prev => ({ ...prev, [id]: { ...prev[id], ...patch } }));
	};

	const effectiveConfigured = (input: MCPServerSetupInput): boolean =>
		reset ? false : isMCPSetupInputConfigured(server, input);

	const validate = (): string => {
		for (const input of inputs) {
			if (!input.required) continue;
			if (effectiveConfigured(input)) continue;

			const row = rows[input.id] ?? emptyRow();
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

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();
		e.stopPropagation();
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

		void onSubmit(values, reset)
			.then(() => {
				requestClose();
			})
			.catch((error: unknown) => {
				setSubmitError(error instanceof Error ? error.message : 'Failed to apply setup.');
			});
	};

	return (
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleDialogClose}
			onCancel={e => {
				e.preventDefault();
			}}
		>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-3xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<div className="mb-4 flex items-center justify-between">
						<div>
							<h3 className="text-lg font-bold">Configure {server.displayName}</h3>
							<p className="text-base-content/70 mt-1 text-sm">
								Provide the values this MCP server needs. Stored secrets are never displayed.
							</p>
						</div>
						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={requestClose}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>

					<form noValidate onSubmit={handleSubmit} className="space-y-4">
						{server.setup?.note && <div className="bg-base-100 rounded-2xl p-3 text-sm">{server.setup.note}</div>}

						{submitError && (
							<div className="alert alert-error rounded-2xl text-sm">
								<div className="flex items-center gap-2">
									<FiAlertCircle size={14} />
									<span>{submitError}</span>
								</div>
							</div>
						)}

						{inputs.map(input => {
							const row = rows[input.id] ?? emptyRow();
							const configured = isMCPSetupInputConfigured(server, input);
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
												className="input input-bordered w-full rounded-xl"
												placeholder="Client ID"
												value={row.clientID}
												autoComplete="off"
												spellCheck="false"
												onChange={e => {
													updateRow(input.id, { clientID: e.target.value });
												}}
											/>
											<input
												type="password"
												className="input input-bordered w-full rounded-xl"
												placeholder={
													input.oauthClientCredentials?.clientSecretRequired
														? 'Client Secret'
														: 'Client Secret (optional)'
												}
												value={row.clientSecret}
												autoComplete="new-password"
												onChange={e => {
													updateRow(input.id, { clientSecret: e.target.value });
												}}
											/>
										</div>
									) : (
										<input
											type={isSecret ? 'password' : 'text'}
											className="input input-bordered w-full rounded-xl"
											placeholder={input.placeholder}
											value={row.value}
											autoComplete={isSecret ? 'new-password' : 'off'}
											spellCheck="false"
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
								/>
								<span className="label-text text-sm">Reset existing setup before applying</span>
							</label>
						)}

						<div className="modal-action">
							<button type="button" className="btn bg-base-300 rounded-xl" onClick={requestClose}>
								Cancel
							</button>
							<button type="submit" className="btn btn-primary rounded-xl">
								Save
							</button>
						</div>
					</form>
				</div>
			</div>
			<ModalBackdrop enabled={false} />
		</dialog>
	);
}

export function MCPServerSetupModal(props: MCPServerSetupModalProps) {
	if (!props.isOpen || !props.server) return null;
	if (typeof document === 'undefined' || !document.body) return null;

	const remountKey = `${props.server.bundleID}:${props.server.id}:${props.server.modifiedAt}`;
	return createPortal(<MCPServerSetupModalContent key={remountKey} {...props} server={props.server} />, document.body);
}
