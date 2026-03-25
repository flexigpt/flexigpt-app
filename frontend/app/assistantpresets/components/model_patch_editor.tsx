import { memo } from 'react';

import { FiAlertCircle, FiHelpCircle, FiRefreshCcw } from 'react-icons/fi';

import {
	OutputFormatKind,
	OutputVerbosity,
	ReasoningLevel,
	ReasoningSummaryStyle,
	ReasoningType,
} from '@/spec/inference';

import { Dropdown } from '@/components/dropdown';

import type { ModelPatchFormData, TriStateBoolean } from '@/assistantpresets/lib/assistant_preset_editor_types';

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

const TRI_STATE_OPTIONS: TriStateBoolean[] = ['', 'true', 'false'];
const REASONING_SUMMARY_STYLE_OPTIONS: Array<'' | ReasoningSummaryStyle> = ['', ...REASONING_SUMMARY_STYLES];
const OUTPUT_VERBOSITY_OPTIONS: Array<'' | OutputVerbosity> = ['', ...OUTPUT_VERBOSITIES];

function buildEnabledDropdownItems<K extends string>(keys: readonly K[]): Record<K, { isEnabled: boolean }> {
	return Object.fromEntries(keys.map(key => [key, { isEnabled: true }])) as Record<K, { isEnabled: boolean }>;
}

function getTriStateDropdownLabel(value: TriStateBoolean, defaultLabel: string): string {
	if (value === 'true') return 'Force On';
	if (value === 'false') return 'Force Off';
	return defaultLabel;
}

const TRI_STATE_DROPDOWN_ITEMS = buildEnabledDropdownItems(TRI_STATE_OPTIONS);
const REASONING_TYPE_DROPDOWN_ITEMS = buildEnabledDropdownItems(REASONING_TYPES);
const REASONING_LEVEL_DROPDOWN_ITEMS = buildEnabledDropdownItems(REASONING_LEVELS);
const REASONING_SUMMARY_STYLE_DROPDOWN_ITEMS = buildEnabledDropdownItems(REASONING_SUMMARY_STYLE_OPTIONS);
const OUTPUT_VERBOSITY_DROPDOWN_ITEMS = buildEnabledDropdownItems(OUTPUT_VERBOSITY_OPTIONS);
const OUTPUT_FORMAT_KIND_DROPDOWN_ITEMS = buildEnabledDropdownItems(OUTPUT_FORMAT_KINDS);

export const AssistantPresetModelPatchEditor = memo(function AssistantPresetModelPatchEditor({
	isViewMode,
	modelPatch,
	error,
	onPatchChange,
	canSeedFromSelectedModel,
	onSeedFromSelectedModel,
}: AssistantPresetModelPatchEditorProps) {
	return (
		<div className="flex flex-col space-y-3">
			<div className="grid grid-cols-12 gap-2">
				<label className="label col-span-3 cursor-pointer">
					<span className="label-text text-sm">Starting Model Patch</span>
					<span
						className="label-text-alt tooltip tooltip-right"
						data-tip="Runtime knob patch only. systemPrompt is intentionally not allowed."
					>
						<FiHelpCircle size={12} />
					</span>
				</label>
				<div className="col-span-9 flex flex-row items-start gap-2">
					<input
						type="checkbox"
						className="toggle toggle-accent"
						checked={modelPatch.enabled}
						disabled={isViewMode}
						onChange={e => {
							onPatchChange({ enabled: e.target.checked });
						}}
					/>

					{!isViewMode && (
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
					)}

					{error && (
						<div className="text-error flex items-center gap-1 text-sm">
							<FiAlertCircle size={12} /> {error}
						</div>
					)}
				</div>
			</div>

			{modelPatch.enabled && (
				<div className="border-base-content/10 space-y-4 rounded-2xl border p-4">
					<div className="grid grid-cols-12 gap-2">
						<div className="col-span-12 md:col-span-4">
							<label className="label py-1">
								<span className="label-text text-sm">Stream Override</span>
							</label>
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
						</div>

						<div className="col-span-12 md:col-span-4">
							<label className="label py-1">
								<span className="label-text text-sm">Max Prompt Length</span>
							</label>
							<input
								type="text"
								className="input input-bordered w-full rounded-xl"
								readOnly={isViewMode}
								value={modelPatch.maxPromptLength}
								onChange={e => {
									onPatchChange({ maxPromptLength: e.target.value });
								}}
								spellCheck="false"
							/>
						</div>

						<div className="col-span-12 md:col-span-4">
							<label className="label py-1">
								<span className="label-text text-sm">Max Output Length</span>
							</label>
							<input
								type="text"
								className="input input-bordered w-full rounded-xl"
								readOnly={isViewMode}
								value={modelPatch.maxOutputLength}
								onChange={e => {
									onPatchChange({ maxOutputLength: e.target.value });
								}}
								spellCheck="false"
							/>
						</div>

						<div className="col-span-12 md:col-span-4">
							<label className="label py-1">
								<span className="label-text text-sm">Temperature</span>
							</label>
							<input
								type="text"
								className="input input-bordered w-full rounded-xl"
								readOnly={isViewMode}
								value={modelPatch.temperature}
								onChange={e => {
									onPatchChange({ temperature: e.target.value });
								}}
								spellCheck="false"
							/>
						</div>

						<div className="col-span-12 md:col-span-4">
							<label className="label py-1">
								<span className="label-text text-sm">Timeout (seconds)</span>
							</label>
							<input
								type="text"
								className="input input-bordered w-full rounded-xl"
								readOnly={isViewMode}
								value={modelPatch.timeout}
								onChange={e => {
									onPatchChange({ timeout: e.target.value });
								}}
								spellCheck="false"
							/>
						</div>

						<div className="col-span-12 md:col-span-4">
							<label className="label py-1">
								<span className="label-text text-sm">Stop Sequences</span>
							</label>
							<textarea
								className="textarea textarea-bordered h-24 w-full rounded-xl"
								readOnly={isViewMode}
								value={modelPatch.stopSequencesText}
								onChange={e => {
									onPatchChange({ stopSequencesText: e.target.value });
								}}
								spellCheck="false"
								placeholder="One per line"
							/>
						</div>

						<div className="col-span-12">
							<label className="label py-1">
								<span className="label-text text-sm">Additional Parameters Raw JSON</span>
							</label>
							<textarea
								className="textarea textarea-bordered h-28 w-full rounded-xl font-mono text-xs"
								readOnly={isViewMode}
								value={modelPatch.additionalParametersRawJSON}
								onChange={e => {
									onPatchChange({
										additionalParametersRawJSON: e.target.value,
									});
								}}
								spellCheck="false"
							/>
						</div>
					</div>

					<div className="divider my-0">Reasoning Override</div>

					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-3 cursor-pointer">
							<span className="label-text text-sm">Override Reasoning</span>
						</label>
						<div className="col-span-9">
							<input
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
						</div>
					</div>

					{modelPatch.reasoningEnabled && (
						<div className="grid grid-cols-12 gap-2">
							<div className="col-span-12 md:col-span-4">
								<label className="label py-1">
									<span className="label-text text-sm">Reasoning Type</span>
								</label>
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
							</div>

							<div className="col-span-12 md:col-span-4">
								<label className="label py-1">
									<span className="label-text text-sm">Reasoning Level</span>
								</label>
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
							</div>

							<div className="col-span-12 md:col-span-4">
								<label className="label py-1">
									<span className="label-text text-sm">Reasoning Tokens</span>
								</label>
								<input
									type="text"
									className="input input-bordered w-full rounded-xl"
									readOnly={isViewMode}
									value={modelPatch.reasoningTokens}
									onChange={e => {
										onPatchChange({ reasoningTokens: e.target.value });
									}}
									spellCheck="false"
								/>
							</div>

							<div className="col-span-12 md:col-span-4">
								<label className="label py-1">
									<span className="label-text text-sm">Summary Style</span>
								</label>
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
							</div>
						</div>
					)}

					<div className="divider my-0">Output Override</div>

					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-3 cursor-pointer">
							<span className="label-text text-sm">Override Output</span>
						</label>
						<div className="col-span-9">
							<input
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
						</div>
					</div>

					{modelPatch.outputEnabled && (
						<div className="grid grid-cols-12 gap-2">
							<div className="col-span-12 md:col-span-4">
								<label className="label py-1">
									<span className="label-text text-sm">Verbosity</span>
								</label>
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
							</div>

							<div className="col-span-12 md:col-span-4">
								<label className="label py-1">
									<span className="label-text text-sm">Override Output Format</span>
								</label>
								<div>
									<input
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
								</div>
							</div>

							{modelPatch.outputFormatEnabled && (
								<>
									<div className="col-span-12 md:col-span-4">
										<label className="label py-1">
											<span className="label-text text-sm">Format Kind</span>
										</label>
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
									</div>

									{modelPatch.outputFormatKind === OutputFormatKind.JSONSchema && (
										<>
											<div className="col-span-12 md:col-span-6">
												<label className="label py-1">
													<span className="label-text text-sm">JSON Schema Name*</span>
												</label>
												<input
													type="text"
													className="input input-bordered w-full rounded-xl"
													readOnly={isViewMode}
													value={modelPatch.outputJSONSchemaName}
													onChange={e => {
														onPatchChange({
															outputJSONSchemaName: e.target.value,
														});
													}}
													spellCheck="false"
												/>
											</div>

											<div className="col-span-12 md:col-span-6">
												<label className="label py-1">
													<span className="label-text text-sm">Strict</span>
												</label>
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
											</div>

											<div className="col-span-12">
												<label className="label py-1">
													<span className="label-text text-sm">JSON Schema Description</span>
												</label>
												<input
													type="text"
													className="input input-bordered w-full rounded-xl"
													readOnly={isViewMode}
													value={modelPatch.outputJSONSchemaDescription}
													onChange={e => {
														onPatchChange({
															outputJSONSchemaDescription: e.target.value,
														});
													}}
													spellCheck="false"
												/>
											</div>

											<div className="col-span-12">
												<label className="label py-1">
													<span className="label-text text-sm">JSON Schema Body</span>
												</label>
												<textarea
													className="textarea textarea-bordered h-32 w-full rounded-xl font-mono text-xs"
													readOnly={isViewMode}
													value={modelPatch.outputJSONSchemaRaw}
													onChange={e => {
														onPatchChange({
															outputJSONSchemaRaw: e.target.value,
														});
													}}
													spellCheck="false"
												/>
											</div>
										</>
									)}
								</>
							)}
						</div>
					)}
				</div>
			)}
		</div>
	);
});
