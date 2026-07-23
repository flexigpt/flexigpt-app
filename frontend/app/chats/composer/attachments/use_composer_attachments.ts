import type { SetStateAction } from 'react';
import { useCallback, useRef, useState } from 'react';

import type {
	Attachment,
	AttachmentContentBlockMode,
	AttachmentsDroppedPayload,
	DirectoryAttachmentsResult,
	DirectoryOverflowInfo,
	PathAttachmentsResult,
	UIAttachment,
} from '@/spec/attachment';
import { AttachmentKind } from '@/spec/attachment';

import { resolveStateUpdate } from '@/lib/hook_utils';

import { backendAPI } from '@/apis/baseapi';

import type { DirectoryAttachmentGroup } from '@/chats/composer/attachments/attachment_editor_utils';
import {
	buildUIAttachmentForLocalPath,
	buildUIAttachmentForURL,
	MAX_DIRECTORY_FILES_TO_SCAN,
	MAX_FILES_PER_DIRECTORY,
	uiAttachmentKey,
} from '@/chats/composer/attachments/attachment_editor_utils';

function mergeUniqueStrings(existing: string[], incoming: string[]): string[] {
	if (incoming.length === 0) {
		return existing;
	}

	const seen = new Set(existing);
	const next = [...existing];
	let changed = false;

	for (const item of incoming) {
		if (seen.has(item)) {
			continue;
		}
		seen.add(item);
		next.push(item);
		changed = true;
	}

	return changed ? next : existing;
}

function mergeUniqueAttachments(existing: UIAttachment[], incoming: UIAttachment[]): UIAttachment[] {
	if (incoming.length === 0) {
		return existing;
	}

	const seen = new Set(
		existing.map(a => {
			return uiAttachmentKey(a);
		})
	);
	const next = [...existing];
	let changed = false;

	for (const attachment of incoming) {
		const key = uiAttachmentKey(attachment);
		if (seen.has(key)) {
			continue;
		}
		seen.add(key);
		next.push(attachment);
		changed = true;
	}

	return changed ? next : existing;
}

function mergeOverflowDirs(
	existing: DirectoryOverflowInfo[],
	incoming: DirectoryOverflowInfo[]
): DirectoryOverflowInfo[] {
	if (incoming.length === 0) {
		return existing;
	}

	const byDirPath = new Map<string, DirectoryOverflowInfo>();
	for (const item of existing) {
		byDirPath.set(item.dirPath, item);
	}

	let changed = false;
	for (const item of incoming) {
		const prev = byDirPath.get(item.dirPath);
		if (prev !== item) {
			changed = true;
		}
		byDirPath.set(item.dirPath, item);
	}

	if (!changed && byDirPath.size === existing.length) {
		return existing;
	}
	return [...byDirPath.values()];
}

function releaseOwnedKeysFromDirectoryGroups(
	groups: DirectoryAttachmentGroup[],
	keysToRelease: Set<string>
): DirectoryAttachmentGroup[] {
	let changed = false;

	const next = groups.map(group => {
		const ownedAttachmentKeys = group.ownedAttachmentKeys.filter(key => !keysToRelease.has(key));
		if (ownedAttachmentKeys.length === group.ownedAttachmentKeys.length) {
			return group;
		}

		changed = true;
		return {
			...group,
			ownedAttachmentKeys,
		};
	});

	return changed ? next : groups;
}

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
	attachPathsAsAttachments: (paths: string[], maxFilesPerDir?: number) => Promise<PathAttachmentsResult | undefined>;
	changeAttachmentMode: (att: UIAttachment, mode: AttachmentContentBlockMode) => void;
	removeAttachment: (att: UIAttachment) => void;
	removeDirectoryGroup: (groupId: string) => void;
	removeDirectoryAttachments: (groupId: string, attachments: UIAttachment[]) => void;
	restoreDirectoryAttachments: (groupId: string, attachments: UIAttachment[]) => void;
	removeOverflowDir: (groupId: string, dirPath: string) => void;
	applyAttachmentsDrop: (payload: AttachmentsDroppedPayload) => void;
	clearAttachments: () => void;
	loadAttachmentsFromMessage: (incomingAttachments: Attachment[] | null | undefined) => void;
}

export function useComposerAttachments({
	isBusy,
	focusEditorAtEnd,
}: UseComposerAttachmentsArgs): UseComposerAttachmentsResult {
	const [attachments, setAttachmentsState] = useState<UIAttachment[]>([]);
	const [directoryGroups, setDirectoryGroupsState] = useState<DirectoryAttachmentGroup[]>([]);
	const attachmentsRef = useRef<UIAttachment[]>([]);
	const directoryGroupsRef = useRef<DirectoryAttachmentGroup[]>([]);

	const setAttachments = useCallback((update: SetStateAction<UIAttachment[]>) => {
		const next = resolveStateUpdate(update, attachmentsRef.current);
		attachmentsRef.current = next;
		setAttachmentsState(prev => (prev === next ? prev : next));
	}, []);

	const setDirectoryGroups = useCallback((update: SetStateAction<DirectoryAttachmentGroup[]>) => {
		const next = resolveStateUpdate(update, directoryGroupsRef.current);
		directoryGroupsRef.current = next;
		setDirectoryGroupsState(prev => (prev === next ? prev : next));
	}, []);

	const clearAttachments = useCallback(() => {
		setAttachments([]);
		setDirectoryGroups([]);
	}, [setAttachments, setDirectoryGroups]);

	const loadAttachmentsFromMessage = useCallback(
		(incomingAttachments: Attachment[] | null | undefined) => {
			setAttachments(() => {
				if (!incomingAttachments || incomingAttachments.length === 0) {
					return [];
				}
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

					if (!ui) {
						continue;
					}

					const key = uiAttachmentKey(ui);
					if (seen.has(key)) {
						continue;
					}
					seen.add(key);
					next.push(ui);
				}
				return next;
			});

			// We don’t attempt to reconstruct directoryGroups; show flat chips instead.
			setDirectoryGroups([]);
		},
		[setAttachments, setDirectoryGroups]
	);

	const applyFileAttachments = useCallback(
		(results: Attachment[]) => {
			if (!results || results.length === 0) {
				return;
			}
			const keysToReleaseFromDirectoryOwnership = new Set<string>();
			setAttachments(prev => {
				const existing = new Set(prev.map(k => uiAttachmentKey(k)));
				const next: UIAttachment[] = [...prev];

				for (const r of results) {
					const att = buildUIAttachmentForLocalPath(r);
					if (!att) {
						continue;
					}

					const key = uiAttachmentKey(att);
					if (existing.has(key)) {
						// If the user explicitly attaches a file that already exists only because
						// it came from a directory selection, treat it as independently retained.
						keysToReleaseFromDirectoryOwnership.add(key);
						continue;
					}

					existing.add(key);
					next.push(att);
				}
				return next;
			});
			if (keysToReleaseFromDirectoryOwnership.size > 0) {
				setDirectoryGroups(prev => releaseOwnedKeysFromDirectoryGroups(prev, keysToReleaseFromDirectoryOwnership));
			}
		},
		[setAttachments, setDirectoryGroups]
	);

	const applyDirectoryAttachments = useCallback(
		(result: DirectoryAttachmentsResult) => {
			if (!result || !result.dirPath) {
				return;
			}

			const { dirPath, attachments: dirAttachments, overflowDirs } = result;
			if (
				(!dirAttachments || dirAttachments.length === 0) &&
				(!overflowDirs || overflowDirs.length === 0) &&
				!result.hasMore
			) {
				return;
			}

			const folderLabel = dirPath.trim().split(/[\\/]/).pop() || dirPath.trim();
			const groupId = crypto.randomUUID?.() ?? `dir-${Date.now()}-${Math.random().toString(16).slice(2)}`;
			const nextOverflowDirs = overflowDirs ?? [];
			const existingGroup = directoryGroupsRef.current.find(group => group.dirPath === dirPath);
			const knownKeys = new Set(existingGroup?.attachmentKeys);
			for (const attachment of existingGroup?.removedAttachments ?? []) {
				knownKeys.add(uiAttachmentKey(attachment));
			}

			const candidates: UIAttachment[] = [];
			const candidateKeys = new Set<string>();
			for (const resultAttachment of dirAttachments ?? []) {
				const attachment = buildUIAttachmentForLocalPath(resultAttachment);
				if (!attachment) {
					continue;
				}

				const key = uiAttachmentKey(attachment);
				if (candidateKeys.has(key) || knownKeys.has(key)) {
					continue;
				}
				candidateKeys.add(key);
				candidates.push(attachment);
			}

			const currentAttachedCount = existingGroup?.attachmentKeys.length ?? 0;
			const availableSlots = Math.max(0, MAX_FILES_PER_DIRECTORY - currentAttachedCount);
			const activeCandidates = candidates.slice(0, availableSlots);
			const removedCandidates = candidates.slice(availableSlots);
			const attachmentKeysForGroup: string[] = [];
			const ownedAttachmentKeysForGroup: string[] = [];

			setAttachments(prev => {
				const existing = new Map<string, UIAttachment>();
				for (const att of prev) {
					existing.set(uiAttachmentKey(att), att);
				}

				const added: UIAttachment[] = [];

				for (const att of activeCandidates) {
					const key = uiAttachmentKey(att);
					attachmentKeysForGroup.push(key);

					if (!existing.has(key)) {
						existing.set(key, att);
						added.push(att);
						ownedAttachmentKeysForGroup.push(key);
					}
				}

				if (added.length === 0) {
					return prev;
				}
				return [...prev, ...added];
			});

			setDirectoryGroups(prev => {
				const existingIndex = prev.findIndex(group => group.dirPath === dirPath);

				if (existingIndex === -1) {
					const removedAttachments = removedCandidates;
					return [
						...prev,
						{
							id: groupId,
							dirPath,
							label: folderLabel,
							attachmentKeys: attachmentKeysForGroup,
							ownedAttachmentKeys: ownedAttachmentKeysForGroup,
							removedAttachments,
							overflowDirs: nextOverflowDirs,
							scannedFileCount: attachmentKeysForGroup.length + removedAttachments.length,
							hasMore: result.hasMore,
						},
					];
				}

				const exGroup = prev[existingIndex];
				const removedAttachments = mergeUniqueAttachments(exGroup.removedAttachments ?? [], removedCandidates);
				const attachmentKeys = mergeUniqueStrings(exGroup.attachmentKeys, attachmentKeysForGroup);
				const mergedGroup: DirectoryAttachmentGroup = {
					...exGroup,
					label: folderLabel,
					attachmentKeys,
					ownedAttachmentKeys: mergeUniqueStrings(exGroup.ownedAttachmentKeys, ownedAttachmentKeysForGroup),
					removedAttachments,
					overflowDirs: mergeOverflowDirs(exGroup.overflowDirs, nextOverflowDirs),
					scannedFileCount: attachmentKeys.length + removedAttachments.length,
					hasMore: exGroup.hasMore || result.hasMore,
				};

				if (
					mergedGroup.label === exGroup.label &&
					mergedGroup.attachmentKeys === exGroup.attachmentKeys &&
					mergedGroup.ownedAttachmentKeys === exGroup.ownedAttachmentKeys &&
					mergedGroup.overflowDirs === exGroup.overflowDirs &&
					mergedGroup.removedAttachments === exGroup.removedAttachments &&
					mergedGroup.scannedFileCount === exGroup.scannedFileCount &&
					mergedGroup.hasMore === exGroup.hasMore
				) {
					return prev;
				}

				const next = [...prev];
				next[existingIndex] = mergedGroup;
				return next;
			});
		},
		[setAttachments, setDirectoryGroups]
	);

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
			result = await backendAPI.openDirectoryAsAttachments(MAX_DIRECTORY_FILES_TO_SCAN);
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
			if (!trimmed) {
				return;
			}

			const bAtt = await backendAPI.openURLAsAttachment(trimmed);
			if (!bAtt) {
				return;
			}
			const att = buildUIAttachmentForURL(bAtt);
			const key = uiAttachmentKey(att);

			setAttachments(prev => {
				const existing = new Set(prev.map(k => uiAttachmentKey(k)));
				if (existing.has(key)) {
					return prev;
				}
				return [...prev, att];
			});

			if (!isBusy) {
				focusEditorAtEnd();
			}
		},
		[focusEditorAtEnd, isBusy, setAttachments]
	);

	const attachPathsAsAttachments = useCallback(
		async (paths: string[], maxFilesPerDir = MAX_DIRECTORY_FILES_TO_SCAN) => {
			const cleanPaths = paths.map(path => path.trim()).filter(Boolean);
			if (cleanPaths.length === 0) {
				return undefined;
			}

			const result = await backendAPI.getPathsAsAttachments(cleanPaths, maxFilesPerDir);
			applyFileAttachments(result.fileAttachments ?? []);
			for (const dirResult of result.dirAttachments ?? []) {
				applyDirectoryAttachments(dirResult);
			}

			if (!isBusy) {
				focusEditorAtEnd();
			}
			return result;
		},
		[applyDirectoryAttachments, applyFileAttachments, focusEditorAtEnd, isBusy]
	);

	const changeAttachmentMode = useCallback(
		(att: UIAttachment, newMode: AttachmentContentBlockMode) => {
			const targetKey = uiAttachmentKey(att);
			setAttachments(prev => prev.map(a => (uiAttachmentKey(a) === targetKey ? { ...a, mode: newMode } : a)));
			focusEditorAtEnd();
		},
		[focusEditorAtEnd, setAttachments]
	);

	const removeAttachment = useCallback(
		(att: UIAttachment) => {
			const targetKey = uiAttachmentKey(att);

			setAttachments(prev => prev.filter(a => uiAttachmentKey(a) !== targetKey));

			// A global removal also moves the file into the "not attached"
			// collection of every directory that referenced it.
			setDirectoryGroups(prevGroups => {
				return prevGroups.map(group => {
					if (!group.attachmentKeys.includes(targetKey)) {
						return group;
					}

					return {
						...group,
						attachmentKeys: group.attachmentKeys.filter(key => key !== targetKey),
						ownedAttachmentKeys: group.ownedAttachmentKeys.filter(key => key !== targetKey),
						removedAttachments: mergeUniqueAttachments(group.removedAttachments ?? [], [att]),
					};
				});
			});
		},
		[setAttachments, setDirectoryGroups]
	);

	const removeDirectoryAttachments = useCallback(
		(groupId: string, attachmentsToRemove: UIAttachment[]) => {
			const requestedByKey = new Map(attachmentsToRemove.map(att => [uiAttachmentKey(att), att]));
			if (requestedByKey.size === 0) {
				return;
			}

			setDirectoryGroups(prevGroups => {
				const group = prevGroups.find(item => item.id === groupId);
				if (!group) {
					return prevGroups;
				}

				const activeKeys = new Set(group.attachmentKeys);
				const removed = [...requestedByKey].filter(([key]) => activeKeys.has(key));
				if (removed.length === 0) {
					return prevGroups;
				}

				const removedKeys = new Set(removed.map(([key]) => key));
				const ownedRemovedKeys = new Set(group.ownedAttachmentKeys.filter(key => removedKeys.has(key)));
				const keysToDeleteGlobally = new Set<string>();
				const ownershipTransfers = new Map<string, string[]>();

				for (const key of ownedRemovedKeys) {
					const recipient = prevGroups.find(item => item.id !== groupId && item.attachmentKeys.includes(key));
					if (recipient) {
						const transferred = ownershipTransfers.get(recipient.id) ?? [];
						transferred.push(key);
						ownershipTransfers.set(recipient.id, transferred);
					} else {
						keysToDeleteGlobally.add(key);
					}
				}

				if (keysToDeleteGlobally.size > 0) {
					setAttachments(prev => prev.filter(att => !keysToDeleteGlobally.has(uiAttachmentKey(att))));
				}

				return prevGroups.map(item => {
					if (item.id === groupId) {
						return {
							...item,
							attachmentKeys: item.attachmentKeys.filter(key => !removedKeys.has(key)),
							ownedAttachmentKeys: item.ownedAttachmentKeys.filter(key => !removedKeys.has(key)),
							removedAttachments: mergeUniqueAttachments(
								item.removedAttachments ?? [],
								removed.map(([, att]) => att)
							),
						};
					}

					const transferredKeys = ownershipTransfers.get(item.id);
					if (!transferredKeys) {
						return item;
					}
					return {
						...item,
						ownedAttachmentKeys: mergeUniqueStrings(item.ownedAttachmentKeys, transferredKeys),
					};
				});
			});
		},
		[setAttachments, setDirectoryGroups]
	);

	const restoreDirectoryAttachments = useCallback(
		(groupId: string, attachmentsToRestore: UIAttachment[]) => {
			const requestedKeys = new Set(
				attachmentsToRestore.map(a => {
					return uiAttachmentKey(a);
				})
			);
			if (requestedKeys.size === 0) {
				return;
			}

			setDirectoryGroups(prevGroups => {
				const groupIndex = prevGroups.findIndex(item => item.id === groupId);
				if (groupIndex === -1) {
					return prevGroups;
				}

				const group = prevGroups[groupIndex];
				const availableSlots = Math.max(0, MAX_FILES_PER_DIRECTORY - group.attachmentKeys.length);
				if (availableSlots === 0) {
					return prevGroups;
				}

				const restored = (group.removedAttachments ?? [])
					.filter(att => requestedKeys.has(uiAttachmentKey(att)))
					.slice(0, availableSlots);
				if (restored.length === 0) {
					return prevGroups;
				}

				const restoredKeys = restored.map(a => {
					return uiAttachmentKey(a);
				});
				const restoredKeySet = new Set(restoredKeys);
				const globallyExisting = new Set(attachmentsRef.current.map(uiAttachmentKey));
				const newlyOwnedKeys = restoredKeys.filter(key => !globallyExisting.has(key));
				const attachmentsToAdd = restored.filter(att => !globallyExisting.has(uiAttachmentKey(att)));

				if (attachmentsToAdd.length > 0) {
					setAttachments(prev => [...prev, ...attachmentsToAdd]);
				}

				const next = [...prevGroups];
				next[groupIndex] = {
					...group,
					attachmentKeys: mergeUniqueStrings(group.attachmentKeys, restoredKeys),
					ownedAttachmentKeys: mergeUniqueStrings(group.ownedAttachmentKeys, newlyOwnedKeys),
					removedAttachments: (group.removedAttachments ?? []).filter(att => !restoredKeySet.has(uiAttachmentKey(att))),
				};
				return next;
			});
		},
		[setAttachments, setDirectoryGroups]
	);

	const removeDirectoryGroup = useCallback(
		(groupId: string) => {
			setDirectoryGroups(prevGroups => {
				const groupToRemove = prevGroups.find(g => g.id === groupId);
				if (!groupToRemove) {
					return prevGroups;
				}

				const remainingGroups = prevGroups.filter(g => g.id !== groupId);

				const keysToDelete = new Set<string>();
				const ownershipTransfers = new Map<string, string[]>();

				for (const key of groupToRemove.ownedAttachmentKeys) {
					const recipient = remainingGroups.find(group => group.attachmentKeys.includes(key));
					if (!recipient) {
						keysToDelete.add(key);
						continue;
					}

					const transferred = ownershipTransfers.get(recipient.id) ?? [];
					transferred.push(key);
					ownershipTransfers.set(recipient.id, transferred);
				}

				if (keysToDelete.size > 0) {
					setAttachments(prevAttachments => prevAttachments.filter(att => !keysToDelete.has(uiAttachmentKey(att))));
				}

				return remainingGroups.map(group => {
					const transferredKeys = ownershipTransfers.get(group.id);
					if (!transferredKeys) {
						return group;
					}
					return Object.assign(group, {
						ownedAttachmentKeys: mergeUniqueStrings(group.ownedAttachmentKeys, transferredKeys),
					});
				});
			});
		},
		[setAttachments, setDirectoryGroups]
	);

	const removeOverflowDir = useCallback(
		(groupId: string, dirPath: string) => {
			setDirectoryGroups(prevGroups => {
				const updated = prevGroups.map(g =>
					g.id !== groupId
						? g
						: {
								...g,
								overflowDirs: g.overflowDirs.filter(od => od.dirPath !== dirPath),
							}
				);
				return updated.filter(
					g => g.attachmentKeys.length > 0 || g.removedAttachments.length > 0 || g.overflowDirs.length > 0
				);
			});
		},
		[setDirectoryGroups]
	);

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
		attachPathsAsAttachments,
		changeAttachmentMode,
		removeAttachment,
		removeDirectoryGroup,
		removeDirectoryAttachments,
		restoreDirectoryAttachments,
		removeOverflowDir,
		applyAttachmentsDrop,
		clearAttachments,
		loadAttachmentsFromMessage,
	};
}
