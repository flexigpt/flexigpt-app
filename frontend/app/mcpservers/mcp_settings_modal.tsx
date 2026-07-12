import type { SubmitEventHandler } from 'react';
import { useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiX } from 'react-icons/fi';

import { useDialogController } from '@/hooks/use_dialog_controller';

import { ModalBackdrop } from '@/components/modal/modal_backdrop';

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
	if (h === 'localhost' || h === '::1' || h === '0:0:0:0:0:0:0:1') {
		return true;
	}
	const parts = h.split('.').map(Number);
	return parts.length === 4 && parts.every(p => Number.isInteger(p) && p >= 0 && p <= 255) && parts[0] === 127;
}

function splitListenAddr(value: string): { host: string; port: string } | undefined {
	if (value.startsWith('[')) {
		const end = value.indexOf(']');
		if (end <= 1 || value[end + 1] !== ':') {
			return undefined;
		}
		return {
			host: value.slice(1, end),
			port: value.slice(end + 2),
		};
	}

	const parts = value.split(':');
	if (parts.length !== 2) {
		return undefined;
	}
	return { host: parts[0], port: parts[1] };
}

function validateListenAddr(raw: string): string {
	const value = raw.trim();
	if (!value) {
		return '';
	}
	const parsed = splitListenAddr(value);
	if (!parsed) {
		return 'Use host:port, for IPv6 use [::1]:37645.';
	}
	const port = Number(parsed.port);
	if (!isLoopbackHost(parsed.host)) {
		return 'Host must be loopback (localhost, 127.0.0.1, or ::1).';
	}
	if (!Number.isInteger(port) || port <= 0 || port > 65535) {
		return 'Port must be 1..65535.';
	}
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
	const [errorState, setErrorState] = useState('');
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);

	const { dialogRef, requestClose, handleClose, handleCancel, unmountingRef } = useDialogController({
		onClose,
		blockCancel: true,
		isBusy: isSubmitting,
	});

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = async e => {
		e.preventDefault();
		e.stopPropagation();

		if (isSubmitting) {
			return;
		}

		setSubmitError('');

		const err = validateListenAddr(listenAddr);
		setErrorState(err);
		if (err) {
			return;
		}

		setIsSubmitting(true);
		try {
			await onSubmit(listenAddr.trim());
			if (!unmountingRef.current) {
				requestClose(true);
			}
		} catch (error) {
			if (!unmountingRef.current) {
				setSubmitError(error instanceof Error ? error.message : 'Failed to save MCP settings.');
			}
		} finally {
			if (!unmountingRef.current) {
				setIsSubmitting(false);
			}
		}
	};

	return (
		<dialog ref={dialogRef} className="modal" onClose={handleClose} onCancel={handleCancel}>
			<div className="modal-box bg-base-200 max-w-2xl rounded-2xl p-0">
				<div className="p-6">
					<div className="mb-4 flex items-center justify-between">
						<h3 className="text-lg font-bold">MCP OAuth Settings</h3>
						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={() => {
								requestClose();
							}}
							aria-label="Close"
							disabled={isSubmitting}
						>
							<FiX size={12} />
						</button>
					</div>

					<form noValidate onSubmit={handleSubmit} className="space-y-8" aria-busy={isSubmitting}>
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
								className={`input w-full rounded-xl ${errorState ? 'input-error' : ''}`}
								value={listenAddr}
								spellCheck="false"
								autoComplete="off"
								placeholder="127.0.0.1:37645 (leave blank for a random port)"
								onChange={e => {
									setListenAddr(e.target.value);
									setErrorState(validateListenAddr(e.target.value));
								}}
							/>
							{errorState && (
								<div className="label">
									<span className="text-error flex items-center gap-1">
										<FiAlertCircle size={12} /> {errorState}
									</span>
								</div>
							)}
							{activeListenAddr && (
								<div className="label">
									<span className="text-base-content/60 text-xs">Currently active: {activeListenAddr}</span>
								</div>
							)}
							<p className="text-base-content/60 mt-1 text-xs">
								Changing this address takes effect after restarting FlexiGPT.
							</p>
						</div>

						<div className="modal-action">
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
						</div>
					</form>
				</div>
			</div>
			<ModalBackdrop enabled={false} />
		</dialog>
	);
}

export function MCPSettingsModal(props: MCPSettingsModalProps) {
	if (!props.isOpen) {
		return null;
	}
	if (typeof document === 'undefined' || !document.body) {
		return null;
	}
	return createPortal(<MCPSettingsModalContent key="mcp-settings-modal" {...props} />, document.body);
}
