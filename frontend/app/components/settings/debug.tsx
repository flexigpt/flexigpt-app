import { useMemo, useRef, useState } from 'react';
import { FiAlertCircle, FiAlertTriangle } from 'react-icons/fi';

import type { DebugSettings } from '@/spec/setting';
import { DebugLogLevel, DEFAULT_DEBUG_SETTINGS } from '@/spec/setting';

import { getErrorMessage } from '@/lib/error_utils';

import { settingstoreAPI } from '@/apis/baseapi';

import { Dropdown } from '@/components/dropdown';

const LOG_LEVEL_OPTIONS: Array<{ value: DebugLogLevel; label: string }> = [
	{ value: DebugLogLevel.Debug, label: 'Debug' },
	{ value: DebugLogLevel.Info, label: 'Info' },
	{ value: DebugLogLevel.Warn, label: 'Warn' },
	{ value: DebugLogLevel.Error, label: 'Error' },
];

const LOG_LEVEL_DROPDOWN_ITEMS = Object.fromEntries(
	LOG_LEVEL_OPTIONS.map(option => [option.value, { isEnabled: true }])
) as Record<DebugLogLevel, { isEnabled: boolean }>;

const LOG_LEVEL_ORDERED_KEYS = LOG_LEVEL_OPTIONS.map(option => option.value);

const getLogLevelDisplayName = (key: DebugLogLevel) =>
	LOG_LEVEL_OPTIONS.find(option => option.value === key)?.label ?? key;

interface DebugSettingsSectionProps {
	value?: DebugSettings | null;
	onChanged?: (value: DebugSettings) => void;
}

export function DebugSettingsSection({ value, onChanged }: DebugSettingsSectionProps) {
	const current = useMemo(() => value ?? DEFAULT_DEBUG_SETTINGS, [value]);

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
		<section className="min-w-0" aria-busy={saving}>
			<h2 className="text-base-content/60 mb-3 text-xs font-medium">Debug</h2>

			<div className="space-y-3">
				{saveError ? (
					<div className="alert alert-error rounded-lg px-3 py-2 text-xs" role="alert">
						<FiAlertCircle className="shrink-0" size={14} aria-hidden="true" />
						<span className="wrap-break-word">{saveError}</span>
					</div>
				) : null}

				{current.logLLMReqResp || current.disableContentStripping ? (
					<div
						className="border-warning/40 bg-warning/10 text-warning-content rounded-lg border p-2 text-xs"
						role="alert"
					>
						<div className="flex items-start gap-2">
							<FiAlertTriangle className="mt-0.5 shrink-0" size={14} aria-hidden="true" />
							<div>
								<div className="font-medium">Debug content may include sensitive data.</div>
								<p className="mt-0.5">
									Raw request/response logs and full message details can expose private chat data.
								</p>
							</div>
						</div>
					</div>
				) : null}

				<div className="flex items-start justify-between gap-3">
					<div className="min-w-0 p-2">
						<div className="text-sm">Log level</div>
						<p className="text-base-content/60 mt-0.5 text-xs">Controls backend log verbosity.</p>
					</div>

					<div className="w-32 shrink-0">
						<Dropdown<DebugLogLevel>
							dropdownItems={LOG_LEVEL_DROPDOWN_ITEMS}
							selectedKey={current.logLevel}
							onChange={key => {
								void save({
									...current,
									logLevel: key,
								});
							}}
							filterDisabled={false}
							orderedKeys={LOG_LEVEL_ORDERED_KEYS}
							getDisplayName={getLogLevelDisplayName}
							title="Select log level"
							disabled={saving}
							inlineMenu
							maxMenuHeight={220}
						/>
					</div>
				</div>

				{/* oxlint-disable-next-line jsx-a11y/label-has-associated-control */}
				<label className="flex cursor-pointer items-center justify-between gap-3 p-2">
					<span className="min-w-0">
						<span className="block text-sm">LLM request and response logging</span>
						<span className="text-base-content/60 mt-0.5 block text-xs">
							Log raw LLM request and response payloads to the app logs.
						</span>
					</span>

					<input
						type="checkbox"
						className="toggle toggle-accent toggle-sm shrink-0"
						checked={current.logLLMReqResp}
						disabled={saving}
						onChange={event => {
							void save({ ...current, logLLMReqResp: event.currentTarget.checked });
						}}
					/>
				</label>

				{/* oxlint-disable-next-line jsx-a11y/label-has-associated-control */}
				<label className="flex cursor-pointer items-center justify-between gap-3 p-2">
					<span className="min-w-0">
						<span className="block text-sm">Disable content stripping</span>
						<span className="text-base-content/60 mt-0.5 block text-xs">
							Include chat text, replies, and thinking in message details.
						</span>
					</span>

					<input
						type="checkbox"
						className="toggle toggle-accent toggle-sm shrink-0"
						checked={current.disableContentStripping}
						disabled={saving}
						onChange={event => {
							void save({ ...current, disableContentStripping: event.currentTarget.checked });
						}}
					/>
				</label>
			</div>
		</section>
	);
}
