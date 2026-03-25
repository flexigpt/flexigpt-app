import type { ReactNode } from 'react';

import { Tooltip, TooltipAnchor, useTooltipStore } from '@ariakit/react';

type HoverTipPlacement =
	| 'top'
	| 'top-start'
	| 'top-end'
	| 'bottom'
	| 'bottom-start'
	| 'bottom-end'
	| 'left'
	| 'left-start'
	| 'left-end'
	| 'right'
	| 'right-start'
	| 'right-end';

interface HoverTipProps {
	content: ReactNode;
	children: ReactNode;
	placement?: HoverTipPlacement;
	gutter?: number;
	disabled?: boolean;
	wrapperClassName?: string;
	wrapperElement?: 'span' | 'div';
	tooltipClassName?: string;
}

export function HoverTip({
	content,
	children,
	placement = 'top',
	gutter = 8,
	disabled = false,
	wrapperClassName = 'inline-flex max-w-full',
	wrapperElement = 'span',
	tooltipClassName = '',
}: HoverTipProps) {
	const tooltip = useTooltipStore({ placement });
	const hasContent = !(content == null || (typeof content === 'string' && content.trim() === ''));

	if (!hasContent || disabled) {
		return <>{children}</>;
	}

	const anchor =
		wrapperElement === 'div' ? <div className={wrapperClassName} /> : <span className={wrapperClassName} />;

	return (
		<>
			<TooltipAnchor store={tooltip} render={anchor}>
				{children}
			</TooltipAnchor>
			<Tooltip
				store={tooltip}
				gutter={gutter}
				portal
				className={`rounded-box bg-base-100 text-base-content border-base-300 z-1000 max-w-xs border px-3 py-2 text-xs leading-4 whitespace-pre-line shadow-xl ${tooltipClassName}`}
			>
				{content}
			</Tooltip>
		</>
	);
}
