import type { PlateElementProps } from 'platejs/react';
import { PlateElement } from 'platejs/react';

export function BlockquoteElement(props: PlateElementProps) {
	return (
		<PlateElement
			as="blockquote"
			className="border-base-300 text-base-content/80 my-1 border-l-2 pl-6 italic"
			{...props}
		/>
	);
}
