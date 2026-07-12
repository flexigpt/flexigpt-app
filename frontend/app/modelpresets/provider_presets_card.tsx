import { useMemo, useState } from 'react';

import { FiCheckCircle, FiChevronDown, FiChevronUp, FiEdit2, FiEye, FiKey, FiPlus, FiTrash2 } from 'react-icons/fi';

import type { ProviderName } from '@/spec/inference';
import { SDK_DISPLAY_NAME } from '@/spec/inference';
import type {
	ModelPreset,
	ModelPresetID,
	PatchModelPresetPayload,
	PostModelPresetPayload,
	ProviderPreset,
} from '@/spec/modelpreset';
import type { AuthKeyMeta } from '@/spec/setting';
import { AuthKeyTypeProvider } from '@/spec/setting';

import { redactSensitiveHTTPHeaders } from '@/lib/http_input_utils';

import { usePendingActions } from '@/hooks/use_pending_actions';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { ActionRow } from '@/components/managementui/action_row';
import { EnabledControl } from '@/components/managementui/enabled_control';
import { ManagementBundleCard } from '@/components/managementui/management_bundle_card';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';

import { AddEditModelPresetModal } from '@/modelpresets/modelpreset_add_edit_modal';
import { AddEditAuthKeyModal } from '@/settings/authkey_add_edit_modal';

interface ProviderPresetCardProps {
	provider: ProviderName;
	preset: ProviderPreset;
	defaultProvider?: ProviderName;
	authKeySet: boolean;
	authKeys: AuthKeyMeta[];
	enabledProviders: ProviderName[];
	allProviderPresets: Record<ProviderName, ProviderPreset>;

	onToggleProvider: (provider: ProviderName, nextEnabled: boolean) => Promise<void>;
	onDeleteProvider: (provider: ProviderName) => Promise<void>;
	onRequestEdit: (provider: ProviderName) => void;
	onSetDefaultModel: (provider: ProviderName, modelPresetID: ModelPresetID) => Promise<void>;
	onToggleModel: (provider: ProviderName, modelPresetID: ModelPresetID, nextEnabled: boolean) => Promise<void>;
	onCreateModel: (
		provider: ProviderName,
		modelPresetID: ModelPresetID,
		payload: PostModelPresetPayload
	) => Promise<void>;
	onPatchModel: (
		provider: ProviderName,
		modelPresetID: ModelPresetID,
		payload: PatchModelPresetPayload
	) => Promise<void>;
	onDeleteModel: (provider: ProviderName, modelPresetID: ModelPresetID) => Promise<void>;
	onProviderAuthKeyChanged: (provider: ProviderName) => Promise<void>;
}

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim()) {
		return error.message;
	}
	return fallback;
}

export function ProviderPresetCard({
	provider,
	preset,
	defaultProvider,
	authKeySet,
	authKeys,
	enabledProviders,
	allProviderPresets,
	onToggleProvider,
	onDeleteProvider,
	onRequestEdit,
	onSetDefaultModel,
	onToggleModel,
	onCreateModel,
	onPatchModel,
	onDeleteModel,
	onProviderAuthKeyChanged,
}: ProviderPresetCardProps) {
	const [expanded, setExpanded] = useState(false);
	const [selectedID, setSelectedID] = useState<ModelPresetID | null>(null);
	const [modelModalMode, setModelModalMode] = useState<'add' | 'edit' | 'view'>('add');

	const [showModModal, setShowModModal] = useState(false);
	const [showDelProv, setShowDelProv] = useState(false);
	const [showDelModel, setShowDelModel] = useState(false);
	const [showDenied, setShowDenied] = useState(false);
	const [deniedMsg, setDeniedMsg] = useState('');
	const [showKeyModal, setShowKeyModal] = useState(false);
	const { isPending, runAction } = usePendingActions();

	const isLastEnabled = preset.isEnabled && enabledProviders.length === 1;
	const providerIsBuiltIn = preset.isBuiltIn;

	const modelPresets = preset.modelPresets;
	const defaultModelPresetID = preset.defaultModelPresetID;
	const modelEntries = Object.entries(modelPresets).toSorted(([, left], [, right]) =>
		(left.displayName || left.name || left.id).localeCompare(right.displayName || right.name || right.id)
	);
	const hasModels = modelEntries.length > 0;
	const canDeleteProvider = !providerIsBuiltIn && !hasModels;

	const allModelPresets = useMemo(() => {
		const out: Record<ProviderName, Record<ModelPresetID, ModelPreset>> = {};
		for (const [prov, providerPreset] of Object.entries(allProviderPresets)) {
			out[prov] = providerPreset.modelPresets;
		}
		return out;
	}, [allProviderPresets]);

	const keyModalInitial = useMemo(
		() => authKeys.find(k => k.type === AuthKeyTypeProvider && k.keyName === provider) ?? null,
		[authKeys, provider]
	);

	const showLocalDenied = (message: string) => {
		setDeniedMsg(message);
		setShowDenied(true);
	};

	const runActionWithAlert = async (key: string, action: () => Promise<void>, fallback: string) => {
		try {
			await runAction(key, action);
		} catch (error) {
			showLocalDenied(getErrorMessage(error, fallback));
			throw error;
		}
	};

	const toggleProviderEnable = (nextEnabled: boolean) => {
		if (!nextEnabled && provider === defaultProvider && preset.isEnabled) {
			showLocalDenied('Cannot disable the default provider. Pick another default first.');
			return;
		}

		if (!nextEnabled && isLastEnabled && preset.isEnabled) {
			showLocalDenied('Cannot disable the last enabled provider.');
			return;
		}

		void runActionWithAlert(
			'bundle:toggle',
			() => onToggleProvider(provider, nextEnabled),
			'Failed toggling provider.'
		).catch(() => undefined);
	};

	const requestDeleteProvider = () => {
		if (providerIsBuiltIn) {
			showLocalDenied('Built-in providers cannot be deleted.');
			return;
		}

		if (provider === defaultProvider) {
			showLocalDenied('Cannot delete the current default provider. Pick another default first.');
			return;
		}

		if (hasModels) {
			showLocalDenied('Only empty providers can be deleted. Remove all model presets first.');
			return;
		}

		setShowDelProv(true);
	};

	const confirmDeleteProvider = async () => {
		await runActionWithAlert('bundle:delete', () => onDeleteProvider(provider), 'Failed deleting provider.');
		setShowDelProv(false);
	};

	const setDefaultModel = (id: ModelPresetID) => {
		void runActionWithAlert(
			`${id}:set-default`,
			() => onSetDefaultModel(provider, id),
			'Failed setting default model.'
		).catch(() => undefined);
	};

	const toggleModelEnable = (id: ModelPresetID, nextEnabled: boolean) => {
		const modelPreset = modelPresets[id];

		if (!nextEnabled && id === defaultModelPresetID && modelPreset.isEnabled) {
			showLocalDenied('Cannot disable the default model preset. Choose another default first.');
			return;
		}

		void runActionWithAlert(
			`${id}:toggle`,
			() => onToggleModel(provider, id, nextEnabled),
			'Failed toggling model.'
		).catch(() => undefined);
	};

	const openAddModel = () => {
		if (providerIsBuiltIn) {
			showLocalDenied('Cannot add model presets to a built-in provider.');
			return;
		}

		setSelectedID(null);
		setModelModalMode('add');
		setShowModModal(true);
	};

	const openEditOrViewModel = (id: ModelPresetID) => {
		setSelectedID(id);
		setModelModalMode(modelPresets[id].isBuiltIn ? 'view' : 'edit');
		setShowModModal(true);
	};

	const openViewModel = (id: ModelPresetID) => {
		setSelectedID(id);
		setModelModalMode('view');
		setShowModModal(true);
	};

	const handleModifyModelSubmit = async (id: ModelPresetID, data: PostModelPresetPayload | PatchModelPresetPayload) => {
		try {
			if (modelModalMode === 'add') {
				await runAction(`${id}:create`, () => onCreateModel(provider, id, data as PostModelPresetPayload));
			} else if (modelModalMode === 'edit') {
				await runAction(`${id}:save`, () => onPatchModel(provider, id, data as PatchModelPresetPayload));
			}
			setShowModModal(false);
		} catch (error) {
			const message = getErrorMessage(error, 'Failed saving model preset.');
			showLocalDenied(message);
			throw error instanceof Error ? error : new Error(message);
		}
	};

	const requestDeleteModel = (id: ModelPresetID) => {
		if (modelPresets[id].isBuiltIn) {
			showLocalDenied('Built-in model presets cannot be deleted.');
			return;
		}

		setSelectedID(id);
		setShowDelModel(true);
	};

	const confirmDeleteModel = async () => {
		if (!selectedID) {
			return;
		}

		await runActionWithAlert(
			`${selectedID}:delete`,
			() => onDeleteModel(provider, selectedID),
			'Failed deleting model preset.'
		);
		setShowDelModel(false);
		setSelectedID(null);
	};

	return (
		<>
			<ManagementBundleCard
				title={preset.displayName || provider}
				identity={<span className="font-mono">{provider}</span>}
				status={
					<>
						<StatusBadge tone={preset.isEnabled ? 'success' : 'neutral'}>
							{preset.isEnabled ? 'Enabled' : 'Disabled'}
						</StatusBadge>
						<StatusBadge>{providerIsBuiltIn ? 'Built-in' : 'Custom'}</StatusBadge>
						{provider === defaultProvider ? <StatusBadge tone="info">Default provider</StatusBadge> : null}
						<StatusBadge tone={authKeySet ? 'success' : 'warning'}>
							{authKeySet ? 'API key configured' : 'API key missing'}
						</StatusBadge>
					</>
				}
				metadata={
					<>
						<MetadataPill label="SDK">{SDK_DISPLAY_NAME[preset.sdkType]}</MetadataPill>
						<MetadataPill label="Models">{modelEntries.length}</MetadataPill>
						<MetadataPill label="Origin" title={preset.origin}>
							{preset.origin}
						</MetadataPill>
					</>
				}
				disclosure={
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						aria-expanded={expanded}
						onClick={() => {
							setExpanded(previous => !previous);
						}}
					>
						<span className="whitespace-nowrap">Models: {modelEntries.length}</span>
						{expanded ? <FiChevronUp /> : <FiChevronDown />}
					</button>
				}
				actionLeading={
					<EnabledControl
						id={`provider-enabled-${provider}`}
						checked={preset.isEnabled}
						onChange={toggleProviderEnable}
						disabled={(preset.isEnabled && provider === defaultProvider) || (preset.isEnabled && isLastEnabled)}
						busy={isPending('bundle:toggle')}
						compact={false}
						title={
							preset.isEnabled && provider === defaultProvider
								? 'Choose another default provider before disabling this one.'
								: preset.isEnabled && isLastEnabled
									? 'At least one provider must remain enabled.'
									: undefined
						}
					/>
				}
				actions={
					<>
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							onClick={() => {
								setShowKeyModal(true);
							}}
						>
							<FiKey size={16} />
							<span>{authKeySet ? 'Update API Key' : 'Set API Key'}</span>
						</button>
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							onClick={() => {
								onRequestEdit(provider);
							}}
						>
							{providerIsBuiltIn ? <FiEye size={16} /> : <FiEdit2 size={16} />}
							<span>{providerIsBuiltIn ? 'View Provider' : 'Edit Provider'}</span>
						</button>
						{!providerIsBuiltIn ? (
							<>
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									onClick={openAddModel}
									disabled={!preset.isEnabled}
								>
									<FiPlus size={16} />
									<span>Add Model</span>
								</button>
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									onClick={requestDeleteProvider}
									disabled={!canDeleteProvider || provider === defaultProvider || isPending('bundle:delete')}
								>
									<FiTrash2 size={16} />
									<span>Delete Provider</span>
								</button>
							</>
						) : null}
					</>
				}
			>
				{expanded && (
					<div className="mt-4 space-y-6">
						<ManagementInfoGrid>
							<ManagementInfoRow label="Provider ID" mono>
								{preset.name}
							</ManagementInfoRow>
							<ManagementInfoRow label="SDK">{SDK_DISPLAY_NAME[preset.sdkType]}</ManagementInfoRow>
							<ManagementInfoRow label="Origin" mono>
								{preset.origin}
							</ManagementInfoRow>
							<ManagementInfoRow label="Chat path" mono>
								{preset.chatCompletionPathPrefix}
							</ManagementInfoRow>
							<ManagementInfoRow label="API key header">{preset.apiKeyHeaderKey || '—'}</ManagementInfoRow>
							<ManagementInfoRow label="Default headers">
								<pre className="bg-base-300 max-h-48 overflow-auto rounded-xl p-3 text-xs whitespace-pre-wrap">
									{JSON.stringify(redactSensitiveHTTPHeaders(preset.defaultHeaders) ?? {}, null, 2)}
								</pre>
							</ManagementInfoRow>
						</ManagementInfoGrid>

						<div className="divider my-0">Model presets</div>
						{hasModels ? (
							<div className="space-y-3">
								{modelEntries.map(([id, modelPreset]) => {
									const canModify = !modelPreset.isBuiltIn;
									const isDefault = id === defaultModelPresetID;

									return (
										<ManagementItemCard
											key={id}
											title={modelPreset.displayName || id}
											subtitle={`${id} · ${modelPreset.name}`}
											status={
												<>
													<StatusBadge tone={modelPreset.isEnabled ? 'success' : 'neutral'}>
														{modelPreset.isEnabled ? 'Enabled' : 'Disabled'}
													</StatusBadge>
													{isDefault ? <StatusBadge tone="info">Default</StatusBadge> : null}
												</>
											}
											metadata={
												<>
													<MetadataPill label="Reasoning">{modelPreset.reasoning ? 'Configured' : 'None'}</MetadataPill>
													<MetadataPill label="Stream">{modelPreset.stream ? 'On' : 'Off'}</MetadataPill>
													<MetadataPill label="Prompt">{modelPreset.maxPromptLength ?? 'Default'}</MetadataPill>
													<MetadataPill label="Output">{modelPreset.maxOutputLength ?? 'Default'}</MetadataPill>
													{modelPreset.isBuiltIn ? <MetadataPill>Built-in</MetadataPill> : null}
												</>
											}
										>
											<ActionRow
												leading={
													<EnabledControl
														id={`model-enabled-${provider}-${id}`}
														checked={modelPreset.isEnabled}
														onChange={enabled => {
															toggleModelEnable(id, enabled);
														}}
														disabled={!preset.isEnabled || (isDefault && modelPreset.isEnabled)}
														busy={isPending(`${id}:toggle`)}
														title={
															isDefault && modelPreset.isEnabled
																? 'Choose another default model before disabling this one.'
																: !preset.isEnabled
																	? 'Enable the provider first.'
																	: undefined
														}
													/>
												}
											>
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													onClick={() => {
														openViewModel(id);
													}}
													title="View Model Preset"
												>
													<FiEye size={16} />
													<span>View</span>
												</button>

												{!isDefault ? (
													<button
														type="button"
														className="btn btn-sm btn-ghost rounded-xl"
														disabled={!preset.isEnabled || !modelPreset.isEnabled || isPending(`${id}:set-default`)}
														onClick={() => {
															setDefaultModel(id);
														}}
													>
														<FiCheckCircle size={16} />
														<span>Set default</span>
													</button>
												) : null}

												{canModify ? (
													<>
														<button
															type="button"
															className="btn btn-sm btn-ghost rounded-xl"
															onClick={() => {
																openEditOrViewModel(id);
															}}
															title="Edit Model Preset"
														>
															<FiEdit2 size={16} />
															<span>Edit</span>
														</button>
														<button
															type="button"
															className="btn btn-sm btn-ghost rounded-xl"
															onClick={() => {
																requestDeleteModel(id);
															}}
															disabled={isPending(`${id}:delete`)}
															title="Delete Model Preset"
														>
															<FiTrash2 size={16} />
															<span>Delete</span>
														</button>
													</>
												) : (
													<span className="text-base-content/60 px-2 text-xs">Read only</span>
												)}
											</ActionRow>
										</ManagementItemCard>
									);
								})}
							</div>
						) : (
							<ManagementEmptyState>No model presets configured for this provider.</ManagementEmptyState>
						)}
					</div>
				)}
			</ManagementBundleCard>

			<DeleteConfirmationModal
				isOpen={showDelProv}
				onClose={() => {
					setShowDelProv(false);
				}}
				onConfirm={confirmDeleteProvider}
				confirmButtonText="Delete"
				title="Delete Provider"
				message={`Delete provider “${provider}”? This action cannot be undone.`}
			/>

			<DeleteConfirmationModal
				isOpen={showDelModel}
				onClose={() => {
					setShowDelModel(false);
				}}
				onConfirm={confirmDeleteModel}
				confirmButtonText="Delete"
				title="Delete Model Preset"
				message={`Delete model preset “${selectedID}”? This action cannot be undone.`}
			/>

			<AddEditModelPresetModal
				isOpen={showModModal}
				mode={modelModalMode}
				onClose={() => {
					setShowModModal(false);
				}}
				onSubmit={handleModifyModelSubmit}
				providerName={provider}
				providerSDKType={preset.sdkType}
				providerCapabilitiesOverride={preset.capabilitiesOverride}
				initialModelID={selectedID ?? undefined}
				initialData={selectedID ? modelPresets[selectedID] : undefined}
				existingModels={modelPresets}
				allModelPresets={allModelPresets}
			/>

			<AddEditAuthKeyModal
				isOpen={showKeyModal}
				initial={keyModalInitial}
				existing={authKeys}
				prefill={!keyModalInitial ? { type: AuthKeyTypeProvider, keyName: provider } : undefined}
				onClose={() => {
					setShowKeyModal(false);
				}}
				onChanged={() => {
					setShowKeyModal(false);
					void (async () => {
						try {
							await onProviderAuthKeyChanged(provider);
						} catch (error) {
							showLocalDenied(getErrorMessage(error, 'Failed refreshing auth key state.'));
						}
					})();
				}}
			/>

			<ActionDeniedAlertModal
				isOpen={showDenied}
				onClose={() => {
					setShowDenied(false);
					setDeniedMsg('');
				}}
				message={deniedMsg}
			/>
		</>
	);
}
