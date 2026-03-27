import { createSlatePlugin, SingleBlockPlugin } from 'platejs';
import type { PlateElementProps } from 'platejs/react';

import { AlignKit } from '@/components/editor/plugins/align_kit';
import { BasicBlocksKit } from '@/components/editor/plugins/basic_blocks_kit';
import { BasicMarksKit } from '@/components/editor/plugins/basic_marks_kit';
import { FloatingToolbarKit } from '@/components/editor/plugins/floating_toolbar_kit';
import { IndentKit } from '@/components/editor/plugins/indent_kit';
import { LineHeightKit } from '@/components/editor/plugins/line_height_kit';
import { ListKit } from '@/components/editor/plugins/list_kit';
import { TabbableKit } from '@/components/editor/plugins/tabbable_kit';

import {
	KEY_TEMPLATE_SELECTION,
	KEY_TEMPLATE_VARIABLE,
	KEY_TOOL_SELECTION,
	type TemplateSelectionElementNode,
	type ToolSelectionElementNode,
} from '@/chats/composer/platedoc/nodes';
import { computeEffectiveTemplate, computeRequirements } from '@/chats/composer/templates/template_processing';
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

/**
 * Hidden inline element; acts as a data carrier for one selected tool.
 * Chips are rendered in the bottom attachments bar, not inline in content.
 */
function ToolSelectionElement(props: PlateElementProps<any>) {
	const { element, attributes, children } = props as any;
	const el = element as ToolSelectionElementNode;

	const display = el.overrides?.displayName ?? el.toolSnapshot?.displayName ?? el.toolSlug;
	const slug = `${el.bundleSlug ?? el.bundleID}/${el.toolSlug}@${el.toolVersion}`;

	return (
		<span
			{...attributes}
			contentEditable={false}
			data-tool-chip
			aria-hidden="true"
			title={`Tool: ${display} • ${slug}`}
			// Absolutely position and zero-size so it contributes no line height.
			style={{
				position: 'absolute',
				width: 0,
				height: 0,
				padding: 0,
				margin: 0,
				overflow: 'hidden',
				border: 0,
				clip: 'rect(0 0 0 0)',
				whiteSpace: 'nowrap',
			}}
		>
			{children}
		</span>
	);
}

const ToolSelectionPlugin = createSlatePlugin({
	key: KEY_TOOL_SELECTION,
	node: { isElement: true, isInline: true, isVoid: true, isSelectable: false },
	editOnly: true,
});

const ToolPlusKit = [ToolSelectionPlugin.withComponent(ToolSelectionElement)];

/**
 * Template selection element (data carrier).
 * We render it as a hidden inline element so it doesn't affect the text layout.
 * The toolbar is responsible for user-facing controls.
 */
function TemplateSelectionElement(props: PlateElementProps<any>) {
	const { element, attributes, children } = props as any;
	const el = element as TemplateSelectionElementNode;

	// We still compute badges for accessibility/title, but hide it from visual flow.
	const { template, variablesSchema } = computeEffectiveTemplate(el);
	const req = computeRequirements(variablesSchema, el.variables);

	return (
		<span
			{...attributes}
			contentEditable={false}
			className="pointer-events-none sr-only"
			data-template-chip
			title={`Template: ${el.overrides?.displayName ?? template?.displayName ?? el.templateSlug} • pending vars: ${req.requiredCount}`}
			aria-hidden="true"
		>
			{/* Invisible info holder */}
			<span className="sr-only">{el.overrides?.displayName ?? template?.displayName ?? el.templateSlug}</span>
			{children}
		</span>
	);
}

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
