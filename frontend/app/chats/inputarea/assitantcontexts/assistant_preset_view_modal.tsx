import { useEffect, useMemo, useRef } from 'react';

import { createPortal } from 'react-dom';

import { FiSliders, FiTool, FiX, FiZap } from 'react-icons/fi';

import { ToolStoreChoiceType } from '@/spec/tool';

import { ModalBackdrop } from '@/components/modal_backdrop';

import {
	type AssistantPresetModificationSummary,
	type AssistantPresetPreparedApplication,
	type AssistantPresetRuntimeSnapshot,
	buildAssistantPresetModelComparisonState,
} from '@/chats/inputarea/assitantcontexts/assistant_preset_runtime';
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

function SectionCard(props: { title: string; isModified: boolean; children: React.ReactNode; icon?: React.ReactNode }) {
	return (
		<section className="border-base-300 rounded-2xl border p-4">
			<div className="mb-3 flex items-center gap-2">
				{props.icon}
				<h4 className="text-sm font-semibold">{props.title}</h4>
				<span className={`badge badge-sm ${props.isModified ? 'badge-warning' : 'badge-success'}`}>
					{props.isModified ? 'Modified' : 'In sync'}
				</span>
			</div>
			{props.children}
		</section>
	);
}

type AssistantPresetViewModalProps = {
	isOpen: boolean;
	onClose: () => void;
	appliedPresetApplication: AssistantPresetPreparedApplication | null;
	currentRuntimeSnapshot: AssistantPresetRuntimeSnapshot;
	currentModel: ReturnType<typeof buildAssistantPresetModelComparisonState> extends infer _X ? any : never;
	currentIncludeModelSystemPrompt: boolean;
	currentSelectedPromptKeys: string[];
	promptItems: SystemPromptItem[];
	modificationSummary: AssistantPresetModificationSummary;
};

export function AssistantPresetViewModal({
	isOpen,
	onClose,
	appliedPresetApplication,
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

	if (!isOpen || !appliedPresetApplication || typeof document === 'undefined') {
		return null;
	}

	const appliedModelState = appliedPresetApplication.comparisonState.model;
	const currentModelState = buildAssistantPresetModelComparisonState(
		appliedPresetApplication.preset,
		currentModel,
		currentIncludeModelSystemPrompt
	);

	const promptLabelItems = (keys: string[]) =>
		keys.map(key => {
			const item = promptItemsByKey.get(key);
			return {
				title: item?.displayName || key,
				meta: item ? `${item.bundleDisplayName} • ${item.templateSlug}@${item.templateVersion}` : key,
			};
		});

	const appliedToolItems = [
		...appliedPresetApplication.runtimeSelections.conversationToolChoices,
		...appliedPresetApplication.runtimeSelections.webSearchChoices,
	].map(formatToolLabel);

	const currentToolItems = [
		...currentRuntimeSnapshot.conversationToolChoices,
		...currentRuntimeSnapshot.webSearchChoices,
	].map(choice =>
		formatToolLabel({
			displayName: choice.displayName,
			toolSlug: choice.toolSlug,
			toolVersion: choice.toolVersion,
			bundleID: choice.bundleID,
			toolType: choice.toolType,
			autoExecute: choice.autoExecute,
			userArgSchemaInstance: choice.userArgSchemaInstance,
		})
	);

	const appliedSkillItems = appliedPresetApplication.runtimeSelections.enabledSkillRefs.map(formatSkillLabel);
	const currentSkillItems = currentRuntimeSnapshot.enabledSkillRefs.map(formatSkillLabel);

	const description = appliedPresetApplication.option.description || appliedPresetApplication.preset.description;

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
							<h3 className="text-lg font-bold">{appliedPresetApplication.option.displayName}</h3>
							<div className="mt-1 text-xs opacity-70">
								{appliedPresetApplication.option.bundleDisplayName} • {appliedPresetApplication.preset.slug}@
								{appliedPresetApplication.preset.version}
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
						</div>
					</div>

					<div className="space-y-4">
						{appliedModelState ? (
							<SectionCard
								title="Model and advanced params"
								isModified={modificationSummary.model}
								icon={<FiSliders size={14} />}
							>
								<div className={`grid gap-4 ${modificationSummary.model ? 'lg:grid-cols-2' : 'grid-cols-1'}`}>
									<div>
										<div className="mb-2 text-xs font-semibold opacity-70">Preset-applied values</div>
										<pre className="bg-base-300 overflow-x-auto rounded-xl p-3 text-xs">
											{JSON.stringify(appliedModelState, null, 2)}
										</pre>
									</div>

									{modificationSummary.model ? (
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

						{appliedPresetApplication.comparisonState.instructions ? (
							<SectionCard title="Instruction templates" isModified={modificationSummary.instructions}>
								<div className={`grid gap-4 ${modificationSummary.instructions ? 'lg:grid-cols-2' : 'grid-cols-1'}`}>
									<div>
										<div className="mb-2 text-xs font-semibold opacity-70">Preset-applied selection</div>
										{renderSimpleList(
											promptLabelItems(appliedPresetApplication.comparisonState.instructions),
											'No instruction templates selected.'
										)}
									</div>

									{modificationSummary.instructions ? (
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

						{appliedPresetApplication.comparisonState.tools ? (
							<SectionCard
								title="Tools and web search"
								isModified={modificationSummary.tools}
								icon={<FiTool size={14} />}
							>
								<div className={`grid gap-4 ${modificationSummary.tools ? 'lg:grid-cols-2' : 'grid-cols-1'}`}>
									<div>
										<div className="mb-2 text-xs font-semibold opacity-70">Preset-applied selection</div>
										{renderSimpleList(appliedToolItems, 'No tool selections.')}
									</div>

									{modificationSummary.tools ? (
										<div>
											<div className="mb-2 text-xs font-semibold opacity-70">Current selection</div>
											{renderSimpleList(currentToolItems, 'No tool selections.')}
										</div>
									) : null}
								</div>
							</SectionCard>
						) : null}

						{appliedPresetApplication.comparisonState.skills ? (
							<SectionCard title="Enabled skills" isModified={modificationSummary.skills} icon={<FiZap size={14} />}>
								<div className={`grid gap-4 ${modificationSummary.skills ? 'lg:grid-cols-2' : 'grid-cols-1'}`}>
									<div>
										<div className="mb-2 text-xs font-semibold opacity-70">Preset-applied selection</div>
										{renderSimpleList(appliedSkillItems, 'No enabled skills.')}
									</div>

									{modificationSummary.skills ? (
										<div>
											<div className="mb-2 text-xs font-semibold opacity-70">Current selection</div>
											{renderSimpleList(currentSkillItems, 'No enabled skills.')}
										</div>
									) : null}
								</div>
							</SectionCard>
						) : null}

						{!appliedModelState &&
						!appliedPresetApplication.comparisonState.instructions &&
						!appliedPresetApplication.comparisonState.tools &&
						!appliedPresetApplication.comparisonState.skills ? (
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
