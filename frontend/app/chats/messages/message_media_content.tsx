// oxlint-disable react/only-export-components
import { FiExternalLink, FiFile, FiImage } from 'react-icons/fi';

import type { OutputUnion } from '@/spec/inference';
import { ContentItemKind, OutputKind } from '@/spec/inference';

import { backendAPI } from '@/apis/baseapi';

import { DownloadButton } from '@/components/download_button';

// oxlint-disable-next-line typescript/consistent-type-definitions
type ContentItemLike = {
	kind: string;
	imageItem?: {
		imageName?: string;
		imageMIME?: string;
		imageURL?: string;
		imageData?: string;
	};
	fileItem?: {
		fileName?: string;
		fileMIME?: string;
		fileURL?: string;
		fileData?: string;
	};
};

export interface MessageMediaItem {
	key: string;
	kind: 'image' | 'file';
	label: string;
	mimeType: string;
	url?: string;
	base64Data?: string;
}

function safeExternalURL(value: string | undefined): string | undefined {
	const raw = value?.trim();
	if (!raw) {
		return undefined;
	}

	try {
		const parsed = new URL(raw);
		if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') {
			return undefined;
		}
		return parsed.href;
	} catch {
		return undefined;
	}
}

function safeFilePrefix(value: string, fallback: string): string {
	const baseName = value
		.trim()
		.split(/[\\/]/)
		.at(-1)
		?.replace(/\.[^.]+$/u, '')
		.trim();

	return baseName || fallback;
}

function normalizedBase64(value: string): string {
	const trimmed = value.trim();
	if (!trimmed.startsWith('data:')) {
		return trimmed.replaceAll(/\s+/g, '');
	}

	const comma = trimmed.indexOf(',');
	return comma >= 0 ? trimmed.slice(comma + 1).replaceAll(/\s+/g, '') : '';
}

function base64Blob(value: string, mimeType: string): Blob {
	const raw = normalizedBase64(value);
	const binary = atob(raw);
	const bytes = new Uint8Array(binary.length);

	for (let index = 0; index < binary.length; index += 1) {
		bytes[index] = binary.charCodeAt(index);
	}

	return new Blob([bytes], {
		type: mimeType || 'application/octet-stream',
	});
}

function imageSource(item: MessageMediaItem): string | undefined {
	const externalURL = safeExternalURL(item.url);
	if (externalURL) {
		return externalURL;
	}

	if (!item.base64Data) {
		return undefined;
	}

	const data = item.base64Data.trim();
	if (data.startsWith('data:')) {
		return data;
	}

	return `data:${item.mimeType || 'image/png'};base64,${normalizedBase64(data)}`;
}

function mediaItemFromContent(item: ContentItemLike, key: string): MessageMediaItem | undefined {
	if (item.kind === (ContentItemKind.Image as string) || item.kind === 'image') {
		const image = item.imageItem;
		const url = safeExternalURL(image?.imageURL);
		const base64Data = image?.imageData?.trim();

		if (!url && !base64Data) {
			return undefined;
		}

		return {
			key,
			kind: 'image',
			label: image?.imageName?.trim() || 'Generated image',
			mimeType: image?.imageMIME?.trim() || 'image/png',
			url,
			base64Data,
		};
	}

	if (item.kind === (ContentItemKind.File as string) || item.kind === 'file') {
		const file = item.fileItem;
		const url = safeExternalURL(file?.fileURL);
		const base64Data = file?.fileData?.trim();

		if (!url && !base64Data) {
			return undefined;
		}

		return {
			key,
			kind: 'file',
			label: file?.fileName?.trim() || 'Generated file',
			mimeType: file?.fileMIME?.trim() || 'application/octet-stream',
			url,
			base64Data,
		};
	}

	return undefined;
}

export function mediaItemsFromOutputUnions(outputs: OutputUnion[] | undefined): MessageMediaItem[] {
	const items: MessageMediaItem[] = [];

	for (const [outputIndex, output] of (outputs ?? []).entries()) {
		if (output.kind !== OutputKind.OutputMessage || !output.outputMessage?.contents) {
			continue;
		}

		for (const [contentIndex, content] of output.outputMessage.contents.entries()) {
			const media = mediaItemFromContent(content, `model-output:${outputIndex}:${contentIndex}`);
			if (media) {
				items.push(media);
			}
		}
	}

	return items;
}

export function mediaItemsFromToolOutputItems(
	items: ContentItemLike[] | undefined,
	keyPrefix = 'tool-output'
): MessageMediaItem[] {
	const media: MessageMediaItem[] = [];

	for (const [index, item] of (items ?? []).entries()) {
		const value = mediaItemFromContent(item, `${keyPrefix}:${index}`);
		if (value) {
			media.push(value);
		}
	}

	return media;
}

function OpenExternalMediaButton({ url }: { url: string }) {
	return (
		<button
			type="button"
			className="btn btn-ghost btn-xs rounded-lg"
			onClick={() => {
				backendAPI.openURL(url);
			}}
			title="Open generated media"
			aria-label="Open generated media"
		>
			<FiExternalLink size={13} />
			Open
		</button>
	);
}

function DownloadGeneratedMediaButton({ item }: { item: MessageMediaItem }) {
	if (!item.base64Data) {
		return null;
	}

	return (
		<DownloadButton
			language=""
			isBinary
			size={13}
			fileprefix={safeFilePrefix(item.label, item.kind === 'image' ? 'generated-image' : 'generated-file')}
			valueFetcher={async () => base64Blob(item.base64Data ?? '', item.mimeType)}
			className="btn btn-ghost btn-xs rounded-lg"
			aria-label={`Download ${item.label}`}
			title={`Download ${item.label}`}
		/>
	);
}

export function MessageMediaContent({ items }: { items: MessageMediaItem[] }) {
	if (items.length === 0) {
		return null;
	}

	return (
		<section className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2" data-message-media aria-label="Images and files">
			{items.map(item => {
				if (item.kind === 'image') {
					const source = imageSource(item);
					if (!source) {
						return null;
					}

					return (
						<div key={item.key} className="border-base-300 bg-base-200 overflow-hidden rounded-xl border">
							<img src={source} alt={item.label} className="max-h-96 w-full object-contain" loading="lazy" />
							<div className="flex min-w-0 items-center gap-2 px-2 py-1">
								<FiImage size={13} className="shrink-0" />
								<span className="min-w-0 flex-1 truncate text-xs">{item.label}</span>
								{item.url ? <OpenExternalMediaButton url={item.url} /> : null}
								<DownloadGeneratedMediaButton item={item} />
							</div>
						</div>
					);
				}

				return (
					<div
						key={item.key}
						className="border-base-300 bg-base-200 flex min-w-0 items-center gap-2 rounded-xl border p-2"
					>
						<FiFile size={16} className="shrink-0" />
						<div className="min-w-0 flex-1">
							<div className="truncate text-xs font-medium">{item.label}</div>
							<div className="text-base-content/60 truncate text-[10px]">{item.mimeType}</div>
						</div>
						{item.url ? <OpenExternalMediaButton url={item.url} /> : null}
						<DownloadGeneratedMediaButton item={item} />
					</div>
				);
			})}
		</section>
	);
}
