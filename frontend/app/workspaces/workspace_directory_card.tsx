import { FiEye, FiTrash2 } from 'react-icons/fi';

import type { WorkspaceDirectoryListItem } from '@/spec/workspace';

import { ManagementPluginCard } from '@/components/managementui/management_plugin_card';
import { StatusBadge } from '@/components/managementui/status_badge';

interface WorkspaceDirectoryCardProps {
	directory: WorkspaceDirectoryListItem;
	isLoading: boolean;
	error?: string;
	onManage: () => void;
	onRequestRemove: () => void;
}

export function WorkspaceDirectoryCard({
	directory,
	isLoading,
	error,
	onManage,
	onRequestRemove,
}: WorkspaceDirectoryCardProps) {
	return (
		<ManagementPluginCard
			title={directory.rootDisplayName}
			description="Repository Workspace"
			status={
				<>
					<StatusBadge tone={directory.enabled ? 'success' : 'warning'}>
						{directory.enabled ? 'Enabled' : 'Paused'}
					</StatusBadge>
					<StatusBadge>Repository</StatusBadge>
				</>
			}
			actions={
				<>
					<button type="button" className="btn btn-sm btn-primary rounded-xl" disabled={isLoading} onClick={onManage}>
						<FiEye size={15} />
						<span>{isLoading ? 'Opening...' : 'Manage'}</span>
					</button>
					<button
						type="button"
						className="btn btn-sm btn-ghost text-error rounded-xl"
						disabled={isLoading}
						onClick={onRequestRemove}
					>
						<FiTrash2 size={15} />
						<span>Remove</span>
					</button>
				</>
			}
		>
			<div className="text-base-content/60 mt-2 text-xs">
				Open Manage to see the exact instruction files, reference files, Skills, and MCP servers included from this
				repository.
			</div>

			{error ? <div className="alert alert-warning mt-3 rounded-2xl text-sm">{error}</div> : null}
		</ManagementPluginCard>
	);
}
