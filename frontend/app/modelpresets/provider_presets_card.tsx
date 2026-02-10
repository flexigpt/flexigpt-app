import { useEffect, useMemo, useState } from 'react';

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
import { type ModelPreset, type ModelPresetID, type ProviderPreset } from '@/spec/modelpreset';
import type { AuthKeyMeta } from '@/spec/setting';
import { AuthKeyTypeProvider } from '@/spec/setting';

import { modelPresetStoreAPI, settingstoreAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Dropdown } from '@/components/dropdown';

import { AddEditModelPresetModal } from '@/modelpresets/modelpreset_add_edit_modal';
import { AddEditAuthKeyModal } from '@/settings/authkey_add_edit_modal';

interface ProviderPresetCardProps {
	provider: ProviderName;
	preset: ProviderPreset;
	defaultProvider: ProviderName;
	authKeySet: boolean;
	enabledProviders: ProviderName[];
	allProviderPresets: Record<ProviderName, ProviderPreset>;

	onProviderPresetChange: (provider: ProviderName, newPreset: ProviderPreset) => void;
	onProviderDelete: (provider: ProviderName) => Promise<void>;
	onRequestEdit: (provider: ProviderName) => void; // opens edit OR view modal in parent
}

export function ProviderPresetCard({
	provider,
	preset,
	defaultProvider,
	authKeySet,
	enabledProviders,
	allProviderPresets,
	onProviderPresetChange,
	onProviderDelete,
	onRequestEdit,
}: ProviderPresetCardProps) {
	const [expanded, setExpanded] = useState(false);

	const [keySet, setKeySet] = useState(authKeySet);
	useEffect(() => {
		setKeySet(authKeySet);
	}, [authKeySet]);

	const [selectedID, setSelectedID] = useState<ModelPresetID | null>(null);
	const [modelModalMode, setModelModalMode] = useState<'add' | 'edit' | 'view'>('add');

	const [showModModal, setShowModModal] = useState(false);
	const [showDelProv, setShowDelProv] = useState(false);
	const [showDelModel, setShowDelModel] = useState(false);
	const [showDenied, setShowDenied] = useState(false);
	const [deniedMsg, setDeniedMsg] = useState('');

	const [showKeyModal, setShowKeyModal] = useState(false);
	const [authKeys, setAuthKeys] = useState<AuthKeyMeta[]>([]);
	const [keyModalInitial, setKeyModalInitial] = useState<AuthKeyMeta | null>(null);

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
		const o: Record<ProviderName, Record<ModelPresetID, ModelPreset>> = {};
		for (const [prov, pp] of Object.entries(allProviderPresets)) {
			o[prov] = pp.modelPresets;
		}
		return o;
	}, [allProviderPresets]);

	const toggleProviderEnable = async () => {
		if (provider === defaultProvider && preset.isEnabled) {
			setDeniedMsg('Cannot disable the default provider. Pick another default first.');
			setShowDenied(true);
			return;
		}
		if (isLastEnabled && preset.isEnabled) {
			setDeniedMsg('Cannot disable the last enabled provider.');
			setShowDenied(true);
			return;
		}

		try {
			await modelPresetStoreAPI.patchProviderPreset(provider, !preset.isEnabled);
			onProviderPresetChange(provider, { ...preset, isEnabled: !preset.isEnabled });
		} catch {
			setDeniedMsg('Failed toggling provider.');
			setShowDenied(true);
		}
	};

	const toggleExpand = () => {
		setExpanded(p => !p);
	};

	const requestDeleteProvider = () => {
		if (providerIsBuiltIn) {
			setDeniedMsg('Built-in providers cannot be deleted.');
			setShowDenied(true);
			return;
		}
		if (hasModels) {
			setDeniedMsg('Only empty providers can be deleted. Remove all model presets first.');
			setShowDenied(true);
			return;
		}
		setShowDelProv(true);
	};

	const confirmDeleteProvider = async () => {
		await onProviderDelete(provider);
		setShowDelProv(false);
	};

	const handleDefaultModelChange = async (id: ModelPresetID) => {
		try {
			await modelPresetStoreAPI.patchProviderPreset(provider, undefined, id);
			onProviderPresetChange(provider, { ...preset, defaultModelPresetID: id });
		} catch {
			setDeniedMsg('Failed setting default model.');
			setShowDenied(true);
		}
	};

	const toggleModelEnable = async (id: ModelPresetID) => {
		const m = modelPresets[id];
		if (id === defaultModelPresetID && m.isEnabled) {
			setDeniedMsg('Cannot disable the default model preset. Choose another default first.');
			setShowDenied(true);
			return;
		}
		try {
			await modelPresetStoreAPI.patchModelPreset(provider, id, !m.isEnabled);
			onProviderPresetChange(provider, {
				...preset,
				modelPresets: { ...modelPresets, [id]: { ...m, isEnabled: !m.isEnabled } },
			});
		} catch {
			setDeniedMsg('Failed toggling model.');
			setShowDenied(true);
		}
	};

	const openAddModel = () => {
		if (providerIsBuiltIn) {
			setDeniedMsg('Cannot add model presets to a built-in provider.');
			setShowDenied(true);
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

	const handleModifyModelSubmit = async (id: ModelPresetID, data: ModelPreset) => {
		try {
			const { id: _id, isBuiltIn: _isBuiltIn, ...payload } = data;
			await modelPresetStoreAPI.putModelPreset(provider, id, payload);

			let newDefault = defaultModelPresetID;
			if (!defaultModelPresetID) {
				await modelPresetStoreAPI.patchProviderPreset(provider, undefined, id);
				newDefault = id;
			}

			onProviderPresetChange(provider, {
				...preset,
				modelPresets: { ...modelPresets, [id]: data },
				defaultModelPresetID: newDefault,
			});

			setShowModModal(false);
		} catch {
			setDeniedMsg('Failed saving model preset.');
			setShowDenied(true);
		}
	};

	const requestDeleteModel = (id: ModelPresetID) => {
		if (modelPresets[id].isBuiltIn) {
			setDeniedMsg('Built-in model presets cannot be deleted.');
			setShowDenied(true);
			return;
		}
		setSelectedID(id);
		setShowDelModel(true);
	};

	const confirmDeleteModel = async () => {
		if (!selectedID) return;
		try {
			await modelPresetStoreAPI.deleteModelPreset(provider, selectedID);
			const { [selectedID]: _removed, ...rest } = modelPresets;

			let newDefault = defaultModelPresetID;
			if (selectedID === defaultModelPresetID) {
				newDefault = Object.keys(rest)[0] ?? '';
				if (newDefault !== '') {
					await modelPresetStoreAPI.patchProviderPreset(provider, undefined, newDefault);
				}
			}

			onProviderPresetChange(provider, {
				...preset,
				modelPresets: rest,
				defaultModelPresetID: newDefault,
			});

			setShowDelModel(false);
		} catch {
			setDeniedMsg('Failed deleting model preset.');
			setShowDenied(true);
		}
	};

	const openSetApiKey = async () => {
		try {
			const settings = await settingstoreAPI.getSettings();
			setAuthKeys(settings.authKeys);

			const meta = settings.authKeys.find(k => k.type === AuthKeyTypeProvider && k.keyName === provider) ?? null;

			setKeyModalInitial(meta);
			setShowKeyModal(true);
		} catch (err) {
			console.error(err);
			setDeniedMsg('Failed loading auth keys.');
			setShowDenied(true);
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
							{keySet ? <FiCheckCircle className="text-success mx-1" /> : <FiXCircle className="text-error mx-1" />}
						</div>

						<div className="flex items-center">
							<span className="text-sm">Details</span>
							{expanded ? <FiChevronUp className="mx-1" /> : <FiChevronDown className="mx-1" />}
						</div>
					</div>
				</div>
			</div>

			{/* body: allow even when provider disabled */}
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
													title={'Delete Provider'}
													disabled={!canDeleteProvider}
												>
													<FiTrash2 />
													<span className="ml-1 hidden md:inline">Delete Provider</span>
												</button>
											</span>

											<div className="flex gap-2">
												<button
													className="btn btn-ghost flex items-center rounded-2xl"
													onClick={openSetApiKey}
													title={keySet ? 'Update API Key' : 'Set API Key'}
												>
													<FiKey />
													<span className="ml-1 hidden md:inline">{keySet ? 'Update API Key' : 'Set API Key'}</span>
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
									{modelEntries.map(([id, m]) => {
										const canModify = !m.isBuiltIn;
										return (
											<tr key={id} className="hover:bg-base-300">
												<td>{m.displayName || id}</td>
												<td>{m.name}</td>
												<td className="text-center">
													<input
														type="checkbox"
														className="toggle toggle-accent"
														checked={m.isEnabled}
														onChange={() => toggleModelEnable(id)}
													/>
												</td>
												<td className="text-center">
													{'reasoning' in m && m.reasoning ? (
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
					setKeySet(true);
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
