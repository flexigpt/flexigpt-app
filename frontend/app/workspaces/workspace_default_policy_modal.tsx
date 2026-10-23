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
					title="Automatic Workspace Setup"
					description="This setup is used when a repository has no workspace.yaml file. It detects common instruction files, documentation, Skills, and integrations."
					onClose={onClose}
				/>

				<div className="app-scrollbar-thin min-h-0 flex-1 space-y-4 overflow-y-auto p-4 sm:p-6">
					{error ? (
						<ManagementResourceError
							title="Automatic Workspace setup could not be loaded"
							error={error}
							isRetrying={isRefreshing}
							onRetry={reloadOrThrow}
						/>
					) : null}

					{isLoading && !policy ? <Loader text="Loading default Workspace policy..." /> : null}

					{policy ? (
						<>
							<ModalSection title="Customize this repository">
								<ul className="text-base-content/70 list-disc space-y-1 pl-5 text-sm">
									<li>Copy this setup to `workspace.yaml` in the repository root.</li>
									<li>Choose the files, Skills, and integrations the repository should add.</li>
									<li>Refresh the repository in Workspace management after saving it.</li>
									<li>
										Add `*.workspace.yaml`, `*.workspace.yml`, or `*.workspace.json` to offer multiple Workspace setups.
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
