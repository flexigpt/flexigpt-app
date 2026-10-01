import { useRef, useState } from 'react';

import type { DebugSettings } from '@/spec/setting';
import { DebugLogLevel, DEFAULT_DEBUG_SETTINGS } from '@/spec/setting';

import { getErrorMessage } from '@/lib/error_utils';

import { settingstoreAPI } from '@/apis/baseapi';

const LOG_LEVEL_OPTIONS: Array<{ value: DebugLogLevel; label: string }> = [
	{ value: DebugLogLevel.Debug, label: 'Debug' },
	{ value: DebugLogLevel.Info, label: 'Info' },
	{ value: DebugLogLevel.Warn, label: 'Warning' },
	{ value: DebugLogLevel.Error, label: 'Error' },
];

interface DebugSettingsSectionProps {
	value?: DebugSettings | null;
	onChanged?: (value: DebugSettings) => void;
}

export function DebugSettingsSection({ value, onChanged }: DebugSettingsSectionProps) {
	const current = value ?? DEFAULT_DEBUG_SETTINGS;

	const [saving, setSaving] = useState(false);
	const [saveError, setSaveError] = useState('');
	const savingRef = useRef(false);

	const save = async (next: DebugSettings) => {
		if (savingRef.current) {
			return;
		}

		savingRef.current = true;
		setSaving(true);
		setSaveError('');

		try {
			await settingstoreAPI.setDebugSettings(next);
			onChanged?.(next);
		} catch (error) {
			console.error('Failed to save debug settings', error);
			setSaveError(getErrorMessage(error, 'Could not save debug settings.'));
		} finally {
			savingRef.current = false;
			setSaving(false);
		}
	};

	return (
		<fieldset className="min-w-0" disabled={saving} aria-busy={saving}>
			<legend className="text-base-content/60 mb-3 text-xs font-medium">Debug</legend>

			<div className="space-y-3">
				<label className="flex items-center justify-between gap-3">
					<span className="text-sm">Log level</span>

					<select
						className="select select-sm w-32"
						value={current.logLevel}
						onChange={event => {
							void save({
								...current,
								logLevel: event.currentTarget.value as DebugLogLevel,
							});
						}}
					>
						{LOG_LEVEL_OPTIONS.map(option => (
							<option key={option.value} value={option.value}>
								{option.label}
							</option>
						))}
					</select>
				</label>

				<label className="flex cursor-pointer items-center justify-between gap-3">
					<span className="min-w-0 text-sm">Log AI requests and responses</span>

					<input
						type="checkbox"
						className="toggle toggle-accent toggle-sm shrink-0"
						checked={current.logLLMReqResp}
						onChange={event => {
							void save({ ...current, logLLMReqResp: event.currentTarget.checked });
						}}
					/>
				</label>

				<label
					className="flex cursor-pointer items-center justify-between gap-3"
					title="Include chat text, replies and thinking in message details."
				>
					<span className="min-w-0 text-sm">Show full message details</span>

					<input
						type="checkbox"
						className="toggle toggle-accent toggle-sm shrink-0"
						checked={current.disableContentStripping}
						onChange={event => {
							void save({ ...current, disableContentStripping: event.currentTarget.checked });
						}}
					/>
				</label>

				{current.logLLMReqResp || current.disableContentStripping ? (
					<output className="border-warning/30 bg-warning/10 rounded-lg border p-2 text-xs">
						Logs and message details may contain private chat data.
					</output>
				) : null}

				{saveError ? (
					<p className="text-error text-xs wrap-break-word" role="alert">
						{saveError}
					</p>
				) : null}
			</div>
		</fieldset>
	);
}
