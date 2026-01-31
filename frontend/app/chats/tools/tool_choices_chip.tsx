import { FiChevronUp, FiTool, FiX } from 'react-icons/fi';

import { Menu, MenuButton, useMenuStore } from '@ariakit/react';
import type { Path } from 'platejs';
import type { PlateEditor } from 'platejs/react';

import { ToolStoreChoiceType } from '@/spec/tool';

import { dispatchOpenToolArgs } from '@/chats/events/open_attached_toolargs';
import {
	computeToolUserArgsStatus,
	removeToolByKey,
	setToolAutoExecuteByKey,
	toolIdentityKey,
	type ToolSelectionElementNode,
} from '@/chats/tools/tool_editor_utils';
import { ToolMenuRow } from '@/chats/tools/tool_menu_row';

interface ToolChoicesChipProps {
	editor: PlateEditor;
	// Entries from getToolNodesWithPath(editor); typed loosely here.
	toolEntries: Array<[ToolSelectionElementNode, Path]>;
	onToolsChanged?: () => void;
	onShowToolDetails?: (node: ToolSelectionElementNode) => void;
}

/**
 * Aggregated "Tools" chip for attached tool choices.
 * - Shows a count of selected tools.
 * - Opens a dropdown listing each tool with an individual remove button.
 * - Has a "remove all" cross that clears all attached tools.
 */
export function ToolChoicesChip({ editor, toolEntries, onToolsChanged, onShowToolDetails }: ToolChoicesChipProps) {
	// Only show "attached tools" that behave like normal tools in this UI.
	// Web search is controlled separately in the bottom bar.
	const visibleEntries = toolEntries.filter(([node]) => node.toolType !== ToolStoreChoiceType.WebSearch);
	const count = visibleEntries.length;

	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });

	const title = `Tools\n${count} tool${count === 1 ? '' : 's'} attached`;
	const handleToggleAutoExecute = (node: ToolSelectionElementNode) => {
		const key = toolIdentityKey(node.bundleID, node.bundleSlug, node.toolSlug, node.toolVersion);
		if (!key) return;
		setToolAutoExecuteByKey(editor, key, !node.autoExecute);
		onToolsChanged?.();
	};

	const handleRemoveSingle = (node: ToolSelectionElementNode) => {
		const key = toolIdentityKey(node.bundleID, node.bundleSlug, node.toolSlug, node.toolVersion);
		if (!key) return;
		removeToolByKey(editor, key);
		onToolsChanged?.();
	};

	const handleRemoveAll = () => {
		const seen = new Set<string>();

		for (const [node] of visibleEntries) {
			const key = toolIdentityKey(node.bundleID, node.bundleSlug, node.toolSlug, node.toolVersion);
			if (!key || seen.has(key)) continue;
			seen.add(key);
			removeToolByKey(editor, key);
		}

		menu.hide();
		onToolsChanged?.();
	};

	if (count === 0) return null;

	return (
		<div
			className="bg-base-200 text-base-content flex shrink-0 items-center gap-1 rounded-2xl px-2 py-0"
			title={title}
			data-attachment-chip="tools-group"
		>
			<FiTool size={14} />
			<span className="max-w-24 truncate">Tools</span>
			<span className="text-base-content/60 whitespace-nowrap">{count}</span>

			<MenuButton
				store={menu}
				className="btn btn-ghost btn-xs px-0 py-0 shadow-none"
				aria-label="Show selected tools"
				title="Show selected tools"
			>
				<FiChevronUp size={14} />
			</MenuButton>

			{/* Remove all tool choices */}
			<button
				type="button"
				className="btn btn-ghost btn-xs text-error shrink-0 px-0 py-0 shadow-none"
				onClick={handleRemoveAll}
				title="Remove all tools"
				aria-label="Remove all tools"
			>
				<FiX size={14} />
			</button>

			<Menu
				store={menu}
				gutter={6}
				className="rounded-box bg-base-100 text-base-content border-base-300 z-50 max-h-72 min-w-80 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
				autoFocusOnShow
			>
				<div className="text-base-content/70 mb-1 text-[11px] font-semibold">Tools</div>

				{visibleEntries.map(([node]) => {
					const rawDisplay: string | undefined = node.toolSnapshot?.displayName ?? node.toolSlug;
					const display = rawDisplay && rawDisplay.length > 0 ? rawDisplay : 'Tool';
					const slug = `${node.bundleSlug ?? node.bundleID}/${node.toolSlug}@${node.toolVersion}`;
					const schema = node.toolSnapshot?.userArgSchema;
					const status = computeToolUserArgsStatus(schema, node.userArgSchemaInstance);
					const supportsAutoExecute =
						node.toolType === ToolStoreChoiceType.Function || node.toolType === ToolStoreChoiceType.Custom;
					const hasArgs = status?.hasSchema ?? false;
					return (
						<ToolMenuRow
							key={node.selectionID}
							store={menu}
							menuItemClassName="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
							contentClassName="grid grid-cols-12 items-center gap-x-2 px-2 py-1"
							dataAttachmentChip="tool-choice"
							dataSelectionId={node.selectionID}
							title={`Tool choice: ${display} (${slug}@${node.toolVersion})`}
							display={display}
							slug={slug}
							isSelected={true}
							supportsAutoExecute={supportsAutoExecute}
							autoExecute={node.autoExecute}
							onAutoExecuteChange={() => {
								handleToggleAutoExecute(node);
							}}
							argsStatus={status}
							// Keep your existing "edit" icon if you prefer:
							editIcon={<FiChevronUp size={12} />}
							onEditOptions={
								hasArgs
									? () => {
											dispatchOpenToolArgs({ kind: 'attached', selectionID: node.selectionID });
										}
									: undefined
							}
							onShowDetails={
								onShowToolDetails
									? () => {
											onShowToolDetails(node);
										}
									: undefined
							}
							primaryAction={{
								kind: 'remove',
								onClick: () => {
									handleRemoveSingle(node);
								},
								title: 'Remove tool choice',
							}}
						/>
					);
				})}
			</Menu>
		</div>
	);
}
