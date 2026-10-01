import { useState } from 'react';
import { FiChevronDown, FiChevronUp, FiEdit2, FiEye, FiKey, FiPlus, FiTrash2 } from 'react-icons/fi';

import { ArtifactState } from '@/spec/artifact';
import { ModelLookupScope } from '@/spec/model';

import { getErrorMessage } from '@/lib/error_utils';

import { usePendingActions } from '@/hooks/use_pending_actions';

import type { ModelManagementItem, ModelProviderManagementItem } from '@/apis/model_management';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { EnabledControl } from '@/components/managementui/enabled_control';
import { ManagementBundleCard } from '@/components/managementui/management_bundle_card';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';

import { getProviderSDKOption } from '@/models/lib/provider_sdk';
import { ModelCard } from '@/models/model_card';

interface ModelProviderCardProps {
	provider: ModelProviderManagementItem;
	models: ModelManagementItem[];
	isGlobalDefault: boolean;
	onViewProvider: (provider: ModelProviderManagementItem) => void;
	onEditProvider: (provider: ModelProviderManagementItem) => void;
	onToggleProvider: (provider: ModelProviderManagementItem) => Promise<void>;
	onDeleteProvider: (provider: ModelProviderManagementItem) => void;
	onCredential: (provider: ModelProviderManagementItem) => void;
	onAddModel: (provider: ModelProviderManagementItem) => void;
	onViewModel: (model: ModelManagementItem) => void;
	onEditModel: (model: ModelManagementItem) => void;
	onToggleModel: (model: ModelManagementItem) => Promise<void>;
	onDeleteModel: (model: ModelManagementItem) => void;
	onSetDefaultModel: (provider: ModelProviderManagementItem, model: ModelManagementItem) => Promise<void>;
}

function providerDefaultModel(
	provider: ModelProviderManagementItem,
	models: ModelManagementItem[]
): ModelManagementItem | undefined {
	const reference = provider.view.defaultModel;
	if (!reference) {
		return undefined;
	}

	return models.find(model => {
		if (model.view.document.name !== reference.name) {
			return false;
		}
		if (reference.scope === ModelLookupScope.Builtin) {
			return model.list.builtIn;
		}
		return !model.list.builtIn && model.list.ref.rootID === provider.list.ref.rootID;
	});
}

export function ModelProviderCard({
	provider,
	models,
	isGlobalDefault,
	onViewProvider,
	onEditProvider,
	onToggleProvider,
	onDeleteProvider,
	onCredential,
	onAddModel,
	onViewModel,
	onEditModel,
	onToggleModel,
	onDeleteModel,
	onSetDefaultModel,
}: ModelProviderCardProps) {
	const { isPending, runAction } = usePendingActions();
	const [expanded, setExpanded] = useState(false);
	const [showDenied, setShowDenied] = useState(false);
	const [deniedMessage, setDeniedMessage] = useState('');
	const mutable = !provider.list.builtIn;
	const available = provider.list.state === ArtifactState.Available;
	const defaultModel = providerDefaultModel(provider, models);
	const defaultModelDisplayName = defaultModel?.list.displayName;
	const canDelete = mutable && models.length === 0 && !isGlobalDefault;
	const toggleKey = 'provider-toggle';
	const sdkOption = getProviderSDKOption(provider.view.document.adapter);

	const showActionError = (message: string) => {
		setDeniedMessage(message);
		setShowDenied(true);
	};

	const runProviderAction = (key: string, action: () => Promise<void>, fallback: string) => {
		void runAction(key, action).catch((error: unknown) => {
			showActionError(getErrorMessage(error, fallback));
		});
	};

	return (
		<>
			<ManagementBundleCard
				title={provider.list.displayName || 'Provider'}
				status={
					<>
						<StatusBadge tone={provider.list.enabled ? 'success' : 'neutral'}>
							{provider.list.enabled ? 'Enabled' : 'Disabled'}
						</StatusBadge>
						<StatusBadge>{provider.list.builtIn ? 'Built-in' : 'Custom'}</StatusBadge>
						{isGlobalDefault ? <StatusBadge tone="info">Default provider</StatusBadge> : null}
						<StatusBadge tone={provider.apiKey.configured ? 'success' : 'warning'}>
							{provider.apiKey.configured ? 'API key configured' : 'API key missing'}
						</StatusBadge>
						{!available ? <StatusBadge tone="warning">Unavailable</StatusBadge> : null}
					</>
				}
				metadata={
					<>
						<MetadataPill label="SDK">{sdkOption?.displayName ?? 'Unsupported compatibility mode'}</MetadataPill>
						<MetadataPill label="Models">{models.length}</MetadataPill>
						<MetadataPill label="Default">{defaultModelDisplayName ?? 'None'}</MetadataPill>
					</>
				}
				disclosure={
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						aria-expanded={expanded}
						onClick={() => {
							setExpanded(value => !value);
						}}
					>
						<span>{models.length} models</span>
						{expanded ? <FiChevronUp size={16} /> : <FiChevronDown size={16} />}
					</button>
				}
				actionLeading={
					<EnabledControl
						id={`provider-enabled-${provider.list.ref.rootID}-${provider.list.ref.artifactID}`}
						checked={provider.list.enabled}
						onChange={() => {
							runProviderAction(toggleKey, () => onToggleProvider(provider), 'Failed changing provider availability.');
						}}
						disabled={isPending(toggleKey) || !available || (isGlobalDefault && provider.list.enabled)}
						busy={isPending(toggleKey)}
						compact={false}
						title={
							isGlobalDefault && provider.list.enabled
								? 'Choose another default provider before disabling this provider.'
								: !available
									? 'This provider artifact is not available.'
									: undefined
						}
					/>
				}
				actions={
					<>
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							disabled={isPending(toggleKey)}
							onClick={() => {
								onCredential(provider);
							}}
						>
							<FiKey size={16} />
							<span>{provider.apiKey.configured ? 'Manage API Key' : 'Set API Key'}</span>
						</button>
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							disabled={isPending(toggleKey)}
							onClick={() => {
								if (mutable) {
									onEditProvider(provider);
									return;
								}
								onViewProvider(provider);
							}}
						>
							{mutable ? <FiEdit2 size={16} /> : <FiEye size={16} />}
							<span>{mutable ? 'Edit Provider' : 'View Provider'}</span>
						</button>
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							disabled={isPending(toggleKey) || !available || !provider.list.enabled}
							title={
								provider.list.builtIn ? 'Create a custom model that references this built-in provider.' : undefined
							}
							onClick={() => {
								onAddModel(provider);
							}}
						>
							<FiPlus size={16} />
							<span>Add Model</span>
						</button>
						{mutable ? (
							<button
								type="button"
								className="btn btn-sm btn-ghost rounded-xl"
								disabled={isPending(toggleKey) || !canDelete}
								title={
									isGlobalDefault
										? 'Choose another default provider before deleting this provider.'
										: !canDelete
											? 'Remove all linked models before deleting this provider.'
											: 'Delete provider'
								}
								onClick={() => {
									onDeleteProvider(provider);
								}}
							>
								<FiTrash2 size={16} />
								<span>Delete Provider</span>
							</button>
						) : null}
					</>
				}
			>
				{expanded ? (
					<div className="mt-4 space-y-5">
						<ManagementInfoGrid>
							<ManagementInfoRow label="SDK">
								{sdkOption?.displayName ?? 'Unsupported compatibility mode'}
							</ManagementInfoRow>
							<ManagementInfoRow label="Origin">{provider.view.document.connection?.origin ?? '—'}</ManagementInfoRow>
							<ManagementInfoRow label="Chat path">
								{provider.view.document.connection?.path ?? 'Adapter default'}
							</ManagementInfoRow>
							<ManagementInfoRow label="Default model">
								{defaultModelDisplayName ?? 'No default model'}
							</ManagementInfoRow>
						</ManagementInfoGrid>

						<div className="divider my-0">Models</div>

						{models.length > 0 ? (
							<div className="space-y-3">
								{models.map(model => (
									<ModelCard
										key={`${model.list.ref.rootID}:${model.list.ref.artifactID}`}
										model={model}
										provider={provider}
										isProviderDefault={defaultModel?.list.ref.artifactID === model.list.ref.artifactID}
										onView={onViewModel}
										onEdit={onEditModel}
										onToggle={onToggleModel}
										onDelete={onDeleteModel}
										onSetDefault={m => {
											return onSetDefaultModel(provider, m);
										}}
										onActionError={showActionError}
									/>
								))}
							</div>
						) : (
							<ManagementEmptyState>No models are linked to this provider.</ManagementEmptyState>
						)}
					</div>
				) : null}
			</ManagementBundleCard>

			<ActionDeniedAlertModal
				isOpen={showDenied}
				onClose={() => {
					setShowDenied(false);
					setDeniedMessage('');
				}}
				message={deniedMessage}
			/>
		</>
	);
}
