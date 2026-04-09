import { useMemo, useState } from 'react';

import {
	FiCheck,
	FiCheckCircle,
	FiChevronDown,
	FiChevronUp,
	FiEdit2,
	FiEye,
	FiKey,
	FiPlus,
	FiTrash2,
	FiX,
	FiXCircle,
} from 'react-icons/fi';

import { type ProviderName, SDK_DISPLAY_NAME } from '@/spec/inference';
import {
	type ModelPreset,
	type ModelPresetID,
	type PatchModelPresetPayload,
	type PostModelPresetPayload,
	type ProviderPreset,
} from '@/spec/modelpreset';
import type { AuthKeyMeta } from '@/spec/setting';
import { AuthKeyTypeProvider } from '@/spec/setting';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Dropdown } from '@/components/dropdown';

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
	if (error instanceof Error && error.message.trim()) return error.message;
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

	const isLastEnabled = preset.isEnabled && enabledProviders.length === 1;
	const providerIsBuiltIn = preset.isBuiltIn;

	const modelPresets = preset.modelPresets;
	const defaultModelPresetID = preset.defaultModelPresetID;
	const modelEntries = Object.entries(modelPresets);
	const hasModels = modelEntries.length > 0;
	const canDeleteProvider = !providerIsBuiltIn && !hasModels;

	const safeDefaultModelID: ModelPresetID =
		defaultModelPresetID && modelPresets[defaultModelPresetID]
			? defaultModelPresetID
			: (Object.keys(modelPresets)[0] ?? '');

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

	const toggleProviderEnable = () => {
		if (provider === defaultProvider && preset.isEnabled) {
			showLocalDenied('Cannot disable the default provider. Pick another default first.');
			return;
		}

		if (isLastEnabled && preset.isEnabled) {
			showLocalDenied('Cannot disable the last enabled provider.');
			return;
		}

		void (async () => {
			try {
				await onToggleProvider(provider, !preset.isEnabled);
			} catch (error) {
				showLocalDenied(getErrorMessage(error, 'Failed toggling provider.'));
			}
		})();
	};

	const toggleExpand = () => {
		setExpanded(prev => !prev);
	};

	const requestDeleteProvider = () => {
		if (providerIsBuiltIn) {
			showLocalDenied('Built-in providers cannot be deleted.');
			return;
		}

		if (hasModels) {
			showLocalDenied('Only empty providers can be deleted. Remove all model presets first.');
			return;
		}

		setShowDelProv(true);
	};

	const confirmDeleteProvider = async () => {
		try {
			await onDeleteProvider(provider);
			setShowDelProv(false);
		} catch (error) {
			showLocalDenied(getErrorMessage(error, 'Failed deleting provider.'));
		}
	};

	const handleDefaultModelChange = (id: ModelPresetID) => {
		void (async () => {
			try {
				await onSetDefaultModel(provider, id);
			} catch (error) {
				showLocalDenied(getErrorMessage(error, 'Failed setting default model.'));
			}
		})();
	};

	const toggleModelEnable = (id: ModelPresetID) => {
		const modelPreset = modelPresets[id];

		if (id === defaultModelPresetID && modelPreset.isEnabled) {
			showLocalDenied('Cannot disable the default model preset. Choose another default first.');
			return;
		}

		void (async () => {
			try {
				await onToggleModel(provider, id, !modelPreset.isEnabled);
			} catch (error) {
				showLocalDenied(getErrorMessage(error, 'Failed toggling model.'));
			}
		})();
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
				await onCreateModel(provider, id, data as PostModelPresetPayload);
			} else if (modelModalMode === 'edit') {
				await onPatchModel(provider, id, data as PatchModelPresetPayload);
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
		if (!selectedID) return;

		try {
			await onDeleteModel(provider, selectedID);
			setShowDelModel(false);
		} catch (error) {
			showLocalDenied(getErrorMessage(error, 'Failed deleting model preset.'));
		}
	};

	return (
		<div className="bg-base-100 mb-8 rounded-2xl p-4 shadow-lg">
			<div className="flex items-center justify-between">
				<div className="flex items-center">
					<h3 className="text-sm font-semibold capitalize">{preset.displayName || provider} </h3>
				</div>

				<div className="flex items-center justify-end gap-4">
					<span className="text-base-content/60 text-xs tracking-wide uppercase">
						{preset.isBuiltIn ? 'Built-in' : 'Custom'}
					</span>

					<div className="flex items-center gap-1">
						<label className="text-sm">Enable</label>
						<input
							type="checkbox"
							className="toggle toggle-accent"
							checked={preset.isEnabled}
							onChange={toggleProviderEnable}
						/>
					</div>

					<div className="flex cursor-pointer items-end justify-end gap-4" onClick={toggleExpand}>
						<div className="flex items-center">
							<span className="text-sm">API-Key</span>
							{authKeySet ? <FiCheckCircle className="text-success mx-1" /> : <FiXCircle className="text-error mx-1" />}
						</div>

						<div className="flex items-center">
							<span className="text-sm">Details</span>
							{expanded ? <FiChevronUp className="mx-1" /> : <FiChevronDown className="mx-1" />}
						</div>
					</div>
				</div>
			</div>

			{expanded && (
				<div className="mt-4 space-y-6">
					<div className="border-base-content/10 mb-4 overflow-x-auto rounded-2xl border">
						<table className="table w-full">
							<tbody>
								<tr>
									<td colSpan={2} className="py-0.5">
										<div className="flex items-center justify-between py-0.5">
											<span
												className="label-text-alt tooltip tooltip-right"
												data-tip={
													providerIsBuiltIn
														? 'Built-in providers cannot be deleted.'
														: hasModels
															? 'Only empty providers can be deleted.'
															: 'Delete Empty Provider'
												}
											>
												<button
													className={`btn btn-ghost flex items-center rounded-2xl ${
														!canDeleteProvider ? 'btn-disabled cursor-not-allowed opacity-50' : ''
													}`}
													onClick={canDeleteProvider ? requestDeleteProvider : undefined}
													title="Delete Provider"
													disabled={!canDeleteProvider}
												>
													<FiTrash2 />
													<span className="ml-1 hidden md:inline">Delete Provider</span>
												</button>
											</span>

											<div className="flex gap-2">
												<button
													className="btn btn-ghost flex items-center rounded-2xl"
													onClick={() => {
														setShowKeyModal(true);
													}}
													title={authKeySet ? 'Update API Key' : 'Set API Key'}
												>
													<FiKey />
													<span className="ml-1 hidden md:inline">{authKeySet ? 'Update API Key' : 'Set API Key'}</span>
												</button>

												<button
													className="btn btn-ghost flex items-center rounded-2xl"
													onClick={() => {
														onRequestEdit(provider);
													}}
													title={providerIsBuiltIn ? 'View Provider' : 'Edit Provider'}
												>
													{providerIsBuiltIn ? <FiEye /> : <FiEdit2 />}
													<span className="ml-1 hidden md:inline">
														{providerIsBuiltIn ? 'View Provider' : 'Edit Provider'}
													</span>
												</button>
											</div>
										</div>
									</td>
								</tr>

								<tr className="hover:bg-base-300">
									<td className="w-1/3 text-sm">ID</td>
									<td className="text-sm">{preset.name}</td>
								</tr>
								<tr className="hover:bg-base-300">
									<td className="w-1/3 text-sm">SDK Type</td>
									<td className="text-sm">{SDK_DISPLAY_NAME[preset.sdkType]}</td>
								</tr>
								<tr className="hover:bg-base-300">
									<td className="w-1/3 text-sm">Origin</td>
									<td className="text-sm">{preset.origin}</td>
								</tr>
								<tr className="hover:bg-base-300">
									<td className="w-1/3 text-sm">Chat Path</td>
									<td className="text-sm">{preset.chatCompletionPathPrefix}</td>
								</tr>
								<tr className="hover:bg-base-300">
									<td className="w-1/3 text-sm">API-Key Header Key</td>
									<td className="text-sm">{preset.apiKeyHeaderKey || '—'}</td>
								</tr>
								<tr className="hover:bg-base-300">
									<td className="w-1/3 text-sm">Default Headers</td>
									<td className="text-sm">
										<pre className="m-0 p-0 text-xs wrap-break-word whitespace-pre-wrap">
											{JSON.stringify(preset.defaultHeaders ?? {}, null, 2)}
										</pre>
									</td>
								</tr>
							</tbody>
						</table>
					</div>

					<div className="border-base-content/10 mb-2 overflow-x-auto rounded-2xl border">
						<div className="grid grid-cols-12 items-center gap-4 px-4 py-2">
							<span className="col-span-3 text-sm font-semibold">Default Model</span>

							<div className="col-span-6 ml-8">
								{hasModels ? (
									<Dropdown<ModelPresetID>
										dropdownItems={modelPresets}
										selectedKey={safeDefaultModelID}
										onChange={handleDefaultModelChange}
										filterDisabled={false}
										title="Select default model"
										getDisplayName={k => modelPresets[k].displayName || k}
									/>
								) : (
									<span className="text-sm italic">No model presets configured.</span>
								)}
							</div>

							<div className="col-span-3 flex justify-end">
								<button
									className={`btn btn-ghost flex items-center rounded-2xl ${
										providerIsBuiltIn ? 'btn-disabled cursor-not-allowed opacity-50' : ''
									}`}
									onClick={openAddModel}
									disabled={providerIsBuiltIn}
									title="Add Model Preset"
								>
									<FiPlus size={16} />
									<span className="ml-1 hidden md:inline">Add Model Preset</span>
								</button>
							</div>
						</div>

						{hasModels && (
							<table className="table-zebra table w-full">
								<thead>
									<tr className="bg-base-300 text-sm font-semibold">
										<th>Model Preset Label</th>
										<th>Model Name</th>
										<th className="text-center">Enabled</th>
										<th className="text-center">Reasoning</th>
										<th className="text-center">Actions</th>
									</tr>
								</thead>

								<tbody>
									{modelEntries.map(([id, modelPreset]) => {
										const canModify = !modelPreset.isBuiltIn;
										return (
											<tr key={id} className="hover:bg-base-300">
												<td>{modelPreset.displayName || id}</td>
												<td>{modelPreset.name}</td>
												<td className="text-center">
													<input
														type="checkbox"
														className="toggle toggle-accent"
														checked={modelPreset.isEnabled}
														onChange={() => {
															toggleModelEnable(id);
														}}
													/>
												</td>
												<td className="text-center">
													{'reasoning' in modelPreset && modelPreset.reasoning ? (
														<FiCheck className="mx-auto" />
													) : (
														<FiX className="mx-auto" />
													)}
												</td>
												<td className="space-x-1 text-center">
													<button
														className="btn btn-sm btn-ghost rounded-2xl"
														onClick={() => {
															openViewModel(id);
														}}
														title="View Model Preset"
													>
														<FiEye size={16} />
													</button>

													{canModify ? (
														<>
															<button
																className="btn btn-sm btn-ghost rounded-2xl"
																onClick={() => {
																	openEditOrViewModel(id);
																}}
																title="Edit Model Preset"
															>
																<FiEdit2 size={16} />
															</button>
															<button
																className="btn btn-sm btn-ghost rounded-2xl"
																onClick={() => {
																	requestDeleteModel(id);
																}}
																title="Delete Model Preset"
															>
																<FiTrash2 size={16} />
															</button>
														</>
													) : (
														<span className="ml-2 text-xs opacity-50">Built-in</span>
													)}
												</td>
											</tr>
										);
									})}
								</tbody>
							</table>
						)}
					</div>
				</div>
			)}

			<DeleteConfirmationModal
				isOpen={showDelProv}
				onClose={() => {
					setShowDelProv(false);
				}}
				onConfirm={confirmDeleteProvider}
				title="Delete Provider"
				message={`Delete provider “${provider}”? This action cannot be undone.`}
				confirmButtonText="Delete"
			/>

			<DeleteConfirmationModal
				isOpen={showDelModel}
				onClose={() => {
					setShowDelModel(false);
				}}
				onConfirm={confirmDeleteModel}
				title="Delete Model Preset"
				message={`Delete model preset “${selectedID}”? This action cannot be undone.`}
				confirmButtonText="Delete"
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
		</div>
	);
}
