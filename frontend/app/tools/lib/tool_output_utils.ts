import { ContentItemKind, ImageDetail, type ToolOutputItemUnion } from '@/spec/inference';
import { ToolOutputKind, type ToolOutputUnion } from '@/spec/tool';

import { getPrettyToolName } from '@/tools/lib/tool_identity_utils';

function mapImageDetail(detail?: string): ImageDetail | undefined {
	if (!detail) return undefined;
	switch (detail.toLowerCase()) {
		case 'high':
			return ImageDetail.High;
		case 'low':
			return ImageDetail.Low;
		case 'auto':
			return ImageDetail.Auto;
		default:
			return undefined;
	}
}

/**
 * Map inference ToolOutput.contents -> ToolOutputUnion[]
 */
export function mapToolOutputItemsToToolOutputs(contents?: ToolOutputItemUnion[]): ToolOutputUnion[] | undefined {
	if (!contents?.length) return undefined;

	const outputs: ToolOutputUnion[] = [];

	for (const item of contents) {
		switch (item.kind) {
			case ContentItemKind.Text: {
				const text = item.textItem?.text;
				if (text != null) {
					outputs.push({
						kind: ToolOutputKind.Text,
						textItem: { text },
					});
				}
				break;
			}

			case ContentItemKind.Image: {
				const img = item.imageItem;
				if (img) {
					outputs.push({
						kind: ToolOutputKind.Image,
						imageItem: {
							detail: (img.detail ?? ImageDetail.Auto) as string,
							imageName: img.imageName ?? '',
							imageMIME: img.imageMIME ?? '',
							imageData: img.imageData ?? '',
						},
					});
				}
				break;
			}

			case ContentItemKind.File: {
				const file = item.fileItem;
				if (file) {
					outputs.push({
						kind: ToolOutputKind.File,
						fileItem: {
							fileName: file.fileName ?? '',
							fileMIME: file.fileMIME ?? '',
							fileData: file.fileData ?? '',
						},
					});
				}
				break;
			}

			default:
				// Refusal / other kinds are ignored for tool-store outputs
				break;
		}
	}

	return outputs.length ? outputs : undefined;
}

/**
 * Map ToolOutputUnion[] -> inference ToolOutputItemUnion[]
 */
export function mapToolOutputsToToolOutputItems(outputs?: ToolOutputUnion[]): ToolOutputItemUnion[] | undefined {
	if (!outputs?.length) return undefined;

	const contents: ToolOutputItemUnion[] = [];

	for (const out of outputs) {
		switch (out.kind) {
			case ToolOutputKind.Text: {
				const text = out.textItem?.text;
				if (text != null) {
					contents.push({
						kind: ContentItemKind.Text,
						textItem: { text },
					});
				}
				break;
			}

			case ToolOutputKind.Image: {
				const img = out.imageItem;
				if (img) {
					contents.push({
						kind: ContentItemKind.Image,
						imageItem: {
							detail: mapImageDetail(img.detail),
							imageName: img.imageName,
							imageMIME: img.imageMIME,
							imageData: img.imageData,
						},
					});
				}
				break;
			}

			case ToolOutputKind.File: {
				const file = out.fileItem;
				if (file) {
					contents.push({
						kind: ContentItemKind.File,
						fileItem: {
							fileName: file.fileName,
							fileMIME: file.fileMIME,
							fileData: file.fileData,
						},
					});
				}
				break;
			}

			case ToolOutputKind.None:
			default:
				// ignore
				break;
		}
	}

	return contents.length ? contents : undefined;
}

export function extractPrimaryTextFromToolOutputs(outputs?: ToolOutputUnion[]): string | undefined {
	if (!outputs?.length) return undefined;

	const texts = outputs
		.filter(o => o.kind === ToolOutputKind.Text && o.textItem?.text)
		.map(o => o.textItem?.text.trim())
		.filter(Boolean);

	if (!texts.length) return undefined;

	return texts.join('\n\n');
}

/**
 * Default summary label for a tool-output chip.
 */
export function formatToolOutputSummary(name: string): string {
	const pretty = getPrettyToolName(name);
	return `Result: ${pretty}`;
}
