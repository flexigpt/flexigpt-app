import type { Tool, ToolStoreChoiceType } from '@/spec/tool';

export const KEY_TOOL_SELECTION = 'toolSelection';

// oxlint-disable-next-line typescript/consistent-type-definitions
export type ToolSelectionElementNode = {
	type: typeof KEY_TOOL_SELECTION;
	choiceID: string;
	bundleID: string;
	bundleSlug?: string;
	toolSlug: string;
	toolVersion: string;
	selectionID: string;
	toolType: ToolStoreChoiceType;
	autoExecute: boolean;
	userArgSchemaInstance?: string;

	toolSnapshot?: Tool;
	overrides?: {
		displayName?: string;
		description?: string;
		tags?: string[];
	};

	// inline+void node needs a text child
	children: [{ text: '' }];
};
