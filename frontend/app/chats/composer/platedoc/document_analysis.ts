import type { PlateEditor } from 'platejs/react';

import type { getFirstTemplateNodeWithPath } from '@/chats/composer/platedoc/templates/template_document_ops';

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
