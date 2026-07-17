import { useMemo } from 'react';

import { FiServer, FiSliders, FiTool, FiZap } from 'react-icons/fi';

import type { AssistantPreset } from '@/spec/assistantpreset';
import type { MCPConversationContext } from '@/spec/mcp';
import { MCPToolExposure } from '@/spec/mcp';
import type { UIChatOption } from '@/spec/modelpreset';
import type { SkillSelection } from '@/spec/skill';
import { ToolStoreChoiceType } from '@/spec/tool';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

import type {
	AssistantPresetModificationSummary,
	AssistantPresetOptionItem,
	AssistantPresetPreparedApplication,
	AssistantPresetRuntimeSnapshot,
} from '@/chats/composer/assistantpresets/assistant_preset_runtime';
import {
	buildAssistantPresetModelComparisonState,
	normalizeAssistantPresetMCPContext,
} from '@/chats/composer/assistantpresets/assistant_preset_runtime';
import type { SystemInstructionSource } from '@/chats/composer/skills/prompt_utils';

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

function getMCPToolExposureLabel(exposure: MCPToolExposure | string | undefined, selectedToolCount: number): string {
	switch (exposure) {
		case MCPToolExposure.MCPToolExposureAll:
			return selectedToolCount > 0 ? `All tools (${selectedToolCount})` : 'All tools';
		case MCPToolExposure.MCPToolExposureSelected:
			return `${selectedToolCount} selected tool${selectedToolCount === 1 ? '' : 's'}`;
		case MCPToolExposure.MCPToolExposureNone:
			return 'No tools';
		default:
			return exposure ? exposure : 'Tools not specified';
	}
}

function formatMCPArgumentSummary(values?: Record<string, string>): string | undefined {
	const entries = Object.entries(values ?? {})
		.filter(([key, value]) => key.trim().length > 0 && value.trim().length > 0)
		.toSorted(([a], [b]) => a.localeCompare(b));

	if (entries.length === 0) {
		return undefined;
	}

	const body = entries
		.map(([key, value]) => {
			const clipped = value.length > 32 ? `${value.slice(0, 29)}...` : value;
			return `${key}=${clipped}`;
		})
		.join(', ');

	return `args: ${body}`;
}

function formatMCPContextLabelItems(context?: MCPConversationContext): Array<{ title: string; meta?: string }> {
	const normalized = normalizeAssistantPresetMCPContext(context);
	if (!normalized) {
		return [];
	}

	const items: Array<{ title: string; meta?: string }> = [];

	for (const server of normalized.servers ?? []) {
		const selectedToolCount = server.selectedTools?.length ?? 0;
		const metaParts = [
			server.bundleID,
			getMCPToolExposureLabel(server.toolExposure, selectedToolCount),
			server.includeServerInstructions ? 'server instructions' : undefined,
			server.snapshotDigest ? `snapshot ${server.snapshotDigest}` : undefined,
		].filter(Boolean);

		items.push({
			title: `Server: ${server.serverID}`,
			meta: metaParts.join(' • '),
		});

		for (const tool of server.selectedTools ?? []) {
			const toolMetaParts = [
				`${tool.bundleID}/${tool.serverID}`,
				tool.providerToolName ? `provider ${tool.providerToolName}` : undefined,
				tool.executionMode ? `execution ${tool.executionMode}` : undefined,
				tool.approvalRule ? `approval ${tool.approvalRule}` : undefined,
				tool.digest ? `digest ${tool.digest}` : undefined,
				tool.appResourceUri ? `app ${tool.appResourceUri}` : undefined,
			].filter(Boolean);

			items.push({
				title: `Tool: ${tool.toolName}`,
				meta: toolMetaParts.join(' • '),
			});
		}
	}

	for (const resource of normalized.resources ?? []) {
		items.push({
			title: `Resource: ${resource.uri}`,
			meta: `${resource.bundleID}/${resource.serverID}${resource.digest ? ` • digest ${resource.digest}` : ''}`,
		});
	}

	for (const template of normalized.resourceTemplates ?? []) {
		items.push({
			title: `Resource template: ${template.uriTemplate}`,
			meta: [template.bundleID + '/' + template.serverID, formatMCPArgumentSummary(template.argumentValues)]
				.filter(Boolean)
				.join(' • '),
		});
	}

	for (const prompt of normalized.prompts ?? []) {
		items.push({
			title: `Prompt: ${prompt.promptName}`,
			meta: [prompt.bundleID + '/' + prompt.serverID, formatMCPArgumentSummary(prompt.argumentValues)]
				.filter(Boolean)
				.join(' • '),
		});
	}

	return items;
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

	if (selection.toolChoicePatch?.autoExecute) {
		metaParts.push('auto');
	}
	if (selection.toolChoicePatch?.userArgSchemaInstance?.trim()) {
		metaParts.push('args');
	}

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

interface AssistantPresetViewModalProps {
	isOpen: boolean;
	onClose: () => void;
	viewedPreset: AssistantPresetOptionItem | null;
	viewedPresetApplication: AssistantPresetPreparedApplication | null;
	isActivePresetView: boolean;

	currentRuntimeSnapshot: AssistantPresetRuntimeSnapshot;
	currentModel: UIChatOption;
	currentIncludeModelSystemPrompt: boolean;
	currentSelectedInstructionSourceKeys: string[];
	instructionSources: SystemInstructionSource[];
	modificationSummary: AssistantPresetModificationSummary;
}

function AssistantPresetViewModalContent({
	isOpen,
	viewedPreset,
	viewedPresetApplication,
	isActivePresetView,
	currentRuntimeSnapshot,
	currentModel,
	currentIncludeModelSystemPrompt,
	currentSelectedInstructionSourceKeys,
	instructionSources,
	modificationSummary,
}: AssistantPresetViewModalProps) {
	const { requestClose } = useModalDialogController();

	const instructionSourcesByKey = useMemo(() => {
		return new Map(
			[...instructionSources, ...(viewedPresetApplication?.preparedInstructionSources ?? [])].map(item => [
				item.identityKey,
				item,
			])
		);
	}, [instructionSources, viewedPresetApplication?.preparedInstructionSources]);

	if (!isOpen || !viewedPreset) {
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
			const item = instructionSourcesByKey.get(key);
			return {
				title: item?.displayName || key,
				meta: item ? `${item.bundleDisplayName} • ${item.sourceSlug}` : key,
			};
		});

	const appliedInstructionKeys = preparedApplication?.comparisonState.instructions ?? [];
	const fallbackAppliedInstructionItems = (viewedPreset.preset.startingSkillSelections ?? [])
		.filter(selection => selection.useAsInstructions)
		.map(selection => formatSkillSelectionLabel(selection));
	const appliedToolItems = preparedApplication
		? [
				...preparedApplication.runtimeSelections.conversationToolChoices,
				...preparedApplication.runtimeSelections.webSearchChoices,
			].map(t => formatToolLabel(t))
		: (viewedPreset.preset.startingToolSelections ?? []).map(p => formatPresetToolSelectionLabel(p));

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

	const appliedSkillItems = (viewedPreset.preset.startingSkillSelections ?? [])
		.filter(selection => !selection.useAsInstructions)
		.map(selection => formatSkillSelectionLabel(selection));
	const currentSkillItems = shouldShowActiveComparison
		? currentRuntimeSnapshot.enabledSkillRefs.map(formatSkillLabel)
		: [];

	const appliedMCPContext =
		preparedApplication?.comparisonState.mcp ??
		normalizeAssistantPresetMCPContext(viewedPreset.preset.startingMCPContext);
	const appliedMCPItems = formatMCPContextLabelItems(appliedMCPContext);
	const currentMCPItems = shouldShowActiveComparison
		? formatMCPContextLabelItems(currentRuntimeSnapshot.mcpContext)
		: [];

	const description = viewedPreset.description || viewedPreset.preset.description;
	const presetStartingText = viewedPreset.preset.startingText ?? '';
	const hasPresetStartingText = presetStartingText.trim().length > 0;
	const showCurrentModel = shouldShowActiveComparison && modificationSummary.model;
	const showCurrentInstructions = shouldShowActiveComparison && modificationSummary.instructions;
	const showCurrentTools = shouldShowActiveComparison && modificationSummary.tools;
	const showCurrentSkills = shouldShowActiveComparison && modificationSummary.skills;
	const showCurrentMCP = shouldShowActiveComparison && modificationSummary.mcp;

	return (
		<>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-4xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<ModalHeader
						title={viewedPreset.displayName}
						description={`${viewedPreset.bundleDisplayName} • ${viewedPreset.preset.slug}@${viewedPreset.preset.version}`}
						onClose={() => {
							requestClose();
						}}
					/>

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
						{hasPresetStartingText ? (
							<SectionCard title="Starting text">
								<div className="mb-2 text-xs font-semibold opacity-70">
									Inserted only when the composer text is empty.
								</div>
								<pre className="bg-base-300 overflow-x-auto rounded-xl p-3 text-xs whitespace-pre-wrap">
									{presetStartingText}
								</pre>
							</SectionCard>
						) : null}

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

						{appliedInstructionKeys.length > 0 || fallbackAppliedInstructionItems.length > 0 ? (
							<SectionCard
								title="System instruction skills"
								showSyncState={shouldShowActiveComparison}
								isModified={modificationSummary.instructions}
							>
								<div className={`grid gap-4 ${showCurrentInstructions ? 'lg:grid-cols-2' : 'grid-cols-1'}`}>
									<div>
										<div className="mb-2 text-xs font-semibold opacity-70">
											{shouldShowActiveComparison ? 'Preset-applied selection' : 'Preset selection'}
										</div>
										{renderSimpleList(
											appliedInstructionKeys.length > 0
												? promptLabelItems(appliedInstructionKeys)
												: fallbackAppliedInstructionItems,
											'No instruction skills selected.'
										)}
									</div>

									{showCurrentInstructions ? (
										<div>
											<div className="mb-2 text-xs font-semibold opacity-70">Current selection</div>
											{renderSimpleList(
												promptLabelItems(currentSelectedInstructionSourceKeys),
												'No instruction skill prompts selected.'
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

						{appliedMCPItems.length > 0 ? (
							<SectionCard
								title="MCP context"
								showSyncState={shouldShowActiveComparison}
								isModified={modificationSummary.mcp}
								icon={<FiServer size={14} />}
							>
								<div className={`grid gap-4 ${showCurrentMCP ? 'lg:grid-cols-2' : 'grid-cols-1'}`}>
									<div>
										<div className="mb-2 text-xs font-semibold opacity-70">
											{shouldShowActiveComparison ? 'Preset-applied MCP context' : 'Preset MCP context'}
										</div>
										{renderSimpleList(appliedMCPItems, 'No MCP context selected.')}
									</div>

									{showCurrentMCP ? (
										<div>
											<div className="mb-2 text-xs font-semibold opacity-70">Current MCP context</div>
											{renderSimpleList(currentMCPItems, 'No MCP context selected.')}
										</div>
									) : null}
								</div>
							</SectionCard>
						) : null}

						{!hasPresetStartingText &&
						!appliedModelState &&
						appliedInstructionKeys.length === 0 &&
						fallbackAppliedInstructionItems.length === 0 &&
						appliedToolItems.length === 0 &&
						appliedSkillItems.length === 0 &&
						appliedMCPItems.length === 0 ? (
							<div className="rounded-2xl border p-4 text-sm opacity-70">
								This preset only carries display metadata right now; it does not manage any current authoring sections.
							</div>
						) : null}
					</div>

					<ModalActions className="-mx-6 mt-6 -mb-6">
						<button
							type="button"
							className="btn bg-base-300 rounded-xl"
							onClick={() => {
								requestClose();
							}}
						>
							Close
						</button>
					</ModalActions>
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</>
	);
}

export function AssistantPresetViewModal(props: AssistantPresetViewModalProps) {
	if (!props.isOpen || !props.viewedPreset) {
		return null;
	}

	return (
		<ModalDialog isOpen={props.isOpen} onClose={props.onClose}>
			<AssistantPresetViewModalContent {...props} />
		</ModalDialog>
	);
}
