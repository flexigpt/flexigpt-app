import { useState } from 'react';

import type { SessionOpaqueJSON, SessionWaitingRequest } from '@/spec/session_base';
import { SessionWaitResolution, SessionWaitStatus } from '@/spec/session_base';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

interface SessionWaitModalProps {
	wait: SessionWaitingRequest | undefined;
	isResolving: boolean;
	error?: string;
	onResolve: (resolution: SessionWaitResolution, input?: SessionOpaqueJSON) => Promise<void>;
}

interface ActiveSessionWaitModalProps extends Omit<SessionWaitModalProps, 'wait'> {
	wait: SessionWaitingRequest;
}

function formatSummary(value: unknown): string {
	if (value === null || value === undefined) {
		return 'No additional details were supplied.';
	}

	if (typeof value === 'string') {
		try {
			return JSON.stringify(JSON.parse(value), null, 2);
		} catch {
			return value;
		}
	}

	try {
		const serialized = JSON.stringify(value, null, 2);
		return serialized ?? 'No additional details were supplied.';
	} catch {
		return 'Additional details could not be displayed.';
	}
}

function labelForResolution(value: SessionWaitResolution): string {
	switch (value) {
		case SessionWaitResolution.ApproveOnce:
			return 'Approve once';
		case SessionWaitResolution.ApproveAlways:
			return 'Approve for this session';
		case SessionWaitResolution.DenyOnce:
			return 'Deny once';
		case SessionWaitResolution.DenyAlways:
			return 'Deny for this session';
		case SessionWaitResolution.Continue:
			return 'Continue';
		case SessionWaitResolution.Cancel:
			return 'Cancel run';
		default:
			return value;
	}
}

function resolutionAcceptsInput(value: SessionWaitResolution): boolean {
	return (
		value !== SessionWaitResolution.DenyOnce &&
		value !== SessionWaitResolution.DenyAlways &&
		value !== SessionWaitResolution.Cancel
	);
}

export function SessionWaitModal({ wait, isResolving, error, onResolve }: SessionWaitModalProps) {
	if (!wait || wait.status !== SessionWaitStatus.Pending) {
		return null;
	}

	return (
		<SessionWaitModalContent key={wait.id} wait={wait} isResolving={isResolving} error={error} onResolve={onResolve} />
	);
}

function SessionWaitModalContent({ wait, isResolving, error, onResolve }: ActiveSessionWaitModalProps) {
	const [inputText, setInputText] = useState('');
	const [inputError, setInputError] = useState<string | undefined>();

	const resolve = (resolution: SessionWaitResolution) => {
		let input: SessionOpaqueJSON = undefined;

		if (resolutionAcceptsInput(resolution) && inputText.trim()) {
			try {
				input = JSON.parse(inputText) as SessionOpaqueJSON;
			} catch {
				setInputError('Response input must be valid JSON.');
				return;
			}
		}

		setInputError(undefined);
		void onResolve(resolution, input);
	};

	return (
		<ModalDialog isOpen={true} isBusy={isResolving} onClose={() => undefined} blockCancel>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-3xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<ModalHeader title={`Run is waiting: ${wait.kind}`} closeDisabled={true} />

					<div className="space-y-4">
						{wait.expiresAt ? (
							<div className="text-base-content/70 text-sm">Expires: {new Date(wait.expiresAt).toLocaleString()}</div>
						) : null}

						<pre className="bg-base-300 max-h-80 overflow-auto rounded-2xl p-3 text-xs whitespace-pre-wrap">
							{formatSummary(wait.summary)}
						</pre>

						{wait.inputSchema !== undefined && wait.inputSchema !== null ? (
							<div className="space-y-2">
								<div className="text-sm font-semibold">Response input</div>
								<div className="text-base-content/70 text-xs">
									This wait accepts optional structured input. Provide JSON when the selected resolution requires a
									response payload.
								</div>
								<pre className="bg-base-300 max-h-48 overflow-auto rounded-2xl p-3 text-xs whitespace-pre-wrap">
									{formatSummary(wait.inputSchema)}
								</pre>
								<textarea
									className={`textarea w-full rounded-2xl font-mono text-xs ${inputError ? 'textarea-error' : ''}`}
									rows={6}
									value={inputText}
									onChange={event => {
										setInputText(event.currentTarget.value);
										if (inputError) {
											setInputError(undefined);
										}
									}}
									placeholder={'{\n  "value": "response"\n}'}
									spellCheck={false}
									disabled={isResolving}
								/>
							</div>
						) : null}

						{error ? <div className="alert alert-error rounded-2xl text-sm">{error}</div> : null}
						{inputError ? <div className="alert alert-error rounded-2xl text-sm">{inputError}</div> : null}
					</div>

					<ModalActions className="-mx-6 mt-6 -mb-6">
						{(wait.allowedResolutions ?? []).map(resolution => (
							<button
								key={resolution}
								type="button"
								className={`btn rounded-xl ${
									resolution.includes('deny') || resolution === SessionWaitResolution.Cancel
										? 'btn-error'
										: resolution.includes('approve') || resolution === SessionWaitResolution.Continue
											? 'btn-primary'
											: 'bg-base-300'
								}`}
								disabled={isResolving}
								onClick={() => {
									resolve(resolution);
								}}
							>
								{labelForResolution(resolution)}
							</button>
						))}
					</ModalActions>
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</ModalDialog>
	);
}
