import { useEffect, useMemo, useRef } from 'react';

import { createPortal } from 'react-dom';

import { FiSliders, FiTool, FiX, FiZap } from 'react-icons/fi';

import type { AssistantPreset } from '@/spec/assistantpreset';
import type { UIChatOption } from '@/spec/modelpreset';
import type { SkillSelection } from '@/spec/skill';
import { ToolStoreChoiceType } from '@/spec/tool';

import { ModalBackdrop } from '@/components/modal_backdrop';

import {
	type AssistantPresetModificationSummary,
	type AssistantPresetOptionItem,
	type AssistantPresetPreparedApplication,
	type AssistantPresetRuntimeSnapshot,
	buildAssistantPresetModelComparisonState,
} from '@/chats/composer/assistantpresets/assistant_preset_runtime';
import { buildPromptTemplateRefKey } from '@/prompts/lib/prompt_template_ref';
import type { SystemPromptItem } from '@/prompts/lib/use_system_prompts';

function closeDialogSafely(dialog: HTMLDialogElement | null): boolean {
	if (!dialog?.open) return false;

	try {
		dialog.close();
		return true;
	} catch {
		return false;
	}
}

function formatToolLabel(choice: {
	displayName?: string;
	toolSlug: string;
	toolVersion: string;
	bundleID: string;
	toolType: ToolStoreChoiceType;
	autoExecute: boolean;
	userArgSchemaInstance?: string;
}) {
	const title = choice.displayName?.trim() || choice.toolSlug;
	const metaParts = [`${choice.bundleID}/${choice.toolSlug}@${choice.toolVersion}`];

	if (choice.toolType === ToolStoreChoiceType.WebSearch) {
		metaParts.push('web-search');
	}

	if (choice.autoExecute) {
		metaParts.push('auto');
	}

	if (choice.userArgSchemaInstance?.trim()) {
		metaParts.push('args');
	}

	return {
		title,
		meta: metaParts.join(' • '),
	};
}

function formatSkillLabel(ref: { bundleID: string; skillSlug: string; skillID: string }) {
	return {
		title: ref.skillSlug,
		meta: `${ref.bundleID} • ${ref.skillID}`,
	};
}

function formatSkillSelectionLabel(sel: SkillSelection) {
	const metaParts = [`${sel.skillRef.bundleID} • ${sel.skillRef.skillID}`];
	if (sel.preLoadAsActive) {
		metaParts.push('Preload as active');
	}
	return {
		title: sel.skillRef.skillSlug,
		meta: metaParts.join(' • '),
	};
}

function renderSimpleList(items: Array<{ title: string; meta?: string }>, emptyText: string): React.ReactNode {
	if (items.length === 0) {
		return <div className="text-xs opacity-70">{emptyText}</div>;
	}

	return (
		<ul className="space-y-2">
			{items.map((item, index) => (
				<li key={`${item.title}-${item.meta ?? ''}-${index}`} className="rounded-xl border p-2">
					<div className="text-sm font-medium">{item.title}</div>
					{item.meta ? <div className="mt-1 text-xs opacity-70">{item.meta}</div> : null}
				</li>
			))}
		</ul>
	);
}

function buildPresetModelPreviewState(preset: AssistantPreset): Record<string, unknown> | undefined {
	const state: Record<string, unknown> = {};

	if (preset.startingModelPresetRef) {
		state.modelRef = preset.startingModelPresetRef;
	}

	if (preset.startingModelPresetPatch) {
		Object.assign(state, preset.startingModelPresetPatch);
	}

	if (preset.startingIncludeModelSystemPrompt !== undefined) {
		state.includeModelSystemPrompt = preset.startingIncludeModelSystemPrompt;
	}

	return Object.keys(state).length > 0 ? state : undefined;
}

function formatPresetToolSelectionLabel(selection: {
	toolRef: {
		bundleID: string;
		toolSlug: string;
		toolVersion: string;
	};
	toolChoicePatch?: {
		autoExecute?: boolean;
		userArgSchemaInstance?: string;
	};
}) {
	const metaParts = [`${selection.toolRef.bundleID}/${selection.toolRef.toolSlug}@${selection.toolRef.toolVersion}`];

	if (selection.toolChoicePatch?.autoExecute) metaParts.push('auto');
	if (selection.toolChoicePatch?.userArgSchemaInstance?.trim()) metaParts.push('args');

	return {
		title: selection.toolRef.toolSlug,
		meta: metaParts.join(' • '),
	};
}

function SectionCard(props: {
	title: string;
	isModified?: boolean;
	showSyncState?: boolean;
	children: React.ReactNode;
	icon?: React.ReactNode;
}) {
	return (
		<section className="border-base-300 rounded-2xl border p-4">
			<div className="mb-3 flex items-center gap-2">
				{props.icon}
				<h4 className="text-sm font-semibold">{props.title}</h4>
				<span
					className={`badge badge-sm ${props.showSyncState ? (props.isModified ? 'badge-warning' : 'badge-success') : 'badge-outline'}`}
				>
					{props.showSyncState ? (props.isModified ? 'Modified' : 'In sync') : 'Preset values'}
				</span>
			</div>
			{props.children}
		</section>
	);
}

type AssistantPresetViewModalProps = {
	isOpen: boolean;
	onClose: () => void;
	viewedPreset: AssistantPresetOptionItem | null;
	viewedPresetApplication: AssistantPresetPreparedApplication | null;
	isActivePresetView: boolean;

	currentRuntimeSnapshot: AssistantPresetRuntimeSnapshot;
	currentModel: UIChatOption;
	currentIncludeModelSystemPrompt: boolean;
	currentSelectedPromptKeys: string[];
	promptItems: SystemPromptItem[];
	modificationSummary: AssistantPresetModificationSummary;
};

export function AssistantPresetViewModal({
	isOpen,
	onClose,
	viewedPreset,
	viewedPresetApplication,
	isActivePresetView,
	currentRuntimeSnapshot,
	currentModel,
	currentIncludeModelSystemPrompt,
	currentSelectedPromptKeys,
	promptItems,
	modificationSummary,
}: AssistantPresetViewModalProps) {
	const dialogRef = useRef<HTMLDialogElement | null>(null);

	useEffect(() => {
		if (!isOpen) return;

		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) {
			dialog.showModal();
		}

		return () => {
			if (dialog.open) {
				dialog.close();
			}
		};
	}, [isOpen]);

	const promptItemsByKey = useMemo(() => {
		return new Map(promptItems.map(item => [item.identityKey, item]));
	}, [promptItems]);

	if (!isOpen || !viewedPreset || typeof document === 'undefined') {
		return null;
	}

	const preparedApplication = viewedPresetApplication;
	const shouldShowActiveComparison = isActivePresetView && preparedApplication !== null;

	const appliedModelState =
		preparedApplication?.comparisonState.model ?? buildPresetModelPreviewState(viewedPreset.preset);
	const currentModelState =
		shouldShowActiveComparison && preparedApplication
			? buildAssistantPresetModelComparisonState(
					preparedApplication.preset,
					currentModel,
					currentIncludeModelSystemPrompt
				)
			: undefined;

	const promptLabelItems = (keys: string[]) =>
		keys.map(key => {
			const item = promptItemsByKey.get(key);
			return {
				title: item?.displayName || key,
				meta: item ? `${item.bundleDisplayName} • ${item.templateSlug}@${item.templateVersion}` : key,
			};
		});

	const appliedInstructionKeys =
		preparedApplication?.comparisonState.instructions ??
		(viewedPreset.preset.startingInstructionTemplateRefs ?? []).map(buildPromptTemplateRefKey);
	const appliedToolItems = preparedApplication
		? [
				...preparedApplication.runtimeSelections.conversationToolChoices,
				...preparedApplication.runtimeSelections.webSearchChoices,
			].map(formatToolLabel)
		: (viewedPreset.preset.startingToolSelections ?? []).map(formatPresetToolSelectionLabel);

	const currentToolItems = shouldShowActiveComparison
		? [...currentRuntimeSnapshot.conversationToolChoices, ...currentRuntimeSnapshot.webSearchChoices].map(choice =>
				formatToolLabel({
					displayName: choice.displayName,
					toolSlug: choice.toolSlug,
					toolVersion: choice.toolVersion,
					bundleID: choice.bundleID,
					toolType: choice.toolType,
					autoExecute: choice.autoExecute,
					userArgSchemaInstance: choice.userArgSchemaInstance,
				})
			)
		: [];

	const appliedSkillItems = (viewedPreset.preset.startingSkillSelections ?? []).map(formatSkillSelectionLabel);
	const currentSkillItems = shouldShowActiveComparison
		? currentRuntimeSnapshot.enabledSkillRefs.map(formatSkillLabel)
		: [];

	const description = viewedPreset.description || viewedPreset.preset.description;
	const showCurrentModel = shouldShowActiveComparison && modificationSummary.model;
	const showCurrentInstructions = shouldShowActiveComparison && modificationSummary.instructions;
	const showCurrentTools = shouldShowActiveComparison && modificationSummary.tools;
	const showCurrentSkills = shouldShowActiveComparison && modificationSummary.skills;

	return createPortal(
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={onClose}
			onCancel={event => {
				event.preventDefault();
				if (!closeDialogSafely(dialogRef.current)) {
					onClose();
				}
			}}
		>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-4xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<div className="mb-4 flex items-center justify-between gap-3">
						<div>
							<h3 className="text-lg font-bold">{viewedPreset.displayName}</h3>
							<div className="mt-1 text-xs opacity-70">
								{viewedPreset.bundleDisplayName} • {viewedPreset.preset.slug}@{viewedPreset.preset.version}
							</div>
						</div>

						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={() => {
								if (!closeDialogSafely(dialogRef.current)) {
									onClose();
								}
							}}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>

					<div className="mb-4 rounded-2xl border p-4">
						{description ? (
							<p className="text-sm">{description}</p>
						) : (
							<p className="text-sm opacity-70">No description.</p>
						)}

						<div className="mt-3 flex flex-wrap items-center gap-2">
							{shouldShowActiveComparison ? (
								<>
									<span
										className={`badge badge-sm ${modificationSummary.any ? 'badge-warning' : 'badge-success'}`}
										title={
											modificationSummary.any
												? `Modified sections: ${modificationSummary.modifiedLabels.join(', ')}`
												: 'Current values still match the applied preset.'
										}
									>
										{modificationSummary.any ? 'Preset overridden' : 'Preset in sync'}
									</span>
									{modificationSummary.modifiedLabels.map(label => (
										<span key={label} className="badge badge-outline badge-sm">
											{label}
										</span>
									))}
								</>
							) : (
								<span className="badge badge-outline badge-sm">Preset preview</span>
							)}

							{!viewedPreset.isSelectable ? <span className="badge badge-warning badge-sm">Unavailable</span> : null}
						</div>

						{!viewedPreset.isSelectable && viewedPreset.availabilityReason ? (
							<div className="text-warning mt-3 text-xs">{viewedPreset.availabilityReason}</div>
						) : null}
					</div>

					<div className="space-y-4">
						{appliedModelState ? (
							<SectionCard
								title="Model and advanced params"
								showSyncState={shouldShowActiveComparison}
								isModified={modificationSummary.model}
								icon={<FiSliders size={14} />}
							>
								<div className={`grid gap-4 ${showCurrentModel ? 'lg:grid-cols-2' : 'grid-cols-1'}`}>
									<div>
										<div className="mb-2 text-xs font-semibold opacity-70">
											{shouldShowActiveComparison ? 'Preset-applied values' : 'Preset values'}
										</div>
										<pre className="bg-base-300 overflow-x-auto rounded-xl p-3 text-xs">
											{JSON.stringify(appliedModelState, null, 2)}
										</pre>
									</div>

									{showCurrentModel ? (
										<div>
											<div className="mb-2 text-xs font-semibold opacity-70">Current values</div>
											<pre className="bg-base-300 overflow-x-auto rounded-xl p-3 text-xs">
												{JSON.stringify(currentModelState, null, 2)}
											</pre>
										</div>
									) : null}
								</div>
							</SectionCard>
						) : null}

						{appliedInstructionKeys.length > 0 ? (
							<SectionCard
								title="Instruction templates"
								showSyncState={shouldShowActiveComparison}
								isModified={modificationSummary.instructions}
							>
								<div className={`grid gap-4 ${showCurrentInstructions ? 'lg:grid-cols-2' : 'grid-cols-1'}`}>
									<div>
										<div className="mb-2 text-xs font-semibold opacity-70">
											{shouldShowActiveComparison ? 'Preset-applied selection' : 'Preset selection'}
										</div>
										{renderSimpleList(promptLabelItems(appliedInstructionKeys), 'No instruction templates selected.')}
									</div>

									{showCurrentInstructions ? (
										<div>
											<div className="mb-2 text-xs font-semibold opacity-70">Current selection</div>
											{renderSimpleList(
												promptLabelItems(currentSelectedPromptKeys),
												'No instruction templates selected.'
											)}
										</div>
									) : null}
								</div>
							</SectionCard>
						) : null}

						{appliedToolItems.length > 0 ? (
							<SectionCard
								title="Tools and web search"
								showSyncState={shouldShowActiveComparison}
								isModified={modificationSummary.tools}
								icon={<FiTool size={14} />}
							>
								<div className={`grid gap-4 ${showCurrentTools ? 'lg:grid-cols-2' : 'grid-cols-1'}`}>
									<div>
										<div className="mb-2 text-xs font-semibold opacity-70">
											{shouldShowActiveComparison ? 'Preset-applied selection' : 'Preset selection'}
										</div>
										{renderSimpleList(appliedToolItems, 'No tool selections.')}
									</div>

									{showCurrentTools ? (
										<div>
											<div className="mb-2 text-xs font-semibold opacity-70">Current selection</div>
											{renderSimpleList(currentToolItems, 'No tool selections.')}
										</div>
									) : null}
								</div>
							</SectionCard>
						) : null}

						{appliedSkillItems.length > 0 ? (
							<SectionCard
								title="Enabled skills"
								showSyncState={shouldShowActiveComparison}
								isModified={modificationSummary.skills}
								icon={<FiZap size={14} />}
							>
								<div className={`grid gap-4 ${showCurrentSkills ? 'lg:grid-cols-2' : 'grid-cols-1'}`}>
									<div>
										<div className="mb-2 text-xs font-semibold opacity-70">
											{shouldShowActiveComparison ? 'Preset-applied selection' : 'Preset selection'}
										</div>
										{renderSimpleList(appliedSkillItems, 'No enabled skills.')}
									</div>

									{showCurrentSkills ? (
										<div>
											<div className="mb-2 text-xs font-semibold opacity-70">Current selection</div>
											{renderSimpleList(currentSkillItems, 'No enabled skills.')}
										</div>
									) : null}
								</div>
							</SectionCard>
						) : null}

						{!appliedModelState &&
						appliedInstructionKeys.length === 0 &&
						appliedToolItems.length === 0 &&
						appliedSkillItems.length === 0 ? (
							<div className="rounded-2xl border p-4 text-sm opacity-70">
								This preset only carries display metadata right now; it does not manage any current authoring sections.
							</div>
						) : null}
					</div>

					<div className="mt-6 flex justify-end">
						<button
							type="button"
							className="btn bg-base-300 rounded-xl"
							onClick={() => {
								if (!closeDialogSafely(dialogRef.current)) {
									onClose();
								}
							}}
						>
							Close
						</button>
					</div>
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</dialog>,
		document.body
	);
}
