import { useMemo, useState } from 'react';

import { DebugLogLevel, type DebugSettings, DEFAULT_DEBUG_SETTINGS } from '@/spec/setting';

import { settingstoreAPI } from '@/apis/baseapi';

import { Dropdown } from '@/components/dropdown';

const LOG_LEVEL_OPTIONS: Array<{ value: DebugLogLevel; label: string }> = [
	{ value: DebugLogLevel.Debug, label: 'Debug' },
	{ value: DebugLogLevel.Info, label: 'Info' },
	{ value: DebugLogLevel.Warn, label: 'Warn' },
	{ value: DebugLogLevel.Error, label: 'Error' },
];

const LOG_LEVEL_DROPDOWN_ITEMS = Object.fromEntries(
	LOG_LEVEL_OPTIONS.map(o => [o.value, { isEnabled: true }])
) as Record<DebugLogLevel, { isEnabled: boolean }>;

const LOG_LEVEL_ORDERED_KEYS = LOG_LEVEL_OPTIONS.map(o => o.value);

const getLogLevelDisplayName = (key: DebugLogLevel) => LOG_LEVEL_OPTIONS.find(o => o.value === key)?.label ?? key;

interface DebugSettingsSectionProps {
	value?: DebugSettings | null;
	onChanged?: (value: DebugSettings) => void;
}

export function DebugSettingsSection({ value, onChanged }: DebugSettingsSectionProps) {
	const current = useMemo(() => value ?? DEFAULT_DEBUG_SETTINGS, [value]);
	const [saving, setSaving] = useState(false);

	const save = async (next: DebugSettings) => {
		setSaving(true);
		try {
			await settingstoreAPI.setDebugSettings(next);
			onChanged?.(next);
		} catch (err) {
			console.error('Failed to save debug settings', err);
		} finally {
			setSaving(false);
		}
	};

	return (
		<div className="flex w-full flex-col gap-4 p-4">
			<div className="flex items-center justify-between gap-4">
				<div>
					<h3 className="text-sm font-semibold">Log level</h3>
					<p className="text-base-content/60 text-xs">Controls the runtime Go backend log verbosity.</p>
				</div>
				<div className="w-60">
					<Dropdown
						dropdownItems={LOG_LEVEL_DROPDOWN_ITEMS}
						selectedKey={current.logLevel}
						onChange={key => {
							void save({ ...current, logLevel: key });
						}}
						orderedKeys={LOG_LEVEL_ORDERED_KEYS}
						getDisplayName={getLogLevelDisplayName}
						disabled={saving}
					/>
				</div>
			</div>

			<div className="flex items-center justify-between gap-4">
				<div>
					<h3 className="text-sm font-semibold">LLM request and response logging</h3>
					<p className="text-base-content/60 text-xs">Log raw LLM request and response payloads to the app logs.</p>
				</div>
				<input
					type="checkbox"
					className="toggle toggle-accent"
					checked={current.logLLMReqResp}
					disabled={saving}
					onChange={e => {
						void save({ ...current, logLLMReqResp: e.target.checked });
					}}
				/>
			</div>

			<div className="flex items-center justify-between gap-4">
				<div>
					<h3 className="text-sm font-semibold">Disable content stripping</h3>
					<p className="text-base-content/60 text-xs">
						Don't strip user, assistant and thinking content from per message details in Chat.
					</p>
				</div>
				<input
					type="checkbox"
					className="toggle toggle-accent"
					checked={current.disableContentStripping}
					disabled={saving}
					onChange={e => {
						void save({ ...current, disableContentStripping: e.target.checked });
					}}
				/>
			</div>
		</div>
	);
}
