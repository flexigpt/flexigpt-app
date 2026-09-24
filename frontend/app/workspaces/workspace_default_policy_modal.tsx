import { useCallback, useState } from 'react';
import { FiCheck, FiCopy } from 'react-icons/fi';

import type { WorkspaceDefaultPolicyView } from '@/spec/workspace';

import { throwIfAborted } from '@/lib/async_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { workspaceManagementAPI } from '@/apis/baseapi';

import { Loader } from '@/components/loader';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

interface WorkspaceDefaultPolicyModalProps {
	isOpen: boolean;
	onClose: () => void;
}

function WorkspaceDefaultPolicyModalContent({ onClose }: Pick<WorkspaceDefaultPolicyModalProps, 'onClose'>) {
	const loadPolicy = useCallback(async (signal: AbortSignal): Promise<WorkspaceDefaultPolicyView> => {
		const policy = await workspaceManagementAPI.getWorkspaceDefaultPolicy();
		throwIfAborted(signal);
		return policy;
	}, []);

	const {
		data: policy,
		error,
		isLoading,
		isRefreshing,
		reloadOrThrow,
	} = useAsyncResource(loadPolicy, {
		initialData: null as WorkspaceDefaultPolicyView | null,
	});
	const [copied, setCopied] = useState(false);

	const copy = async () => {
		if (!policy) {
			return;
		}

		await navigator.clipboard.writeText(policy.yaml);
		setCopied(true);
		window.setTimeout(() => {
			setCopied(false);
		}, 1_500);
	};

	return (
		<ModalDialog isOpen onClose={onClose} blockCancel>
			<div className="modal-box bg-base-200 flex max-h-[85vh] w-[calc(100%-1rem)] max-w-5xl flex-col overflow-hidden rounded-2xl p-0">
				<ModalHeader
					title="Default Workspace Policy"
					description="This policy is applied only when the selected directory has no Workspace manifest. Copy it into the repository as workspace.yaml to customize it."
					onClose={onClose}
				/>

				<div className="app-scrollbar-thin min-h-0 flex-1 space-y-4 overflow-y-auto p-4 sm:p-6">
					{error ? (
						<ManagementResourceError
							title="Default Workspace policy could not be loaded"
							error={error}
							isRetrying={isRefreshing}
							onRetry={reloadOrThrow}
						/>
					) : null}

					{isLoading && !policy ? <Loader text="Loading default Workspace policy..." /> : null}

					{policy ? (
						<>
							<ModalSection title="How to customize a repository">
								<ul className="text-base-content/70 list-disc space-y-1 pl-5 text-sm">
									<li>Copy this document to `workspace.yaml` in the repository root.</li>
									<li>Edit its members to select the declarations and source-backed Text you want.</li>
									<li>Refresh the Workspace directory after saving the manifest.</li>
									<li>
										Add `*.workspace.yaml`, `*.workspace.yml`, or `*.workspace.json` files to expose multiple peer
										Workspaces for one repository.
									</li>
								</ul>
							</ModalSection>

							<pre className="bg-base-100 max-h-[50vh] overflow-auto rounded-2xl p-4 text-xs whitespace-pre-wrap">
								{policy.yaml}
							</pre>
						</>
					) : null}
				</div>

				<ModalActions>
					<button type="button" className="btn btn-ghost rounded-xl" onClick={onClose}>
						Close
					</button>
					<button type="button" className="btn btn-primary rounded-xl" disabled={!policy} onClick={() => void copy()}>
						{copied ? <FiCheck size={14} /> : <FiCopy size={14} />}
						{copied ? 'Copied' : 'Copy workspace.yaml'}
					</button>
				</ModalActions>
			</div>
		</ModalDialog>
	);
}

export function WorkspaceDefaultPolicyModal({ isOpen, onClose }: WorkspaceDefaultPolicyModalProps) {
	if (!isOpen) {
		return null;
	}

	return <WorkspaceDefaultPolicyModalContent onClose={onClose} />;
}
