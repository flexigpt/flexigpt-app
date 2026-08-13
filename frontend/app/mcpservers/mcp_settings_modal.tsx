import type { SubmitEventHandler } from 'react';
import { useState } from 'react';

import { FiAlertCircle } from 'react-icons/fi';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

interface MCPSettingsModalProps {
	isOpen: boolean;
	initialListenAddr?: string;
	activeListenAddr?: string;
	oauthRedirectURL?: string;
	onClose: () => void;
	onSubmit: (oauthLoopbackListenAddr: string) => Promise<void>;
}

function isLoopbackHost(host: string): boolean {
	const normalized = host.trim().toLocaleLowerCase().replace(/^\[/, '').replace(/\]$/, '');

	if (normalized === 'localhost' || normalized === '::1' || normalized === '0:0:0:0:0:0:0:1') {
		return true;
	}

	const octets = normalized.split('.').map(Number);
	return (
		octets.length === 4 &&
		octets.every(octet => Number.isInteger(octet) && octet >= 0 && octet <= 255) &&
		octets[0] === 127
	);
}

function splitListenAddress(value: string): { host: string; port: string } | undefined {
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

	return {
		host: parts[0],
		port: parts[1],
	};
}

function validateListenAddress(raw: string): string {
	const value = raw.trim();

	if (!value) {
		return '';
	}

	const parsed = splitListenAddress(value);

	if (!parsed) {
		return 'Use host:port. For IPv6, use [::1]:37645.';
	}

	const port = Number(parsed.port);

	if (!isLoopbackHost(parsed.host)) {
		return 'Host must be localhost, 127.0.0.1, or ::1.';
	}

	if (!Number.isInteger(port) || port <= 0 || port > 65535) {
		return 'Port must be between 1 and 65535.';
	}

	return '';
}

function MCPSettingsModalContent({
	initialListenAddr,
	activeListenAddr,
	oauthRedirectURL,
	onSubmit,
}: Omit<MCPSettingsModalProps, 'isOpen' | 'onClose'>) {
	const [listenAddress, setListenAddress] = useState(initialListenAddr ?? '');
	const [validationError, setValidationError] = useState('');
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);
	const { requestClose, unmountingRef } = useModalDialogController();

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = async event => {
		event.preventDefault();
		event.stopPropagation();

		if (isSubmitting) {
			return;
		}

		const error = validateListenAddress(listenAddress);
		setValidationError(error);
		setSubmitError('');

		if (error) {
			return;
		}

		setIsSubmitting(true);

		try {
			await onSubmit(listenAddress.trim());

			if (!unmountingRef.current) {
				requestClose(true);
			}
		} catch (cause) {
			if (!unmountingRef.current) {
				setSubmitError(cause instanceof Error ? cause.message : 'Failed to save MCP OAuth settings.');
			}
		} finally {
			if (!unmountingRef.current) {
				setIsSubmitting(false);
			}
		}
	};

	return (
		<>
			<div className="modal-box bg-base-200 max-w-2xl rounded-2xl p-0">
				<div className="p-6">
					<ModalHeader
						title="MCP OAuth Settings"
						onClose={() => {
							requestClose();
						}}
						closeDisabled={isSubmitting}
					/>

					<form noValidate onSubmit={handleSubmit} className="space-y-8" aria-busy={isSubmitting}>
						{submitError ? (
							<div className="alert alert-error rounded-2xl text-sm">
								<FiAlertCircle size={14} />
								<span>{submitError}</span>
							</div>
						) : null}

						{oauthRedirectURL ? (
							<div>
								<div className="text-base-content/70 mb-1 text-xs font-semibold uppercase">OAuth Redirect URL</div>
								<div className="bg-base-300 rounded-2xl p-3 text-xs break-all">{oauthRedirectURL}</div>
								<p className="text-base-content/60 mt-2 text-xs">
									Register this URL with providers that require a fixed loopback redirect URI.
								</p>
							</div>
						) : null}

						<div>
							<div className="text-base-content/70 mb-1 text-xs font-semibold uppercase">
								OAuth Loopback Listen Address
							</div>
							<input
								type="text"
								className={`input w-full rounded-xl ${validationError ? 'input-error' : ''}`}
								value={listenAddress}
								placeholder="127.0.0.1:37645, or leave blank for a random port"
								spellCheck="false"
								autoComplete="off"
								disabled={isSubmitting}
								onChange={event => {
									setListenAddress(event.target.value);
									setValidationError(validateListenAddress(event.target.value));
								}}
							/>
							{validationError ? (
								<div className="label">
									<span className="text-error flex items-center gap-1">
										<FiAlertCircle size={12} />
										{validationError}
									</span>
								</div>
							) : null}
							{activeListenAddr ? (
								<p className="text-base-content/60 mt-2 text-xs">Currently active: {activeListenAddr}</p>
							) : null}
							<p className="text-base-content/60 mt-2 text-xs">
								Changing this address takes effect after restarting FlexiGPT.
							</p>
						</div>

						<ModalActions className="-mx-6 mt-6 -mb-6">
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

export function MCPSettingsModal(props: MCPSettingsModalProps) {
	if (!props.isOpen) {
		return null;
	}

	return (
		<ModalDialog isOpen={props.isOpen} onClose={props.onClose} blockCancel>
			<MCPSettingsModalContent key="mcp-settings-modal" {...props} />
		</ModalDialog>
	);
}
