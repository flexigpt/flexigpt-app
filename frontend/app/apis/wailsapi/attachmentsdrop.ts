import type { AttachmentsDroppedPayload, PathAttachmentsResult } from '@/spec/attachment';

import { getUUIDv7 } from '@/lib/uuid_utils';

import type { IAttachmentsDropAPI } from '@/apis/interface';
import { requireWailsBody, requireWailsString, wailsArrayOrEmpty } from '@/apis/wailsapi/transport';
import { GetPathsAsAttachments } from '@/apis/wailsjs/go/main/App';
import { EventsOff, EventsOn } from '@/apis/wailsjs/runtime/runtime';

import { MAX_DIRECTORY_FILES_TO_SCAN } from '@/chats/composer/attachments/attachment_editor_utils';

type DropTarget = (payload: AttachmentsDroppedPayload) => void;

const FILE_DROP_EVENT = 'wails:file-drop';

let inited = false;
let listenerUsers = 0;
let activeTarget: DropTarget | null = null;
let pending: AttachmentsDroppedPayload[] = [];
let onNoTarget: ((payload: AttachmentsDroppedPayload) => void) | null = null;

function initWailsDropListener(): () => void {
	if (!inited) {
		EventsOn(FILE_DROP_EVENT, (x: number, y: number, paths: string[]) => {
			void handleFileDrop(x, y, paths);
		});
		inited = true;
	}

	listenerUsers += 1;

	async function handleFileDrop(x: number, y: number, paths: string[]) {
		try {
			const normalizedPaths = wailsArrayOrEmpty<string>(paths, 'wails:file-drop.paths').map((path, index) =>
				requireWailsString(path, `wails:file-drop.paths[${index}]`)
			);

			if (normalizedPaths.length === 0) {
				console.error('empty paths in file drop');
				return;
			}

			const pathResults = await GetPathsAsAttachments(normalizedPaths, MAX_DIRECTORY_FILES_TO_SCAN);
			const r = requireWailsBody(pathResults as PathAttachmentsResult | null | undefined, 'GetPathsAsAttachments');

			const dropID = getUUIDv7();
			const payload: AttachmentsDroppedPayload = {
				dropID: dropID,
				x: x,
				y: y,
				files: wailsArrayOrEmpty(r.fileAttachments, 'GetPathsAsAttachments.fileAttachments'),
				directories: wailsArrayOrEmpty(r.dirAttachments, 'GetPathsAsAttachments.dirAttachments'),
				errors: wailsArrayOrEmpty(r.errors, 'GetPathsAsAttachments.errors'),
				maxFilesPerDirectory: MAX_DIRECTORY_FILES_TO_SCAN,
			};

			const target = activeTarget;
			if (target) {
				try {
					target(payload);
				} catch (error) {
					pending.push(payload);
					console.error('Failed to apply active attachment drop; queued for retry', error);
					onNoTarget?.(payload);
				}
				return;
			}

			// No chat composer active: queue it so it’s never lost
			pending.push(payload);
			onNoTarget?.(payload);
		} catch (e) {
			console.error('error in building attachments', e);
		}
	}

	let stopped = false;
	return () => {
		if (stopped) {
			return;
		}

		stopped = true;
		listenerUsers = Math.max(0, listenerUsers - 1);

		if (listenerUsers === 0 && inited) {
			EventsOff(FILE_DROP_EVENT);
			inited = false;
		}
	};
}

export class WailsAttachmentsDropAPI implements IAttachmentsDropAPI {
	startListener(): () => void {
		return initWailsDropListener();
	}

	registerDropTarget(fn: (payload: AttachmentsDroppedPayload) => void): () => void {
		activeTarget = fn;

		// Flush pending drops immediately when chat becomes active
		const toFlush = pending;
		pending = [];
		for (const p of toFlush) {
			try {
				fn(p);
			} catch (e) {
				console.error('Failed to apply pending drop', e);
				pending.push(p);
			}
		}

		return () => {
			if (activeTarget === fn) {
				activeTarget = null;
			}
		};
	}

	setNoTargetHandler(fn: ((payload: AttachmentsDroppedPayload) => void) | null): void {
		onNoTarget = fn;
	}
}
