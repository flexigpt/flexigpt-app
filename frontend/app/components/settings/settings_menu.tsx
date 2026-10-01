import { FiAlertCircle, FiSettings } from 'react-icons/fi';

import { Popover, PopoverDisclosure, PopoverHeading, usePopoverStore } from '@ariakit/react';

import type { SettingsSchema } from '@/spec/setting';

import { throwIfAborted } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { settingstoreAPI } from '@/apis/baseapi';

import { DebugSettingsSection } from '@/components/settings/debug';
import { ThemeSelector } from '@/components/settings/theme';

async function loadSettings(signal: AbortSignal): Promise<SettingsSchema> {
	const settings = await settingstoreAPI.getSettings();
	throwIfAborted(signal);
	return settings;
}

export function SettingsMenu() {
	const popover = usePopoverStore({ placement: 'bottom-end' });

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

	const settingsBusy = isLoading || isRefreshing;

	return (
		<>
			<PopoverDisclosure
				store={popover}
				type="button"
				className="app-no-drag btn btn-ghost btn-xs btn-circle aria-expanded:bg-base-300 mr-1 shrink-0 p-0 opacity-80 hover:opacity-100 aria-expanded:opacity-100"
				aria-label="Settings"
				title="Settings"
			>
				<FiSettings size={14} aria-hidden="true" />
			</PopoverDisclosure>

			<Popover
				store={popover}
				portal
				modal={false}
				gutter={6}
				unmountOnHide={false}
				className="app-no-drag app-scrollbar-thin bg-base-100 text-base-content border-base-300 z-50 max-h-[calc(100dvh-3rem)] w-96 max-w-[calc(100vw-1rem)] overflow-y-auto rounded-xl border p-4 text-sm shadow-xl outline-none"
			>
				<div className="mb-3 flex items-center justify-between gap-3">
					<PopoverHeading className="text-sm font-semibold">Settings</PopoverHeading>
				</div>

				<ThemeSelector />

				<div className="border-base-300 mt-3 border-t pt-3">
					{settingsLoadError ? (
						<div className="space-y-2">
							<div className="alert alert-error rounded-lg px-3 py-2 text-xs" role="alert">
								<FiAlertCircle className="shrink-0" size={14} aria-hidden="true" />
								<span className="wrap-break-word">
									{getErrorMessage(settingsLoadError, 'Could not load settings.')}
								</span>
							</div>

							<button
								type="button"
								className="btn btn-ghost btn-xs rounded-lg"
								disabled={settingsBusy}
								onClick={() => {
									void reloadOrThrow().catch(() => undefined);
								}}
							>
								{settingsBusy ? 'Loading...' : 'Retry'}
							</button>
						</div>
					) : settings ? (
						<DebugSettingsSection
							value={settings.debug}
							onChanged={debug => {
								setSettings(previous => (previous ? { ...previous, debug } : previous));
							}}
						/>
					) : (
						<output className="text-base-content/60 flex items-center gap-2 text-xs">
							<span className="loading loading-dots loading-sm" aria-hidden="true" />
							<span>Loading settings...</span>
						</output>
					)}
				</div>
			</Popover>
		</>
	);
}
