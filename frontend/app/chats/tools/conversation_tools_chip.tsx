import { FiChevronUp, FiEdit2, FiTool, FiX } from 'react-icons/fi';

import { Menu, MenuButton, useMenuStore } from '@ariakit/react';

import { type Tool, type ToolStoreChoice, ToolStoreChoiceType, type UIToolUserArgsStatus } from '@/spec/tool';

import { dispatchOpenToolArgs } from '@/chats/events/open_attached_toolargs';
import { computeToolUserArgsStatus, toolIdentityKey } from '@/chats/tools/tool_editor_utils';
import { ToolMenuRow } from '@/chats/tools/tool_menu_row';

export interface ConversationToolStateEntry {
	key: string;
	toolStoreChoice: ToolStoreChoice;
	enabled: boolean;
	/** Optional full tool definition, used for arg schema etc. */
	toolDefinition?: Tool;
	/** Cached status of userArgSchemaInstance vs schema. */
	argStatus?: UIToolUserArgsStatus;
}

/**
 * Initialize UI state from an array of ToolStoreChoice coming from history
 * (e.g. last user message's toolChoices).
 */
export function initConversationToolsStateFromChoices(choices: ToolStoreChoice[]): ConversationToolStateEntry[] {
	const out: ConversationToolStateEntry[] = [];
	const seen = new Set<string>();

	for (const t of choices ?? []) {
		if (t.toolType !== ToolStoreChoiceType.WebSearch) {
			const key = toolIdentityKey(t.bundleID, undefined, t.toolSlug, t.toolVersion);
			if (seen.has(key)) continue;
			seen.add(key);
			out.push({ key, toolStoreChoice: t, enabled: true });
		}
	}

	return out;
}

/**
 * Extract only the ENABLED tools, deduped by identity, for attachment to a message.
 */
export function conversationToolsToChoices(entries: ConversationToolStateEntry[]): ToolStoreChoice[] {
	if (!entries || entries.length === 0) return [];
	const out: ToolStoreChoice[] = [];
	const seen = new Set<string>();

	for (const e of entries) {
		if (!e.enabled) continue;
		const t = e.toolStoreChoice;
		if (t.toolType !== ToolStoreChoiceType.WebSearch) {
			const key = toolIdentityKey(t.bundleID, undefined, t.toolSlug, t.toolVersion);
			if (seen.has(key)) continue;
			seen.add(key);
			out.push(t);
		}
	}

	return out;
}

/**
 * After a send, merge any newly used tools into the UI state.
 * - Preserves existing enabled/disabled flags.
 * - Adds brand-new tools as enabled=true.
 */
export function mergeConversationToolsWithNewChoices(
	prev: ConversationToolStateEntry[],
	newTools: ToolStoreChoice[]
): ConversationToolStateEntry[] {
	if (!newTools || newTools.length === 0) return prev;

	const next = [...prev];
	const indexByKey = new Map<string, number>();
	for (let i = 0; i < next.length; i += 1) {
		indexByKey.set(next[i].key, i);
	}

	for (const t of newTools) {
		if (t.toolType !== ToolStoreChoiceType.WebSearch) {
			const key = toolIdentityKey(t.bundleID, undefined, t.toolSlug, t.toolVersion);
			const existingIdx = indexByKey.get(key);
			if (existingIdx != null) {
				// Refresh metadata but keep enabled flag.
				next[existingIdx] = {
					...next[existingIdx],
					toolStoreChoice: { ...next[existingIdx].toolStoreChoice, ...t },
				};
			} else {
				next.push({ key, toolStoreChoice: t, enabled: true });
			}
		}
	}

	return next;
}

interface ConversationToolsChipProps {
	tools: ConversationToolStateEntry[];
	onChange?: (next: ConversationToolStateEntry[]) => void;
	onShowToolDetails?: (entry: ConversationToolStateEntry) => void;
}

/**
 * Conversation-level tools chip.
 * - First chip in the composer chips row.
 * - Tinted differently (primary-ish) vs per-message Tools chip.
 * - Dropdown:
 *   - per-tool enable/disable toggle
 *   - per-tool remove
 *   - "remove all" in the chip header
 *
 * All state here is UI-only; it controls what gets attached on the next send,
 * but does not rewrite existing messages.
 */
export function ConversationToolsChip({ tools, onChange, onShowToolDetails }: ConversationToolsChipProps) {
	const count = tools.length;
	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });

	const title = `Conversation tools\n${count} tool${count === 1 ? '' : 's'} in this conversation`;

	const handleToggleAutoExecute = (key: string) => {
		if (!onChange) return;
		const next = tools.map(entry =>
			entry.key === key
				? {
						...entry,
						toolStoreChoice: {
							...entry.toolStoreChoice,
							autoExecute: !entry.toolStoreChoice.autoExecute,
						},
					}
				: entry
		);
		onChange(next);
	};
	const handleRemoveSingle = (key: string) => {
		if (!onChange) return;
		const next = tools.filter(entry => entry.key !== key);
		onChange(next);
	};

	const handleRemoveAll = () => {
		if (!onChange) return;
		onChange([]);
		menu.hide();
	};

	if (count === 0) return null;

	return (
		<div
			className="bg-primary/10 text-base-content border-primary/40 flex shrink-0 items-center gap-1 rounded-2xl border px-2 py-0"
			title={title}
			data-attachment-chip="conversation-tools-group"
		>
			<FiTool size={14} />
			<span className="max-w-36 truncate">Conversation tools</span>
			<span className="text-base-content/60 whitespace-nowrap">{count}</span>

			<MenuButton
				store={menu}
				className="btn btn-ghost btn-xs px-0 py-0 shadow-none"
				aria-label="Show conversation tools"
				title="Show conversation tools"
			>
				<FiChevronUp size={14} />
			</MenuButton>

			{/* Remove all conversation tools */}
			<button
				type="button"
				className="btn btn-ghost btn-xs text-error shrink-0 px-0 py-0 shadow-none"
				onClick={handleRemoveAll}
				title="Remove all conversation tools"
				aria-label="Remove all conversation tools"
			>
				<FiX size={14} />
			</button>

			<Menu
				store={menu}
				gutter={6}
				className="rounded-box bg-base-100 text-base-content border-base-300 z-50 max-h-72 min-w-70 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
				autoFocusOnShow
			>
				<div className="text-base-content/70 mb-2 text-[11px] font-semibold">Conversation tools</div>

				{tools.map(entry => {
					const { key, toolStoreChoice } = entry;
					const display =
						(toolStoreChoice.displayName && toolStoreChoice.displayName.length > 0
							? toolStoreChoice.displayName
							: toolStoreChoice.toolSlug) || 'Tool';
					const slug = `${toolStoreChoice.bundleID ?? 'bundle'}/${toolStoreChoice.toolSlug}@${toolStoreChoice.toolVersion}`;

					const supportsAutoExecute =
						toolStoreChoice.toolType === ToolStoreChoiceType.Function ||
						toolStoreChoice.toolType === ToolStoreChoiceType.Custom;

					// If argStatus is precomputed, use it; otherwise compute on the fly if we have definition.
					const status =
						entry.toolDefinition && entry.toolDefinition.userArgSchema
							? computeToolUserArgsStatus(
									entry.toolDefinition.userArgSchema,
									entry.toolStoreChoice.userArgSchemaInstance
								)
							: undefined;
					const hasArgs = status?.hasSchema ?? false;

					return (
						<ToolMenuRow
							key={key}
							store={menu}
							menuItemClassName="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
							contentClassName="grid grid-cols-12 items-center gap-x-2 px-2 py-1"
							dataAttachmentChip="conversation-tool"
							title={`Conversation tool: ${display} (${slug}@${toolStoreChoice.toolVersion})`}
							display={display}
							slug={slug}
							isSelected={entry.enabled}
							supportsAutoExecute={supportsAutoExecute}
							autoExecute={entry.toolStoreChoice.autoExecute}
							onAutoExecuteChange={() => {
								handleToggleAutoExecute(key);
							}}
							argsStatus={status}
							editIcon={<FiEdit2 size={12} />}
							onEditOptions={
								hasArgs
									? () => {
											dispatchOpenToolArgs({ kind: 'conversation', key: entry.key });
										}
									: undefined
							}
							onShowDetails={
								onShowToolDetails
									? () => {
											onShowToolDetails(entry);
										}
									: undefined
							}
							primaryAction={{
								kind: 'remove',
								onClick: () => {
									handleRemoveSingle(key);
								},
								title: 'Remove conversation tool',
							}}
						/>
					);
				})}
			</Menu>
		</div>
	);
}
