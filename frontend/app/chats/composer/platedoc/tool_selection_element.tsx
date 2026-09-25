import type { PlateElementProps } from 'platejs/react';

import type { ToolSelectionElementNode } from '@/chats/composer/platedoc/nodes';

/**
 * Hidden inline element; acts as a data carrier for one selected tool.
 * Chips are rendered in the bottom attachments bar, not inline in content.
 */
export function ToolSelectionElement(props: PlateElementProps<any>) {
	const { element, attributes, children } = props as any;
	const el = element as ToolSelectionElementNode;

	const display = el.overrides?.displayName ?? el.displayName ?? el.target.name;
	const label = `${el.collectionName ? `${el.collectionName}/` : ''}${el.target.name}${
		el.toolVersion ? `@${el.toolVersion}` : ''
	}`;

	return (
		<span
			{...attributes}
			contentEditable={false}
			data-tool-chip
			aria-hidden="true"
			title={`Tool: ${display} (${label})`}
			style={{
				position: 'absolute',
				width: 0,
				height: 0,
				padding: 0,
				margin: 0,
				overflow: 'hidden',
				border: 0,
				clipPath: 'inset(50%)',
				whiteSpace: 'nowrap',
			}}
		>
			{children}
		</span>
	);
}
