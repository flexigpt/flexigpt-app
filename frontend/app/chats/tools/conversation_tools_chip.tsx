import { FiCheck, FiChevronUp, FiCode, FiEdit2, FiTool, FiX } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore } from '@ariakit/react';

import { type Tool, type ToolStoreChoice, ToolStoreChoiceType, type UIToolUserArgsStatus } from '@/spec/tool';

import { dispatchOpenToolArgs } from '@/chats/events/open_attached_toolargs';
import { computeToolUserArgsStatus, toolIdentityKey } from '@/chats/tools/tool_editor_utils';

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
				className="rounded-box bg-primary/5 text-base-content border-primary/40 z-50 max-h-72 min-w-70 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
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
					const truncatedDisplay = display.length > 40 ? `${display.slice(0, 37)}…` : display;
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
					const argsLabel =
						!status || !status.hasSchema
							? ''
							: status.requiredKeys.length === 0
								? 'Args: Optional'
								: status.isSatisfied
									? 'Args: Ok'
									: `Args: ${status.missingRequired.length} Missing`;
					const argsClass =
						!status || !status.hasSchema
							? 'text-xs p-0'
							: status.requiredKeys.length === 0
								? 'badge badge-ghost badge-xs text-xs p-0'
								: status.isSatisfied
									? 'badge badge-success badge-xs text-xs p-0'
									: 'badge badge-warning badge-xs text-xs p-0';

					return (
						<MenuItem
							key={key}
							store={menu}
							hideOnClick={false}
							className="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
						>
							<div
								className="grid grid-cols-12 items-center gap-x-2 px-2 py-1"
								title={`Conversation tool: ${display} (${slug}@${toolStoreChoice.toolVersion})`}
								data-attachment-chip="conversation-tool"
							>
								<div className="col-span-8 flex items-center gap-1">
									<FiTool className="justify-start" size={14} />
									<div className="flex-1 justify-start truncate">
										<div className="truncate text-xs font-medium">{truncatedDisplay}</div>
										<div className="text-base-content/70 truncate text-[11px]">{slug}</div>
									</div>

									{/* tick (selected/attached) */}
									<div className="justify-end" title={entry.enabled ? 'Enabled' : 'Disabled'}>
										{entry.enabled ? <FiCheck className="justify-end" size={14} /> : null}
									</div>
								</div>

								{/* Auto-execute column aligned for all tool types */}
								<div className="col-span-2 shrink-0 justify-self-center whitespace-nowrap">
									{supportsAutoExecute ? (
										<label
											className="flex items-center gap-1 text-[11px]"
											title="Automatically run tool calls for this tool"
											onPointerDown={e => {
												e.stopPropagation();
											}}
											onClick={e => {
												e.stopPropagation();
											}}
										>
											<span className="text-base-content/60">Auto</span>
											<input
												type="checkbox"
												className="toggle toggle-xs"
												checked={entry.toolStoreChoice.autoExecute}
												onChange={() => {
													handleToggleAutoExecute(key);
												}}
											/>
										</label>
									) : (
										<span className="text-base-content/40 text-[11px]" title="Auto-exec not applicable">
											—
										</span>
									)}
								</div>

								<div className="col-span-2 flex items-center justify-end gap-1">
									{/* Args status + edit */}
									{hasArgs && <span className={argsClass}>{argsLabel}</span>}
									{hasArgs && (
										<button
											type="button"
											className="btn btn-ghost btn-xs p-0 shadow-none"
											onClick={e => {
												e.preventDefault(); // don’t submit the composer form
												e.stopPropagation(); // don’t trigger any parent click handlers
												dispatchOpenToolArgs({ kind: 'conversation', key: entry.key });
											}}
											title="Edit tool options"
											aria-label="Edit tool options"
										>
											<FiEdit2 size={12} />
										</button>
									)}
									{/* JSON details */}
									{onShowToolDetails && (
										<button
											type="button"
											className="btn btn-ghost btn-xs shrink-0 px-1 py-0 shadow-none"
											onClick={() => {
												onShowToolDetails(entry);
											}}
											title="Show tool details"
											aria-label="Show tool details"
										>
											<FiCode size={12} />
										</button>
									)}

									{/* Remove from conversation tools */}
									<button
										type="button"
										className="btn btn-ghost btn-xs text-error shrink-0 px-1 py-0 shadow-none"
										onClick={() => {
											handleRemoveSingle(key);
										}}
										title="Remove conversation tool"
										aria-label="Remove conversation tool"
									>
										<FiX size={12} />
									</button>
								</div>
							</div>
						</MenuItem>
					);
				})}
			</Menu>
		</div>
	);
}
