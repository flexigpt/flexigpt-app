import type { ToolOutputItemUnion } from '@/spec/inference';
import type { ToolOutputUnion } from '@/spec/tool';
import { ContentItemKind, ImageDetail } from '@/spec/inference';
import { ToolOutputKind } from '@/spec/tool';

import { getPrettyToolName } from '@/tools/lib/tool_identity_utils';

/**
 * Map inference ToolOutput.contents -> ToolOutputUnion[]
 */
export function mapToolOutputItemsToToolOutputs(contents?: ToolOutputItemUnion[]): ToolOutputUnion[] | undefined {
	if (!contents?.length) {
		return undefined;
	}

	const outputs: ToolOutputUnion[] = [];

	for (const item of contents) {
		switch (item.kind) {
			case ContentItemKind.Text: {
				const text = item.textItem?.text;

				if (text !== null) {
					outputs.push({
						kind: ToolOutputKind.Text,
						textItem: { text: text ?? '' },
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

	return outputs.length > 0 ? outputs : undefined;
}

export function extractPrimaryTextFromToolOutputs(outputs?: ToolOutputUnion[]): string | undefined {
	if (!outputs?.length) {
		return undefined;
	}

	const texts = outputs
		.filter(o => o.kind === ToolOutputKind.Text && o.textItem?.text)
		.map(o => o.textItem?.text.trim())
		.filter(Boolean);

	if (texts.length === 0) {
		return undefined;
	}

	return texts.join('\n\n');
}

/**
 * Default summary label for a tool-output chip.
 */
export function formatToolOutputSummary(name: string): string {
	const pretty = getPrettyToolName(name);
	return `Result: ${pretty}`;
}
