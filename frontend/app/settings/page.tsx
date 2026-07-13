import { useCallback, useState } from 'react';

import { FiPlus } from 'react-icons/fi';

import type { AuthKeyMeta, SettingsSchema } from '@/spec/setting';

import { throwIfAborted } from '@/lib/async_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { settingstoreAPI } from '@/apis/baseapi';

import { DownloadButton } from '@/components/download_button';
import { Loader } from '@/components/loader';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { PageFrame } from '@/components/page_frame';

import { AddEditAuthKeyModal } from '@/settings/authkey_add_edit_modal';
import { AuthKeyTable } from '@/settings/authkey_table';
import { DebugSettingsSection } from '@/settings/debug';
import { ThemeSelector } from '@/settings/theme';

async function exportSettings() {
	const settings = await settingstoreAPI.getSettings();
	return JSON.stringify(
		{
			appTheme: settings.appTheme,
			debug: settings.debug,
			authKeys: settings.authKeys.map(({ type, keyName, sha256, nonEmpty }) => ({
				type,
				keyName,
				sha256,
				nonEmpty,
			})),
		},
		null,
		2
	);
}

// oxlint-disable-next-line no-restricted-exports
export default function SettingsPage() {
	const loadSettings = useCallback(async (signal: AbortSignal): Promise<SettingsSchema> => {
		const settings = await settingstoreAPI.getSettings();
		throwIfAborted(signal);
		return settings;
	}, []);

	const {
		data: settings,
		error: settingsLoadError,
		isLoading,
		isRefreshing,
		reloadOrThrow,
		setData: setSettings,
	} = useAsyncResource(loadSettings, {
		initialData: null as SettingsSchema | null,
	});

	const [isModalOpen, setIsModalOpen] = useState(false);
	const [modalInitial, setModalInitial] = useState<AuthKeyMeta | null>(null);

	const authKeys = settings?.authKeys ?? [];
	const debugSettings = settings?.debug ?? null;

	const refresh = () => {
		void reloadOrThrow().catch((error: unknown) => {
			console.error('Failed to refresh settings', error);
		});
	};
	const showAddModal = () => {
		setModalInitial(null);
		setIsModalOpen(true);
	};
	const showEditModal = (meta: AuthKeyMeta) => {
		setModalInitial(meta);
		setIsModalOpen(true);
	};

	if (isLoading && settings === null) {
		return <Loader text="Loading settings..." />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="Settings"
					description="Manage appearance, provider secrets, and backend diagnostics."
					actions={
						<DownloadButton
							title="Download Settings"
							language="json"
							valueFetcher={exportSettings}
							size={18}
							fileprefix="settings"
							className="btn btn-sm btn-ghost rounded-xl"
							isBinary={false}
						/>
					}
				/>

				<ManagementPageContent>
					{settingsLoadError ? (
						<ManagementResourceError
							title="Settings could not be loaded"
							error={settingsLoadError}
							isRetrying={isRefreshing}
							onRetry={async () => {
								await reloadOrThrow();
							}}
						/>
					) : null}

					<section className="border-base-content/10 bg-base-100 flex flex-col gap-3 rounded-2xl border p-4 shadow-sm sm:flex-row sm:items-center">
						<h2 className="font-semibold">Theme</h2>
						<ThemeSelector />
					</section>

					<section className="border-base-content/10 bg-base-100 rounded-2xl border p-4 shadow-sm">
						<div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
							<h2 className="font-semibold">Auth Keys</h2>
							<button type="button" className="btn btn-ghost flex items-center rounded-xl" onClick={showAddModal}>
								<FiPlus className="mr-1" /> Add Key
							</button>
						</div>

						<AuthKeyTable authKeys={authKeys} onEdit={showEditModal} onChanged={refresh} />
					</section>

					<section className="border-base-content/10 bg-base-100 rounded-2xl border p-4 shadow-sm">
						<h2 className="font-semibold">Debug</h2>
						<DebugSettingsSection
							value={debugSettings}
							onChanged={debug => {
								setSettings(previous => (previous ? { ...previous, debug } : previous));
							}}
						/>
					</section>
				</ManagementPageContent>

				<AddEditAuthKeyModal
					isOpen={isModalOpen}
					initial={modalInitial}
					existing={authKeys}
					onClose={() => {
						setIsModalOpen(false);
					}}
					onChanged={refresh}
				/>
			</div>
		</PageFrame>
	);
}
