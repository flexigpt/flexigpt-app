import { useMemo } from 'react';
import { FiAlertCircle } from 'react-icons/fi';

import type {
	CacheControlKind,
	ProviderSDKType,
	ReasoningLevel,
	ReasoningSummaryStyle,
	ReasoningType,
} from '@/spec/inference';
import type { ModelCapabilities } from '@/spec/model';
import {
	OutputFormatKind as OutputFormatKindValue,
	OutputVerbosity as OutputVerbosityValue,
	ReasoningLevel as ReasoningLevelValue,
	ReasoningSummaryStyle as ReasoningSummaryStyleValue,
	ReasoningType as ReasoningTypeValue,
} from '@/spec/inference';

import { Dropdown } from '@/components/dropdown';
import { ModalField } from '@/components/modal/modal_field';
import { ModalSection } from '@/components/modal/modal_section';
import { ReadOnlyValue } from '@/components/read_only_value';

import type { CacheControlTTLSelection } from '@/models/lib/cache_control';
import type {
	ModelRuntimeDefaultsErrors,
	ModelRuntimeDefaultsForm,
	OutputFormatSelection,
	OutputVerbositySelection,
	StrictSelection,
} from '@/models/lib/model_runtime_defaults';
import {
	buildCacheControlKindDropdownItems,
	buildCacheControlTTLDropdownItems,
	resolveSupportedCacheControlKinds,
	resolveSupportedCacheControlTTLs,
} from '@/models/lib/cache_control';
import {
	getEffectiveCacheCapabilities,
	getSupportedOutputFormats,
	getTopLevelCacheControlCapabilities,
	supportsOutputVerbosity,
	supportsReasoningSummaryStyle,
} from '@/models/lib/capabilities';
import {
	getOutputFormatDisplayName,
	getOutputVerbosityDisplayName,
	getReasoningLevelDisplayName,
} from '@/models/lib/model_option_labels';
import {
	OPTIONAL_BOOLEAN_UNSET,
	OUTPUT_FORMAT_NONE,
	OUTPUT_VERBOSITY_NONE,
	parseStopSequences,
} from '@/models/lib/model_runtime_defaults';

function FieldError({ error }: { error?: string }) {
	if (!error) {
		return null;
	}

	return (
		<div className="label">
			<span className="text-error flex items-center gap-1 text-xs">
				<FiAlertCircle size={12} />
				{error}
			</span>
		</div>
	);
}

interface ModelRuntimeDefaultsFieldsProps {
	value: ModelRuntimeDefaultsForm;
	errors: ModelRuntimeDefaultsErrors;
	enabled: boolean;
	readOnly: boolean;
	disabled: boolean;
	providerSDKType: ProviderSDKType;
	capabilities?: ModelCapabilities;
	onEnabledChange: (enabled: boolean) => void;
	onChange: (patch: Partial<ModelRuntimeDefaultsForm>) => void;
}

export function ModelRuntimeDefaultsFields({
	value,
	errors,
	enabled,
	readOnly,
	disabled,
	providerSDKType,
	capabilities,
	onEnabledChange,
	onChange,
}: ModelRuntimeDefaultsFieldsProps) {
	const cacheCapabilities = useMemo(
		() => getEffectiveCacheCapabilities(providerSDKType, capabilities),
		[capabilities, providerSDKType]
	);
	const topLevelCacheCapabilities = useMemo(
		() => getTopLevelCacheControlCapabilities(providerSDKType, capabilities),
		[capabilities, providerSDKType]
	);
	const supportedCacheKinds = useMemo(
		() => resolveSupportedCacheControlKinds(topLevelCacheCapabilities?.supportedKinds),
		[topLevelCacheCapabilities?.supportedKinds]
	);
	const supportedCacheTTLs = useMemo(
		() => resolveSupportedCacheControlTTLs(topLevelCacheCapabilities?.supportedTTLs),
		[topLevelCacheCapabilities?.supportedTTLs]
	);
	const supportsManualCacheControl = supportedCacheKinds.length > 0;
	const supportsCacheKey = topLevelCacheCapabilities?.supportsKey === true;
	const supportsAutomaticCaching = cacheCapabilities?.supportsAutomaticCaching === true;
	const supportedFormats = useMemo(() => getSupportedOutputFormats(capabilities), [capabilities]);
	const outputFormatItems = useMemo<Record<OutputFormatSelection, { isEnabled: boolean; displayName: string }>>(
		() => ({
			[OUTPUT_FORMAT_NONE]: { isEnabled: true, displayName: 'Default' },
			[OutputFormatKindValue.Text]: {
				isEnabled: !supportedFormats || supportedFormats.includes(OutputFormatKindValue.Text),
				displayName: getOutputFormatDisplayName(OutputFormatKindValue.Text),
			},
			[OutputFormatKindValue.JSONSchema]: {
				isEnabled: !supportedFormats || supportedFormats.includes(OutputFormatKindValue.JSONSchema),
				displayName: getOutputFormatDisplayName(OutputFormatKindValue.JSONSchema),
			},
		}),
		[supportedFormats]
	);
	const outputVerbosityItems: Record<OutputVerbositySelection, { isEnabled: boolean; displayName: string }> = {
		[OUTPUT_VERBOSITY_NONE]: { isEnabled: true, displayName: 'Default' },
		[OutputVerbosityValue.Low]: {
			isEnabled: true,
			displayName: getOutputVerbosityDisplayName(OutputVerbosityValue.Low),
		},
		[OutputVerbosityValue.Medium]: {
			isEnabled: true,
			displayName: getOutputVerbosityDisplayName(OutputVerbosityValue.Medium),
		},
		[OutputVerbosityValue.High]: {
			isEnabled: true,
			displayName: getOutputVerbosityDisplayName(OutputVerbosityValue.High),
		},
		[OutputVerbosityValue.XHigh]: {
			isEnabled: true,
			displayName: getOutputVerbosityDisplayName(OutputVerbosityValue.XHigh),
		},
		[OutputVerbosityValue.Max]: {
			isEnabled: true,
			displayName: getOutputVerbosityDisplayName(OutputVerbosityValue.Max),
		},
	};
	const reasoningTypeItems: Record<ReasoningType, { isEnabled: boolean; displayName: string }> = {
		[ReasoningTypeValue.SingleWithLevels]: { isEnabled: true, displayName: 'Reasoning with levels' },
		[ReasoningTypeValue.HybridWithTokens]: { isEnabled: true, displayName: 'Hybrid with tokens' },
	};
	const reasoningLevelItems: Record<ReasoningLevel, { isEnabled: boolean; displayName: string }> = {
		[ReasoningLevelValue.None]: {
			isEnabled: true,
			displayName: getReasoningLevelDisplayName(ReasoningLevelValue.None),
		},
		[ReasoningLevelValue.Minimal]: {
			isEnabled: true,
			displayName: getReasoningLevelDisplayName(ReasoningLevelValue.Minimal),
		},
		[ReasoningLevelValue.Low]: { isEnabled: true, displayName: getReasoningLevelDisplayName(ReasoningLevelValue.Low) },
		[ReasoningLevelValue.Medium]: {
			isEnabled: true,
			displayName: getReasoningLevelDisplayName(ReasoningLevelValue.Medium),
		},
		[ReasoningLevelValue.High]: {
			isEnabled: true,
			displayName: getReasoningLevelDisplayName(ReasoningLevelValue.High),
		},
		[ReasoningLevelValue.XHigh]: {
			isEnabled: true,
			displayName: getReasoningLevelDisplayName(ReasoningLevelValue.XHigh),
		},
		[ReasoningLevelValue.Max]: { isEnabled: true, displayName: getReasoningLevelDisplayName(ReasoningLevelValue.Max) },
	};
	const reasoningSummaryItems: Record<ReasoningSummaryStyle, { isEnabled: boolean; displayName: string }> = {
		[ReasoningSummaryStyleValue.Auto]: {
			isEnabled: supportsReasoningSummaryStyle(capabilities),
			displayName: 'Auto',
		},
		[ReasoningSummaryStyleValue.Concise]: {
			isEnabled: supportsReasoningSummaryStyle(capabilities),
			displayName: 'Concise',
		},
		[ReasoningSummaryStyleValue.Detailed]: {
			isEnabled: supportsReasoningSummaryStyle(capabilities),
			displayName: 'Detailed',
		},
		[ReasoningSummaryStyleValue.Omitted]: {
			isEnabled: supportsReasoningSummaryStyle(capabilities),
			displayName: 'Concise (legacy default)',
		},
	};
	const strictItems: Record<StrictSelection, { isEnabled: boolean; displayName: string }> = {
		[OPTIONAL_BOOLEAN_UNSET]: { isEnabled: true, displayName: 'Leave unset' },
		true: { isEnabled: true, displayName: 'Strict' },
		false: { isEnabled: true, displayName: 'Non-strict' },
	};
	const cacheKindItems = useMemo(() => buildCacheControlKindDropdownItems(supportedCacheKinds), [supportedCacheKinds]);
	const cacheTTLItems = useMemo(() => buildCacheControlTTLDropdownItems(supportedCacheTTLs), [supportedCacheTTLs]);

	return (
		<>
			<ModalSection
				title="Presentation"
				description="Control whether the model is selectable and whether responses stream."
			>
				<ModalField label="Enabled">
					{readOnly ? (
						<ReadOnlyValue value={enabled ? 'Enabled' : 'Disabled'} />
					) : (
						<input
							type="checkbox"
							className="toggle toggle-accent"
							checked={enabled}
							onChange={event => {
								onEnabledChange(event.target.checked);
							}}
							disabled={disabled}
						/>
					)}
				</ModalField>

				<ModalField label="Streaming">
					{readOnly ? (
						<ReadOnlyValue value={value.stream ? 'Enabled' : 'Disabled'} />
					) : (
						<input
							type="checkbox"
							className="toggle toggle-accent"
							checked={value.stream}
							onChange={event => {
								onChange({ stream: event.target.checked });
							}}
							disabled={disabled}
						/>
					)}
				</ModalField>
			</ModalSection>

			<ModalSection title="Reasoning" description="Configure reasoning when this model supports it.">
				<ModalField label="Supports Reasoning">
					{readOnly ? (
						<ReadOnlyValue value={value.reasoningEnabled ? 'Enabled' : 'Disabled'} />
					) : (
						<input
							type="checkbox"
							className="toggle toggle-accent"
							checked={value.reasoningEnabled}
							onChange={event => {
								onChange({ reasoningEnabled: event.target.checked });
							}}
							disabled={disabled}
						/>
					)}
				</ModalField>

				{value.reasoningEnabled ? (
					<>
						<ModalField label="Reasoning Type">
							{readOnly ? (
								<ReadOnlyValue value={reasoningTypeItems[value.reasoningType].displayName} />
							) : (
								<Dropdown<ReasoningType>
									dropdownItems={reasoningTypeItems}
									selectedKey={value.reasoningType}
									onChange={reasoningType => {
										onChange({ reasoningType });
									}}
									filterDisabled={false}
									title="Select reasoning type"
									getDisplayName={key => reasoningTypeItems[key].displayName}
									disabled={disabled}
								/>
							)}
						</ModalField>

						{value.reasoningType === ReasoningTypeValue.SingleWithLevels ? (
							<ModalField label="Reasoning Level">
								{readOnly ? (
									<ReadOnlyValue value={getReasoningLevelDisplayName(value.reasoningLevel)} />
								) : (
									<Dropdown<ReasoningLevel>
										dropdownItems={reasoningLevelItems}
										selectedKey={value.reasoningLevel}
										onChange={reasoningLevel => {
											onChange({ reasoningLevel });
										}}
										filterDisabled={false}
										title="Select reasoning level"
										getDisplayName={key => reasoningLevelItems[key].displayName}
										disabled={disabled}
									/>
								)}
							</ModalField>
						) : (
							<ModalField label="Reasoning Tokens" error={errors.reasoningTokens}>
								{readOnly ? (
									<ReadOnlyValue value={value.reasoningTokens} />
								) : (
									<>
										<input
											className={`input w-full rounded-xl ${errors.reasoningTokens ? 'input-error' : ''}`}
											value={value.reasoningTokens}
											onChange={event => {
												onChange({ reasoningTokens: event.target.value });
											}}
											placeholder="1024"
											spellCheck="false"
											disabled={disabled}
										/>
										<FieldError error={errors.reasoningTokens} />
									</>
								)}
							</ModalField>
						)}

						<ModalField label="Reasoning Summary">
							{readOnly ? (
								<ReadOnlyValue value={reasoningSummaryItems[value.reasoningSummaryStyle].displayName} />
							) : (
								<Dropdown<ReasoningSummaryStyle>
									dropdownItems={reasoningSummaryItems}
									selectedKey={value.reasoningSummaryStyle}
									onChange={reasoningSummaryStyle => {
										onChange({ reasoningSummaryStyle });
									}}
									filterDisabled={false}
									title="Select reasoning summary style"
									getDisplayName={key => reasoningSummaryItems[key].displayName}
									disabled={disabled}
								/>
							)}
						</ModalField>
					</>
				) : null}
			</ModalSection>

			<ModalSection title="Runtime Limits" description="Set sampling, request timeout, and token limits.">
				<ModalField label="Temperature" htmlFor="model-temperature" error={errors.temperature}>
					{readOnly ? (
						<ReadOnlyValue value={value.temperature || 'Default'} />
					) : (
						<>
							<input
								id="model-temperature"
								className={`input w-full rounded-xl ${errors.temperature ? 'input-error' : ''}`}
								value={value.temperature}
								onChange={event => {
									onChange({ temperature: event.target.value });
								}}
								placeholder="0.1"
								spellCheck="false"
								disabled={disabled}
							/>
							<FieldError error={errors.temperature} />
						</>
					)}
				</ModalField>

				<ModalField label="Timeout (seconds)" htmlFor="model-timeout" error={errors.timeoutSeconds}>
					{readOnly ? (
						<ReadOnlyValue value={value.timeoutSeconds || 'Default'} />
					) : (
						<>
							<input
								id="model-timeout"
								className={`input w-full rounded-xl ${errors.timeoutSeconds ? 'input-error' : ''}`}
								value={value.timeoutSeconds}
								onChange={event => {
									onChange({ timeoutSeconds: event.target.value });
								}}
								placeholder="300"
								spellCheck="false"
								disabled={disabled}
							/>
							<FieldError error={errors.timeoutSeconds} />
						</>
					)}
				</ModalField>

				<ModalField label="Max Prompt Tokens" htmlFor="model-max-prompt" error={errors.maxPromptTokens}>
					{readOnly ? (
						<ReadOnlyValue value={value.maxPromptTokens || 'Default'} />
					) : (
						<>
							<input
								id="model-max-prompt"
								className={`input w-full rounded-xl ${errors.maxPromptTokens ? 'input-error' : ''}`}
								value={value.maxPromptTokens}
								onChange={event => {
									onChange({ maxPromptTokens: event.target.value });
								}}
								placeholder="2048"
								spellCheck="false"
								disabled={disabled}
							/>
							<FieldError error={errors.maxPromptTokens} />
						</>
					)}
				</ModalField>

				<ModalField label="Max Output Tokens" htmlFor="model-max-output" error={errors.maxOutputTokens}>
					{readOnly ? (
						<ReadOnlyValue value={value.maxOutputTokens || 'Default'} />
					) : (
						<>
							<input
								id="model-max-output"
								className={`input w-full rounded-xl ${errors.maxOutputTokens ? 'input-error' : ''}`}
								value={value.maxOutputTokens}
								onChange={event => {
									onChange({ maxOutputTokens: event.target.value });
								}}
								placeholder="1024"
								spellCheck="false"
								disabled={disabled}
							/>
							<FieldError error={errors.maxOutputTokens} />
						</>
					)}
				</ModalField>
			</ModalSection>

			{supportsManualCacheControl || supportsAutomaticCaching ? (
				<ModalSection
					title="Cache Control"
					description="Configure manual request caching when the provider supports it."
				>
					{supportsManualCacheControl ? (
						<ModalField label="Cache Control">
							{readOnly ? (
								<ReadOnlyValue value={value.cacheControlEnabled ? 'Enabled' : 'Disabled'} />
							) : (
								<input
									type="checkbox"
									className="toggle toggle-accent"
									checked={value.cacheControlEnabled}
									onChange={event => {
										onChange({
											cacheControlEnabled: event.target.checked,
											cacheControlKind: value.cacheControlKind || supportedCacheKinds[0] || '',
										});
									}}
									disabled={disabled}
								/>
							)}
						</ModalField>
					) : (
						<p className="text-sm opacity-70">Manual cache control is not available for this provider.</p>
					)}

					{value.cacheControlEnabled && supportsManualCacheControl ? (
						<>
							<ModalField label="Cache Kind" error={errors.cacheControlKind}>
								{readOnly ? (
									<ReadOnlyValue
										value={cacheKindItems[value.cacheControlKind as CacheControlKind]?.displayName ?? '—'}
									/>
								) : (
									<Dropdown<CacheControlKind>
										dropdownItems={cacheKindItems}
										selectedKey={(value.cacheControlKind || supportedCacheKinds[0]) as CacheControlKind}
										onChange={cacheControlKind => {
											onChange({ cacheControlKind });
										}}
										filterDisabled={false}
										title="Select cache kind"
										getDisplayName={key => cacheKindItems[key].displayName}
										disabled={disabled}
									/>
								)}
							</ModalField>

							<ModalField label="Cache TTL">
								{readOnly ? (
									<ReadOnlyValue value={cacheTTLItems[value.cacheControlTTL].displayName} />
								) : (
									<Dropdown<CacheControlTTLSelection>
										dropdownItems={cacheTTLItems}
										selectedKey={value.cacheControlTTL}
										onChange={cacheControlTTL => {
											onChange({ cacheControlTTL });
										}}
										filterDisabled={false}
										title="Select cache TTL"
										getDisplayName={key => cacheTTLItems[key].displayName}
										disabled={disabled}
									/>
								)}
							</ModalField>

							{supportsCacheKey ? (
								<ModalField label="Cache Key">
									{readOnly ? (
										<ReadOnlyValue value={value.cacheControlKey || '—'} />
									) : (
										<input
											className="input w-full rounded-xl"
											value={value.cacheControlKey}
											onChange={event => {
												onChange({ cacheControlKey: event.target.value });
											}}
											placeholder="Optional request cache key"
											autoComplete="off"
											spellCheck="false"
											disabled={disabled}
										/>
									)}
								</ModalField>
							) : null}
						</>
					) : null}
				</ModalSection>
			) : null}

			<ModalSection title="Prompt and Stopping" description="Set the default system prompt and stop sequences.">
				<ModalField label="System Prompt" htmlFor="model-system-prompt">
					{readOnly ? (
						<ReadOnlyValue value={value.systemPrompt || '—'} />
					) : (
						<textarea
							id="model-system-prompt"
							className="textarea h-24 w-full rounded-xl"
							value={value.systemPrompt}
							onChange={event => {
								onChange({ systemPrompt: event.target.value });
							}}
							placeholder="Optional default instructions"
							spellCheck="false"
							disabled={disabled}
						/>
					)}
				</ModalField>

				<ModalField label="Stop Sequences" htmlFor="model-stop-sequences" error={errors.stopSequencesRaw}>
					{readOnly ? (
						<ReadOnlyValue value={parseStopSequences(value.stopSequencesRaw).join(', ') || '—'} />
					) : (
						<>
							<textarea
								id="model-stop-sequences"
								className={`textarea h-20 w-full rounded-xl ${errors.stopSequencesRaw ? 'textarea-error' : ''}`}
								value={value.stopSequencesRaw}
								onChange={event => {
									onChange({ stopSequencesRaw: event.target.value });
								}}
								placeholder={'One per line\ne.g.\n###\n</final>'}
								spellCheck="false"
								disabled={disabled}
							/>
							<FieldError error={errors.stopSequencesRaw} />
						</>
					)}
				</ModalField>
			</ModalSection>

			<ModalSection title="Output Behavior" description="Configure output format and verbosity defaults.">
				<ModalField label="Output Format" error={errors.outputFormatKind}>
					{readOnly ? (
						<ReadOnlyValue
							value={getOutputFormatDisplayName(
								value.outputFormatKind === OUTPUT_FORMAT_NONE ? undefined : value.outputFormatKind
							)}
						/>
					) : (
						<>
							<Dropdown<OutputFormatSelection>
								dropdownItems={outputFormatItems}
								selectedKey={value.outputFormatKind}
								onChange={outputFormatKind => {
									onChange({ outputFormatKind });
								}}
								filterDisabled={false}
								title="Select output format"
								getDisplayName={key => getOutputFormatDisplayName(key === OUTPUT_FORMAT_NONE ? undefined : key)}
								disabled={disabled}
							/>
							<FieldError error={errors.outputFormatKind} />
						</>
					)}
				</ModalField>

				{value.outputFormatKind === OutputFormatKindValue.JSONSchema ? (
					<>
						<ModalField label="JSON Schema Name" required error={errors.outputJSONSchemaName}>
							{readOnly ? (
								<ReadOnlyValue value={value.outputJSONSchemaName || '—'} />
							) : (
								<>
									<input
										className={`input w-full rounded-xl ${errors.outputJSONSchemaName ? 'input-error' : ''}`}
										value={value.outputJSONSchemaName}
										onChange={event => {
											onChange({ outputJSONSchemaName: event.target.value });
										}}
										autoComplete="off"
										spellCheck="false"
										disabled={disabled}
									/>
									<FieldError error={errors.outputJSONSchemaName} />
								</>
							)}
						</ModalField>

						<ModalField label="Strict Mode">
							{readOnly ? (
								<ReadOnlyValue value={strictItems[value.outputJSONSchemaStrict].displayName} />
							) : (
								<Dropdown<StrictSelection>
									dropdownItems={strictItems}
									selectedKey={value.outputJSONSchemaStrict}
									onChange={outputJSONSchemaStrict => {
										onChange({ outputJSONSchemaStrict });
									}}
									filterDisabled={false}
									title="Select JSON Schema strictness"
									getDisplayName={key => strictItems[key].displayName}
									disabled={disabled}
								/>
							)}
						</ModalField>

						<ModalField label="JSON Schema Description">
							{readOnly ? (
								<ReadOnlyValue value={value.outputJSONSchemaDescription || '—'} />
							) : (
								<input
									className="input w-full rounded-xl"
									value={value.outputJSONSchemaDescription}
									onChange={event => {
										onChange({ outputJSONSchemaDescription: event.target.value });
									}}
									autoComplete="off"
									spellCheck="false"
									disabled={disabled}
								/>
							)}
						</ModalField>

						<ModalField label="JSON Schema Body" required error={errors.outputJSONSchemaRaw}>
							{readOnly ? (
								<pre className="bg-base-300 max-h-56 overflow-auto rounded-xl p-3 text-xs whitespace-pre-wrap">
									{value.outputJSONSchemaRaw || '—'}
								</pre>
							) : (
								<>
									<textarea
										className={`textarea h-40 w-full rounded-xl font-mono text-xs ${
											errors.outputJSONSchemaRaw ? 'textarea-error' : ''
										}`}
										value={value.outputJSONSchemaRaw}
										onChange={event => {
											onChange({ outputJSONSchemaRaw: event.target.value });
										}}
										placeholder='{"type":"object","properties":{}}'
										spellCheck="false"
										disabled={disabled}
									/>
									<FieldError error={errors.outputJSONSchemaRaw} />
								</>
							)}
						</ModalField>
					</>
				) : null}

				{supportsOutputVerbosity(capabilities) ? (
					<ModalField label="Output Verbosity" error={errors.outputVerbosity}>
						{readOnly ? (
							<ReadOnlyValue
								value={getOutputVerbosityDisplayName(
									value.outputVerbosity === OUTPUT_VERBOSITY_NONE ? undefined : value.outputVerbosity
								)}
							/>
						) : (
							<>
								<Dropdown<OutputVerbositySelection>
									dropdownItems={outputVerbosityItems}
									selectedKey={value.outputVerbosity}
									onChange={outputVerbosity => {
										onChange({ outputVerbosity });
									}}
									filterDisabled={false}
									title="Select output verbosity"
									getDisplayName={key => getOutputVerbosityDisplayName(key === OUTPUT_VERBOSITY_NONE ? undefined : key)}
									disabled={disabled}
								/>
								<FieldError error={errors.outputVerbosity} />
							</>
						)}
					</ModalField>
				) : null}
			</ModalSection>
		</>
	);
}
