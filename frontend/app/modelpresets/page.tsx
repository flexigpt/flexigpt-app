import { useCallback, useEffect, useRef, useState } from 'react';

import { FiPlus } from 'react-icons/fi';

import type { ProviderName } from '@/spec/inference';
import type {
	ModelPresetID,
	PatchModelPresetPayload,
	PatchProviderPresetPayload,
	PostModelPresetPayload,
	PostProviderPresetPayload,
	ProviderPreset,
} from '@/spec/modelpreset';
import type { AuthKeyMeta } from '@/spec/setting';
import { AuthKeyTypeProvider } from '@/spec/setting';

import { throwIfAborted } from '@/lib/async_utils';
import { redactSensitiveHTTPHeaders } from '@/lib/http_input_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { aggregateAPI, modelPresetStoreAPI, settingstoreAPI } from '@/apis/baseapi';
import { getAllProviderPresetsMap } from '@/apis/list_helper';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DownloadButton } from '@/components/download_button';
import type { DropdownItem } from '@/components/dropdown';
import { Dropdown } from '@/components/dropdown';
import { Loader } from '@/components/loader';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { PageFrame } from '@/components/page_frame';

import { AddEditProviderPresetModal } from '@/modelpresets/provider_add_edit_modal';
import { ProviderPresetCard } from '@/modelpresets/provider_presets_card';

const EMPTY_PROVIDER_PRESETS = {} as Record<ProviderName, ProviderPreset>;
const EMPTY_AUTH_KEYS: AuthKeyMeta[] = [];

const sortByDisplayName = ([, a]: [string, ProviderPreset], [, b]: [string, ProviderPreset]) =>
	a.displayName.localeCompare(b.displayName);

interface CanonicalPageData {
	settings: Awaited<ReturnType<typeof settingstoreAPI.getSettings>>;
	defaultProvider: ProviderName | undefined;
	providerPresets: Record<ProviderName, ProviderPreset>;
}

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

// oxlint-disable-next-line no-restricted-exports
export default function ModelPresetsPage() {
	const loadCanonicalData = useCallback(async (signal: AbortSignal): Promise<CanonicalPageData> => {
		const data = await fetchCanonicalPageData();
		throwIfAborted(signal);
		return data;
	}, []);

	const {
		data: canonicalData,
		error: canonicalLoadError,
		isLoading,
		isRefreshing,
		reloadOrThrow: reloadCanonicalData,
	} = useAsyncResource(loadCanonicalData, {
		initialData: null as CanonicalPageData | null,
	});

	/*
	 * Action callbacks intentionally read from this ref rather than capturing
	 * `providerPresets` and `defaultProvider`. This keeps callback identities
	 * stable while ensuring actions use the latest committed server state.
	 */
	const canonicalDataRef = useRef<CanonicalPageData | null>(canonicalData);

	useEffect(() => {
		canonicalDataRef.current = canonicalData;
	}, [canonicalData]);

	const defaultProvider = canonicalData?.defaultProvider;
	const providerPresets = canonicalData?.providerPresets ?? EMPTY_PROVIDER_PRESETS;
	const authKeys = canonicalData?.settings.authKeys ?? EMPTY_AUTH_KEYS;
	const providerKeySet = buildProviderKeySet(authKeys);

	const [showDenied, setShowDenied] = useState(false);
	const [deniedMsg, setDeniedMsg] = useState('');

	const [modalOpen, setModalOpen] = useState(false);
	const [modalMode, setModalMode] = useState<'add' | 'edit' | 'view'>('add');
	const [editProvider, setEditProvider] = useState<ProviderName | null>(null);

	const showGlobalDenied = useCallback((message: string) => {
		setDeniedMsg(message);
		setShowDenied(true);
	}, []);

	const refreshCanonicalData = useCallback(async () => {
		await reloadCanonicalData();
	}, [reloadCanonicalData]);

	const refreshCanonicalDataSafely = useCallback(async () => {
		try {
			await refreshCanonicalData();
		} catch (refreshError) {
			console.error('refresh canonical data error', refreshError);
		}
	}, [refreshCanonicalData]);

	const enabledProviderNames = Object.values(providerPresets)
		.filter(providerPreset => providerPreset.isEnabled)
		.map(providerPreset => providerPreset.name);

	const enabledProviderPresets: Partial<Record<ProviderName, ProviderPreset>> = {};

	for (const [name, preset] of Object.entries(providerPresets)) {
		if (preset.isEnabled) {
			enabledProviderPresets[name as ProviderName] = preset;
		}
	}

	const safeDefaultKey: ProviderName | undefined =
		defaultProvider && enabledProviderPresets[defaultProvider] ? defaultProvider : enabledProviderNames[0];

	const defaultProviderNeedsRepair = Boolean(safeDefaultKey && safeDefaultKey !== defaultProvider);

	const handleDefaultProviderChange = useCallback(
		async (providerName: ProviderName) => {
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
			const currentCanonicalData = canonicalDataRef.current;
			const currentProviderPresets = currentCanonicalData?.providerPresets ?? EMPTY_PROVIDER_PRESETS;
			const currentDefaultProvider = currentCanonicalData?.defaultProvider;
			const providerPreset = currentProviderPresets[providerName];

			if (!providerPreset) {
				throw new Error('Provider not found.');
			}

			if (!nextEnabled) {
				if (providerName === currentDefaultProvider) {
					throw new Error('Cannot disable the current default provider. Pick another default first.');
				}

				const enabledCount = Object.values(currentProviderPresets).filter(preset => preset.isEnabled).length;

				if (providerPreset.isEnabled && enabledCount === 1) {
					throw new Error('Cannot disable the last enabled provider.');
				}
			}

			try {
				await modelPresetStoreAPI.patchProviderPreset(providerName, { isEnabled: nextEnabled });

				if (nextEnabled && !currentDefaultProvider) {
					await modelPresetStoreAPI.patchDefaultProvider(providerName);
				}

				await refreshCanonicalData();
			} catch (toggleError) {
				console.error(toggleError);
				await refreshCanonicalDataSafely();
				throw new Error('Failed toggling provider.', { cause: toggleError });
			}
		},
		[refreshCanonicalData, refreshCanonicalDataSafely]
	);

	const handleDeleteProvider = useCallback(
		async (providerName: ProviderName) => {
			const currentCanonicalData = canonicalDataRef.current;
			const currentProviderPresets = currentCanonicalData?.providerPresets ?? EMPTY_PROVIDER_PRESETS;
			const currentDefaultProvider = currentCanonicalData?.defaultProvider;
			const providerPreset = currentProviderPresets[providerName];

			if (!providerPreset) {
				throw new Error('Provider not found.');
			}

			if (providerPreset.isBuiltIn) {
				throw new Error('Built-in providers cannot be deleted.');
			}

			if (providerName === currentDefaultProvider) {
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
				throw new Error('Failed deleting provider.', { cause: deleteError });
			}
		},
		[refreshCanonicalData, refreshCanonicalDataSafely]
	);

	const handleSetDefaultModel = useCallback(
		async (providerName: ProviderName, modelPresetID: ModelPresetID) => {
			const currentProviderPresets = canonicalDataRef.current?.providerPresets ?? EMPTY_PROVIDER_PRESETS;
			const providerPreset = currentProviderPresets[providerName];

			if (!providerPreset) {
				throw new Error('Provider not found.');
			}

			if (!providerPreset.isEnabled) {
				throw new Error('Enable the provider before selecting its default model.');
			}

			if (!providerPreset.modelPresets[modelPresetID]) {
				throw new Error('Model preset not found.');
			}

			if (!providerPreset.modelPresets[modelPresetID].isEnabled) {
				throw new Error('Enable the model preset before selecting it as the default.');
			}

			try {
				await modelPresetStoreAPI.patchProviderPreset(providerName, {
					defaultModelPresetID: modelPresetID,
				});
				await refreshCanonicalData();
			} catch (changeError) {
				console.error(changeError);
				await refreshCanonicalDataSafely();
				throw new Error('Failed setting default model.', { cause: changeError });
			}
		},
		[refreshCanonicalData, refreshCanonicalDataSafely]
	);

	const handleToggleModel = useCallback(
		async (providerName: ProviderName, modelPresetID: ModelPresetID, nextEnabled: boolean) => {
			const currentProviderPresets = canonicalDataRef.current?.providerPresets ?? EMPTY_PROVIDER_PRESETS;
			const providerPreset = currentProviderPresets[providerName];

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
				await modelPresetStoreAPI.patchModelPreset(providerName, modelPresetID, {
					isEnabled: nextEnabled,
				});
				await refreshCanonicalData();
			} catch (toggleError) {
				console.error(toggleError);
				await refreshCanonicalDataSafely();
				throw new Error('Failed toggling model.', { cause: toggleError });
			}
		},
		[refreshCanonicalData, refreshCanonicalDataSafely]
	);

	const handleCreateModel = useCallback(
		async (providerName: ProviderName, modelPresetID: ModelPresetID, payload: PostModelPresetPayload) => {
			const currentProviderPresets = canonicalDataRef.current?.providerPresets ?? EMPTY_PROVIDER_PRESETS;
			const providerPreset = currentProviderPresets[providerName];

			if (!providerPreset) {
				throw new Error('Provider not found.');
			}

			let modelWasCreated = false;
			try {
				await modelPresetStoreAPI.postModelPreset(providerName, modelPresetID, payload);
				modelWasCreated = true;

				if (!providerPreset.defaultModelPresetID) {
					await modelPresetStoreAPI.patchProviderPreset(providerName, {
						defaultModelPresetID: modelPresetID,
					});
				}
			} catch (saveError) {
				console.error(saveError);
				await refreshCanonicalDataSafely();

				if (modelWasCreated) {
					showGlobalDenied(
						'The model preset was saved, but its default-model update failed. Select a default model manually before using it.'
					);
					return;
				}

				throw new Error('Failed saving model preset.', { cause: saveError });
			}

			try {
				await refreshCanonicalData();
			} catch (refreshError) {
				console.error('Model preset was saved but provider refresh failed:', refreshError);
				showGlobalDenied(
					'The model preset was saved, but the provider could not be refreshed. Reload before making more changes.'
				);
			}
		},
		[refreshCanonicalData, refreshCanonicalDataSafely, showGlobalDenied]
	);

	const handlePatchModel = useCallback(
		async (providerName: ProviderName, modelPresetID: ModelPresetID, payload: PatchModelPresetPayload) => {
			const currentProviderPresets = canonicalDataRef.current?.providerPresets ?? EMPTY_PROVIDER_PRESETS;
			const providerPreset = currentProviderPresets[providerName];

			if (!providerPreset) {
				throw new Error('Provider not found.');
			}

			if (Object.keys(payload).length === 0) {
				return;
			}

			try {
				await modelPresetStoreAPI.patchModelPreset(providerName, modelPresetID, payload);
				await refreshCanonicalData();
			} catch (patchError) {
				console.error(patchError);
				await refreshCanonicalDataSafely();
				throw new Error('Failed updating model preset.', { cause: patchError });
			}
		},
		[refreshCanonicalData, refreshCanonicalDataSafely]
	);

	const handleDeleteModel = useCallback(
		async (providerName: ProviderName, modelPresetID: ModelPresetID) => {
			const currentProviderPresets = canonicalDataRef.current?.providerPresets ?? EMPTY_PROVIDER_PRESETS;
			const providerPreset = currentProviderPresets[providerName];

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

			if (providerPreset.defaultModelPresetID === modelPresetID) {
				throw new Error('Choose another default model before deleting the current default model preset.');
			}

			try {
				await modelPresetStoreAPI.deleteModelPreset(providerName, modelPresetID);
				await refreshCanonicalData();
			} catch (deleteError) {
				console.error(deleteError);
				await refreshCanonicalDataSafely();
				throw new Error('Failed deleting model preset.', { cause: deleteError });
			}
		},
		[refreshCanonicalData, refreshCanonicalDataSafely]
	);

	const handleCreateProvider = useCallback(
		async (providerName: ProviderName, payload: PostProviderPresetPayload, apiKey: string | null) => {
			const currentDefaultProvider = canonicalDataRef.current?.defaultProvider;

			let providerWasCreated = false;
			try {
				await aggregateAPI.postProviderPreset(providerName, payload);
				providerWasCreated = true;

				if (apiKey?.trim()) {
					await aggregateAPI.setAuthKey(AuthKeyTypeProvider, providerName, apiKey.trim());
				}

				if (!currentDefaultProvider && payload.isEnabled) {
					await modelPresetStoreAPI.patchDefaultProvider(providerName);
				}
			} catch (createError) {
				console.error(createError);

				if (providerWasCreated) {
					await refreshCanonicalDataSafely();
					showGlobalDenied(
						'The provider was created, but its API key or default-provider setup was not completed. Review the provider before sending requests.'
					);
					return;
				}

				const message = 'Failed adding provider.';
				showGlobalDenied(message);
				throw new Error(message, { cause: createError });
			}

			try {
				await refreshCanonicalData();
			} catch (refreshError) {
				console.error('Provider was created but page refresh failed:', refreshError);
				showGlobalDenied(
					'The provider was created, but the page could not be refreshed. Reload before making more changes.'
				);
			}
		},
		[refreshCanonicalData, refreshCanonicalDataSafely, showGlobalDenied]
	);

	const handlePatchProvider = useCallback(
		async (providerName: ProviderName, payload: PatchProviderPresetPayload, apiKey: string | null) => {
			const currentDefaultProvider = canonicalDataRef.current?.defaultProvider;

			if (providerName === currentDefaultProvider && payload.isEnabled === false) {
				const message = 'Cannot disable the current default provider. Pick another default first.';
				showGlobalDenied(message);
				throw new Error(message);
			}

			const hasStorePatch = Object.keys(payload).length > 0;
			const hasKeyPatch = Boolean(apiKey?.trim());

			if (!hasStorePatch && !hasKeyPatch) {
				return;
			}

			try {
				if (hasStorePatch) {
					await modelPresetStoreAPI.patchProviderPreset(providerName, payload);
				}

				if (apiKey?.trim()) {
					await aggregateAPI.setAuthKey(AuthKeyTypeProvider, providerName, apiKey.trim());
				}

				if (!currentDefaultProvider && payload.isEnabled === true) {
					await modelPresetStoreAPI.patchDefaultProvider(providerName);
				}

				await refreshCanonicalData();
			} catch (patchError) {
				console.error(patchError);
				await refreshCanonicalDataSafely();

				const message = 'Failed updating provider.';
				showGlobalDenied(message);

				throw new Error(message, { cause: patchError });
			}
		},
		[refreshCanonicalData, refreshCanonicalDataSafely, showGlobalDenied]
	);

	const handleProviderAuthKeyChanged = useCallback(
		async (_providerName: ProviderName) => {
			try {
				await refreshCanonicalData();
			} catch (refreshError) {
				console.error(refreshError);
				throw new Error('Failed refreshing auth key state.', { cause: refreshError });
			}
		},
		[refreshCanonicalData]
	);

	const fetchValue = useCallback(async () => {
		try {
			const data = await fetchCanonicalPageData();

			return JSON.stringify(
				{
					defaultProvider: data.defaultProvider,
					providers: Object.fromEntries(
						Object.entries(data.providerPresets).map(([name, provider]) => [
							name,
							{
								...provider,
								defaultHeaders: redactSensitiveHTTPHeaders(provider.defaultHeaders) ?? {},
							},
						])
					),
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

	if (isLoading && canonicalData === null) {
		return <Loader text="Loading model presets…" />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="Model Presets"
					description="Configure providers, API compatibility, defaults, and model runtime parameters."
					actions={
						<>
							<DownloadButton
								title="Download Model Presets"
								language="json"
								valueFetcher={fetchValue}
								size={18}
								fileprefix="modelpresets"
								className="btn btn-sm btn-ghost rounded-xl"
							/>

							<button type="button" className="btn btn-ghost rounded-xl" onClick={openAddModal}>
								<FiPlus size={18} />
								<span>Add Provider</span>
							</button>
						</>
					}
				/>

				<ManagementPageContent>
					{canonicalLoadError ? (
						<ManagementResourceError
							title="Model presets could not be loaded"
							error={canonicalLoadError}
							isRetrying={isRefreshing}
							onRetry={async () => {
								await reloadCanonicalData();
							}}
						/>
					) : null}

					<div className="bg-base-100 mb-8 rounded-2xl px-4 py-2 shadow-lg">
						<div className="flex flex-col gap-3 sm:flex-row sm:items-center">
							<div className="text-sm font-medium sm:w-48">Default Provider</div>

							<div className="min-w-0 grow">
								{enabledProviderNames.length > 0 && safeDefaultKey ? (
									<Dropdown<ProviderName>
										dropdownItems={enabledProviderPresets as Record<string, DropdownItem>}
										selectedKey={safeDefaultKey}
										onChange={handleDefaultProviderChange}
										filterDisabled={false}
										title="Select default provider"
										getDisplayName={key => enabledProviderPresets[key]?.displayName ?? ''}
									/>
								) : (
									<span className="text-error text-sm">Enable at least one provider first.</span>
								)}
							</div>

							{defaultProviderNeedsRepair && safeDefaultKey ? (
								<button
									type="button"
									className="btn btn-sm rounded-xl"
									disabled={isRefreshing}
									onClick={() => {
										void handleDefaultProviderChange(safeDefaultKey);
									}}
								>
									Save fallback default
								</button>
							) : null}
						</div>
					</div>

					{!canonicalLoadError &&
						Object.entries(providerPresets)
							.toSorted(sortByDisplayName)
							.map(([name, preset]) => (
								<ProviderPresetCard
									key={name}
									provider={name as ProviderName}
									preset={preset}
									defaultProvider={defaultProvider}
									authKeySet={providerKeySet[name as ProviderName]}
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
				</ManagementPageContent>

				<AddEditProviderPresetModal
					isOpen={modalOpen}
					mode={modalMode}
					onClose={() => {
						setModalOpen(false);
					}}
					onSubmit={async (name, payload, apiKey) => {
						if (modalMode === 'view') {
							return;
						}

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
