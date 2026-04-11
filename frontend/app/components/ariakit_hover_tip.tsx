import type { FocusEventHandler, MouseEventHandler, ReactNode } from 'react';

import { Tooltip, useTooltipStore } from '@ariakit/react';

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
	overflowPadding?: number;
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
	overflowPadding = 8,
	disabled = false,
	wrapperClassName = 'inline-flex max-w-full',
	wrapperElement = 'span',
	tooltipClassName = '',
}: HoverTipProps) {
	const tooltip = useTooltipStore({ placement });
	const Wrapper = wrapperElement ?? 'span';
	const hasContent = !(content == null || (typeof content === 'string' && content.trim() === ''));

	const hideTip = () => {
		tooltip.hide();
		tooltip.setAnchorElement(null);
	};

	const showForCurrentTarget: MouseEventHandler<HTMLElement> = event => {
		tooltip.setAnchorElement(event.currentTarget);
		tooltip.show();
	};

	const showForFocusedTarget: FocusEventHandler<HTMLElement> = event => {
		tooltip.setAnchorElement(event.target as HTMLElement);
		tooltip.show();
	};

	const hideOnBlurCapture: FocusEventHandler<HTMLElement> = event => {
		const nextTarget = event.relatedTarget as Node | null;
		if (nextTarget && event.currentTarget.contains(nextTarget)) {
			return;
		}
		hideTip();
	};

	if (!hasContent || disabled) {
		return <>{children}</>;
	}

	return (
		<>
			<Wrapper
				className={wrapperClassName}
				onMouseEnter={showForCurrentTarget}
				onMouseLeave={hideTip}
				onFocusCapture={showForFocusedTarget}
				onBlurCapture={hideOnBlurCapture}
			>
				{children}
			</Wrapper>
			<Tooltip
				store={tooltip}
				gutter={gutter}
				overflowPadding={overflowPadding}
				portal
				className={`rounded-box bg-base-100 text-base-content border-base-300 z-1000 max-w-xs border px-3 py-2 text-xs leading-4 whitespace-pre-line shadow-xl ${tooltipClassName}`}
			>
				{content}
			</Tooltip>
		</>
	);
}
