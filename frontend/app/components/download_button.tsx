// DownloadButton.tsx
import type { ButtonHTMLAttributes } from 'react';
import { useEffect, useRef, useState } from 'react';
import { FiDownload } from 'react-icons/fi';

import type { FileFilter } from '@/spec/attachment';

import { Base64EncodeUTF8, GenerateRandomString } from '@/lib/encode_decode';
import { MimeTypeMap, ProgrammingLanguages } from '@/lib/extension_utils';

import { backendAPI } from '@/apis/baseapi';

export interface DownloadButtonValue {
	content: string | Blob;
	fileName?: string;
	extension?: string;
	filterDisplayName?: string;
}

type DownloadButtonContent = string | Blob | DownloadButtonValue | null | undefined;

interface DownloadButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
	language?: string;
	valueFetcher: () => Promise<DownloadButtonContent>;
	size: number;
	fileprefix?: string;
	isBinary?: boolean;
	label?: string;
	downloadingLabel?: string;
	onDownloaded?: () => void;
	onDownloadError?: (error: unknown) => void;
}

function normalizeExtension(extension?: string) {
	return extension?.replace(/^\./, '') ?? '';
}

function getFileNameExtension(fileName?: string) {
	const match = fileName?.match(/\.([^.]+)$/);
	return match?.[1] ?? '';
}

function isDownloadButtonValue(value: string | Blob | DownloadButtonValue): value is DownloadButtonValue {
	return typeof value === 'object' && value !== null && 'content' in value;
}

export function DownloadButton({
	language = '',
	valueFetcher,
	size,
	fileprefix = 'file',
	isBinary = false,
	label,
	downloadingLabel = 'Downloading',
	onDownloaded,
	onDownloadError,
	children,
	disabled = false,
	onClick,
	'aria-label': ariaLabel,
	title,
	...buttonProps
}: DownloadButtonProps) {
	const [downloading, setDownloading] = useState(false);
	const mountedRef = useRef(false);

	useEffect(() => {
		mountedRef.current = true;
		return () => {
			mountedRef.current = false;
		};
	}, []);

	const downloadAsFile = async () => {
		if (disabled || downloading) {
			return;
		}

		setDownloading(true);
		try {
			const fetchedValue = await valueFetcher();
			if (fetchedValue === null || fetchedValue === undefined) {
				return;
			}

			const download = isDownloadButtonValue(fetchedValue) ? fetchedValue : { content: fetchedValue };
			const content = download.content;
			let contentBase64: string;
			let fileExtension: string;
			fileExtension = normalizeExtension(download.extension) || getFileNameExtension(download.fileName);

			if (isBinary && content instanceof Blob) {
				const mimeType = content.type || 'application/octet-stream';
				contentBase64 = await new Promise<string>((resolve, reject) => {
					const reader = new FileReader();
					reader.onloadend = () => {
						const dataUrl = reader.result;
						if (typeof dataUrl !== 'string') {
							reject(new TypeError('Binary content could not be encoded.'));
							return;
						}

						const base64Data = dataUrl.split(',')[1];
						if (base64Data === null || base64Data === undefined) {
							reject(new TypeError('Binary content could not be encoded.'));
							return;
						}
						resolve(base64Data);
					};
					reader.onerror = reject;
					reader.readAsDataURL(content);
				});
				if (!fileExtension) {
					fileExtension = normalizeExtension(MimeTypeMap[mimeType]);
				}
			} else if (typeof content === 'string') {
				const languageKey = language.toLowerCase();
				const langInfo =
					languageKey in ProgrammingLanguages
						? ProgrammingLanguages[languageKey]
						: {
								extension: '.txt',
								mimeType: 'text/plain',
							};
				if (!fileExtension) {
					fileExtension = normalizeExtension(langInfo.extension);
				}
				contentBase64 = Base64EncodeUTF8(content);
			} else {
				throw new TypeError('Unsupported content type for download');
			}

			const extensionSuffix = fileExtension ? `.${fileExtension}` : '';
			const suggestedFileName = download.fileName || `${fileprefix}-${GenerateRandomString(3, true)}${extensionSuffix}`;
			const additionalFilters: FileFilter[] = fileExtension
				? [
						{
							DisplayName: `${download.filterDisplayName ?? 'Files'} (*.${fileExtension})`,
							Extensions: [fileExtension],
						},
					]
				: [];

			await backendAPI.saveFile(suggestedFileName, contentBase64, additionalFilters);
			if (mountedRef.current) {
				onDownloaded?.();
			}
		} catch (error: unknown) {
			console.error('Download failed:', error);
			if (mountedRef.current) {
				onDownloadError?.(error);
			}
		} finally {
			if (mountedRef.current) {
				setDownloading(false);
			}
		}
	};

	const buttonLabel = downloading ? downloadingLabel : label;

	return (
		<button
			{...buttonProps}
			type="button"
			aria-label={ariaLabel ?? buttonLabel ?? 'Download'}
			title={title ?? buttonLabel ?? 'Download'}
			disabled={disabled || downloading}
			onClick={event => {
				onClick?.(event);
				if (!event.defaultPrevented) {
					void downloadAsFile();
				}
			}}
		>
			{downloading ? (
				<span className="loading loading-spinner loading-xs" aria-hidden="true" />
			) : (
				<FiDownload size={size} aria-hidden="true" />
			)}
			{children ?? (label ? <span>{buttonLabel}</span> : null)}
		</button>
	);
}
