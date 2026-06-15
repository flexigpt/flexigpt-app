import { type SubmitEventHandler, useEffect, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiX } from 'react-icons/fi';

import { ModalBackdrop } from '@/components/modal_backdrop';

interface MCPSettingsModalProps {
	isOpen: boolean;
	initialListenAddr?: string;
	activeListenAddr?: string;
	oauthRedirectURL?: string;
	onClose: () => void;
	onSubmit: (oauthLoopbackListenAddr: string) => Promise<void>;
}

function isLoopbackHost(host: string): boolean {
	const h = host.trim().toLowerCase().replace(/^\[/, '').replace(/\]$/, '');
	if (h === 'localhost' || h === '::1' || h === '0:0:0:0:0:0:0:1') return true;
	const parts = h.split('.').map(Number);
	return parts.length === 4 && parts.every(p => Number.isInteger(p) && p >= 0 && p <= 255) && parts[0] === 127;
}

function validateListenAddr(raw: string): string {
	const value = raw.trim();
	if (!value) return '';
	const idx = value.lastIndexOf(':');
	if (idx <= 0) return 'Use host:port, for example 127.0.0.1:37645.';
	const host = value.slice(0, idx);
	const port = Number(value.slice(idx + 1));
	if (!isLoopbackHost(host)) return 'Host must be loopback (localhost or 127.0.0.1).';
	if (!Number.isInteger(port) || port <= 0 || port > 65535) return 'Port must be 1..65535.';
	return '';
}

function MCPSettingsModalContent({
	initialListenAddr,
	activeListenAddr,
	oauthRedirectURL,
	onClose,
	onSubmit,
}: MCPSettingsModalProps) {
	const [listenAddr, setListenAddr] = useState(initialListenAddr ?? '');
	const [error, setError] = useState('');
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

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();
		e.stopPropagation();
		setSubmitError('');

		const err = validateListenAddr(listenAddr);
		setError(err);
		if (err) return;

		void onSubmit(listenAddr.trim())
			.then(() => {
				requestClose();
			})
			.catch((error: unknown) => {
				setSubmitError(error instanceof Error ? error.message : 'Failed to save MCP settings.');
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
			<div className="modal-box bg-base-200 max-w-2xl rounded-2xl p-0">
				<div className="p-6">
					<div className="mb-4 flex items-center justify-between">
						<h3 className="text-lg font-bold">MCP OAuth Settings</h3>
						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={requestClose}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>

					<form noValidate onSubmit={handleSubmit} className="space-y-8">
						{submitError && (
							<div className="alert alert-error rounded-2xl text-sm">
								<div className="flex items-center gap-2">
									<FiAlertCircle size={14} />
									<span>{submitError}</span>
								</div>
							</div>
						)}

						{oauthRedirectURL && (
							<div className="flex flex-col gap-2">
								<div className="text-base-content/70 mb-1 text-xs font-semibold uppercase">OAuth Redirect URL</div>
								<div className="bg-base-300 rounded-2xl p-3 text-xs break-all">{oauthRedirectURL}</div>
								<p className="text-base-content/60 mt-1 text-xs">
									Register this URL with providers that require a fixed redirect URI.
								</p>
							</div>
						)}

						<div className="flex flex-col gap-2">
							<div className="text-base-content/70 mb-1 text-xs font-semibold uppercase">
								OAuth Loopback Listen Address
							</div>
							<input
								type="text"
								className={`input input-bordered w-full rounded-xl ${error ? 'input-error' : ''}`}
								value={listenAddr}
								spellCheck="false"
								autoComplete="off"
								placeholder="127.0.0.1:37645 (leave blank for a random port)"
								onChange={e => {
									setListenAddr(e.target.value);
									setError(validateListenAddr(e.target.value));
								}}
							/>
							{error && (
								<div className="label">
									<span className="label-text-alt text-error flex items-center gap-1">
										<FiAlertCircle size={12} /> {error}
									</span>
								</div>
							)}
							{activeListenAddr && (
								<div className="label">
									<span className="label-text-alt text-base-content/60 text-xs">
										Currently active: {activeListenAddr}
									</span>
								</div>
							)}
							<p className="text-base-content/60 mt-1 text-xs">
								Changing this address takes effect after restarting FlexiGPT.
							</p>
						</div>

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

export function MCPSettingsModal(props: MCPSettingsModalProps) {
	if (!props.isOpen) return null;
	if (typeof document === 'undefined' || !document.body) return null;
	return createPortal(<MCPSettingsModalContent key="mcp-settings-modal" {...props} />, document.body);
}
