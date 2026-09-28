import { useModalDialogController } from '@/hooks/use_dialog_controller';

import type { AgentCatalogOption } from '@/apis/agent_management';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

interface AgentViewModalProps {
	isOpen: boolean;
	onClose: () => void;
	agent: AgentCatalogOption | null;
}

function AgentViewModalContent({ agent }: Omit<AgentViewModalProps, 'isOpen' | 'onClose'>) {
	const { requestClose } = useModalDialogController();

	if (!agent) {
		return null;
	}

	const artifactIdentity = `${agent.ref.rootID}/${agent.ref.artifactID}`;

	return (
		<>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-2xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<ModalHeader
						title={agent.displayName}
						description={`${agent.agent.name} / revision ${agent.agent.revision}`}
						onClose={() => {
							requestClose();
						}}
					/>

					<div className="mb-4 rounded-2xl border p-4">
						{agent.description ? (
							<p className="text-sm">{agent.description}</p>
						) : (
							<p className="text-sm opacity-70">No description supplied.</p>
						)}

						<div className="mt-3 flex flex-wrap items-center gap-2">
							<span className={`badge badge-sm ${agent.isSelectable ? 'badge-success' : 'badge-warning'}`}>
								{agent.isSelectable ? 'Available' : 'Unavailable'}
							</span>
							{agent.agent.builtIn ? <span className="badge badge-outline badge-sm">Built-in</span> : null}
							{agent.agent.managed ? <span className="badge badge-outline badge-sm">Managed</span> : null}
						</div>

						{!agent.isSelectable && agent.availabilityReason ? (
							<div className="text-warning mt-3 text-xs">{agent.availabilityReason}</div>
						) : null}
					</div>

					<section className="border-base-300 rounded-2xl border p-4">
						<h4 className="mb-3 text-sm font-semibold">Agent metadata</h4>

						<dl className="grid gap-4 sm:grid-cols-2">
							<div>
								<dt className="text-xs font-semibold opacity-70">Artifact</dt>
								<dd className="mt-1 text-sm break-all">
									<code>{artifactIdentity}</code>
								</dd>
							</div>

							<div>
								<dt className="text-xs font-semibold opacity-70">Logical name</dt>
								<dd className="mt-1 text-sm">{agent.agent.name}</dd>
							</div>

							<div>
								<dt className="text-xs font-semibold opacity-70">Artifact state</dt>
								<dd className="mt-1 text-sm">{agent.agent.state}</dd>
							</div>

							<div>
								<dt className="text-xs font-semibold opacity-70">Definition digest</dt>
								<dd className="mt-1 text-sm break-all">
									<code>{agent.agent.definitionDigest ?? 'Not supplied'}</code>
								</dd>
							</div>
						</dl>
					</section>

					<ModalActions className="-mx-6 mt-6 -mb-6">
						<button
							type="button"
							className="btn bg-base-300 rounded-xl"
							onClick={() => {
								requestClose();
							}}
						>
							Close
						</button>
					</ModalActions>
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</>
	);
}

export function AgentViewModal({ isOpen, onClose, agent }: AgentViewModalProps) {
	if (!isOpen || !agent) {
		return null;
	}

	return (
		<ModalDialog isOpen={isOpen} onClose={onClose}>
			<AgentViewModalContent agent={agent} />
		</ModalDialog>
	);
}
