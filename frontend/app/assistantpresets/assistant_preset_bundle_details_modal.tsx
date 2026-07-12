import type { AssistantPresetBundle } from '@/spec/assistantpreset';

import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';

import { formatDateish } from '@/assistantpresets/lib/assistant_preset_utils';

interface AssistantPresetBundleDetailsModalProps {
	isOpen: boolean;
	onClose: () => void;
	bundle: AssistantPresetBundle | null;
}

export function AssistantPresetBundleDetailsModal({ isOpen, onClose, bundle }: AssistantPresetBundleDetailsModalProps) {
	if (!isOpen || !bundle) {
		return null;
	}

	return (
		<ManagementDetailsModal
			isOpen={isOpen}
			onClose={onClose}
			title="Assistant Preset Bundle Details"
			modalKey={`assistant-preset-bundle:${bundle.id}:${String(bundle.modifiedAt)}`}
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
				<ManagementInfoRow label="Created">{formatDateish(bundle.createdAt)}</ManagementInfoRow>
				<ManagementInfoRow label="Modified">{formatDateish(bundle.modifiedAt)}</ManagementInfoRow>
				{bundle.softDeletedAt ? (
					<ManagementInfoRow label="Soft deleted">{formatDateish(bundle.softDeletedAt)}</ManagementInfoRow>
				) : null}
			</ManagementInfoGrid>
		</ManagementDetailsModal>
	);
}
