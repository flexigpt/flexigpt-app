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

interface HoverTipSection {
	id: string;
	title: ReactNode;
	items: ReactNode[];
}

interface HoverTipContentProps {
	title: ReactNode;
	description: ReactNode;
	sections: HoverTipSection[];
}

export function HoverTipContent({ title, description, sections }: HoverTipContentProps) {
	return (
		<div className="space-y-2 whitespace-normal">
			<div className="text-xs/4 font-semibold">{title}</div>

			<div className="border-info/30 bg-info/10 text-base-content/80 rounded-lg border px-2 py-1.5 text-[11px]/4">
				<div>{description}</div>
			</div>

			{sections.map(section => (
				<section key={section.id} className="border-base-300 border-t pt-2">
					<div className="text-base-content/70 text-[10px] font-semibold tracking-wide uppercase">{section.title}</div>
					<ul className="mt-1 space-y-1">
						{section.items.map((item, index) => (
							<li key={`${section.id}-${index}`} className="flex items-start gap-1.5 text-[11px]/4">
								<span className="text-base-content/50 shrink-0" aria-hidden="true">
									-
								</span>
								<span>{item}</span>
							</li>
						))}
					</ul>
				</section>
			))}
		</div>
	);
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
	const hasContent = !(content === null || (typeof content === 'string' && content.trim() === ''));

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
		// oxlint-disable-next-line react/jsx-no-useless-fragment
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
				className={`rounded-box bg-base-100 text-base-content border-base-300 z-1000 max-w-xs border px-3 py-2 text-xs/4 whitespace-pre-line shadow-xl ${tooltipClassName}`}
			>
				{content}
			</Tooltip>
		</>
	);
}
