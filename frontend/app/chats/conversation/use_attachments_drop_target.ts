import { type RefObject, useEffect } from 'react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';

import { attachmentsDropAPI } from '@/apis/baseapi';

type UseAttachmentsDropTargetArgs = {
	selectedTabIdRef: RefObject<string>;
	tabExists: (tabId: string) => boolean;
	tryApplyDropToTab: (tabId: string, payload: AttachmentsDroppedPayload) => boolean;
	queuePendingDrop: (tabId: string, payload: AttachmentsDroppedPayload) => void;
	flushPendingDrops: () => void;
};

export function useAttachmentsDropTarget({
	selectedTabIdRef,
	tabExists,
	tryApplyDropToTab,
	queuePendingDrop,
	flushPendingDrops,
}: UseAttachmentsDropTargetArgs) {
	useEffect(() => {
		const unregister = attachmentsDropAPI.registerDropTarget((payload: AttachmentsDroppedPayload) => {
			const tabId = selectedTabIdRef.current;

			if (!tabId || !tabExists(tabId)) {
				queuePendingDrop(tabId, payload);
				window.setTimeout(() => {
					flushPendingDrops();
				}, 50);
				return;
			}

			const applied = tryApplyDropToTab(tabId, payload);
			if (!applied) {
				queuePendingDrop(tabId, payload);
				window.setTimeout(() => {
					flushPendingDrops();
				}, 0);
			}
		});

		return unregister;
	}, [flushPendingDrops, queuePendingDrop, selectedTabIdRef, tabExists, tryApplyDropToTab]);
}
