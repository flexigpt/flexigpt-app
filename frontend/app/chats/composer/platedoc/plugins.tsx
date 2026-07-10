import { createSlatePlugin, SingleBlockPlugin } from 'platejs';

import { AlignKit } from '@/components/editor/plugins/align_kit';
import { BasicBlocksKit } from '@/components/editor/plugins/basic_blocks_kit';
import { BasicMarksKit } from '@/components/editor/plugins/basic_marks_kit';
import { FloatingToolbarKit } from '@/components/editor/plugins/floating_toolbar_kit';
import { IndentKit } from '@/components/editor/plugins/indent_kit';
import { LineHeightKit } from '@/components/editor/plugins/line_height_kit';
import { ListKit } from '@/components/editor/plugins/list_kit';
import { TabbableKit } from '@/components/editor/plugins/tabbable_kit';

import { KEY_TOOL_SELECTION } from '@/chats/composer/platedoc/nodes';
import { ToolSelectionElement } from '@/chats/composer/platedoc/tool_selection_element';

export const createComposerEditorPlugins = () => [
	SingleBlockPlugin,
	...BasicBlocksKit,
	...BasicMarksKit,
	...LineHeightKit,
	...AlignKit,
	...IndentKit,
	...ListKit,
	// ...AutoformatKit, // Don't want any formatting on typing
	...TabbableKit,
	...ToolPlusKit,
	...FloatingToolbarKit,
];

const ToolSelectionPlugin = createSlatePlugin({
	key: KEY_TOOL_SELECTION,
	node: { isElement: true, isInline: true, isVoid: true, isSelectable: false },
	editOnly: true,
});

const ToolPlusKit = [ToolSelectionPlugin.withComponent(ToolSelectionElement)];
