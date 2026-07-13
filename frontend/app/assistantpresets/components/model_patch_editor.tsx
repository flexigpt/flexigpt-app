import { memo } from 'react';

import { FiRefreshCcw } from 'react-icons/fi';

import {
	CacheControlKind,
	CacheControlTTL,
	OutputFormatKind,
	OutputVerbosity,
	ReasoningLevel,
	ReasoningSummaryStyle,
	ReasoningType,
} from '@/spec/inference';

import { Dropdown } from '@/components/dropdown';
import { ModalField } from '@/components/modal/modal_field';
import { ModalSection } from '@/components/modal/modal_section';

import type { ModelPatchFormData, TriStateBoolean } from '@/assistantpresets/lib/assistant_preset_editor_types';
import { CACHE_CONTROL_KIND_LABELS, CACHE_CONTROL_TTL_LABELS } from '@/modelpresets/lib/capabilities_override';

interface AssistantPresetModelPatchEditorProps {
	isViewMode: boolean;
	modelPatch: ModelPatchFormData;
	error?: string;
	onPatchChange: (patch: Partial<ModelPatchFormData>) => void;
	canSeedFromSelectedModel: boolean;
	onSeedFromSelectedModel: () => void;
}

const REASONING_TYPES = Object.values(ReasoningType) as ReasoningType[];
const REASONING_LEVELS = Object.values(ReasoningLevel) as ReasoningLevel[];
const REASONING_SUMMARY_STYLES = Object.values(ReasoningSummaryStyle) as ReasoningSummaryStyle[];
const OUTPUT_VERBOSITIES = Object.values(OutputVerbosity) as OutputVerbosity[];
const OUTPUT_FORMAT_KINDS = Object.values(OutputFormatKind) as OutputFormatKind[];
const CACHE_CONTROL_KINDS = Object.values(CacheControlKind) as CacheControlKind[];
const CACHE_CONTROL_TTLS = Object.values(CacheControlTTL) as CacheControlTTL[];

const TRI_STATE_OPTIONS: TriStateBoolean[] = ['', 'true', 'false'];
const REASONING_SUMMARY_STYLE_OPTIONS: Array<'' | ReasoningSummaryStyle> = ['', ...REASONING_SUMMARY_STYLES];
const OUTPUT_VERBOSITY_OPTIONS: Array<'' | OutputVerbosity> = ['', ...OUTPUT_VERBOSITIES];
const CACHE_CONTROL_TTL_OPTIONS: Array<'' | CacheControlTTL> = ['', ...CACHE_CONTROL_TTLS];

function buildEnabledDropdownItems<K extends string>(keys: readonly K[]): Record<K, { isEnabled: boolean }> {
	return Object.fromEntries(keys.map(key => [key, { isEnabled: true }])) as Record<K, { isEnabled: boolean }>;
}

function getTriStateDropdownLabel(value: TriStateBoolean, defaultLabel: string): string {
	if (value === 'true') {
		return 'Force On';
	}
	if (value === 'false') {
		return 'Force Off';
	}
	return defaultLabel;
}

const TRI_STATE_DROPDOWN_ITEMS = buildEnabledDropdownItems(TRI_STATE_OPTIONS);
const REASONING_TYPE_DROPDOWN_ITEMS = buildEnabledDropdownItems(REASONING_TYPES);
const REASONING_LEVEL_DROPDOWN_ITEMS = buildEnabledDropdownItems(REASONING_LEVELS);
const REASONING_SUMMARY_STYLE_DROPDOWN_ITEMS = buildEnabledDropdownItems(REASONING_SUMMARY_STYLE_OPTIONS);
const OUTPUT_VERBOSITY_DROPDOWN_ITEMS = buildEnabledDropdownItems(OUTPUT_VERBOSITY_OPTIONS);
const OUTPUT_FORMAT_KIND_DROPDOWN_ITEMS = buildEnabledDropdownItems(OUTPUT_FORMAT_KINDS);
const CACHE_CONTROL_KIND_DROPDOWN_ITEMS = buildEnabledDropdownItems(CACHE_CONTROL_KINDS);
const CACHE_CONTROL_TTL_DROPDOWN_ITEMS = buildEnabledDropdownItems(CACHE_CONTROL_TTL_OPTIONS);

export const AssistantPresetModelPatchEditor = memo(function AssistantPresetModelPatchEditor({
	isViewMode,
	modelPatch,
	error,
	onPatchChange,
	canSeedFromSelectedModel,
	onSeedFromSelectedModel,
}: AssistantPresetModelPatchEditorProps) {
	return (
		<div className="space-y-4">
			<ModalSection
				title="Starting model patch"
				description="Override runtime knobs only. System prompt and capability overrides are intentionally excluded from assistant presets."
				actions={
					!isViewMode ? (
						<button
							type="button"
							className="btn btn-ghost btn-sm rounded-xl"
							disabled={!canSeedFromSelectedModel}
							onClick={onSeedFromSelectedModel}
							title={
								canSeedFromSelectedModel
									? modelPatch.enabled
										? 'Reset patch fields from the selected model preset defaults'
										: 'Load patch fields from the selected model preset defaults'
									: 'Select a starting model preset first'
							}
						>
							<FiRefreshCcw size={14} />
							<span className="ml-1">
								{modelPatch.enabled ? 'Reset from selected model' : 'Load selected model defaults'}
							</span>
						</button>
					) : null
				}
			>
				<ModalField label="Enable starting model patch" htmlFor="assistant-model-patch-enabled" error={error}>
					<input
						id="assistant-model-patch-enabled"
						type="checkbox"
						className="toggle toggle-accent"
						checked={modelPatch.enabled}
						disabled={isViewMode}
						onChange={e => {
							onPatchChange({ enabled: e.target.checked });
						}}
					/>
				</ModalField>
			</ModalSection>

			{modelPatch.enabled ? (
				<>
					<ModalSection
						title="Core runtime overrides"
						description="Leave a field unset to preserve the selected model preset or provider default."
					>
						<ModalField label="Stream Override" hint="Not Set preserves the model default.">
							<Dropdown<TriStateBoolean>
								dropdownItems={TRI_STATE_DROPDOWN_ITEMS}
								orderedKeys={TRI_STATE_OPTIONS}
								selectedKey={modelPatch.stream}
								onChange={stream => {
									onPatchChange({ stream });
								}}
								disabled={isViewMode}
								placeholderLabel="Leave Default"
								title="Stream override"
								getDisplayName={value => getTriStateDropdownLabel(value, 'Leave Default')}
							/>
						</ModalField>

						<ModalField label="Max Prompt Length" htmlFor="assistant-model-patch-max-prompt">
							<input
								id="assistant-model-patch-max-prompt"
								type="text"
								className="input w-full rounded-xl"
								readOnly={isViewMode}
								value={modelPatch.maxPromptLength}
								onChange={e => {
									onPatchChange({ maxPromptLength: e.target.value });
								}}
								spellCheck="false"
							/>
						</ModalField>

						<ModalField label="Max Output Length" htmlFor="assistant-model-patch-max-output">
							<input
								id="assistant-model-patch-max-output"
								type="text"
								className="input w-full rounded-xl"
								readOnly={isViewMode}
								value={modelPatch.maxOutputLength}
								onChange={e => {
									onPatchChange({ maxOutputLength: e.target.value });
								}}
								spellCheck="false"
							/>
						</ModalField>

						<ModalField label="Temperature" htmlFor="assistant-model-patch-temperature">
							<input
								id="assistant-model-patch-temperature"
								type="text"
								className="input w-full rounded-xl"
								readOnly={isViewMode}
								value={modelPatch.temperature}
								onChange={e => {
									onPatchChange({ temperature: e.target.value });
								}}
								spellCheck="false"
							/>
						</ModalField>

						<ModalField label="Timeout" htmlFor="assistant-model-patch-timeout" hint="Positive seconds.">
							<input
								id="assistant-model-patch-timeout"
								type="text"
								className="input w-full rounded-xl"
								readOnly={isViewMode}
								value={modelPatch.timeout}
								onChange={e => {
									onPatchChange({ timeout: e.target.value });
								}}
								spellCheck="false"
							/>
						</ModalField>

						<ModalField
							label="Stop Sequences"
							htmlFor="assistant-model-patch-stop-sequences"
							hint="Use one stop sequence per line."
							align="start"
						>
							<textarea
								id="assistant-model-patch-stop-sequences"
								className="textarea h-24 w-full rounded-xl"
								readOnly={isViewMode}
								value={modelPatch.stopSequencesText}
								onChange={e => {
									onPatchChange({ stopSequencesText: e.target.value });
								}}
								spellCheck="false"
								placeholder="One per line"
							/>
						</ModalField>

						<ModalField
							label="Additional Parameters Raw JSON"
							htmlFor="assistant-model-patch-additional-json"
							hint="Optional provider-specific JSON parameters. This value must be valid JSON."
							align="start"
						>
							<textarea
								id="assistant-model-patch-additional-json"
								className="textarea h-28 w-full rounded-xl font-mono text-xs"
								readOnly={isViewMode}
								value={modelPatch.additionalParametersRawJSON}
								onChange={e => {
									onPatchChange({
										additionalParametersRawJSON: e.target.value,
									});
								}}
								spellCheck="false"
							/>
						</ModalField>
					</ModalSection>

					<ModalSection
						title="Cache-control override"
						description="Override request-level cache behavior. Provider capability validation still applies at runtime."
					>
						<ModalField label="Override Cache Control" htmlFor="assistant-model-patch-cache-enabled">
							<input
								id="assistant-model-patch-cache-enabled"
								type="checkbox"
								className="toggle toggle-accent"
								checked={modelPatch.cacheControlEnabled}
								disabled={isViewMode}
								onChange={event => {
									onPatchChange({ cacheControlEnabled: event.currentTarget.checked });
								}}
							/>
						</ModalField>

						{modelPatch.cacheControlEnabled ? (
							<>
								<ModalField label="Cache Kind">
									<Dropdown<CacheControlKind>
										dropdownItems={CACHE_CONTROL_KIND_DROPDOWN_ITEMS}
										orderedKeys={CACHE_CONTROL_KINDS}
										selectedKey={modelPatch.cacheControlKind}
										onChange={cacheControlKind => {
											onPatchChange({ cacheControlKind });
										}}
										disabled={isViewMode}
										title="Cache-control kind"
										getDisplayName={value => CACHE_CONTROL_KIND_LABELS[value] ?? value}
									/>
								</ModalField>

								<ModalField label="Cache TTL" hint="Provider default leaves the explicit TTL unset.">
									<Dropdown<'' | CacheControlTTL>
										dropdownItems={CACHE_CONTROL_TTL_DROPDOWN_ITEMS}
										orderedKeys={CACHE_CONTROL_TTL_OPTIONS}
										selectedKey={modelPatch.cacheControlTTL}
										onChange={cacheControlTTL => {
											onPatchChange({ cacheControlTTL });
										}}
										disabled={isViewMode}
										placeholderLabel="Provider Default"
										title="Cache-control TTL"
										getDisplayName={value => (value ? (CACHE_CONTROL_TTL_LABELS[value] ?? value) : 'Provider Default')}
									/>
								</ModalField>

								<ModalField
									label="Cache Key"
									htmlFor="assistant-model-patch-cache-key"
									hint="Optional provider request-cache key."
								>
									<input
										id="assistant-model-patch-cache-key"
										type="text"
										className="input w-full rounded-xl"
										readOnly={isViewMode}
										value={modelPatch.cacheControlKey}
										onChange={event => {
											onPatchChange({ cacheControlKey: event.currentTarget.value });
										}}
										spellCheck="false"
										autoComplete="off"
									/>
								</ModalField>
							</>
						) : null}
					</ModalSection>

					<ModalSection
						title="Reasoning override"
						description="Override the selected model's reasoning configuration for chats started from this assistant preset."
					>
						<ModalField label="Override Reasoning" htmlFor="assistant-model-patch-reasoning-enabled">
							<input
								id="assistant-model-patch-reasoning-enabled"
								type="checkbox"
								className="toggle toggle-accent"
								checked={modelPatch.reasoningEnabled}
								disabled={isViewMode}
								onChange={e => {
									onPatchChange({
										reasoningEnabled: e.target.checked,
									});
								}}
							/>
						</ModalField>

						{modelPatch.reasoningEnabled ? (
							<>
								<ModalField label="Reasoning Type">
									<Dropdown<ReasoningType>
										dropdownItems={REASONING_TYPE_DROPDOWN_ITEMS}
										orderedKeys={REASONING_TYPES}
										selectedKey={modelPatch.reasoningType}
										onChange={reasoningType => {
											onPatchChange({ reasoningType });
										}}
										disabled={isViewMode}
										placeholderLabel="Select reasoning type"
										title="Reasoning type"
										getDisplayName={value => value}
									/>
								</ModalField>

								<ModalField label="Reasoning Level">
									<Dropdown<ReasoningLevel>
										dropdownItems={REASONING_LEVEL_DROPDOWN_ITEMS}
										orderedKeys={REASONING_LEVELS}
										selectedKey={modelPatch.reasoningLevel}
										onChange={reasoningLevel => {
											onPatchChange({ reasoningLevel });
										}}
										disabled={isViewMode}
										placeholderLabel="Select reasoning level"
										title="Reasoning level"
										getDisplayName={value => value}
									/>
								</ModalField>

								<ModalField label="Reasoning Tokens" htmlFor="assistant-model-patch-reasoning-tokens">
									<input
										id="assistant-model-patch-reasoning-tokens"
										type="text"
										className="input w-full rounded-xl"
										readOnly={isViewMode}
										value={modelPatch.reasoningTokens}
										onChange={e => {
											onPatchChange({ reasoningTokens: e.target.value });
										}}
										spellCheck="false"
									/>
								</ModalField>

								<ModalField label="Summary Style">
									<Dropdown<'' | ReasoningSummaryStyle>
										dropdownItems={REASONING_SUMMARY_STYLE_DROPDOWN_ITEMS}
										orderedKeys={REASONING_SUMMARY_STYLE_OPTIONS}
										selectedKey={modelPatch.reasoningSummaryStyle}
										onChange={reasoningSummaryStyle => {
											onPatchChange({ reasoningSummaryStyle });
										}}
										disabled={isViewMode}
										placeholderLabel="Leave Default"
										title="Reasoning summary style"
										getDisplayName={value => value || 'Leave Default'}
									/>
								</ModalField>
							</>
						) : null}
					</ModalSection>

					<ModalSection
						title="Output override"
						description="Override response verbosity and structured-output settings for chats started from this assistant preset."
					>
						<ModalField label="Override Output" htmlFor="assistant-model-patch-output-enabled">
							<input
								id="assistant-model-patch-output-enabled"
								type="checkbox"
								className="toggle toggle-accent"
								checked={modelPatch.outputEnabled}
								disabled={isViewMode}
								onChange={e => {
									onPatchChange({
										outputEnabled: e.target.checked,
									});
								}}
							/>
						</ModalField>

						{modelPatch.outputEnabled ? (
							<>
								<ModalField label="Verbosity">
									<Dropdown<'' | OutputVerbosity>
										dropdownItems={OUTPUT_VERBOSITY_DROPDOWN_ITEMS}
										orderedKeys={OUTPUT_VERBOSITY_OPTIONS}
										selectedKey={modelPatch.outputVerbosity}
										onChange={outputVerbosity => {
											onPatchChange({ outputVerbosity });
										}}
										disabled={isViewMode}
										placeholderLabel="Leave Default"
										title="Output verbosity"
										getDisplayName={value => value || 'Leave Default'}
									/>
								</ModalField>

								<ModalField label="Override Output Format" htmlFor="assistant-model-patch-output-format-enabled">
									<input
										id="assistant-model-patch-output-format-enabled"
										type="checkbox"
										className="toggle toggle-accent"
										checked={modelPatch.outputFormatEnabled}
										disabled={isViewMode}
										onChange={e => {
											onPatchChange({
												outputFormatEnabled: e.target.checked,
											});
										}}
									/>
								</ModalField>

								{modelPatch.outputFormatEnabled ? (
									<>
										<ModalField label="Format Kind">
											<Dropdown<OutputFormatKind>
												dropdownItems={OUTPUT_FORMAT_KIND_DROPDOWN_ITEMS}
												orderedKeys={OUTPUT_FORMAT_KINDS}
												selectedKey={modelPatch.outputFormatKind}
												onChange={outputFormatKind => {
													onPatchChange({ outputFormatKind });
												}}
												disabled={isViewMode}
												placeholderLabel="Select format kind"
												title="Output format kind"
												getDisplayName={value => value}
											/>
										</ModalField>

										{modelPatch.outputFormatKind === OutputFormatKind.JSONSchema ? (
											<>
												<ModalField label="JSON Schema Name" htmlFor="assistant-model-patch-schema-name" required>
													<input
														id="assistant-model-patch-schema-name"
														type="text"
														className="input w-full rounded-xl"
														readOnly={isViewMode}
														value={modelPatch.outputJSONSchemaName}
														onChange={e => {
															onPatchChange({
																outputJSONSchemaName: e.target.value,
															});
														}}
														spellCheck="false"
													/>
												</ModalField>

												<ModalField label="Strict Mode">
													<Dropdown<TriStateBoolean>
														dropdownItems={TRI_STATE_DROPDOWN_ITEMS}
														orderedKeys={TRI_STATE_OPTIONS}
														selectedKey={modelPatch.outputJSONSchemaStrictMode}
														onChange={outputJSONSchemaStrictMode => {
															onPatchChange({ outputJSONSchemaStrictMode });
														}}
														disabled={isViewMode}
														placeholderLabel="Leave Default"
														title="JSON schema strict mode"
														getDisplayName={value => getTriStateDropdownLabel(value, 'Leave Default')}
													/>
												</ModalField>

												<ModalField label="JSON Schema Description" htmlFor="assistant-model-patch-schema-description">
													<input
														id="assistant-model-patch-schema-description"
														type="text"
														className="input w-full rounded-xl"
														readOnly={isViewMode}
														value={modelPatch.outputJSONSchemaDescription}
														onChange={e => {
															onPatchChange({
																outputJSONSchemaDescription: e.target.value,
															});
														}}
														spellCheck="false"
													/>
												</ModalField>

												<ModalField label="JSON Schema Body" htmlFor="assistant-model-patch-schema-body" align="start">
													<textarea
														id="assistant-model-patch-schema-body"
														className="textarea h-32 w-full rounded-xl font-mono text-xs"
														readOnly={isViewMode}
														value={modelPatch.outputJSONSchemaRaw}
														onChange={e => {
															onPatchChange({
																outputJSONSchemaRaw: e.target.value,
															});
														}}
														spellCheck="false"
													/>
												</ModalField>
											</>
										) : null}
									</>
								) : null}
							</>
						) : null}
					</ModalSection>
				</>
			) : null}
		</div>
	);
});
