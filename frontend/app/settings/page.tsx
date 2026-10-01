import { useCallback } from 'react';

import type { SettingsSchema } from '@/spec/setting';

import { throwIfAborted } from '@/lib/async_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { settingstoreAPI } from '@/apis/baseapi';

import { DownloadButton } from '@/components/download_button';
import { Loader } from '@/components/loader';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { PageFrame } from '@/components/page_frame';

import { DebugSettingsSection } from '@/settings/debug';
import { ThemeSelector } from '@/settings/theme';

async function exportSettings() {
	const settings = await settingstoreAPI.getSettings();
	return JSON.stringify(
		{
			appTheme: settings.appTheme,
			debug: settings.debug,
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

	const debugSettings = settings?.debug ?? null;

	if (isLoading && settings === null) {
		return <Loader text="Loading settings..." />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="Settings"
					description="Manage appearance and backend diagnostics."
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
						<h2 className="font-semibold">Debug</h2>
						<DebugSettingsSection
							value={debugSettings}
							onChanged={debug => {
								setSettings(previous => (previous ? { ...previous, debug } : previous));
							}}
						/>
					</section>
				</ManagementPageContent>
			</div>
		</PageFrame>
	);
}
