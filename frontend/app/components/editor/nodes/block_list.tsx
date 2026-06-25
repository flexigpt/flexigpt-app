// oxlint-disable-next-line no-restricted-imports
import type { FC } from 'react';

import { isOrderedList } from '@platejs/list';
import { useTodoListElement, useTodoListElementState } from '@platejs/list/react';
import type { TListElement } from 'platejs';
import { type PlateElementProps, type RenderNodeWrapper, useReadOnly } from 'platejs/react';

// Tiny helper to merge class names (no external deps)
function cx(...classes: Array<string | undefined | null | false>) {
	return classes.filter(Boolean).join(' ');
}

const config: Record<
	string,
	{
		Li: FC<PlateElementProps>;
		Marker: FC<PlateElementProps>;
	}
> = {
	todo: {
		Li: TodoLi,
		Marker: TodoMarker,
	},
};

function List(props: PlateElementProps) {
	const { listStart, listStyleType } = props.element as TListElement;
	const { Li, Marker } = config[listStyleType] ?? {};
	const OList = isOrderedList(props.element) ? 'ol' : 'ul';

	return (
		<OList className="relative m-1 p-1 pl-2" style={{ listStyleType }} start={listStart}>
			{Marker && <Marker {...props} />}
			{Li ? <Li {...props} /> : <li>{props.children}</li>}
		</OList>
	);
}

// oxlint-disable-next-line react/display-name
export const BlockList: RenderNodeWrapper = props => {
	if (!props.element.listStyleType) return;

	// oxlint-disable-next-line no-shadow
	return props => <List {...props} />;
};

/**
 * @public
 */
export function TodoMarker(props: PlateElementProps) {
	const state = useTodoListElementState({ element: props.element });
	const { checkboxProps } = useTodoListElement(state);
	const readOnly = useReadOnly();

	return (
		<div contentEditable={false}>
			<input
				type="checkbox"
				className={cx('checkbox checkbox-sm checkbox-primary absolute top-1 -left-6')}
				disabled={readOnly}
				aria-disabled={readOnly || undefined}
				tabIndex={readOnly ? -1 : 0}
				{...checkboxProps}
			/>
		</div>
	);
}

/**
 * @public
 */
export function TodoLi(props: PlateElementProps) {
	const isChecked =
		typeof props.element === 'object' &&
		'checked' in props.element &&
		Boolean((props.element as { checked?: boolean }).checked);

	return <li className={cx('list-none', isChecked && 'line-through opacity-60')}>{props.children}</li>;
}
