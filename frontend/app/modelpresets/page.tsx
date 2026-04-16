import { useCallback, useEffect, useMemo, useState } from 'react';

import { FiPlus } from 'react-icons/fi';

import type { ProviderName } from '@/spec/inference';
import {
	type ModelPresetID,
	type PatchModelPresetPayload,
	type PatchProviderPresetPayload,
	type PostModelPresetPayload,
	type PostProviderPresetPayload,
	type ProviderPreset,
} from '@/spec/modelpreset';
import type { AuthKeyMeta } from '@/spec/setting';
import { AuthKeyTypeProvider } from '@/spec/setting';

import { aggregateAPI, modelPresetStoreAPI, settingstoreAPI } from '@/apis/baseapi';
import { getAllProviderPresetsMap } from '@/apis/list_helper';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DownloadButton } from '@/components/download_button';
import { Dropdown, type DropdownItem } from '@/components/dropdown';
import { Loader } from '@/components/loader';
import { PageFrame } from '@/components/page_frame';

import { AddEditProviderPresetModal } from '@/modelpresets/provider_add_edit_modal';
import { ProviderPresetCard } from '@/modelpresets/provider_presets_card';

const sortByDisplayName = ([, a]: [string, ProviderPreset], [, b]: [string, ProviderPreset]) =>
	a.displayName.localeCompare(b.displayName);

type CanonicalPageData = {
	settings: Awaited<ReturnType<typeof settingstoreAPI.getSettings>>;
	defaultProvider: ProviderName | undefined;
	providerPresets: Record<ProviderName, ProviderPreset>;
};

const buildProviderKeySet = (authKeys: AuthKeyMeta[]): Record<ProviderName, boolean> =>
	Object.fromEntries(
		authKeys.filter(key => key.type === AuthKeyTypeProvider).map(key => [key.keyName, key.nonEmpty])
	) as Record<ProviderName, boolean>;

async function fetchCanonicalPageData(): Promise<CanonicalPageData> {
	const [settings, defaultProvider, providerPresets] = await Promise.all([
		settingstoreAPI.getSettings(),
		modelPresetStoreAPI.getDefaultProvider(),
		getAllProviderPresetsMap(true),
	]);

	return {
		settings,
		defaultProvider,
		providerPresets,
	};
}

// eslint-disable-next-line no-restricted-exports
export default function ModelPresetsPage() {
	const [defaultProvider, setDefaultProvider] = useState<ProviderName | undefined>(undefined);
	const [providerPresets, setProviderPresets] = useState<Record<ProviderName, ProviderPreset>>({});
	const [providerKeySet, setProviderKeySet] = useState<Record<ProviderName, boolean>>({});
	const [authKeys, setAuthKeys] = useState<AuthKeyMeta[]>([]);

	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [showDenied, setShowDenied] = useState(false);
	const [deniedMsg, setDeniedMsg] = useState('');
	const [isRepairingDefaultProvider, setIsRepairingDefaultProvider] = useState(false);

	const [modalOpen, setModalOpen] = useState(false);
	const [modalMode, setModalMode] = useState<'add' | 'edit' | 'view'>('add');
	const [editProvider, setEditProvider] = useState<ProviderName | null>(null);

	const showGlobalDenied = useCallback((message: string) => {
		setDeniedMsg(message);
		setShowDenied(true);
	}, []);

	const applyCanonicalPageData = useCallback((data: CanonicalPageData) => {
		setDefaultProvider(data.defaultProvider);
		setProviderPresets(data.providerPresets);
		setAuthKeys(data.settings.authKeys);
		setProviderKeySet(buildProviderKeySet(data.settings.authKeys));
	}, []);

	const refreshCanonicalData = useCallback(async () => {
		const data = await fetchCanonicalPageData();
		applyCanonicalPageData(data);
	}, [applyCanonicalPageData]);

	const refreshCanonicalDataSafely = useCallback(async () => {
		try {
			await refreshCanonicalData();
		} catch (refreshError) {
			console.error('refresh canonical data error', refreshError);
		}
	}, [refreshCanonicalData]);

	const refreshAuthKeys = useCallback(async () => {
		const settings = await settingstoreAPI.getSettings();
		setAuthKeys(settings.authKeys);
		setProviderKeySet(buildProviderKeySet(settings.authKeys));
	}, []);

	useEffect(() => {
		let isActive = true;

		const loadInitialData = async () => {
			try {
				setLoading(true);
				setError(null);

				const data = await fetchCanonicalPageData();
				if (!isActive) return;

				applyCanonicalPageData(data);
			} catch (loadError) {
				console.error('init model presets error', loadError);
				if (!isActive) return;
				setError('Failed to load provider presets. Try again.');
			} finally {
				if (isActive) {
					setLoading(false);
				}
			}
		};

		void loadInitialData();

		return () => {
			isActive = false;
		};
	}, [applyCanonicalPageData]);

	const enabledProviderNames = useMemo(
		() =>
			Object.values(providerPresets)
				.filter(providerPreset => providerPreset.isEnabled)
				.map(providerPreset => providerPreset.name),
		[providerPresets]
	);

	const enabledProviderPresets = useMemo<Partial<Record<ProviderName, ProviderPreset>>>(() => {
		const out: Partial<Record<ProviderName, ProviderPreset>> = {};
		for (const [name, preset] of Object.entries(providerPresets)) {
			if (preset.isEnabled) {
				out[name] = preset;
			}
		}
		return out;
	}, [providerPresets]);

	const safeDefaultKey: ProviderName | undefined =
		defaultProvider && enabledProviderPresets[defaultProvider] ? defaultProvider : undefined;

	useEffect(() => {
		if (loading || isRepairingDefaultProvider) return;
		if (enabledProviderNames.length === 0) return;
		if (defaultProvider && providerPresets[defaultProvider]?.isEnabled) return;

		const fallbackProvider = enabledProviderNames[0];
		if (!fallbackProvider) return;

		let isActive = true;

		const repairDefaultProvider = async () => {
			try {
				setIsRepairingDefaultProvider(true);
				await modelPresetStoreAPI.patchDefaultProvider(fallbackProvider);
				if (isActive) {
					await refreshCanonicalData();
				}
			} catch (repairError) {
				console.error(repairError);
				if (isActive) {
					showGlobalDenied('Failed setting default provider.');
				}
			} finally {
				if (isActive) {
					setIsRepairingDefaultProvider(false);
				}
			}
		};

		void repairDefaultProvider();

		return () => {
			isActive = false;
		};
	}, [
		defaultProvider,
		enabledProviderNames,
		isRepairingDefaultProvider,
		loading,
		providerPresets,
		refreshCanonicalData,
		showGlobalDenied,
	]);

	const handleDefaultProviderChange = useCallback(
		async (providerName: ProviderName) => {
			setDefaultProvider(providerName);

			try {
				await modelPresetStoreAPI.patchDefaultProvider(providerName);
				await refreshCanonicalData();
			} catch (changeError) {
				console.error(changeError);
				await refreshCanonicalDataSafely();
				showGlobalDenied('Failed changing default provider.');
			}
		},
		[refreshCanonicalData, refreshCanonicalDataSafely, showGlobalDenied]
	);

	const handleToggleProvider = useCallback(
		async (providerName: ProviderName, nextEnabled: boolean) => {
			const providerPreset = providerPresets[providerName];
			if (!providerPreset) {
				throw new Error('Provider not found.');
			}

			if (!nextEnabled) {
				if (providerName === defaultProvider) {
					throw new Error('Cannot disable the current default provider. Pick another default first.');
				}

				const enabledCount = Object.values(providerPresets).filter(preset => preset.isEnabled).length;
				if (providerPreset.isEnabled && enabledCount === 1) {
					throw new Error('Cannot disable the last enabled provider.');
				}
			}

			try {
				await modelPresetStoreAPI.patchProviderPreset(providerName, { isEnabled: nextEnabled });
				if (nextEnabled && !defaultProvider) {
					await modelPresetStoreAPI.patchDefaultProvider(providerName);
				}

				await refreshCanonicalData();
			} catch (toggleError) {
				console.error(toggleError);
				await refreshCanonicalDataSafely();
				throw new Error('Failed toggling provider.');
			}
		},
		[defaultProvider, providerPresets, refreshCanonicalData, refreshCanonicalDataSafely]
	);

	const handleDeleteProvider = useCallback(
		async (providerName: ProviderName) => {
			const providerPreset = providerPresets[providerName];
			if (!providerPreset) {
				throw new Error('Provider not found.');
			}

			if (providerPreset.isBuiltIn) {
				throw new Error('Built-in providers cannot be deleted.');
			}

			if (providerName === defaultProvider) {
				throw new Error('Cannot delete the current default provider. Pick another default first.');
			}

			if (Object.keys(providerPreset.modelPresets).length > 0) {
				throw new Error('Only empty providers can be deleted. Remove all model presets first.');
			}

			try {
				await aggregateAPI.deleteProviderPreset(providerName);
				await aggregateAPI.deleteAuthKey(AuthKeyTypeProvider, providerName).catch(() => void 0);
				await refreshCanonicalData();
			} catch (deleteError) {
				console.error(deleteError);
				await refreshCanonicalDataSafely();
				throw new Error('Failed deleting provider.');
			}
		},
		[defaultProvider, providerPresets, refreshCanonicalData, refreshCanonicalDataSafely]
	);

	const handleSetDefaultModel = useCallback(
		async (providerName: ProviderName, modelPresetID: ModelPresetID) => {
			const providerPreset = providerPresets[providerName];
			if (!providerPreset) {
				throw new Error('Provider not found.');
			}

			if (!providerPreset.modelPresets[modelPresetID]) {
				throw new Error('Model preset not found.');
			}

			try {
				await modelPresetStoreAPI.patchProviderPreset(providerName, { defaultModelPresetID: modelPresetID });
				await refreshCanonicalData();
			} catch (changeError) {
				console.error(changeError);
				await refreshCanonicalDataSafely();
				throw new Error('Failed setting default model.');
			}
		},
		[providerPresets, refreshCanonicalData, refreshCanonicalDataSafely]
	);

	const handleToggleModel = useCallback(
		async (providerName: ProviderName, modelPresetID: ModelPresetID, nextEnabled: boolean) => {
			const providerPreset = providerPresets[providerName];
			if (!providerPreset) {
				throw new Error('Provider not found.');
			}

			const modelPreset = providerPreset.modelPresets[modelPresetID];
			if (!modelPreset) {
				throw new Error('Model preset not found.');
			}

			if (!nextEnabled && providerPreset.defaultModelPresetID === modelPresetID && modelPreset.isEnabled) {
				throw new Error('Cannot disable the default model preset. Choose another default first.');
			}

			try {
				await modelPresetStoreAPI.patchModelPreset(providerName, modelPresetID, { isEnabled: nextEnabled });
				await refreshCanonicalData();
			} catch (toggleError) {
				console.error(toggleError);
				await refreshCanonicalDataSafely();
				throw new Error('Failed toggling model.');
			}
		},
		[providerPresets, refreshCanonicalData, refreshCanonicalDataSafely]
	);

	const handleCreateModel = useCallback(
		async (providerName: ProviderName, modelPresetID: ModelPresetID, payload: PostModelPresetPayload) => {
			const providerPreset = providerPresets[providerName];
			if (!providerPreset) {
				throw new Error('Provider not found.');
			}

			try {
				await modelPresetStoreAPI.postModelPreset(providerName, modelPresetID, payload);

				if (!providerPreset.defaultModelPresetID) {
					await modelPresetStoreAPI.patchProviderPreset(providerName, { defaultModelPresetID: modelPresetID });
				}

				await refreshCanonicalData();
			} catch (saveError) {
				console.error(saveError);
				await refreshCanonicalDataSafely();
				throw new Error('Failed saving model preset.');
			}
		},
		[providerPresets, refreshCanonicalData, refreshCanonicalDataSafely]
	);

	const handlePatchModel = useCallback(
		async (providerName: ProviderName, modelPresetID: ModelPresetID, payload: PatchModelPresetPayload) => {
			const providerPreset = providerPresets[providerName];
			if (!providerPreset) {
				throw new Error('Provider not found.');
			}

			if (Object.keys(payload).length === 0) return;

			try {
				await modelPresetStoreAPI.patchModelPreset(providerName, modelPresetID, payload);
				await refreshCanonicalData();
			} catch (patchError) {
				console.error(patchError);
				await refreshCanonicalDataSafely();
				throw new Error('Failed updating model preset.');
			}
		},
		[providerPresets, refreshCanonicalData, refreshCanonicalDataSafely]
	);

	const handleDeleteModel = useCallback(
		async (providerName: ProviderName, modelPresetID: ModelPresetID) => {
			const providerPreset = providerPresets[providerName];
			if (!providerPreset) {
				throw new Error('Provider not found.');
			}

			const modelPreset = providerPreset.modelPresets[modelPresetID];
			if (!modelPreset) {
				throw new Error('Model preset not found.');
			}

			if (modelPreset.isBuiltIn) {
				throw new Error('Built-in model presets cannot be deleted.');
			}

			const remainingModelIDs = Object.keys(providerPreset.modelPresets).filter(id => id !== modelPresetID);

			try {
				await modelPresetStoreAPI.deleteModelPreset(providerName, modelPresetID);

				if (providerPreset.defaultModelPresetID === modelPresetID && remainingModelIDs.length > 0) {
					await modelPresetStoreAPI.patchProviderPreset(providerName, { defaultModelPresetID: remainingModelIDs[0] });
				}

				await refreshCanonicalData();
			} catch (deleteError) {
				console.error(deleteError);
				await refreshCanonicalDataSafely();
				throw new Error('Failed deleting model preset.');
			}
		},
		[providerPresets, refreshCanonicalData, refreshCanonicalDataSafely]
	);

	const handleCreateProvider = useCallback(
		async (providerName: ProviderName, payload: PostProviderPresetPayload, apiKey: string | null) => {
			try {
				await aggregateAPI.postProviderPreset(providerName, payload);

				if (apiKey && apiKey.trim()) {
					await aggregateAPI.setAuthKey(AuthKeyTypeProvider, providerName, apiKey.trim());
				}

				if (!defaultProvider && payload.isEnabled) {
					await modelPresetStoreAPI.patchDefaultProvider(providerName);
				}

				await refreshCanonicalData();
			} catch (createError) {
				console.error(createError);
				const message = 'Failed adding provider.';
				showGlobalDenied(message);
				throw new Error(message);
			}
		},
		[defaultProvider, refreshCanonicalData, showGlobalDenied]
	);

	const handlePatchProvider = useCallback(
		async (providerName: ProviderName, payload: PatchProviderPresetPayload, apiKey: string | null) => {
			if (providerName === defaultProvider && payload.isEnabled === false) {
				const message = 'Cannot disable the current default provider. Pick another default first.';
				showGlobalDenied(message);
				throw new Error(message);
			}

			const hasStorePatch = Object.keys(payload).length > 0;
			const hasKeyPatch = Boolean(apiKey && apiKey.trim());
			if (!hasStorePatch && !hasKeyPatch) return;

			try {
				if (hasStorePatch) {
					await modelPresetStoreAPI.patchProviderPreset(providerName, payload);
				}

				if (apiKey && apiKey.trim()) {
					await aggregateAPI.setAuthKey(AuthKeyTypeProvider, providerName, apiKey.trim());
				}

				if (!defaultProvider && payload.isEnabled === true) {
					await modelPresetStoreAPI.patchDefaultProvider(providerName);
				}

				await refreshCanonicalData();
			} catch (patchError) {
				console.error(patchError);
				await refreshCanonicalDataSafely();
				const message = 'Failed updating provider.';

				showGlobalDenied(message);
				throw new Error(message);
			}
		},
		[defaultProvider, refreshCanonicalData, refreshCanonicalDataSafely, showGlobalDenied]
	);

	const handleProviderAuthKeyChanged = useCallback(
		async (providerName: ProviderName) => {
			setProviderKeySet(prev => ({ ...prev, [providerName]: true }));

			try {
				await refreshAuthKeys();
			} catch (refreshError) {
				console.error(refreshError);
				throw new Error('Failed refreshing auth key state.');
			}
		},
		[refreshAuthKeys]
	);

	const fetchValue = useCallback(async () => {
		try {
			const data = await fetchCanonicalPageData();
			return JSON.stringify(
				{
					defaultProvider: data.defaultProvider,
					providers: data.providerPresets,
				},
				null,
				2
			);
		} catch (fetchError) {
			console.log('fetch preset error', fetchError);
			showGlobalDenied('Failed fetching presets.');
			return '';
		}
	}, [showGlobalDenied]);

	const openAddModal = () => {
		setModalMode('add');
		setEditProvider(null);
		setModalOpen(true);
	};

	const openEditModal = (providerName: ProviderName) => {
		const providerPreset = providerPresets[providerName];
		if (!providerPreset) {
			showGlobalDenied('Provider not found.');
			return;
		}

		setModalMode(providerPreset.isBuiltIn ? 'view' : 'edit');
		setEditProvider(providerName);
		setModalOpen(true);
	};

	if (loading) return <Loader text="Loading model presets…" />;

	return (
		<PageFrame>
			<div className="flex h-full w-full flex-col items-center">
				<div className="fixed mt-8 flex w-10/12 items-center p-2 lg:w-2/3">
					<h1 className="flex grow items-center justify-center text-xl font-semibold">Model Presets</h1>
					<DownloadButton
						title="Download Model Presets"
						language="json"
						valueFetcher={fetchValue}
						size={20}
						fileprefix="modelpresets"
						className="btn btn-sm btn-ghost"
					/>
				</div>

				<div
					className="mt-24 flex w-full grow flex-col items-center overflow-y-auto"
					style={{ maxHeight: 'calc(100vh - 128px)' }}
				>
					<div className="flex w-5/6 flex-col space-y-4 xl:w-2/3">
						<div className="bg-base-100 mb-8 rounded-2xl px-4 py-2 shadow-lg">
							<div className="grid grid-cols-12 items-center gap-4">
								<label className="col-span-3 text-sm font-medium">Default Provider</label>

								<div className="col-span-6">
									{enabledProviderNames.length > 0 && safeDefaultKey ? (
										<Dropdown<ProviderName>
											dropdownItems={enabledProviderPresets as Record<string, DropdownItem>}
											selectedKey={safeDefaultKey}
											onChange={handleDefaultProviderChange}
											filterDisabled={false}
											title="Select default provider"
											getDisplayName={k => enabledProviderPresets[k]?.displayName ?? ''}
										/>
									) : (
										<span className="text-error text-sm">Enable at least one provider first.</span>
									)}
								</div>

								<div className="col-span-3 flex justify-end">
									<button className="btn btn-ghost flex items-center rounded-2xl" onClick={openAddModal}>
										<FiPlus /> <span className="ml-1">Add Provider</span>
									</button>
								</div>
							</div>
						</div>

						{error && <p className="text-error mt-8 text-center">{error}</p>}

						{!error &&
							Object.entries(providerPresets)
								.sort(sortByDisplayName)
								.map(([name, preset]) => (
									<ProviderPresetCard
										key={name}
										provider={name}
										preset={preset}
										defaultProvider={defaultProvider}
										authKeySet={providerKeySet[name]}
										authKeys={authKeys}
										enabledProviders={enabledProviderNames}
										allProviderPresets={providerPresets}
										onToggleProvider={handleToggleProvider}
										onDeleteProvider={handleDeleteProvider}
										onRequestEdit={openEditModal}
										onSetDefaultModel={handleSetDefaultModel}
										onToggleModel={handleToggleModel}
										onCreateModel={handleCreateModel}
										onPatchModel={handlePatchModel}
										onDeleteModel={handleDeleteModel}
										onProviderAuthKeyChanged={handleProviderAuthKeyChanged}
									/>
								))}
					</div>
				</div>

				<AddEditProviderPresetModal
					isOpen={modalOpen}
					mode={modalMode}
					onClose={() => {
						setModalOpen(false);
					}}
					onSubmit={async (name, payload, apiKey) => {
						if (modalMode === 'view') return;
						if (modalMode === 'add') {
							await handleCreateProvider(name, payload as PostProviderPresetPayload, apiKey);
							return;
						}

						await handlePatchProvider(name, payload as PatchProviderPresetPayload, apiKey);
					}}
					existingProviderNames={Object.keys(providerPresets)}
					allProviderPresets={providerPresets}
					initialPreset={editProvider ? providerPresets[editProvider] : undefined}
					apiKeyAlreadySet={editProvider ? providerKeySet[editProvider] : false}
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
		</PageFrame>
	);
}
