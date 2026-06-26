import type { ComponentProps } from 'react';

import type { FloatingToolbarState } from '@platejs/floating';
import { flip, offset, useFloatingToolbar, useFloatingToolbarState } from '@platejs/floating';
import { cn, useComposedRef } from '@udecode/cn';
import { KEYS } from 'platejs';
import { useEditorId, useEventEditorValue, usePluginOption } from 'platejs/react';

import { Toolbar } from '@/components/editor/nodes/toolbar';

export function FloatingToolbar({
	children,
	className,
	state,
	...props
}: ComponentProps<typeof Toolbar> & {
	state?: FloatingToolbarState;
}) {
	const editorId = useEditorId();
	const focusedEditorId = useEventEditorValue('focus');
	const isFloatingLinkOpen = !!usePluginOption({ key: KEYS.link }, 'mode');
	const isAIChatOpen = usePluginOption({ key: KEYS.aiChat }, 'open');

	const floatingToolbarState = useFloatingToolbarState({
		editorId,
		focusedEditorId,
		hideToolbar: isFloatingLinkOpen || isAIChatOpen,
		...state,
		floatingOptions: {
			middleware: [
				offset(12),
				flip({
					fallbackPlacements: ['top-start', 'top-end', 'bottom-start', 'bottom-end'],
					padding: 12,
				}),
			],
			placement: 'top',
			...state?.floatingOptions,
		},
	});

	const { clickOutsideRef, hidden, props: rootProps, ref: floatingRef } = useFloatingToolbar(floatingToolbarState);

	// Keep composed ref behavior (props.ref from parent + floatingRef)
	const ref = useComposedRef<HTMLDivElement>(props.ref, floatingRef);

	if (hidden) {
		return null;
	}

	return (
		<div ref={clickOutsideRef}>
			<Toolbar
				{...props}
				{...rootProps}
				ref={ref}
				className={cn(
					// Positioning/box
					'absolute p-1 whitespace-nowrap',
					// DaisyUI surface
					'border-base-300 bg-base-100 rounded-xl border shadow-sm',
					// Misc
					'print:hidden',
					className
				)}
			>
				{children}
			</Toolbar>
		</div>
	);
}
