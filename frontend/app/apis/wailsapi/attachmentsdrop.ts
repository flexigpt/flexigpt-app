import { Events } from '@wailsio/runtime';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';

import { getUUIDv7 } from '@/lib/uuid_utils';

import type { IAttachmentsDropAPI } from '@/apis/interface';
import { pathAttachmentsResultFromWails } from '@/apis/wailsapi/attachment_projection';
import {
	requiredObject,
	requireWailsFiniteNumber,
	requireWailsString,
	wailsArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import { GetPathsAsAttachments } from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/cmd/agentgo/app';

import { MAX_DIRECTORY_FILES_TO_SCAN } from '@/chats/composer/attachments/attachment_editor_utils';

type DropTarget = (payload: AttachmentsDroppedPayload) => void;

const FILE_DROP_EVENT = 'wails:file-drop';
const nativeFileDropSupported = import.meta.env.VITE_WAILS_NATIVE_FILE_DROP !== 'disabled';

let inited = false;
let listenerUsers = 0;
let activeTarget: DropTarget | null = null;
let fileDropUnsubscribe: (() => void) | undefined;
let pending: AttachmentsDroppedPayload[] = [];
let onNoTarget: ((payload: AttachmentsDroppedPayload) => void) | null = null;

async function handleFileDrop(x: number, y: number, paths: string[]) {
	try {
		const normalizedPaths = wailsArrayOrEmpty<string>(paths, 'wails:file-drop.paths').map((path, index) =>
			requireWailsString(path, `wails:file-drop.paths[${index}]`)
		);

		if (normalizedPaths.length === 0) {
			console.error('empty paths in file drop');
			return;
		}

		const pathResults = pathAttachmentsResultFromWails(
			await GetPathsAsAttachments(normalizedPaths, MAX_DIRECTORY_FILES_TO_SCAN),
			'GetPathsAsAttachments'
		);

		const dropID = getUUIDv7();
		const payload: AttachmentsDroppedPayload = {
			dropID: dropID,
			x: x,
			y: y,
			files: pathResults.fileAttachments,
			directories: pathResults.dirAttachments,
			errors: pathResults.errors,
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

function nativeFileDropFromWails(args: readonly unknown[]): {
	x: number;
	y: number;
	paths: string[];
} {
	const [first, second, third] = args;

	// Wails v2 delivered x, y, paths as positional arguments. Retaining this
	// branch makes the bridge tolerant while moving the subscription itself to
	// the v3 event API.
	if (typeof first === 'number' && typeof second === 'number') {
		return {
			x: requireWailsFiniteNumber(first, `${FILE_DROP_EVENT}.x`),
			y: requireWailsFiniteNumber(second, `${FILE_DROP_EVENT}.y`),
			paths: wailsArrayOrEmpty<string>(third, `${FILE_DROP_EVENT}.paths`),
		};
	}

	const event = requiredObject<Record<string, unknown>>(first, FILE_DROP_EVENT);
	const payload =
		event.paths !== undefined || event.Paths !== undefined
			? event
			: requiredObject<Record<string, unknown>>(event.data, `${FILE_DROP_EVENT}.data`);

	return {
		x: requireWailsFiniteNumber(payload.x ?? payload.X, `${FILE_DROP_EVENT}.x`),
		y: requireWailsFiniteNumber(payload.y ?? payload.Y, `${FILE_DROP_EVENT}.y`),
		paths: wailsArrayOrEmpty<string>(payload.paths ?? payload.Paths, `${FILE_DROP_EVENT}.paths`),
	};
}

function initWailsDropListener(): () => void {
	if (!inited) {
		fileDropUnsubscribe = Events.On(FILE_DROP_EVENT, (...args: unknown[]) => {
			try {
				const drop = nativeFileDropFromWails(args);
				void handleFileDrop(drop.x, drop.y, drop.paths);
			} catch (error) {
				console.error('Invalid Wails file-drop event payload', error);
			}
		});
		inited = true;
	}

	listenerUsers += 1;

	let stopped = false;

	return () => {
		if (stopped) {
			return;
		}

		stopped = true;
		listenerUsers = Math.max(0, listenerUsers - 1);

		if (listenerUsers === 0 && inited) {
			const unsubscribe = fileDropUnsubscribe;
			fileDropUnsubscribe = undefined;
			inited = false;

			unsubscribe?.();
		}
	};
}

export class WailsAttachmentsDropAPI implements IAttachmentsDropAPI {
	startListener(): () => void {
		if (!nativeFileDropSupported) {
			return () => undefined;
		}

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
