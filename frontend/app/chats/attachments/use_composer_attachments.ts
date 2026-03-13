import { useCallback, useState } from 'react';

import type {
	Attachment,
	AttachmentContentBlockMode,
	AttachmentsDroppedPayload,
	DirectoryAttachmentsResult,
	UIAttachment,
} from '@/spec/attachment';
import { AttachmentKind } from '@/spec/attachment';

import { backendAPI } from '@/apis/baseapi';

import {
	buildUIAttachmentForLocalPath,
	buildUIAttachmentForURL,
	type DirectoryAttachmentGroup,
	MAX_FILES_PER_DIRECTORY,
	uiAttachmentKey,
} from '@/chats/attachments/attachment_editor_utils';

interface UseComposerAttachmentsArgs {
	isBusy: boolean;
	focusEditorAtEnd: () => void;
}

interface UseComposerAttachmentsResult {
	attachments: UIAttachment[];
	directoryGroups: DirectoryAttachmentGroup[];
	attachFiles: () => Promise<void>;
	attachDirectory: () => Promise<void>;
	attachURL: (url: string) => Promise<void>;
	changeAttachmentMode: (att: UIAttachment, mode: AttachmentContentBlockMode) => void;
	removeAttachment: (att: UIAttachment) => void;
	removeDirectoryGroup: (groupId: string) => void;
	removeOverflowDir: (groupId: string, dirPath: string) => void;
	applyAttachmentsDrop: (payload: AttachmentsDroppedPayload) => void;
	clearAttachments: () => void;
	loadAttachmentsFromMessage: (incomingAttachments: Attachment[] | null | undefined) => void;
}

export function useComposerAttachments({
	isBusy,
	focusEditorAtEnd,
}: UseComposerAttachmentsArgs): UseComposerAttachmentsResult {
	const [attachments, setAttachments] = useState<UIAttachment[]>([]);
	const [directoryGroups, setDirectoryGroups] = useState<DirectoryAttachmentGroup[]>([]);

	const clearAttachments = useCallback(() => {
		setAttachments([]);
		setDirectoryGroups([]);
	}, []);

	const loadAttachmentsFromMessage = useCallback((incomingAttachments: Attachment[] | null | undefined) => {
		setAttachments(() => {
			if (!incomingAttachments || incomingAttachments.length === 0) return [];
			const next: UIAttachment[] = [];
			const seen = new Set<string>();

			for (const att of incomingAttachments) {
				let ui: UIAttachment | undefined = undefined;

				if (att.kind === AttachmentKind.url) {
					if (att.urlRef) {
						ui = buildUIAttachmentForURL(att);
					} else {
						continue;
					}
				} else if (att.kind === AttachmentKind.file || att.kind === AttachmentKind.image) {
					ui = buildUIAttachmentForLocalPath(att);
				}

				if (!ui) continue;

				const key = uiAttachmentKey(ui);
				if (seen.has(key)) continue;
				seen.add(key);
				next.push(ui);
			}
			return next;
		});

		// We don’t attempt to reconstruct directoryGroups; show flat chips instead.
		setDirectoryGroups([]);
	}, []);

	const applyFileAttachments = useCallback((results: Attachment[]) => {
		if (!results || results.length === 0) return;

		setAttachments(prev => {
			const existing = new Set(prev.map(uiAttachmentKey));
			const next: UIAttachment[] = [...prev];

			for (const r of results) {
				const att = buildUIAttachmentForLocalPath(r);
				if (!att) continue;

				const key = uiAttachmentKey(att);
				if (existing.has(key)) continue;

				existing.add(key);
				next.push(att);
			}
			return next;
		});
	}, []);

	const applyDirectoryAttachments = useCallback((result: DirectoryAttachmentsResult) => {
		if (!result || !result.dirPath) return;

		const { dirPath, attachments: dirAttachments, overflowDirs } = result;
		if ((!dirAttachments || dirAttachments.length === 0) && (!overflowDirs || overflowDirs.length === 0)) {
			return;
		}

		const folderLabel = dirPath.trim().split(/[\\/]/).pop() || dirPath.trim();
		const groupId = crypto.randomUUID?.() ?? `dir-${Date.now()}-${Math.random().toString(16).slice(2)}`;

		const attachmentKeysForGroup: string[] = [];
		const ownedAttachmentKeysForGroup: string[] = [];
		const seenKeysForGroup = new Set<string>();

		setAttachments(prev => {
			const existing = new Map<string, UIAttachment>();
			for (const att of prev) existing.set(uiAttachmentKey(att), att);

			const added: UIAttachment[] = [];

			for (const r of dirAttachments ?? []) {
				const att = buildUIAttachmentForLocalPath(r);
				if (!att) continue;

				const key = uiAttachmentKey(att);
				if (seenKeysForGroup.has(key)) continue;
				seenKeysForGroup.add(key);

				attachmentKeysForGroup.push(key);

				if (!existing.has(key)) {
					existing.set(key, att);
					added.push(att);
					ownedAttachmentKeysForGroup.push(key);
				}
			}

			return [...prev, ...added];
		});

		setDirectoryGroups(prev => [
			...prev,
			{
				id: groupId,
				dirPath,
				label: folderLabel,
				attachmentKeys: attachmentKeysForGroup,
				ownedAttachmentKeys: ownedAttachmentKeysForGroup,
				overflowDirs: overflowDirs ?? [],
			},
		]);
	}, []);

	const attachFiles = useCallback(async () => {
		let results: Attachment[];
		try {
			results = await backendAPI.openMultipleFilesAsAttachments(true);
		} catch {
			return;
		}

		applyFileAttachments(results);
		focusEditorAtEnd();
	}, [applyFileAttachments, focusEditorAtEnd]);

	const attachDirectory = useCallback(async () => {
		let result: DirectoryAttachmentsResult;
		try {
			result = await backendAPI.openDirectoryAsAttachments(MAX_FILES_PER_DIRECTORY);
		} catch {
			// Backend canceled or errored; nothing to do.
			return;
		}

		applyDirectoryAttachments(result);
		focusEditorAtEnd();
	}, [applyDirectoryAttachments, focusEditorAtEnd]);

	const attachURL = useCallback(
		async (rawUrl: string) => {
			const trimmed = rawUrl.trim();
			if (!trimmed) return;

			const bAtt = await backendAPI.openURLAsAttachment(trimmed);
			if (!bAtt) return;
			const att = buildUIAttachmentForURL(bAtt);
			const key = uiAttachmentKey(att);

			setAttachments(prev => {
				const existing = new Set(prev.map(uiAttachmentKey));
				if (existing.has(key)) return prev;
				return [...prev, att];
			});

			if (!isBusy) {
				focusEditorAtEnd();
			}
		},
		[focusEditorAtEnd, isBusy]
	);

	const changeAttachmentMode = useCallback(
		(att: UIAttachment, newMode: AttachmentContentBlockMode) => {
			const targetKey = uiAttachmentKey(att);
			setAttachments(prev => prev.map(a => (uiAttachmentKey(a) === targetKey ? { ...a, mode: newMode } : a)));
			focusEditorAtEnd();
		},
		[focusEditorAtEnd]
	);

	const removeAttachment = useCallback((att: UIAttachment) => {
		const targetKey = uiAttachmentKey(att);

		setAttachments(prev => prev.filter(a => uiAttachmentKey(a) !== targetKey));

		// Also detach from any directory groups (and drop empty groups)
		setDirectoryGroups(prevGroups => {
			const updated = prevGroups.map(g => ({
				...g,
				attachmentKeys: g.attachmentKeys.filter(k => k !== targetKey),
				ownedAttachmentKeys: g.ownedAttachmentKeys.filter(k => k !== targetKey),
			}));
			return updated.filter(g => g.attachmentKeys.length > 0 || g.overflowDirs.length > 0);
		});
	}, []);

	const removeDirectoryGroup = useCallback((groupId: string) => {
		setDirectoryGroups(prevGroups => {
			const groupToRemove = prevGroups.find(g => g.id === groupId);
			if (!groupToRemove) return prevGroups;

			const remainingGroups = prevGroups.filter(g => g.id !== groupId);

			// Keys owned by other groups (so we don't delete shared attachments).
			const keysOwnedByOtherGroups = new Set<string>();
			for (const g of remainingGroups) {
				for (const key of g.ownedAttachmentKeys) {
					keysOwnedByOtherGroups.add(key);
				}
			}

			if (groupToRemove.ownedAttachmentKeys.length > 0) {
				setAttachments(prevAttachments =>
					prevAttachments.filter(att => {
						const key = uiAttachmentKey(att);
						if (!groupToRemove.ownedAttachmentKeys.includes(key)) return true;
						// If other groups still own this attachment, keep it.
						if (keysOwnedByOtherGroups.has(key)) return true;
						// Otherwise, drop it when this folder is removed.
						return false;
					})
				);
			}

			return remainingGroups;
		});
	}, []);

	const removeOverflowDir = useCallback((groupId: string, dirPath: string) => {
		setDirectoryGroups(prevGroups => {
			const updated = prevGroups.map(g =>
				g.id !== groupId
					? g
					: {
							...g,
							overflowDirs: g.overflowDirs.filter(od => od.dirPath !== dirPath),
						}
			);
			return updated.filter(g => g.attachmentKeys.length > 0 || g.overflowDirs.length > 0);
		});
	}, []);

	const applyAttachmentsDrop = useCallback(
		(payload: AttachmentsDroppedPayload) => {
			applyFileAttachments(payload.files ?? []);
			for (const dir of payload.directories ?? []) {
				applyDirectoryAttachments(dir);
			}

			// Don’t steal focus while the tab is generating; just attach chips.
			if (!isBusy) {
				focusEditorAtEnd();
			}
		},
		[applyDirectoryAttachments, applyFileAttachments, focusEditorAtEnd, isBusy]
	);

	return {
		attachments,
		directoryGroups,
		attachFiles,
		attachDirectory,
		attachURL,
		changeAttachmentMode,
		removeAttachment,
		removeDirectoryGroup,
		removeOverflowDir,
		applyAttachmentsDrop,
		clearAttachments,
		loadAttachmentsFromMessage,
	};
}
