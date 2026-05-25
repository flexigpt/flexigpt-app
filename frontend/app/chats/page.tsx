import { useEffect, useMemo, useRef } from 'react';

import { useSearchParams } from 'react-router';

import { useTitleBarContent } from '@/hooks/use_title_bar';

import { PageFrame } from '@/components/page_frame';

import { ConversationArea, type ConversationAreaHandle } from '@/chats/conversation/conversation_area';
import {
	parseChatWorkflowStarterSearchParams,
	removeChatWorkflowStarterSearchParams,
} from '@/chats/conversation/starter_intent';
import { ConversationSearch, type ConversationSearchHandle } from '@/chats/search/conversation_search';
import { ChatTabsBar } from '@/chats/tabs/chat_tabs_bar';
import { useChatsController } from '@/chats/tabs/use_chats_controller';

// eslint-disable-next-line no-restricted-exports
export default function ChatsPage() {
	const conversationAreaRef = useRef<ConversationAreaHandle | null>(null);
	const searchRef = useRef<ConversationSearchHandle | null>(null);

	const [searchParams, setSearchParams] = useSearchParams();
	const workflowStarterSearch = searchParams.toString();
	const appliedWorkflowStarterSearchRef = useRef<string | null>(null);
	const workflowStarter = useMemo(
		() => parseChatWorkflowStarterSearchParams(new URLSearchParams(workflowStarterSearch)),
		[workflowStarterSearch]
	);

	const {
		tabStore,
		tabs,
		selectedTabId,
		mountedInputTabIds,
		initialScrollTopByTab,
		shortcutConfig,
		updateTab,
		saveUpdatedConversation,
		selectTab,
		openNewTab,
		closeTab,
		renameTabTitle,
		handleSelectConversation,
		getConversationForExport,
		tabBarItems,
		openConversationIds,
		searchRefreshKey,
		maxTabs,
		preloadWorkflowStarter,
	} = useChatsController({
		conversationAreaRef,
		searchRef,
	});

	useEffect(() => {
		if (!workflowStarter) {
			appliedWorkflowStarterSearchRef.current = null;
			return;
		}

		if (appliedWorkflowStarterSearchRef.current === workflowStarterSearch) {
			return;
		}

		appliedWorkflowStarterSearchRef.current = workflowStarterSearch;

		void preloadWorkflowStarter(workflowStarter)
			.then(applied => {
				if (!applied) {
					appliedWorkflowStarterSearchRef.current = null;
					return;
				}

				const nextParams = removeChatWorkflowStarterSearchParams(new URLSearchParams(workflowStarterSearch));
				setSearchParams(nextParams, { replace: true });
			})
			.catch((error: unknown) => {
				console.error('Failed to preload workflow starter:', error);
				appliedWorkflowStarterSearchRef.current = null;
			});
	}, [preloadWorkflowStarter, setSearchParams, workflowStarter, workflowStarterSearch]);

	useTitleBarContent(
		{
			center: (
				<div className="mx-auto flex w-4/5 items-center justify-center">
					<ConversationSearch
						ref={searchRef}
						compact={true}
						onSelectConversation={handleSelectConversation}
						refreshKey={searchRefreshKey}
						openConversationIds={openConversationIds}
					/>
				</div>
			),
		},
		[handleSelectConversation, openConversationIds, searchRefreshKey]
	);

	return (
		<PageFrame contentScrollable={false}>
			<div className="grid h-full w-full grid-rows-[auto_1fr_auto] overflow-hidden">
				<div className="relative row-start-1 row-end-2 min-h-0 min-w-0 p-0">
					<ChatTabsBar
						store={tabStore}
						selectedTabId={selectedTabId}
						tabs={tabBarItems}
						maxTabs={maxTabs}
						onSelectTab={selectTab}
						onNewTab={openNewTab}
						onCloseTab={closeTab}
						onRenameTab={renameTabTitle}
						getConversationForExport={getConversationForExport}
					/>
				</div>

				<ConversationArea
					ref={conversationAreaRef}
					tabs={tabs}
					selectedTabId={selectedTabId}
					mountedInputTabIds={mountedInputTabIds}
					shortcutConfig={shortcutConfig}
					initialScrollTopByTab={initialScrollTopByTab}
					updateTab={updateTab}
					saveUpdatedConversation={saveUpdatedConversation}
				/>
			</div>
		</PageFrame>
	);
}
