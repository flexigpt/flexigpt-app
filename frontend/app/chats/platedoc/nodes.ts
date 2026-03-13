import type { MessageBlock, PromptTemplate, PromptVariable } from '@/spec/prompt';
import type { Tool, ToolStoreChoiceType } from '@/spec/tool';

export const KEY_TOOL_SELECTION = 'toolSelection';

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

export const KEY_TEMPLATE_SELECTION = 'templateSelection';
export const KEY_TEMPLATE_VARIABLE = 'templateVariable';

export type TemplateVariableElementNode = {
	type: typeof KEY_TEMPLATE_VARIABLE;
	bundleID: string;
	templateSlug: string;
	templateVersion: string;
	selectionID: string;
	name: string;
	// for layout only (computed again at render)
	required?: boolean;
	children: [{ text: '' }];
};

export type TemplateSelectionElementNode = {
	type: typeof KEY_TEMPLATE_SELECTION;
	bundleID: string;
	templateSlug: string;
	templateVersion: string;
	selectionID: string;

	// User-provided variable values
	variables: Record<string, unknown>;

	// Captured template at insertion time
	templateSnapshot?: PromptTemplate;

	// Local per-chip overrides
	overrides?: {
		displayName?: string;
		description?: string;
		tags?: string[];
		blocks?: MessageBlock[];
		variables?: PromptVariable[];
	};

	// Slate text children
	children: [{ text: '' }];
};
