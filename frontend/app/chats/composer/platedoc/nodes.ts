import type { TElement, TText } from 'platejs';

import type { ToolView, UIToolStoreChoice } from '@/spec/tool';

export const KEY_TOOL_SELECTION = 'toolSelection';

/**
 * Plate requires inserted element nodes to structurally satisfy TElement.
 * Extending TElement also gives the node the index signature expected by
 * Plate's generic transform APIs.
 */
export type ToolSelectionElementNode = TElement &
	UIToolStoreChoice & {
		type: typeof KEY_TOOL_SELECTION;
		toolSnapshot?: ToolView;
		overrides?: {
			displayName?: string;
			description?: string;
			tags?: string[];
		};

		// Inline void elements need one text child.
		children: [TText];
	};
