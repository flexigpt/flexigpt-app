import type { ReactNode } from 'react';
import { memo } from 'react';
import {
	FiAlertTriangle,
	FiChevronRight,
	FiCode,
	FiFileText,
	FiGlobe,
	FiImage,
	FiLink,
	FiPaperclip,
	FiTerminal,
	FiTool,
} from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import type { Attachment } from '@/spec/attachment';
import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import type { SessionOperationResolutionSummary } from '@/spec/session_base';
import type { MCPAppContextUpdate } from '@/spec/session_mcp';
import type { ToolSelectionIssue, ToolStoreChoice } from '@/spec/tool';
import { AttachmentContentBlockMode, AttachmentKind } from '@/spec/attachment';
import { MCPExecutionMode } from '@/spec/mcp';
import { ToolStoreChoiceType } from '@/spec/tool';

import { getPrettyToolName } from '@/apis/tool_management';

import type { MessageBarChipTone } from '@/chats/messages/message_bar_chip_utils';
import { getAttachmentDisplayLabel } from '@/chats/composer/attachments/attachment_editor_utils';
import {
	getAttachmentContentBlockModeLabel,
	getAttachmentContentBlockModeTooltip,
} from '@/chats/composer/attachments/attachment_mode_menu_utils';
import { MCPMessageContextChip } from '@/chats/messages/mcp_message_context_chip';
import { getMessageBarChipClassName } from '@/chats/messages/message_bar_chip_utils';
import {
	MCPAppContextResolutionChip,
	MessageSkillsResolutionChip,
	MessageWorkspaceResolutionChip,
} from '@/chats/messages/message_resolution_chips';

/**
 * Get a path/URL for tooltip display, similar to getUIAttachmentPath
 * but for persisted Conversation attachments.
 */
function getAttachmentPath(att: Attachment): string {
	if (att.kind === AttachmentKind.file && att.fileRef) {
		return att.fileRef.origPath || att.fileRef.path || att.fileRef.name || '';
	}
	if (att.kind === AttachmentKind.image && att.imageRef) {
		return att.imageRef.origPath || att.imageRef.path || att.imageRef.name || '';
	}
	if (att.kind === AttachmentKind.url && att.urlRef) {
		return att.urlRef.origNormalized || att.urlRef.normalized || att.urlRef.url || '';
	}
	return '';
}

interface MessageBarChipProps {
	icon: ReactNode;
	label: ReactNode;
	title?: string;
	dataMessageChip: string;
	fullWidth?: boolean;
	onClick?: () => void;
	trailing?: ReactNode;
	children?: ReactNode;
	tone?: MessageBarChipTone;
	maxLabelWidthClass?: string;
}

/** Shared visual and interaction shell for every chip in the message attachments bar. */
function MessageBarChip({
	icon,
	label,
	title,
	dataMessageChip,
	fullWidth = false,
	onClick,
	trailing,
	children,
	tone = 'default',
	maxLabelWidthClass = 'max-w-44',
}: MessageBarChipProps) {
	const className = getMessageBarChipClassName(tone, fullWidth, Boolean(onClick));
	const content = (
		<>
			<span className="shrink-0">{icon}</span>
			<span className={fullWidth ? 'min-w-0 flex-1 truncate' : `${maxLabelWidthClass} truncate`}>{label}</span>
			{trailing}
			{children}
		</>
	);

	if (onClick) {
		return (
			<button type="button" className={className} title={title} data-message-chip={dataMessageChip} onClick={onClick}>
				{content}
			</button>
		);
	}

	return (
		<div className={className} title={title} data-message-chip={dataMessageChip}>
			{content}
		</div>
	);
}

interface MessageBarGroupChipProps<T> {
	items: T[];
	icon: ReactNode;
	label: string;
	menuLabel: string;
	title: string;
	ariaLabel: string;
	dataMessageChip: string;
	itemKey: (item: T, index: number) => string;
	renderItem: (item: T, options: { fullWidth: boolean; onClick?: () => void }) => ReactNode;
	onItemDetails?: (item: T) => void;
	maxLabelWidthClass?: string;
	countClassName?: string;
}

/**
 * Shared one-or-many behavior for attachment-bar groups. A single item
 * renders directly; multiple items use the same lazy Ariakit menu.
 */
function MessageBarGroupChip<T>({
	items,
	icon,
	label,
	menuLabel,
	title,
	ariaLabel,
	dataMessageChip,
	itemKey,
	renderItem,
	onItemDetails,
	maxLabelWidthClass = 'max-w-24',
	countClassName = 'text-base-content/60 whitespace-nowrap',
}: MessageBarGroupChipProps<T>) {
	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });
	const open = useStoreState(menu, 'open');
	const count = items.length;

	const getItemClick = (item: T) =>
		onItemDetails
			? () => {
					onItemDetails(item);
				}
			: undefined;

	if (count === 0) {
		return null;
	}

	if (count === 1) {
		const item = items[0];
		return renderItem(item, { fullWidth: false, onClick: getItemClick(item) });
	}

	return (
		<div className="shrink-0">
			<MenuButton
				store={menu}
				className={getMessageBarChipClassName('info', false, true)}
				title={title}
				data-message-chip={dataMessageChip}
				aria-label={ariaLabel}
			>
				<span className="shrink-0">{icon}</span>
				<span className={`${maxLabelWidthClass} truncate`}>{label}</span>
				<span className={countClassName}>{count}</span>
				<FiChevronRight
					className={`shrink-0 transition-transform ${open ? 'rotate-90' : ''}`}
					aria-hidden="true"
					size={14}
				/>
			</MenuButton>

			{open ? (
				<Menu
					store={menu}
					gutter={8}
					overflowPadding={8}
					className="rounded-box bg-base-200 text-base-content border-base-content z-50 max-h-72 max-w-lg min-w-60 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
					autoFocusOnShow
				>
					<div className="text-base-content/70 mb-1 text-xs font-semibold">{menuLabel}</div>

					{items.map((item, index) => (
						<MenuItem
							key={itemKey(item, index)}
							store={menu}
							hideOnClick={Boolean(onItemDetails)}
							className="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
							onClick={getItemClick(item)}
						>
							{renderItem(item, { fullWidth: true })}
						</MenuItem>
					))}
				</Menu>
			) : null}
		</div>
	);
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
	const { kind } = attachment;
	const label = getAttachmentDisplayLabel(attachment);
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
	if (isLabelTruncated) {
		tooltipLines.push(label);
	}
	if (path && path !== label) {
		tooltipLines.push(path);
	}

	const title = tooltipLines.length > 0 ? tooltipLines.join('\n') : undefined;

	const mode = attachment.mode ?? AttachmentContentBlockMode.notReadable;
	const modeLabel = getAttachmentContentBlockModeLabel(mode);
	const modeTooltip = getAttachmentContentBlockModeTooltip(mode);

	return (
		<MessageBarChip
			icon={icon}
			label={truncated}
			title={title}
			dataMessageChip="attachment"
			fullWidth={fullWidth}
			trailing={
				<span className="badge badge-ghost badge-xs" title={modeTooltip} data-attachment-mode-pill>
					{modeLabel}
				</span>
			}
		/>
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
	const name = tool.displayName || tool.target.name;
	const slug = `${tool.pluginName ? `${tool.pluginName}/` : ''}${tool.target.name}${tool.toolVersion ? `@${tool.toolVersion}` : ''}`;
	const tooltipLines: string[] = [name, slug];
	if (tool.description) {
		tooltipLines.push(tool.description);
	}

	return (
		<MessageBarChip
			icon={<FiTool size={14} />}
			label={name}
			title={tooltipLines.join('\n')}
			dataMessageChip="tool-choice"
			fullWidth={fullWidth}
			onClick={onClick}
			trailing={<FiCode className="text-base-content/60" title="Details" size={14} />}
		/>
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
	const label = call.displayName?.trim() || 'Tool';

	const statusLabel = call.status ? ` (${call.status})` : '';
	const isAutoExecute =
		Boolean(call.automatic) ||
		Boolean(call.toolStoreChoice?.autoExecute) ||
		call.mcpToolSelection?.executionMode === MCPExecutionMode.Auto;
	const autoLabel = isAutoExecute ? ' • Auto-execute: enabled' : '';
	const title = `Suggested tool call: ${label}${statusLabel}${autoLabel}`;

	return (
		<MessageBarChip
			icon={<FiTerminal size={14} />}
			label={label}
			title={title}
			dataMessageChip="tool-suggested"
			fullWidth={fullWidth}
			onClick={onClick}
			trailing={
				<>
					{isAutoExecute ? <span className="badge badge-ghost badge-xs">Auto</span> : null}
					<FiCode className="text-base-content/60" title="Details" size={14} />
				</>
			}
		/>
	);
}

interface MessageToolOutputChipProps {
	output: UIToolOutput;
	fullWidth?: boolean;
	onClick?: () => void;
}

/**
 * Read-only chip for a tool result shown with the assistant's tool activity.
 * The details action opens the recorded output.
 */
function MessageToolOutputChip({ output, fullWidth = false, onClick }: MessageToolOutputChipProps) {
	const displayName = output.displayName?.trim() || 'Tool';
	const label = output.summary || `Result: ${displayName}`;
	const titleLines = [label, `Tool: ${displayName}`];

	const title = titleLines.join('\n');

	return (
		<MessageBarChip
			icon={<FiTool size={14} />}
			label={label}
			title={title}
			dataMessageChip="tool-output"
			fullWidth={fullWidth}
			onClick={onClick}
			trailing={<FiCode className="text-base-content/60" title="Details" size={14} />}
		/>
	);
}

interface MessageWebSearchOutputChipProps {
	output: UIToolOutput;
	fullWidth?: boolean;
	onClick?: () => void;
}

function MessageWebSearchOutputChip({ output, fullWidth = false, onClick }: MessageWebSearchOutputChipProps) {
	const resultCount = output.webSearchToolOutputItems?.length ?? 0;
	const label = resultCount > 0 ? `${resultCount} result${resultCount === 1 ? '' : 's'}` : 'Web search results';

	const title = 'Web search results';

	return (
		<MessageBarChip
			icon={<FiGlobe size={14} />}
			label={label}
			title={title}
			dataMessageChip="websearch-output"
			fullWidth={fullWidth}
			onClick={onClick}
			trailing={<FiCode className="text-base-content/60" title="Details" size={14} />}
		/>
	);
}

interface MessageWebSearchCallChipProps {
	call: UIToolCall;
	fullWidth?: boolean;
	onClick?: () => void;
}

function MessageWebSearchCallChip({ call, fullWidth = false, onClick }: MessageWebSearchCallChipProps) {
	// Prefer web-search query if present; fall back to generic label
	const items = call.webSearchToolCallItems ?? [];
	const firstQuery =
		items.find(it => it?.searchItem?.query)?.searchItem?.query ??
		items.find(it => it?.findItem?.pattern)?.findItem?.pattern;

	const fallback = getPrettyToolName(call.name);
	const label = firstQuery || fallback;

	const title = `Web search query: ${label}`;

	return (
		<MessageBarChip
			icon={<FiGlobe size={14} />}
			label={label}
			title={title}
			dataMessageChip="websearch-call"
			fullWidth={fullWidth}
			onClick={onClick}
			trailing={<FiCode className="text-base-content/60" title="Details" size={14} />}
		/>
	);
}

function MessageWebSearchToolChoiceChip({ tool, fullWidth = false, onClick }: MessageToolChoiceChipProps) {
	const name = tool.displayName || tool.target.name;
	const slug = `${tool.pluginName ? `${tool.pluginName}/` : ''}${tool.target.name}${tool.toolVersion ? `@${tool.toolVersion}` : ''}`;
	const title = [name, slug, tool.description].filter(Boolean).join('\n');

	return (
		<MessageBarChip
			icon={<FiGlobe size={14} />}
			label={name}
			title={title}
			dataMessageChip="websearch-tool-choice"
			fullWidth={fullWidth}
			onClick={onClick}
			trailing={<FiCode className="text-base-content/60" title="Details" size={14} />}
		/>
	);
}

interface AttachmentsGroupChipProps {
	attachments: Attachment[];
}

function AttachmentsGroupChip({ attachments }: AttachmentsGroupChipProps) {
	const count = attachments.length;

	return (
		<MessageBarGroupChip
			items={attachments}
			icon={<FiPaperclip size={14} />}
			label="Attachments"
			menuLabel="Attachments"
			title={['Attachments', `${count} item${count === 1 ? '' : 's'} attached`].join('\n')}
			ariaLabel="Show attachments for this message"
			dataMessageChip="attachments-group"
			itemKey={(attachment, index) => `${attachment.kind}:${attachment.label}:${index}`}
			renderItem={(attachment, options) => (
				<MessageAttachmentInfoChip attachment={attachment} fullWidth={options.fullWidth} />
			)}
			countClassName="text-base-content/70 text-xs whitespace-nowrap"
		/>
	);
}

interface ToolChoicesGroupChipProps {
	tools: ToolStoreChoice[];
	onToolChoiceDetails?: (choice: ToolStoreChoice) => void;
}

function ToolChoicesGroupChip({ tools, onToolChoiceDetails }: ToolChoicesGroupChipProps) {
	const count = tools.length;

	return (
		<MessageBarGroupChip
			items={tools}
			icon={<FiTool size={14} />}
			label="Tools"
			menuLabel="Tools"
			title={['Tools', `${count} tool${count === 1 ? '' : 's'} used for this turn`].join('\n')}
			ariaLabel="Show tools for this message"
			dataMessageChip="tools-group"
			itemKey={tool => tool.choiceID}
			renderItem={(tool, options) => (
				<MessageToolChoiceChip tool={tool} fullWidth={options.fullWidth} onClick={options.onClick} />
			)}
			onItemDetails={onToolChoiceDetails}
		/>
	);
}

interface ToolOutputsGroupChipProps {
	outputs: UIToolOutput[];
	onToolOutputDetails?: (output: UIToolOutput) => void;
}

function ToolOutputsGroupChip({ outputs, onToolOutputDetails }: ToolOutputsGroupChipProps) {
	const count = outputs.length;

	return (
		<MessageBarGroupChip
			items={outputs}
			icon={<FiTool size={14} />}
			label="Tool results"
			menuLabel="Tool results"
			title={['Tool outputs', `${count} result${count === 1 ? '' : 's'} used for this turn`].join('\n')}
			ariaLabel="Show tool results for this message"
			dataMessageChip="tool-outputs-group"
			itemKey={output => output.id}
			renderItem={(output, options) => (
				<MessageToolOutputChip output={output} fullWidth={options.fullWidth} onClick={options.onClick} />
			)}
			onItemDetails={onToolOutputDetails}
		/>
	);
}

interface ToolCallsGroupChipProps {
	calls: UIToolCall[];
	onToolCallDetails?: (call: UIToolCall) => void;
}

function ToolCallsGroupChip({ calls, onToolCallDetails }: ToolCallsGroupChipProps) {
	const count = calls.length;

	return (
		<MessageBarGroupChip
			items={calls}
			icon={<FiTerminal size={14} />}
			label="Tool calls"
			menuLabel="Suggested tool calls"
			title={['Suggested tool calls', `${count} suggestion${count === 1 ? '' : 's'} from assistant`].join('\n')}
			ariaLabel="Show suggested tool calls for this message"
			dataMessageChip="tool-calls-group"
			itemKey={call => call.id || call.callID}
			renderItem={(call, options) => (
				<MessageToolCallChip call={call} fullWidth={options.fullWidth} onClick={options.onClick} />
			)}
			onItemDetails={onToolCallDetails}
		/>
	);
}

interface WebSearchOutputsGroupChipProps {
	outputs: UIToolOutput[];
	onOutputDetails?: (output: UIToolOutput) => void;
}

function WebSearchOutputsGroupChip({ outputs, onOutputDetails }: WebSearchOutputsGroupChipProps) {
	const count = outputs.length;

	return (
		<MessageBarGroupChip
			items={outputs}
			icon={<FiGlobe size={14} />}
			label="Web results"
			menuLabel="Web search results"
			title={['Web search results', `${count} result set${count === 1 ? '' : 's'} for this turn`].join('\n')}
			ariaLabel="Show web-search results for this turn"
			dataMessageChip="websearch-outputs-group"
			itemKey={output => output.id}
			renderItem={(output, options) => (
				<MessageWebSearchOutputChip output={output} fullWidth={options.fullWidth} onClick={options.onClick} />
			)}
			onItemDetails={onOutputDetails}
			maxLabelWidthClass="max-w-32"
			countClassName="text-xs whitespace-nowrap opacity-80"
		/>
	);
}

interface WebSearchCallsGroupChipProps {
	calls: UIToolCall[];
	onCallDetails?: (call: UIToolCall) => void;
}

function WebSearchCallsGroupChip({ calls, onCallDetails }: WebSearchCallsGroupChipProps) {
	const count = calls.length;

	return (
		<MessageBarGroupChip
			items={calls}
			icon={<FiGlobe size={14} />}
			label="Web search"
			menuLabel="Web search queries"
			title={['Web search activity', `${count} web-search quer${count === 1 ? 'y' : 'ies'} this turn`].join('\n')}
			ariaLabel="Show web-search queries for this turn"
			dataMessageChip="websearch-calls-group"
			itemKey={call => call.id || call.callID}
			renderItem={(call, options) => (
				<MessageWebSearchCallChip call={call} fullWidth={options.fullWidth} onClick={options.onClick} />
			)}
			onItemDetails={onCallDetails}
			countClassName="text-xs whitespace-nowrap opacity-80"
		/>
	);
}

interface WebSearchChoicesGroupChipProps {
	choices: ToolStoreChoice[];
	onChoiceDetails?: (choice: ToolStoreChoice) => void;
}

function WebSearchChoicesGroupChip({ choices, onChoiceDetails }: WebSearchChoicesGroupChipProps) {
	const count = choices.length;

	return (
		<MessageBarGroupChip
			items={choices}
			icon={<FiGlobe size={14} />}
			label="Web search"
			menuLabel="Web search configuration"
			title={['Web search configuration', `${count} web-search tool${count === 1 ? '' : 's'} in this turn`].join('\n')}
			ariaLabel="Show web-search configuration for this turn"
			dataMessageChip="websearch-tools-group"
			itemKey={choice => choice.choiceID}
			renderItem={(choice, options) => (
				<MessageWebSearchToolChoiceChip tool={choice} fullWidth={options.fullWidth} onClick={options.onClick} />
			)}
			onItemDetails={onChoiceDetails}
			countClassName="text-xs whitespace-nowrap opacity-80"
		/>
	);
}

interface MessageAttachmentsBarProps {
	isUser: boolean;
	attachments?: Attachment[];
	toolChoices?: ToolStoreChoice[];
	toolSelectionIssues?: ToolSelectionIssue[];
	onToolSelectionIssueDetails?: (issue: ToolSelectionIssue) => void;
	operationResolution?: SessionOperationResolutionSummary;
	appContextUpdates?: MCPAppContextUpdate[];
	toolCalls?: UIToolCall[];
	toolOutputs?: UIToolOutput[];
	onToolChoiceDetails?: (choice: ToolStoreChoice) => void;
	onToolCallDetails?: (call: UIToolCall) => void;
	onToolOutputDetails?: (output: UIToolOutput) => void;
}

/**
 * Read‑only toolbar under a message bubble:
 * - User messages show attachments and recorded send-time selections.
 * - Assistant messages show attachments, tool calls, and tool results.
 *
 * Uses compact dropdown chips similar to the composer, but without
 * any remove / edit actions.
 */
export const MessageAttachmentsBar = memo(function MessageAttachmentsBar({
	isUser,
	attachments,
	toolChoices,
	toolSelectionIssues,
	onToolSelectionIssueDetails,
	operationResolution,
	appContextUpdates,
	toolCalls,
	toolOutputs,
	onToolChoiceDetails,
	onToolCallDetails,
	onToolOutputDetails,
}: MessageAttachmentsBarProps) {
	const choices = isUser ? (toolChoices ?? []) : [];
	const calls = isUser ? [] : (toolCalls ?? []);
	const outputs = isUser ? [] : (toolOutputs ?? []);
	const unavailableSelections = isUser ? (toolSelectionIssues ?? []) : [];

	const normalToolChoices = choices.filter(c => c.toolType !== ToolStoreChoiceType.WebSearch);
	const webSearchChoices = choices.filter(c => c.toolType === ToolStoreChoiceType.WebSearch);
	const normalToolCalls = calls.filter(c => c.type !== ToolStoreChoiceType.WebSearch);
	const webSearchCalls = calls.filter(c => c.type === ToolStoreChoiceType.WebSearch);
	const normalToolOutputs = outputs.filter(o => o.type !== ToolStoreChoiceType.WebSearch);
	const webSearchOutputs = outputs.filter(o => o.type === ToolStoreChoiceType.WebSearch);

	const hasAttachments = !!attachments && attachments.length > 0;
	const hasTools = normalToolChoices.length > 0;
	const hasMCP = isUser && (operationResolution?.mcp?.length ?? 0) > 0;
	const hasMCPAppContext = isUser && (appContextUpdates?.length ?? 0) > 0;
	const hasSkillContext = isUser && (operationResolution?.skills?.length ?? 0) > 0;
	const hasWorkspace = isUser && Boolean(operationResolution?.workspace);

	const hasWebSearchTools = webSearchChoices.length > 0;
	const hasToolCalls = normalToolCalls.length > 0;
	const hasWebSearchCalls = webSearchCalls.length > 0;
	const hasToolOutputs = normalToolOutputs.length > 0;
	const hasWebSearchOutputs = webSearchOutputs.length > 0;

	if (
		!hasAttachments &&
		unavailableSelections.length === 0 &&
		!hasTools &&
		!hasWorkspace &&
		!hasMCP &&
		!hasMCPAppContext &&
		!hasSkillContext &&
		!hasWebSearchTools &&
		!hasToolCalls &&
		!hasWebSearchCalls &&
		!hasToolOutputs &&
		!hasWebSearchOutputs
	) {
		return null;
	}

	return (
		<div
			className="flex min-h-8 max-w-full min-w-0 items-center gap-1 overflow-x-auto text-xs"
			style={{ scrollbarGutter: 'stable' }}
		>
			{hasWorkspace ? <MessageWorkspaceResolutionChip resolution={operationResolution} /> : null}

			{hasAttachments && <AttachmentsGroupChip attachments={attachments ?? []} />}

			{hasMCP && <MCPMessageContextChip resolution={operationResolution} />}
			{hasMCPAppContext && <MCPAppContextResolutionChip updates={appContextUpdates ?? []} />}
			{hasSkillContext && <MessageSkillsResolutionChip resolution={operationResolution} />}

			{unavailableSelections.map(issue => (
				<MessageBarChip
					key={issue.selection.choiceID}
					icon={<FiAlertTriangle size={14} />}
					label={issue.selection.target.name}
					title={`Tool selection unavailable: ${issue.message}`}
					dataMessageChip="tool-selection-unavailable"
					tone="secondary"
					onClick={
						onToolSelectionIssueDetails
							? () => {
									onToolSelectionIssueDetails(issue);
								}
							: undefined
					}
					trailing={<span className="badge badge-warning badge-xs">Unavailable</span>}
				/>
			))}

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
});
