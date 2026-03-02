import { useEffect, useMemo, useState } from 'react';

import { FiPlus } from 'react-icons/fi';

import type { ProviderName } from '@/spec/inference';
import { type ProviderPreset } from '@/spec/modelpreset';
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

// eslint-disable-next-line no-restricted-exports
export default function ModelPresetsPage() {
	const [defaultProvider, setDefaultProvider] = useState<ProviderName | undefined>(undefined);
	const [providerPresets, setProviderPresets] = useState<Record<ProviderName, ProviderPreset>>({});
	const [providerKeySet, setProviderKeySet] = useState<Record<ProviderName, boolean>>({});

	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [showDenied, setShowDenied] = useState(false);
	const [deniedMsg, setDeniedMsg] = useState('');

	const [modalOpen, setModalOpen] = useState(false);
	const [modalMode, setModalMode] = useState<'add' | 'edit' | 'view'>('add');
	const [editProvider, setEditProvider] = useState<ProviderName | null>(null);

	useEffect(() => {
		(async () => {
			try {
				setLoading(true);
				const [settings, defProv, presets] = await Promise.all([
					settingstoreAPI.getSettings(),
					modelPresetStoreAPI.getDefaultProvider(),
					getAllProviderPresetsMap(true),
				]);

				setDefaultProvider(defProv);
				setProviderPresets(presets);
				setProviderKeySet(
					Object.fromEntries(
						settings.authKeys.filter(k => k.type === AuthKeyTypeProvider).map(k => [k.keyName, k.nonEmpty])
					)
				);
			} catch (err) {
				console.error('init model presets error', err);
				setError('Failed to load provider presets. Try again.');
			} finally {
				setLoading(false);
			}
		})();
	}, []);

	const enabledProviderNames = useMemo(
		() =>
			Object.values(providerPresets)
				.filter(p => p.isEnabled)
				.map(p => p.name),
		[providerPresets]
	);

	const enabledProviderPresets = useMemo<Partial<Record<ProviderName, ProviderPreset>>>(() => {
		const obj: Partial<Record<ProviderName, ProviderPreset>> = {};
		for (const [name, preset] of Object.entries(providerPresets)) {
			if (preset.isEnabled) obj[name] = preset;
		}
		return obj;
	}, [providerPresets]);

	const safeDefaultKey: ProviderName | undefined =
		defaultProvider && defaultProvider in enabledProviderPresets ? defaultProvider : undefined;

	useEffect(() => {
		if (enabledProviderNames.length === 0) return;

		if (!defaultProvider || !providerPresets[defaultProvider].isEnabled) {
			const first = enabledProviderNames[0];
			(async () => {
				try {
					await modelPresetStoreAPI.patchDefaultProvider(first);
					setDefaultProvider(first);
				} catch (err) {
					console.error(err);
					setDeniedMsg('Failed setting default provider.');
					setShowDenied(true);
				}
			})();
		}
	}, [enabledProviderNames, defaultProvider, providerPresets]);

	const handleDefaultProviderChange = async (prov: ProviderName) => {
		try {
			setDefaultProvider(prov);
			await modelPresetStoreAPI.patchDefaultProvider(prov);
		} catch (err) {
			console.error(err);
			setDeniedMsg('Failed changing default provider.');
			setShowDenied(true);
		}
	};

	const handleProviderPresetChange = (name: ProviderName, newPreset: ProviderPreset) => {
		if (name === defaultProvider && !newPreset.isEnabled) {
			setDeniedMsg('Cannot disable the current default provider. Pick another default first.');
			setShowDenied(true);
			return;
		}

		setProviderPresets(prev => ({ ...prev, [name]: newPreset }));

		if (!defaultProvider && newPreset.isEnabled) {
			handleDefaultProviderChange(name);
		}
	};

	const handleProviderDelete = async (name: ProviderName) => {
		if (providerPresets[name].isBuiltIn) {
			setDeniedMsg('Built-in providers cannot be deleted.');
			setShowDenied(true);
			return;
		}

		if (name === defaultProvider) {
			setDeniedMsg('Cannot delete the current default provider. Pick another default first.');
			setShowDenied(true);
			return;
		}

		try {
			await aggregateAPI.deleteProviderPreset(name);
			await aggregateAPI.deleteAuthKey(AuthKeyTypeProvider, name).catch(() => void 0);

			setProviderPresets(prev => {
				const { [name]: _removed, ...rest } = prev;
				return rest;
			});
			setProviderKeySet(prev => {
				const { [name]: _removed, ...rest } = prev;
				return rest;
			});
		} catch (err) {
			console.error(err);
			setDeniedMsg('Failed deleting provider.');
			setShowDenied(true);
		}
	};

	const handleProviderModalSubmit = async (
		name: ProviderName,
		payload: Omit<ProviderPreset, 'isBuiltIn' | 'defaultModelPresetID' | 'modelPresets' | 'name'>,
		apiKey: string | null,
		isEdit: boolean
	) => {
		if (isEdit && name === defaultProvider && !payload.isEnabled) {
			setDeniedMsg('Cannot disable the current default provider. Pick another default first.');
			setShowDenied(true);
			return;
		}

		try {
			await aggregateAPI.putProviderPreset(name, payload);
			if (apiKey && apiKey.trim()) {
				await aggregateAPI.setAuthKey(AuthKeyTypeProvider, name, apiKey.trim());
			}

			setProviderPresets(prev => {
				const existing = prev[name];
				return {
					...prev,
					[name]: {
						...(existing ?? {}),
						...payload,

						// Editing should NEVER wipe these:
						defaultModelPresetID: existing?.defaultModelPresetID ?? '',
						modelPresets: existing?.modelPresets ?? {},

						// Only custom providers go through this modal in edit/add
						isBuiltIn: false,
					} as ProviderPreset,
				};
			});

			if (apiKey && apiKey.trim()) setProviderKeySet(prev => ({ ...prev, [name]: true }));

			if (!defaultProvider && payload.isEnabled) {
				await modelPresetStoreAPI.patchDefaultProvider(name);
				setDefaultProvider(name);
			}
		} catch (err) {
			console.error(err);
			setDeniedMsg(isEdit ? 'Failed updating provider.' : 'Failed adding provider.');
			setShowDenied(true);
		}
	};

	const fetchValue = async () => {
		try {
			const [providerPresetMap, defProv] = await Promise.all([
				getAllProviderPresetsMap(true),
				modelPresetStoreAPI.getDefaultProvider(),
			]);
			return JSON.stringify({ defaultProvider: defProv, providers: providerPresetMap }, null, 2);
		} catch (err) {
			console.log('fetch preset error', err);
			setDeniedMsg('Failed fetching presets.');
			setShowDenied(true);
			return '';
		}
	};

	const openAddModal = () => {
		setModalMode('add');
		setEditProvider(null);
		setModalOpen(true);
	};

	const openEditModal = (name: ProviderName) => {
		const isBuiltIn = providerPresets[name].isBuiltIn;
		setModalMode(isBuiltIn ? 'view' : 'edit');
		setEditProvider(name);
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
								.map(([name, p]) => (
									<ProviderPresetCard
										key={name}
										provider={name}
										preset={p}
										defaultProvider={defaultProvider ?? ''}
										authKeySet={providerKeySet[name]}
										enabledProviders={enabledProviderNames}
										onProviderPresetChange={handleProviderPresetChange}
										onProviderDelete={handleProviderDelete}
										onRequestEdit={openEditModal}
										allProviderPresets={providerPresets}
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
					onSubmit={(n, payload, key) => {
						if (modalMode === 'view') return;
						return handleProviderModalSubmit(n, payload, key, modalMode === 'edit');
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
