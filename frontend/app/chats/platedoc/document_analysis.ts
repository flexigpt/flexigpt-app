import type { PlateEditor } from 'platejs/react';

import { getFirstTemplateNodeWithPath, getTemplateSelections } from '@/chats/platedoc/template_document_ops';

export interface ComposerDocumentSelectionInfo {
	tplNodeWithPath: ReturnType<typeof getFirstTemplateNodeWithPath>;
	hasTemplate: boolean;
	requiredCount: number;
	firstPendingVar: { name: string; selectionID?: string } | undefined;
}

export const isSelectionOnlyEditorChange = (editor: PlateEditor): boolean => {
	const operations = editor.operations ?? [];
	return operations.length > 0 && operations.every(op => op.type === 'set_selection');
};

export function analyzeTemplateSelectionInfo(editor: PlateEditor): ComposerDocumentSelectionInfo {
	// Fast path: if the document contains no template-selection elements at all,
	// short-circuit instead of running the heavier helpers.
	const tplNodeWithPath = getFirstTemplateNodeWithPath(editor);
	if (!tplNodeWithPath) {
		return {
			tplNodeWithPath: undefined,
			hasTemplate: false,
			requiredCount: 0,
			firstPendingVar: undefined,
		};
	}

	const selections = getTemplateSelections(editor);
	const hasTemplate = selections.length > 0;

	let requiredCount = 0;
	let firstPendingVar: { name: string; selectionID?: string } | undefined = undefined;

	for (const s of selections) {
		requiredCount += s.requiredCount;

		if (!firstPendingVar && s.requiredVariables.length > 0) {
			firstPendingVar = {
				name: s.requiredVariables[0],
				selectionID: s.selectionID,
			};
		}
	}

	return {
		tplNodeWithPath,
		hasTemplate,
		requiredCount,
		firstPendingVar,
	};
}
