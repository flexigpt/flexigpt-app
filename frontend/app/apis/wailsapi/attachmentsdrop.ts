import type { AttachmentsDroppedPayload, PathAttachmentsResult } from '@/spec/attachment';

import { getUUIDv7 } from '@/lib/uuid_utils';

import type { IAttachmentsDropAPI } from '@/apis/interface';
import { GetPathsAsAttachments } from '@/apis/wailsjs/go/main/App';
import { EventsOn } from '@/apis/wailsjs/runtime/runtime';

type DropTarget = (payload: AttachmentsDroppedPayload) => void;

let inited = false;
let activeTarget: DropTarget | null = null;
let pending: AttachmentsDroppedPayload[] = [];
let onNoTarget: ((payload: AttachmentsDroppedPayload) => void) | null = null;

function initWailsDropListener(): () => void {
	if (inited) return () => {};
	inited = true;

	EventsOn('wails:file-drop', (x: number, y: number, paths: string[]) => {
		void handleFileDrop(x, y, paths);
	});

	async function handleFileDrop(x: number, y: number, paths: string[]) {
		if (paths.length === 0) {
			console.error('empty paths in file drop');
			return;
		}

		try {
			console.log('got attachments drop', x, y, paths);
			const pathResults = await GetPathsAsAttachments(paths, 128);

			const r = pathResults as PathAttachmentsResult;
			const dropID = getUUIDv7();
			const payload: AttachmentsDroppedPayload = {
				dropID: dropID,
				x: x,
				y: y,
				files: r.fileAttachments,
				directories: r.dirAttachments,
				errors: r.errors,
				maxFilesPerDirectory: 128,
			};
			if (activeTarget) {
				activeTarget(payload);
				return;
			}

			// No chat composer active: queue it so it’s never lost
			pending.push(payload);
			onNoTarget?.(payload);
		} catch (e) {
			console.error('error in building attachments', e);
		}
	}

	return () => {};
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
			}
		}

		return () => {
			if (activeTarget === fn) activeTarget = null;
		};
	}

	setNoTargetHandler(fn: ((payload: AttachmentsDroppedPayload) => void) | null): void {
		onNoTarget = fn;
	}
}
