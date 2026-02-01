import { FiChevronDown, FiCode, FiFileText, FiGlobe, FiImage, FiLink, FiPaperclip, FiTool } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore } from '@ariakit/react';

import type { Attachment } from '@/spec/attachment';
import { AttachmentContentBlockMode, AttachmentKind } from '@/spec/attachment';
import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import { type ToolStoreChoice, ToolStoreChoiceType } from '@/spec/tool';

import {
	getAttachmentContentBlockModeLabel,
	getAttachmentContentBlockModePillClasses,
	getAttachmentContentBlockModeTooltip,
} from '@/chats/attachments/attachment_mode_menu';
import { formatToolCallLabel, getPrettyToolName } from '@/chats/tools/tool_editor_utils';

/**
 * Get a path/URL for tooltip display, similar to getUIAttachmentPath
 * but for persisted Conversation attachments.
 */
function getAttachmentPath(att: Attachment): string {
	if (att.kind === AttachmentKind.file && att.fileRef) {
		return att.fileRef.path;
	}
	if (att.kind === AttachmentKind.image && att.imageRef) {
		return att.imageRef.path;
	}
	if (att.kind === AttachmentKind.url && att.urlRef) {
		return att.urlRef.url;
	}
	return '';
}

interface MessageAttachmentInfoChipProps {
	attachment: Attachment;
	fullWidth?: boolean;
}

/**
 * Read‑only chip for a single attachment (file/image/url).
 * No remove, no mode menu — just info.
 */
function MessageAttachmentInfoChip({ attachment, fullWidth = false }: MessageAttachmentInfoChipProps) {
	const { kind, label } = attachment;

	const icon =
		kind === AttachmentKind.image ? (
			<FiImage size={14} />
		) : kind === AttachmentKind.url ? (
			<FiLink size={14} />
		) : (
			<FiFileText size={14} />
		);

	const isLabelTruncated = label.length > 40;
	const truncated = isLabelTruncated ? label.slice(0, 37) + '…' : label;

	const path = getAttachmentPath(attachment);

	const tooltipLines: string[] = [];
	if (isLabelTruncated) tooltipLines.push(label);
	if (path && path !== label) tooltipLines.push(path);

	const title = tooltipLines.length > 0 ? tooltipLines.join('\n') : undefined;

	const mode = attachment.mode ?? AttachmentContentBlockMode.notReadable;
	const modeLabel = getAttachmentContentBlockModeLabel(mode);
	const modeTooltip = getAttachmentContentBlockModeTooltip(mode);

	const containerClasses = [
		'bg-base-200 text-base-content flex items-center gap-2 rounded-2xl px-2 py-0',
		fullWidth ? 'w-full' : 'shrink-0',
	]
		.filter(Boolean)
		.join(' ');

	const labelClasses = fullWidth ? 'min-w-0 flex-1 truncate' : 'max-w-64 truncate';

	return (
		<div className={containerClasses} title={title} data-message-chip="attachment">
			<span className="shrink-0">{icon}</span>
			<span className={labelClasses}>{truncated}</span>
			<span
				className={getAttachmentContentBlockModePillClasses(mode, false)}
				title={modeTooltip}
				data-attachment-mode-pill
			>
				{modeLabel}
			</span>
		</div>
	);
}

interface MessageToolChoiceChipProps {
	tool: ToolStoreChoice;
	fullWidth?: boolean;
	onClick?: () => void;
}

/**
 * Read‑only chip for a tool choice used for this message.
 */
function MessageToolChoiceChip({ tool, fullWidth = false, onClick }: MessageToolChoiceChipProps) {
	const name = tool.displayName || tool.toolSlug;
	const slug = `${tool.bundleID}/${tool.toolSlug}@${tool.toolVersion}`;
	const tooltipLines: string[] = [name, slug];
	if (tool.description) tooltipLines.push(tool.description);

	const containerClasses = [
		'bg-base-200 text-base-content flex items-center justify-between gap-2 rounded-2xl px-2 py-0',
		fullWidth ? 'w-full' : 'shrink-0',
	]
		.filter(Boolean)
		.join(' ');

	const labelClasses = fullWidth ? 'min-w-0 flex-1 truncate' : 'max-w-64 truncate';

	return (
		<div className={containerClasses} title={tooltipLines.join('\n')} data-message-chip="tool-choice" onClick={onClick}>
			<FiTool size={14} />
			<span className={labelClasses}>{name}</span>
			<span className="text-base-content/60 flex gap-1 text-[10px] uppercase">
				Details <FiCode size={12} />
			</span>
		</div>
	);
}

interface MessageToolCallChipProps {
	call: UIToolCall;
	fullWidth?: boolean;
	onClick?: () => void;
}

/**
 * Read‑only chip for an assistant-suggested tool call under the assistant bubble.
 */
function MessageToolCallChip({ call, fullWidth = false, onClick }: MessageToolCallChipProps) {
	const tmpCall: UIToolCall = {
		id: call.id || call.callID,
		callID: call.callID,
		name: call.name,
		arguments: call.arguments,
		webSearchToolCallItems: call.webSearchToolCallItems,
		type: call.type,
		choiceID: call.choiceID,
		status: call.status,
		toolStoreChoice: call.toolStoreChoice,
		errorMessage: call.errorMessage,
	};

	const label = formatToolCallLabel(tmpCall);

	const statusLabel = call.status ? ` (${call.status})` : '';
	const autoLabel = call.toolStoreChoice?.autoExecute ? ' • Auto-execute: enabled' : '';
	const title = `Suggested tool call: ${label}${statusLabel}${autoLabel}`;

	const containerClasses = [
		'bg-base-200 text-base-content flex items-center justify-between gap-2 rounded-2xl px-2 py-0',
		fullWidth ? 'w-full' : 'shrink-0',
	]
		.filter(Boolean)
		.join(' ');

	const labelClasses = fullWidth ? 'min-w-0 flex-1 truncate' : 'max-w-64 truncate';

	return (
		<div className={containerClasses} title={title} data-message-chip="tool-suggested" onClick={onClick}>
			<FiTool size={14} />
			<span className={labelClasses}>{label}</span>
			{call.toolStoreChoice?.autoExecute ? <span className="badge badge-ghost badge-xs">Auto</span> : null}
			<span className="text-base-content/60 flex gap-1 text-[10px] uppercase">
				Details <FiCode size={12} />
			</span>
		</div>
	);
}

interface MessageToolOutputChipProps {
	output: UIToolOutput;
	fullWidth?: boolean;
	onClick?: () => void;
}

/**
 * Read‑only chip for a tool output that was attached to this user turn.
 * History chips are not interactive; the full output was already used
 * when the turn was sent.
 */
function MessageToolOutputChip({ output, fullWidth = false, onClick }: MessageToolOutputChipProps) {
	const prettyName = getPrettyToolName(output.name);
	const label = output.summary || `Result: ${prettyName}`;
	const titleLines = [label, `Tool: ${output.name}`, `Call ID: ${output.callID}`];
	const title = titleLines.join('\n');

	const containerClasses = [
		'bg-base-200 text-base-content flex items-center justify-between gap-2 rounded-2xl px-2 py-0',
		fullWidth ? 'w-full' : 'shrink-0',
	]
		.filter(Boolean)
		.join(' ');

	const labelClasses = fullWidth ? 'min-w-0 flex-1 truncate' : 'max-w-64 truncate';

	return (
		<div className={containerClasses} title={title} data-message-chip="tool-output" onClick={onClick}>
			<FiTool size={14} />
			<span className={labelClasses}>{label}</span>
			<span className="text-base-content/60 flex gap-1 text-[10px] uppercase">
				Details <FiCode size={12} />
			</span>
		</div>
	);
}

interface AttachmentsGroupChipProps {
	attachments: Attachment[];
}

function AttachmentsGroupChip({ attachments }: AttachmentsGroupChipProps) {
	const count = attachments.length;

	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });

	const titleLines = ['Attachments', `${count} item${count === 1 ? '' : 's'} attached`];
	const title = titleLines.join('\n');
	if (count === 0) return null;
	return (
		<div
			className="bg-info/10 text-base-content border-info/50 flex shrink-0 items-center gap-1 rounded-2xl border px-2 py-0"
			title={title}
			data-message-chip="attachments-group"
		>
			<FiPaperclip size={14} />
			<span className="max-w-24 truncate">Attachments</span>
			<span className="text-base-content/70 text-[11px] whitespace-nowrap">{count}</span>

			<MenuButton
				store={menu}
				className="btn btn-ghost btn-xs px-0 py-0 shadow-none"
				aria-label="Show attachments for this message"
				title="Show attachments for this message"
			>
				<FiChevronDown size={14} />
			</MenuButton>

			<Menu
				store={menu}
				gutter={6}
				className="rounded-box bg-base-200 text-base-content border-base-content z-50 max-h-72 min-w-65 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
				autoFocusOnShow
			>
				<div className="text-base-content/70 mb-1 text-[11px] font-semibold">Attachments</div>

				{attachments.map((att, index) => (
					<MenuItem
						key={`${att.kind}:${att.label}:${index}`}
						store={menu}
						hideOnClick={false}
						className="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
					>
						<MessageAttachmentInfoChip attachment={att} fullWidth />
					</MenuItem>
				))}
			</Menu>
		</div>
	);
}

interface ToolChoicesGroupChipProps {
	tools: ToolStoreChoice[];
	onToolChoiceDetails?: (choice: ToolStoreChoice) => void;
}

function ToolChoicesGroupChip({ tools, onToolChoiceDetails }: ToolChoicesGroupChipProps) {
	const count = tools.length;

	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });

	const titleLines = ['Tools', `${count} tool${count === 1 ? '' : 's'} used for this turn`];
	const title = titleLines.join('\n');
	if (count === 0) return null;

	return (
		<div
			className="bg-info/10 text-base-content border-info/50 flex shrink-0 items-center gap-1 rounded-2xl border px-2 py-0"
			title={title}
			data-message-chip="tools-group"
		>
			<FiTool size={14} />
			<span className="max-w-24 truncate">Tools</span>
			<span className="text-base-content/60 whitespace-nowrap"> {count}</span>

			<MenuButton
				store={menu}
				className="btn btn-ghost btn-xs px-0 py-0 shadow-none"
				aria-label="Show tools for this message"
				title="Show tools for this message"
			>
				<FiChevronDown size={14} />
			</MenuButton>

			<Menu
				store={menu}
				gutter={6}
				className="rounded-box bg-base-200 text-base-content border-base-content z-50 max-h-72 min-w-65 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
				autoFocusOnShow
			>
				<div className="text-base-content/70 mb-1 text-[11px] font-semibold">Tools</div>

				{tools.map(tool => (
					<MenuItem
						key={tool.toolID ?? `${tool.bundleID}-${tool.toolSlug}-${tool.toolVersion}`}
						store={menu}
						hideOnClick={false}
						className="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
						onClick={
							onToolChoiceDetails
								? () => {
										onToolChoiceDetails(tool);
									}
								: undefined
						}
					>
						<MessageToolChoiceChip tool={tool} fullWidth />
					</MenuItem>
				))}
			</Menu>
		</div>
	);
}

interface ToolOutputsGroupChipProps {
	outputs: UIToolOutput[];
	onToolOutputDetails?: (output: UIToolOutput) => void;
}

function ToolOutputsGroupChip({ outputs, onToolOutputDetails }: ToolOutputsGroupChipProps) {
	const count = outputs.length;

	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });

	const titleLines = ['Tool outputs', `${count} result${count === 1 ? '' : 's'} used for this turn`];
	const title = titleLines.join('\n');
	if (count === 0) return null;

	return (
		<div
			className="bg-info/10 text-base-content border-info/50 flex shrink-0 items-center gap-1 rounded-2xl border px-2 py-0"
			title={title}
			data-message-chip="tool-outputs-group"
		>
			<FiTool size={14} />
			<span className="max-w-24 truncate">Tool results</span>
			<span className="text-base-content/60 whitespace-nowrap">{count}</span>

			<MenuButton
				store={menu}
				className="btn btn-ghost btn-xs px-0 py-0 shadow-none"
				aria-label="Show tool results for this message"
				title="Show tool results for this message"
			>
				<FiChevronDown size={14} />
			</MenuButton>

			<Menu
				store={menu}
				gutter={6}
				className="rounded-box bg-base-200 text-base-content border-base-content z-50 max-h-72 min-w-65 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
				autoFocusOnShow
			>
				<div className="text-base-content/70 mb-1 text-[11px] font-semibold">Tool results</div>

				{outputs.map(out => (
					<MenuItem
						key={out.id}
						store={menu}
						hideOnClick={false}
						className="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
						onClick={
							onToolOutputDetails
								? () => {
										onToolOutputDetails(out);
									}
								: undefined
						}
					>
						<MessageToolOutputChip output={out} fullWidth />
					</MenuItem>
				))}
			</Menu>
		</div>
	);
}

interface ToolCallsGroupChipProps {
	calls: UIToolCall[];
	onToolCallDetails?: (call: UIToolCall) => void;
}

function ToolCallsGroupChip({ calls, onToolCallDetails }: ToolCallsGroupChipProps) {
	const count = calls.length;

	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });

	const titleLines = ['Suggested tool calls', `${count} suggestion${count === 1 ? '' : 's'} from assistant`];
	const title = titleLines.join('\n');
	if (count === 0) return null;

	return (
		<div
			className="bg-info/10 text-base-content border-info/50 flex shrink-0 items-center gap-1 rounded-2xl border px-2 py-0"
			title={title}
			data-message-chip="tool-calls-group"
		>
			<FiTool size={14} />
			<span className="max-w-24 truncate">Tool calls</span>
			<span className="text-base-content/60 whitespace-nowrap">{count}</span>

			<MenuButton
				store={menu}
				className="btn btn-ghost btn-xs px-0 py-0 shadow-none"
				aria-label="Show suggested tool calls for this message"
				title="Show suggested tool calls for this message"
			>
				<FiChevronDown size={14} />
			</MenuButton>

			<Menu
				store={menu}
				gutter={6}
				className="rounded-box bg-base-200 text-base-content border-base-content z-50 max-h-72 min-w-65 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
				autoFocusOnShow
			>
				<div className="text-base-content/70 mb-1 text-[11px] font-semibold">Suggested tool calls</div>

				{calls.map(call => (
					<MenuItem
						key={call.id || call.callID}
						store={menu}
						hideOnClick={false}
						className="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
						onClick={
							onToolCallDetails
								? () => {
										onToolCallDetails(call);
									}
								: undefined
						}
					>
						<MessageToolCallChip call={call} fullWidth />
					</MenuItem>
				))}
			</Menu>
		</div>
	);
}

interface WebSearchOutputsGroupChipProps {
	outputs: UIToolOutput[];
	onOutputDetails?: (output: UIToolOutput) => void;
}

function WebSearchOutputsGroupChip({ outputs, onOutputDetails }: WebSearchOutputsGroupChipProps) {
	const count = outputs.length;

	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });

	const title = ['Web search results', `${count} result set${count === 1 ? '' : 's'} for this turn`].join('\n');
	if (count === 0) return null;

	return (
		<div
			className="bg-info/10 text-base-content border-info/50 flex shrink-0 items-center gap-1 rounded-2xl border px-2 py-0"
			title={title}
			data-message-chip="websearch-outputs-group"
		>
			<FiGlobe size={14} />
			<span className="max-w-32 truncate">Web results</span>
			<span className="text-xs whitespace-nowrap opacity-80">{count}</span>

			<MenuButton
				store={menu}
				className="btn btn-ghost btn-xs px-0 py-0 shadow-none"
				aria-label="Show web‑search results for this turn"
				title="Show web‑search results for this turn"
			>
				<FiChevronDown size={14} />
			</MenuButton>

			<Menu
				store={menu}
				gutter={6}
				className="rounded-box bg-base-200 text-base-content border-base-content z-50 max-h-72 min-w-65 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
				autoFocusOnShow
			>
				<div className="text-base-content/70 mb-1 text-[11px] font-semibold">Web search results</div>

				{outputs.map(out => (
					<MenuItem
						key={out.id}
						store={menu}
						hideOnClick={false}
						className="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
						onClick={
							onOutputDetails
								? () => {
										onOutputDetails(out);
									}
								: undefined
						}
					>
						<MessageWebSearchOutputChip output={out} fullWidth />
					</MenuItem>
				))}
			</Menu>
		</div>
	);
}

interface MessageWebSearchOutputChipProps {
	output: UIToolOutput;
	fullWidth?: boolean;
	onClick?: () => void;
}

function MessageWebSearchOutputChip({ output, fullWidth = false, onClick }: MessageWebSearchOutputChipProps) {
	const containerClasses = [
		'bg-base-200 text-base-content flex items-center justify-between gap-2 rounded-2xl px-2 py-0',
		fullWidth ? 'w-full' : 'shrink-0',
	].join(' ');

	const labelClasses = fullWidth ? 'min-w-0 flex-1 truncate' : 'max-w-64 truncate';

	const resultCount = output.webSearchToolOutputItems?.length ?? 0;
	const label = resultCount > 0 ? `${resultCount} result${resultCount === 1 ? '' : 's'}` : 'Web search results';

	const title = [`Web search results`, `Tool: ${output.name}`, `Call ID: ${output.callID}`].join('\n');

	return (
		<div className={containerClasses} title={title} data-message-chip="websearch-output" onClick={onClick}>
			<FiGlobe size={14} />
			<span className={labelClasses}>{label}</span>
			<span className="text-base-content/60 flex gap-1 text-[10px] uppercase">
				Details <FiCode size={12} />
			</span>
		</div>
	);
}

interface WebSearchCallsGroupChipProps {
	calls: UIToolCall[];
	onCallDetails?: (call: UIToolCall) => void;
}

function WebSearchCallsGroupChip({ calls, onCallDetails }: WebSearchCallsGroupChipProps) {
	const count = calls.length;

	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });

	const title = ['Web search activity', `${count} web‑search quer${count === 1 ? 'y' : 'ies'} this turn`].join('\n');
	if (count === 0) return null;

	return (
		<div
			className="bg-info/10 text-base-content border-info/50 flex shrink-0 items-center gap-1 rounded-2xl border px-2 py-0"
			title={title}
			data-message-chip="websearch-calls-group"
		>
			<FiGlobe size={14} />
			<span className="max-w-24 truncate">Web search</span>
			<span className="text-xs whitespace-nowrap opacity-80">{count}</span>

			<MenuButton
				store={menu}
				className="btn btn-ghost btn-xs px-0 py-0 shadow-none"
				aria-label="Show web‑search queries for this turn"
				title="Show web‑search queries for this turn"
			>
				<FiChevronDown size={14} />
			</MenuButton>

			<Menu
				store={menu}
				gutter={6}
				className="rounded-box bg-base-200 text-base-content border-base-content z-50 max-h-72 min-w-65 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
				autoFocusOnShow
			>
				<div className="text-base-content/70 mb-1 text-[11px] font-semibold">Web search queries</div>

				{calls.map(call => (
					<MenuItem
						key={call.id || call.callID}
						store={menu}
						hideOnClick={false}
						className="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
						onClick={
							onCallDetails
								? () => {
										onCallDetails(call);
									}
								: undefined
						}
					>
						<MessageWebSearchCallChip call={call} fullWidth />
					</MenuItem>
				))}
			</Menu>
		</div>
	);
}

interface MessageWebSearchCallChipProps {
	call: UIToolCall;
	fullWidth?: boolean;
	onClick?: () => void;
}

function MessageWebSearchCallChip({ call, fullWidth = false, onClick }: MessageWebSearchCallChipProps) {
	const containerClasses = [
		'bg-base-200 text-base-content flex items-center justify-between gap-2 rounded-2xl px-2 py-0',
		fullWidth ? 'w-full' : 'shrink-0',
	].join(' ');

	const labelClasses = fullWidth ? 'min-w-0 flex-1 truncate' : 'max-w-64 truncate';

	// Prefer web-search query if present; fall back to generic label
	const items = call.webSearchToolCallItems ?? [];
	const firstQuery =
		items.find(it => it?.searchItem?.query)?.searchItem?.query ??
		items.find(it => it?.findItem?.pattern)?.findItem?.pattern;

	const fallback = formatToolCallLabel(call);
	const label = firstQuery || fallback;

	const title = `Web search query: ${label}`;

	return (
		<div className={containerClasses} title={title} data-message-chip="websearch-call" onClick={onClick}>
			<FiGlobe size={14} />
			<span className={labelClasses}>{label}</span>
			<span className="text-base-content/60 flex gap-1 text-[10px] uppercase">
				Details <FiCode size={12} />
			</span>
		</div>
	);
}

interface WebSearchChoicesGroupChipProps {
	choices: ToolStoreChoice[];
	onChoiceDetails?: (choice: ToolStoreChoice) => void;
}

function WebSearchChoicesGroupChip({ choices, onChoiceDetails }: WebSearchChoicesGroupChipProps) {
	const count = choices.length;

	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });

	const title = ['Web search configuration', `${count} web‑search tool${count === 1 ? '' : 's'} in this turn`].join(
		'\n'
	);
	if (count === 0) return null;

	return (
		<div
			className="bg-info/10 text-base-content border-info/50 flex shrink-0 items-center gap-1 rounded-2xl border px-2 py-0"
			title={title}
			data-message-chip="websearch-tools-group"
		>
			<FiGlobe size={14} />
			<span className="max-w-24 truncate">Web search</span>
			<span className="text-xs whitespace-nowrap opacity-80">{count}</span>

			<MenuButton
				store={menu}
				className="btn btn-ghost btn-xs px-0 py-0 shadow-none"
				aria-label="Show web‑search configuration for this turn"
				title="Show web‑search configuration for this turn"
			>
				<FiChevronDown size={14} />
			</MenuButton>

			<Menu
				store={menu}
				gutter={6}
				className="rounded-box bg-base-200 text-base-content border-base-content z-50 max-h-72 min-w-65 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
				autoFocusOnShow
			>
				<div className="text-base-content/70 mb-1 text-[11px] font-semibold">Web search configuration</div>

				{choices.map(choice => (
					<MenuItem
						key={choice.toolID ?? `${choice.bundleID}-${choice.toolSlug}-${choice.toolVersion}`}
						store={menu}
						hideOnClick={false}
						className="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
						onClick={
							onChoiceDetails
								? () => {
										onChoiceDetails(choice);
									}
								: undefined
						}
					>
						<MessageWebSearchToolChoiceChip tool={choice} fullWidth />
					</MenuItem>
				))}
			</Menu>
		</div>
	);
}

function MessageWebSearchToolChoiceChip({ tool, fullWidth = false, onClick }: MessageToolChoiceChipProps) {
	const name = tool.displayName || tool.toolSlug;
	const slug = `${tool.bundleID}/${tool.toolSlug}@${tool.toolVersion}`;
	const title = [name, slug, tool.description].filter(Boolean).join('\n');
	const containerClasses = [
		'bg-base-200 text-base-content flex items-center justify-between gap-2 rounded-2xl px-2 py-0',
		fullWidth ? 'w-full' : 'shrink-0',
	].join(' ');
	const labelClasses = fullWidth ? 'min-w-0 flex-1 truncate' : 'max-w-64 truncate';
	return (
		<div className={containerClasses} title={title} data-message-chip="websearch-tool-choice" onClick={onClick}>
			<FiGlobe size={14} />
			<span className={labelClasses}>{name}</span>
			<span className="text-base-content/60 flex gap-1 text-[10px] uppercase">
				Details <FiCode size={12} />
			</span>
		</div>
	);
}

interface MessageAttachmentsBarProps {
	attachments?: Attachment[];
	toolChoices?: ToolStoreChoice[];
	toolCalls?: UIToolCall[];
	toolOutputs?: UIToolOutput[];
	onToolChoiceDetails?: (choice: ToolStoreChoice) => void;
	onToolCallDetails?: (call: UIToolCall) => void;
	onToolOutputDetails?: (output: UIToolOutput) => void;
}

/**
 * Read‑only toolbar under a message bubble:
 * - For user messages: files, tool choices, and tool outputs.
 * - For assistant messages: files and suggested tool calls.
 *
 * Uses compact dropdown chips similar to the composer, but without
 * any remove / edit actions.
 */
export function MessageAttachmentsBar({
	attachments,
	toolChoices,
	toolCalls,
	toolOutputs,
	onToolChoiceDetails,
	onToolCallDetails,
	onToolOutputDetails,
}: MessageAttachmentsBarProps) {
	const choices = toolChoices ?? [];
	const calls = toolCalls ?? [];
	const outputs = toolOutputs ?? [];

	const normalToolChoices = choices.filter(c => c.toolType !== ToolStoreChoiceType.WebSearch);
	const webSearchChoices = choices.filter(c => c.toolType === ToolStoreChoiceType.WebSearch);

	const normalToolCalls = calls.filter(c => c.type !== ToolStoreChoiceType.WebSearch);
	const webSearchCalls = calls.filter(c => c.type === ToolStoreChoiceType.WebSearch);

	const normalToolOutputs = outputs.filter(o => o.type !== ToolStoreChoiceType.WebSearch);
	const webSearchOutputs = outputs.filter(o => o.type === ToolStoreChoiceType.WebSearch);

	const hasAttachments = !!attachments && attachments.length > 0;
	const hasTools = normalToolChoices.length > 0;
	const hasWebSearchTools = webSearchChoices.length > 0;
	const hasToolCalls = normalToolCalls.length > 0;
	const hasWebSearchCalls = webSearchCalls.length > 0;
	const hasToolOutputs = normalToolOutputs.length > 0;
	const hasWebSearchOutputs = webSearchOutputs.length > 0;

	if (
		!hasAttachments &&
		!hasTools &&
		!hasWebSearchTools &&
		!hasToolCalls &&
		!hasWebSearchCalls &&
		!hasToolOutputs &&
		!hasWebSearchOutputs
	) {
		return null;
	}

	return (
		<div className="flex min-h-8 max-w-full min-w-0 items-center gap-1 overflow-x-auto text-xs">
			{hasAttachments && <AttachmentsGroupChip attachments={attachments ?? []} />}

			{/* Regular tools for this turn */}
			{hasTools && <ToolChoicesGroupChip tools={normalToolChoices} onToolChoiceDetails={onToolChoiceDetails} />}

			{/* Web‑search config for this turn */}
			{hasWebSearchTools && (
				<WebSearchChoicesGroupChip choices={webSearchChoices} onChoiceDetails={onToolChoiceDetails} />
			)}

			{/* Tool outputs (non‑web) */}
			{hasToolOutputs && <ToolOutputsGroupChip outputs={normalToolOutputs} onToolOutputDetails={onToolOutputDetails} />}

			{/* Web‑search outputs */}
			{hasWebSearchOutputs && (
				<WebSearchOutputsGroupChip outputs={webSearchOutputs} onOutputDetails={onToolOutputDetails} />
			)}

			{/* Suggested function/custom tool calls */}
			{hasToolCalls && <ToolCallsGroupChip calls={normalToolCalls} onToolCallDetails={onToolCallDetails} />}

			{/* Web‑search calls (already executed by provider) */}
			{hasWebSearchCalls && <WebSearchCallsGroupChip calls={webSearchCalls} onCallDetails={onToolCallDetails} />}
		</div>
	);
}
