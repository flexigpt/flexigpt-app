import { memo, type ReactNode, type RefObject, useCallback, useEffect, useRef, useState } from 'react';

import { FiEdit2, FiPlus, FiX } from 'react-icons/fi';

import type { TabStore } from '@ariakit/react/tab';
import { Tab, TabList } from '@ariakit/react/tab';

import { sanitizeConversationTitle } from '@/lib/text_utils';

import { HoverTip } from '@/components/ariakit_hover_tip';
import { BusyDot } from '@/components/busy_dot';
import { DownloadButton } from '@/components/download_button';

interface ChatTabBarItem {
	tabId: string;
	title: string;
	isBusy: boolean;
	isEmpty: boolean; // messages.length === 0
	renameEnabled: boolean; // messages.length > 0
}

interface ChatTabsBarProps {
	store: TabStore;
	selectedTabId: string;
	tabs: ChatTabBarItem[];

	maxTabs: number;
	onNewTab: () => void;
	onCloseTab: (tabId: string) => void;
	onRenameTab: (tabId: string, newTitle: string) => void;
	getConversationForExport: () => Promise<string>;
}

interface ChatTabsBarContentProps extends ChatTabsBarProps {
	setTabEl: (id: string) => (el: HTMLElement | null) => void;
	tabsViewportRef: RefObject<HTMLDivElement | null>;
}

function ChatTabsBarContent({
	store,
	selectedTabId,
	tabs,
	maxTabs,
	onNewTab,
	onCloseTab,
	onRenameTab,
	getConversationForExport,
	setTabEl,
	tabsViewportRef,
}: ChatTabsBarContentProps) {
	const [editingTabId, setEditingTabId] = useState<string | null>(null);
	const [draftTitle, setDraftTitle] = useState('');

	const finishRename = useCallback(() => {
		if (!editingTabId) return;

		const currentTitle = tabs.find(tab => tab.tabId === editingTabId)?.title ?? '';
		const cleaned = sanitizeConversationTitle(draftTitle.trim());

		if (cleaned && cleaned !== currentTitle) {
			onRenameTab(editingTabId, cleaned);
		}

		setEditingTabId(null);
	}, [draftTitle, editingTabId, onRenameTab, tabs]);

	const elements: ReactNode[] = [];

	for (const t of tabs) {
		const isActive = t.tabId === selectedTabId;
		const canRename = isActive && t.renameEnabled && !t.isBusy;
		const isEditing = isActive && editingTabId === t.tabId;

		elements.push(
			<Tab
				key={t.tabId}
				ref={setTabEl(t.tabId)}
				store={store}
				id={t.tabId}
				// render as <div> so we can safely place an <input> inside the tab (no <input> inside <button>)
				render={<div />}
				className={[
					'relative flex h-8 w-44 items-center p-0',
					'select-none',
					'focus-visible:outline-primary focus-visible:outline focus-visible:outline-offset-2',
					// Firefox-ish feel: rounded top + active lifted
					isActive
						? 'bg-base-100 text-base-content border-base-300 rounded-xl border shadow-xs'
						: 'bg-base-200/80 text-base-content/80 hover:bg-base-200 border-0',
					t.isBusy ? 'cursor-progress' : '',
				].join(' ')}
			>
				{/* Title / Rename */}
				<div className="min-w-0 flex-1 px-2 text-sm">
					{isEditing ? (
						<input
							data-disable-chat-shortcuts="true"
							autoFocus
							value={draftTitle}
							onChange={e => {
								setDraftTitle(e.target.value);
							}}
							onBlur={finishRename}
							onKeyDown={e => {
								e.stopPropagation();
								if (e.key === 'Enter') finishRename();
								if (e.key === 'Escape') setEditingTabId(null);
							}}
							onMouseDown={e => {
								e.stopPropagation();
							}}
							className="input input-sm bg-base-100 w-full p-0"
						/>
					) : (
						<div className="flex min-w-0" title={t.title}>
							<span className="truncate">{t.title}</span>
						</div>
					)}
				</div>

				{/* Right end: spinner OR rename icon in same slot, then close */}
				<div className="flex items-center gap-1 pr-1">
					{(t.isBusy || canRename) && (
						<div className="flex w-6 shrink-0 items-center justify-center">
							{t.isBusy ? (
								<HoverTip content="Response in progress" placement="bottom">
									<span className="inline-flex">
										<BusyDot />
									</span>
								</HoverTip>
							) : canRename ? (
								<HoverTip content="Rename tab" placement="bottom">
									<button
										type="button"
										className="btn btn-ghost btn-xs btn-circle p-0 opacity-70 hover:opacity-100"
										aria-label="Rename tab"
										onMouseDown={e => {
											e.stopPropagation();
										}}
										onClick={e => {
											e.stopPropagation();
											setEditingTabId(t.tabId);
											setDraftTitle(t.title);
										}}
									>
										<FiEdit2 size={14} />
									</button>
								</HoverTip>
							) : null}
						</div>
					)}
					<HoverTip content="Close tab" placement="bottom">
						<button
							type="button"
							className="btn btn-ghost btn-xs btn-circle shrink-0 p-0 opacity-80 hover:opacity-100"
							aria-label="Close tab"
							onMouseDown={e => {
								e.stopPropagation();
							}}
							onClick={e => {
								e.stopPropagation();
								onCloseTab(t.tabId);
							}}
						>
							<FiX size={14} />
						</button>
					</HoverTip>
				</div>
			</Tab>
		);
	}

	return (
		<div className="border-base-300 flex h-9 w-full items-center gap-2 border-b bg-inherit">
			<div className="flex min-w-0 flex-1 flex-nowrap items-center overflow-hidden">
				{/* Scroll ONLY the tabs. Reserve bottom space so scrollbar doesn't clip tab content. */}
				<div
					ref={tabsViewportRef}
					className="scrollbar-custom-thin min-w-0 overflow-x-auto overflow-y-hidden overscroll-contain pb-1"
					style={{ scrollbarGutter: 'stable' }}
				>
					<TabList store={store} aria-label="Chat tabs" className="flex h-9 w-max items-end gap-0 pr-1">
						{elements}
					</TabList>
				</div>
				<HoverTip
					content={
						tabs.length >= maxTabs ? `New chat (reuses the scratch tab at the ${maxTabs}-tab limit)` : 'New chat'
					}
					placement="left"
				>
					<button
						type="button"
						className="btn btn-ghost btn-circle btn-xs shrink-0 p-0 opacity-80 hover:opacity-100"
						onClick={onNewTab}
						aria-label="New chat"
					>
						<FiPlus size={18} />
					</button>
				</HoverTip>
			</div>
			<HoverTip content="Export current chat as JSON" placement="left">
				<DownloadButton
					language="json"
					valueFetcher={getConversationForExport}
					size={18}
					fileprefix="conversation"
					className="btn btn-ghost btn-circle btn-xs shrink-0 p-0 opacity-80 hover:opacity-100"
					aria-label="Export Chat"
					title="Export Chat"
				/>
			</HoverTip>
		</div>
	);
}

export const ChatTabsBar = memo(function ChatTabsBar({
	store,
	selectedTabId,
	tabs,
	maxTabs,
	onNewTab,
	onCloseTab,
	onRenameTab,
	getConversationForExport,
}: ChatTabsBarProps) {
	const tabElById = useRef(new Map<string, HTMLElement | null>());
	const tabsViewportRef = useRef<HTMLDivElement | null>(null);

	const setTabEl = useCallback(
		(id: string) => (el: HTMLElement | null) => {
			if (!el) tabElById.current.delete(id);
			else tabElById.current.set(id, el);
		},
		[]
	);

	useEffect(() => {
		const tabEl = tabElById.current.get(selectedTabId);
		const viewportEl = tabsViewportRef.current;
		if (!tabEl || !viewportEl) return;

		const viewportRect = viewportEl.getBoundingClientRect();
		const tabRect = tabEl.getBoundingClientRect();
		const fullyVisible = tabRect.left >= viewportRect.left && tabRect.right <= viewportRect.right;

		if (fullyVisible) return;

		tabEl.scrollIntoView({
			behavior: 'auto',
			block: 'nearest',
			inline: 'nearest',
		});
	}, [selectedTabId]);

	return (
		<ChatTabsBarContent
			store={store}
			selectedTabId={selectedTabId}
			tabs={tabs}
			maxTabs={maxTabs}
			onNewTab={onNewTab}
			onCloseTab={onCloseTab}
			onRenameTab={onRenameTab}
			getConversationForExport={getConversationForExport}
			setTabEl={setTabEl}
			tabsViewportRef={tabsViewportRef}
		/>
	);
});
