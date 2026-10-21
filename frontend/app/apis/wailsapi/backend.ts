// oxlint-disable max-classes-per-file
import { Application, Browser, Window } from '@wailsio/runtime';
import { sprintf } from 'sprintf-js';

import type { Attachment, DirectoryAttachmentsResult, FileFilter, PathAttachmentsResult } from '@/spec/attachment';
import type { ApplyUnifiedDiffArgs, ApplyUnifiedDiffOut } from '@/spec/unified_diff';

import type { IBackendAPI, ILogger } from '@/apis/interface';
import {
	attachmentFromWails,
	directoryAttachmentsResultFromWails,
	pathAttachmentsResultFromWails,
} from '@/apis/wailsapi/attachment_projection';
import {
	requireNonBlankString,
	requireWailsBody,
	requireWailsBoolean,
	requireWailsString,
	wailsArrayOrEmpty,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	ApplyUnifiedDiff,
	GetAppVersion,
	GetPathsAsAttachments,
	OpenDirectoryAsAttachments,
	OpenMultipleFilesAsAttachments,
	OpenURLAsAttachment,
	PickDirectory,
	PickFiles,
	Ping,
	SaveFile,
} from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/cmd/agentgo/app';
import {
	Debug as FrontendLogDebug,
	Error as FrontendLogError,
	Info as FrontendLogInfo,
	Warn as FrontendLogWarn,
} from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/cmd/agentgo/logservice';

function stringifyLogArg(arg: unknown): string {
	if (typeof arg === 'string') {
		return arg;
	}

	if (arg instanceof Error) {
		return arg.stack || `${arg.name}: ${arg.message}`;
	}

	try {
		const serialized = JSON.stringify(arg);
		return serialized ?? String(arg);
	} catch {
		return String(arg);
	}
}
function formatMessage(args: unknown[]): string {
	if (args.length === 0) {
		return '';
	}

	if (args.length > 0 && typeof args[0] === 'string') {
		if (args.length === 1) {
			return args[0];
		}

		try {
			return sprintf(args[0], ...args.slice(1));
		} catch (error: unknown) {
			console.error(`Error formatting message: ${JSON.stringify(error, null, 2)}`);
			return args.map(a => stringifyLogArg(a)).join(' ');
		}
	}
	return args.map(a => stringifyLogArg(a)).join(' ');
}

function toWailsFileFilters(filters?: readonly FileFilter[]): Array<{ DisplayName: string; Extensions: string[] }> {
	return (filters ?? []).map(filter => ({
		DisplayName: filter.DisplayName,
		Extensions: [...filter.Extensions],
	}));
}

function submitFrontendLog(operation: string, request: PromiseLike<void>): void {
	void Promise.resolve(request).catch((error: unknown) => {
		console.error(`Failed to write frontend ${operation} log`, error);
	});
}

function writeFrontendLog(level: 'debug' | 'error' | 'info' | 'warn', args: unknown[]): void {
	const message = formatMessage(args);

	if (!message) {
		return;
	}

	switch (level) {
		case 'debug':
			submitFrontendLog('debug', FrontendLogDebug(message));
			return;
		case 'error':
			// oxlint-disable-next-line unicorn/throw-new-error
			submitFrontendLog('error', FrontendLogError(message));
			return;
		case 'warn':
			submitFrontendLog('warn', FrontendLogWarn(message));
			return;
		case 'info':
			submitFrontendLog('info', FrontendLogInfo(message));
	}
}

export class WailsLogger implements ILogger {
	log(...args: unknown[]): void {
		writeFrontendLog('info', args);
	}

	error(...args: unknown[]): void {
		writeFrontendLog('error', args);
	}

	info(...args: unknown[]): void {
		writeFrontendLog('info', args);
	}

	debug(...args: unknown[]): void {
		writeFrontendLog('debug', args);
	}

	warn(...args: unknown[]): void {
		writeFrontendLog('warn', args);
	}
}

export class WailsBackendAPI implements IBackendAPI {
	appQuit(): void {
		void Application.Quit();
	}

	appWindowMinimise(): void {
		void Window.Minimise();
	}

	appWindowToggleMaximise(): void {
		void Window.ToggleMaximise();
	}

	async isAppWindowMaximised(): Promise<boolean> {
		return requireWailsBoolean(await Window.IsMaximised(), 'Window.IsMaximised');
	}

	async getAppVersion(): Promise<string> {
		return requireWailsString(await GetAppVersion(), 'GetAppVersion');
	}

	async ping(): Promise<string> {
		return requireWailsString(await Ping(), 'Ping');
	}

	async pickDirectoryPath(): Promise<string | undefined> {
		const result = await PickDirectory();
		if (result === null || result === undefined) {
			return undefined;
		}

		const path = requireWailsString(result, 'PickDirectory').trim();
		return path || undefined;
	}

	async pickFilePaths(allowMultiple: boolean): Promise<string[]> {
		const paths = await PickFiles(allowMultiple);

		return wailsArrayOrEmpty(paths, 'PickFiles')
			.map((path, index) => requireWailsString(path, `PickFiles[${index}]`).trim())
			.filter(Boolean);
	}

	// Implement the log method
	log(level: string, ...args: unknown[]): void {
		switch (level.toLowerCase()) {
			case 'debug':
				writeFrontendLog('debug', args);
				return;
			case 'error':
				writeFrontendLog('error', args);
				return;
			case 'warn':
			case 'warning':
				writeFrontendLog('warn', args);
				return;
			default:
				writeFrontendLog('info', args);
		}
	}

	openURL(url: string): void {
		void Browser.OpenURL(requireNonBlankString(url, 'url'));
	}

	async openURLAsAttachment(rawURL: string): Promise<Attachment | undefined> {
		const attachment = await OpenURLAsAttachment(rawURL);
		if (attachment === null || attachment === undefined) {
			return undefined;
		}

		return attachmentFromWails(attachment, 'OpenURLAsAttachment');
	}

	async saveFile(defaultFilename: string, contentBase64: string, additionalFilters?: Array<FileFilter>): Promise<void> {
		await SaveFile(
			defaultFilename,
			contentBase64,
			toWailsFileFilters(additionalFilters) as unknown as Parameters<typeof SaveFile>[2]
		);
	}

	async openMultipleFilesAsAttachments(
		allowMultiple: boolean,
		additionalFilters?: Array<FileFilter>
	): Promise<Attachment[]> {
		const attachments = await OpenMultipleFilesAsAttachments(
			allowMultiple,
			toWailsFileFilters(additionalFilters) as unknown as Parameters<typeof OpenMultipleFilesAsAttachments>[1]
		);
		return wailsObjectArrayOrEmpty(attachments, 'OpenMultipleFilesAsAttachments').map((attachment, index) =>
			attachmentFromWails(attachment, `OpenMultipleFilesAsAttachments[${index}]`)
		);
	}

	async openDirectoryAsAttachments(maxFiles: number): Promise<DirectoryAttachmentsResult> {
		return directoryAttachmentsResultFromWails(
			await OpenDirectoryAsAttachments(maxFiles),
			'OpenDirectoryAsAttachments'
		);
	}

	async getPathsAsAttachments(paths: string[], maxFilesPerDir: number): Promise<PathAttachmentsResult> {
		return pathAttachmentsResultFromWails(await GetPathsAsAttachments(paths, maxFilesPerDir), 'GetPathsAsAttachments');
	}

	async applyUnifiedDiff(args: ApplyUnifiedDiffArgs): Promise<ApplyUnifiedDiffOut> {
		const resp = await ApplyUnifiedDiff(args as Parameters<typeof ApplyUnifiedDiff>[0]);
		return requireWailsBody(resp as ApplyUnifiedDiffOut | null | undefined, 'ApplyUnifiedDiff');
	}
}
