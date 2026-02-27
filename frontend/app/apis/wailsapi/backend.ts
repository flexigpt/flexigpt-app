import { sprintf } from 'sprintf-js';

import type { Attachment, DirectoryAttachmentsResult, FileFilter, PathAttachmentsResult } from '@/spec/attachment';

import type { IBackendAPI, ILogger } from '@/apis/interface';
import {
	GetAppVersion,
	GetPathsAsAttachments,
	OpenDirectoryAsAttachments,
	OpenMultipleFilesAsAttachments,
	OpenURLAsAttachment,
	Ping,
	SaveFile,
} from '@/apis/wailsjs/go/main/App';
import {
	BrowserOpenURL,
	LogDebug,
	LogError,
	LogInfo,
	LogWarning,
	Quit,
	WindowIsMaximised,
	WindowMinimise,
	WindowToggleMaximise,
} from '@/apis/wailsjs/runtime/runtime';

function formatMessage(args: unknown[]): string {
	if (args.length > 0 && typeof args[0] === 'string') {
		try {
			return sprintf(args[0], ...args.slice(1));
		} catch (error) {
			console.error(`Error formatting message: ${error}`);
			return '';
		}
	} else {
		console.error(`Invalid format string or arguments: ${args}`);
		return '';
	}
}

function isRuntimeAvailable(): boolean {
	return typeof window !== 'undefined' && 'runtime' in window;
}

/**
 * @public
 */
export class WailsLogger implements ILogger {
	log(...args: unknown[]): void {
		const msg = formatMessage(args);
		if (isRuntimeAvailable()) {
			if (msg) {
				LogInfo(msg);
			}
		} else {
			console.log(...args);
		}
	}

	error(...args: unknown[]): void {
		const msg = formatMessage(args);
		if (isRuntimeAvailable()) {
			if (msg) {
				LogError(msg);
			}
		} else {
			console.error(...args);
		}
	}

	info(...args: unknown[]): void {
		const msg = formatMessage(args);
		if (isRuntimeAvailable()) {
			if (msg) {
				LogInfo(msg);
			}
		} else {
			console.info(...args);
		}
	}

	debug(...args: unknown[]): void {
		const msg = formatMessage(args);
		if (isRuntimeAvailable()) {
			if (msg) {
				LogDebug(msg);
			}
		} else {
			console.debug(...args);
		}
	}

	warn(...args: unknown[]): void {
		const msg = formatMessage(args);
		if (isRuntimeAvailable()) {
			if (msg) {
				LogWarning(msg);
			}
		} else {
			console.warn(...args);
		}
	}
}

/**
 * @public
 */
export class WailsBackendAPI implements IBackendAPI {
	appQuit(): void {
		Quit();
	}

	appWindowMinimise(): void {
		WindowMinimise();
	}

	appWindowToggleMaximise(): void {
		WindowToggleMaximise();
	}

	async isAppWindowMaximised(): Promise<boolean> {
		return WindowIsMaximised();
	}

	async getAppVersion(): Promise<string> {
		return GetAppVersion();
	}

	ping(): Promise<string> {
		return Ping();
	}

	// Implement the log method
	log(level: string, ...args: unknown[]): void {
		const msg = formatMessage(args);
		if (isRuntimeAvailable()) {
			if (msg) {
				switch (level) {
					case 'info':
						LogInfo(msg);
						break;
					case 'error':
						LogError(msg);
						break;
					case 'debug':
						LogDebug(msg);
						break;
					case 'warn':
						LogWarning(msg);
						break;
				}
			}
		} else {
			console.log(...args);
		}
	}

	openURL(url: string): void {
		BrowserOpenURL(url);
	}

	async openURLAsAttachment(rawURL: string): Promise<Attachment | undefined> {
		try {
			const att = await OpenURLAsAttachment(rawURL);
			return att as Attachment;
		} catch (err) {
			console.error('Error saving file:', err);
		}
		return undefined;
	}

	async saveFile(defaultFilename: string, contentBase64: string, additionalFilters?: Array<FileFilter>): Promise<void> {
		// Call the Go backend method to save the file
		try {
			await SaveFile(defaultFilename, contentBase64, additionalFilters ?? []);
		} catch (err) {
			console.error('Error saving file:', err);
		}
		return;
	}

	async openMultipleFilesAsAttachments(
		allowMultiple: boolean,
		additionalFilters?: Array<FileFilter>
	): Promise<Attachment[]> {
		try {
			const attachments = await OpenMultipleFilesAsAttachments(allowMultiple, additionalFilters ?? []);

			return (attachments as Attachment[]) ?? [];
		} catch (err) {
			console.error('Error opening file dialog:', err);
			return [];
		}
	}

	async openDirectoryAsAttachments(maxFiles: number): Promise<DirectoryAttachmentsResult> {
		try {
			const dirResults = await OpenDirectoryAsAttachments(maxFiles);

			return (dirResults as DirectoryAttachmentsResult) ?? [];
		} catch (err) {
			console.error('Error opening dir dialog:', err);
			const res: DirectoryAttachmentsResult = {
				dirPath: '',
				attachments: [],
				overflowDirs: [],
				maxFiles: 0,
				totalSize: 0,
				hasMore: false,
			};
			return res;
		}
	}

	async getPathsAsAttachments(paths: string[], maxFilesPerDir: number): Promise<PathAttachmentsResult> {
		const pathResults = await GetPathsAsAttachments(paths, maxFilesPerDir);
		return pathResults as PathAttachmentsResult;
	}
}
