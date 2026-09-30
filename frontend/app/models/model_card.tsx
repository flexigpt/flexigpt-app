import { FiCheck, FiEdit2, FiEye, FiTrash2 } from 'react-icons/fi';

import { ArtifactState } from '@/spec/artifact';

import { getErrorMessage } from '@/lib/error_utils';

import { usePendingActions } from '@/hooks/use_pending_actions';

import type { ModelManagementItem, ModelProviderManagementItem } from '@/apis/model_management';

import { ActionRow } from '@/components/managementui/action_row';
import { EnabledControl } from '@/components/managementui/enabled_control';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';

interface ModelCardProps {
	model: ModelManagementItem;
	provider: ModelProviderManagementItem;
	isProviderDefault: boolean;
	onView: (model: ModelManagementItem) => void;
	onEdit: (model: ModelManagementItem) => void;
	onToggle: (model: ModelManagementItem) => Promise<void>;
	onDelete: (model: ModelManagementItem) => void;
	onSetDefault: (model: ModelManagementItem) => Promise<void>;
	onActionError: (message: string) => void;
}

export function ModelCard({
	model,
	provider,
	isProviderDefault,
	onView,
	onEdit,
	onToggle,
	onDelete,
	onSetDefault,
	onActionError,
}: ModelCardProps) {
	const { isPending, runAction } = usePendingActions();
	const mutable = !model.list.builtIn;
	const available = model.list.state === ArtifactState.Available;
	const providerAvailable = provider.list.state === ArtifactState.Available;
	const providerEnabled = provider.list.enabled && providerAvailable;
	const toggleKey = 'toggle';
	const defaultKey = 'set-default';
	const canChangeEnabled = available && providerEnabled && !(isProviderDefault && model.list.enabled);
	const canSetDefault = !provider.list.builtIn && providerEnabled && available && model.list.enabled;

	const runModelAction = (key: string, action: () => Promise<void>, fallback: string) => {
		void runAction(key, action).catch((error: unknown) => {
			onActionError(getErrorMessage(error, fallback));
		});
	};

	return (
		<ManagementItemCard
			title={model.list.displayName || 'Model'}
			subtitle={model.view.document.providerModelID}
			status={
				<>
					<StatusBadge tone={model.list.enabled ? 'success' : 'neutral'}>
						{model.list.enabled ? 'Enabled' : 'Disabled'}
					</StatusBadge>
					{isProviderDefault ? <StatusBadge tone="info">Default model</StatusBadge> : null}
					{!available ? <StatusBadge tone="warning">Unavailable</StatusBadge> : null}
					{model.list.builtIn ? <StatusBadge>Built-in</StatusBadge> : null}
				</>
			}
			metadata={
				<>
					<MetadataPill label="Stream">{model.view.document.defaults?.stream === false ? 'Off' : 'On'}</MetadataPill>
					<MetadataPill label="Prompt">{model.view.document.defaults?.maxPromptTokens ?? 'Default'}</MetadataPill>
					<MetadataPill label="Output">{model.view.document.defaults?.maxOutputTokens ?? 'Default'}</MetadataPill>
					<MetadataPill label="Reasoning">
						{model.view.document.defaults?.reasoning ? 'Configured' : 'None'}
					</MetadataPill>
				</>
			}
		>
			<ActionRow
				leading={
					<EnabledControl
						id={`model-enabled-${model.list.ref.rootID}-${model.list.ref.artifactID}`}
						checked={model.list.enabled}
						onChange={() => {
							runModelAction(toggleKey, () => onToggle(model), 'Failed changing model availability.');
						}}
						disabled={isPending(toggleKey) || !canChangeEnabled}
						busy={isPending(toggleKey)}
						title={
							isProviderDefault && model.list.enabled
								? 'Choose another default model before disabling this model.'
								: !providerEnabled
									? 'Enable the provider before changing this model.'
									: !available
										? 'This model artifact is not available.'
										: undefined
						}
					/>
				}
			>
				<button
					type="button"
					className="btn btn-sm btn-ghost rounded-xl"
					disabled={isPending(toggleKey) || isPending(defaultKey)}
					onClick={() => {
						onView(model);
					}}
				>
					<FiEye size={16} />
					<span>View</span>
				</button>

				{mutable ? (
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						disabled={isPending(toggleKey) || isPending(defaultKey)}
						onClick={() => {
							onEdit(model);
						}}
					>
						<FiEdit2 size={16} />
						<span>Edit</span>
					</button>
				) : null}

				{!isProviderDefault && !provider.list.builtIn ? (
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						disabled={isPending(defaultKey) || !canSetDefault}
						title={!canSetDefault ? 'Enable the provider and model before selecting a default.' : undefined}
						onClick={() => {
							runModelAction(defaultKey, () => onSetDefault(model), 'Failed selecting the default model.');
						}}
					>
						<FiCheck size={16} />
						<span>Set default</span>
					</button>
				) : null}

				{mutable ? (
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						disabled={isPending(toggleKey) || isPending(defaultKey) || isProviderDefault}
						title={isProviderDefault ? 'Choose another default model before deleting this model.' : 'Delete model'}
						onClick={() => {
							onDelete(model);
						}}
					>
						<FiTrash2 size={16} />
						<span>Delete</span>
					</button>
				) : null}
			</ActionRow>
		</ManagementItemCard>
	);
}
