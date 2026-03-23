import { memo } from 'react';

import { FiAlertCircle } from 'react-icons/fi';

import {
	OutputFormatKind,
	OutputVerbosity,
	ReasoningLevel,
	ReasoningSummaryStyle,
	ReasoningType,
} from '@/spec/inference';

import type { ModelPatchFormData, TriStateBoolean } from '@/assistantpresets/lib/assistant_preset_editor_types';

interface AssistantPresetModelPatchEditorProps {
	isViewMode: boolean;
	modelPatch: ModelPatchFormData;
	error?: string;
	onPatchChange: (patch: Partial<ModelPatchFormData>) => void;
}

const REASONING_TYPES = Object.values(ReasoningType);
const REASONING_LEVELS = Object.values(ReasoningLevel);
const REASONING_SUMMARY_STYLES = Object.values(ReasoningSummaryStyle);
const OUTPUT_VERBOSITIES = Object.values(OutputVerbosity);
const OUTPUT_FORMAT_KINDS = Object.values(OutputFormatKind);

export const AssistantPresetModelPatchEditor = memo(function AssistantPresetModelPatchEditor({
	isViewMode,
	modelPatch,
	error,
	onPatchChange,
}: AssistantPresetModelPatchEditorProps) {
	return (
		<div className="col-span-9 space-y-3">
			<input
				type="checkbox"
				className="toggle toggle-accent"
				checked={modelPatch.enabled}
				disabled={isViewMode}
				onChange={e => {
					onPatchChange({ enabled: e.target.checked });
				}}
			/>

			<p className="text-base-content/70 text-xs">
				Assistant presets store refs and starter knobs only. This patch is limited to runtime model parameters and may
				not include <code>systemPrompt</code> or <code>capabilitiesOverride</code>.
			</p>

			{error && (
				<div className="text-error flex items-center gap-1 text-sm">
					<FiAlertCircle size={12} /> {error}
				</div>
			)}

			{modelPatch.enabled && (
				<div className="border-base-content/10 space-y-4 rounded-2xl border p-4">
					<div className="grid grid-cols-12 gap-2">
						<div className="col-span-12 md:col-span-4">
							<label className="label py-1">
								<span className="label-text text-sm">Stream Override</span>
							</label>
							<select
								className="select select-bordered w-full rounded-xl"
								disabled={isViewMode}
								value={modelPatch.stream}
								onChange={e => {
									onPatchChange({
										stream: e.target.value as TriStateBoolean,
									});
								}}
							>
								<option value="">Leave Default</option>
								<option value="true">Force On</option>
								<option value="false">Force Off</option>
							</select>
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
								<select
									className="select select-bordered w-full rounded-xl"
									disabled={isViewMode}
									value={modelPatch.reasoningType}
									onChange={e => {
										onPatchChange({
											reasoningType: e.target.value as ReasoningType,
										});
									}}
								>
									{REASONING_TYPES.map(value => (
										<option key={value} value={value}>
											{value}
										</option>
									))}
								</select>
							</div>

							<div className="col-span-12 md:col-span-4">
								<label className="label py-1">
									<span className="label-text text-sm">Reasoning Level</span>
								</label>
								<select
									className="select select-bordered w-full rounded-xl"
									disabled={isViewMode}
									value={modelPatch.reasoningLevel}
									onChange={e => {
										onPatchChange({
											reasoningLevel: e.target.value as ReasoningLevel,
										});
									}}
								>
									{REASONING_LEVELS.map(value => (
										<option key={value} value={value}>
											{value}
										</option>
									))}
								</select>
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
								<select
									className="select select-bordered w-full rounded-xl"
									disabled={isViewMode}
									value={modelPatch.reasoningSummaryStyle}
									onChange={e => {
										onPatchChange({
											reasoningSummaryStyle: e.target.value as '' | ReasoningSummaryStyle,
										});
									}}
								>
									<option value="">Leave Default</option>
									{REASONING_SUMMARY_STYLES.map(value => (
										<option key={value} value={value}>
											{value}
										</option>
									))}
								</select>
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
								<select
									className="select select-bordered w-full rounded-xl"
									disabled={isViewMode}
									value={modelPatch.outputVerbosity}
									onChange={e => {
										onPatchChange({
											outputVerbosity: e.target.value as '' | OutputVerbosity,
										});
									}}
								>
									<option value="">Leave Default</option>
									{OUTPUT_VERBOSITIES.map(value => (
										<option key={value} value={value}>
											{value}
										</option>
									))}
								</select>
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
										<select
											className="select select-bordered w-full rounded-xl"
											disabled={isViewMode}
											value={modelPatch.outputFormatKind}
											onChange={e => {
												onPatchChange({
													outputFormatKind: e.target.value as OutputFormatKind,
												});
											}}
										>
											{OUTPUT_FORMAT_KINDS.map(value => (
												<option key={value} value={value}>
													{value}
												</option>
											))}
										</select>
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
												<select
													className="select select-bordered w-full rounded-xl"
													disabled={isViewMode}
													value={modelPatch.outputJSONSchemaStrictMode}
													onChange={e => {
														onPatchChange({
															outputJSONSchemaStrictMode: e.target.value as TriStateBoolean,
														});
													}}
												>
													<option value="">Leave Default</option>
													<option value="true">Force On</option>
													<option value="false">Force Off</option>
												</select>
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
