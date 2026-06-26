import { createSlatePlugin, SingleBlockPlugin } from 'platejs';

import { AlignKit } from '@/components/editor/plugins/align_kit';
import { BasicBlocksKit } from '@/components/editor/plugins/basic_blocks_kit';
import { BasicMarksKit } from '@/components/editor/plugins/basic_marks_kit';
import { FloatingToolbarKit } from '@/components/editor/plugins/floating_toolbar_kit';
import { IndentKit } from '@/components/editor/plugins/indent_kit';
import { LineHeightKit } from '@/components/editor/plugins/line_height_kit';
import { ListKit } from '@/components/editor/plugins/list_kit';
import { TabbableKit } from '@/components/editor/plugins/tabbable_kit';

import { KEY_TEMPLATE_SELECTION, KEY_TEMPLATE_VARIABLE, KEY_TOOL_SELECTION } from '@/chats/composer/platedoc/nodes';
import { TemplateSelectionElement } from '@/chats/composer/platedoc/template_selection_element';
import { ToolSelectionElement } from '@/chats/composer/platedoc/tool_selection_element';
import { TemplateVariableElement } from '@/chats/composer/templates/template_variables_inline';

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
	...TemplateSlashKit,
	...ToolPlusKit,
	...FloatingToolbarKit,
];

const ToolSelectionPlugin = createSlatePlugin({
	key: KEY_TOOL_SELECTION,
	node: { isElement: true, isInline: true, isVoid: true, isSelectable: false },
	editOnly: true,
});

const ToolPlusKit = [ToolSelectionPlugin.withComponent(ToolSelectionElement)];

const TemplateSelectionPlugin = createSlatePlugin({
	key: KEY_TEMPLATE_SELECTION,
	// this is the chip “schema”
	node: { isElement: true, isInline: true, isVoid: true },
});

const TemplateVariablePlugin = createSlatePlugin({
	key: KEY_TEMPLATE_VARIABLE,
	node: { isElement: true, isInline: true, isVoid: true },
});

const TemplateSlashKit = [
	TemplateSelectionPlugin.withComponent(TemplateSelectionElement),
	TemplateVariablePlugin.withComponent(TemplateVariableElement),
];
