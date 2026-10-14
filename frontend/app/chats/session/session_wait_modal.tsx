import type { SessionWaitingRequest } from '@/spec/session_base';
import { SessionWaitResolution, SessionWaitStatus } from '@/spec/session_base';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

interface SessionWaitModalProps {
	wait: SessionWaitingRequest | undefined;
	isResolving: boolean;
	error?: string;
	onResolve: (resolution: SessionWaitResolution) => Promise<void>;
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
		return JSON.stringify(value, null, 2);
	} catch {
		// oxlint-disable-next-line typescript/no-base-to-string
		return String(value);
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

export function SessionWaitModal({ wait, isResolving, error, onResolve }: SessionWaitModalProps) {
	if (!wait || wait.status !== SessionWaitStatus.Pending) {
		return null;
	}

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

						{error ? <div className="alert alert-error rounded-2xl text-sm">{error}</div> : null}
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
									void onResolve(resolution);
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
