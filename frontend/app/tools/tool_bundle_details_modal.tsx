import type { ToolBundle } from '@/spec/tool';

import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';

interface ToolBundleDetailsModalProps {
	isOpen: boolean;
	onClose: () => void;
	bundle: ToolBundle | null;
}

export function ToolBundleDetailsModal({ isOpen, onClose, bundle }: ToolBundleDetailsModalProps) {
	if (!isOpen || !bundle) {
		return null;
	}

	return (
		<ManagementDetailsModal
			isOpen={isOpen}
			onClose={onClose}
			title="Tool Bundle Details"
			modalKey={`tool-bundle:${bundle.id}:${bundle.modifiedAt}`}
		>
			<ManagementInfoGrid>
				<ManagementInfoRow label="Display Name">{bundle.displayName || '—'}</ManagementInfoRow>
				<ManagementInfoRow label="Slug" mono>
					{bundle.slug}
				</ManagementInfoRow>
				<ManagementInfoRow label="ID" mono>
					{bundle.id}
				</ManagementInfoRow>
				<ManagementInfoRow label="Built-in">{bundle.isBuiltIn ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Enabled">{bundle.isEnabled ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Description">
					<span className="whitespace-pre-wrap">{bundle.description || '—'}</span>
				</ManagementInfoRow>
				<ManagementInfoRow label="Created">{bundle.createdAt}</ManagementInfoRow>
				<ManagementInfoRow label="Modified">{bundle.modifiedAt}</ManagementInfoRow>
			</ManagementInfoGrid>
		</ManagementDetailsModal>
	);
}
