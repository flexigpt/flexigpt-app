import type { FocusEventHandler, KeyboardEventHandler, MouseEventHandler, ReactNode } from 'react';
import { useCallback, useEffect, useRef } from 'react';

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
	showDelay?: number;
	hideDelay?: number;
	disabled?: boolean;
	suspended?: boolean;
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
	showDelay = 400,
	hideDelay = 100,
	disabled = false,
	suspended = false,
	wrapperClassName = 'inline-flex max-w-full',
	wrapperElement = 'span',
	tooltipClassName = '',
}: HoverTipProps) {
	const tooltip = useTooltipStore({ placement });
	const Wrapper = wrapperElement ?? 'span';
	const hasContent =
		content !== null &&
		content !== undefined &&
		content !== false &&
		content !== true &&
		!(typeof content === 'string' && content.trim() === '');
	const showTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
	const hideTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
	const isAnchorHoveredRef = useRef(false);
	const isTooltipHoveredRef = useRef(false);
	const isFocusWithinRef = useRef(false);
	const suppressFocusUntilRef = useRef(0);

	const cancelScheduledShow = useCallback(() => {
		if (showTimerRef.current === null) {
			return;
		}
		clearTimeout(showTimerRef.current);
		showTimerRef.current = null;
	}, []);

	const cancelScheduledHide = useCallback(() => {
		if (hideTimerRef.current === null) {
			return;
		}
		clearTimeout(hideTimerRef.current);
		hideTimerRef.current = null;
	}, []);

	const hideTip = useCallback(() => {
		cancelScheduledShow();
		cancelScheduledHide();
		tooltip.hide();
		tooltip.setAnchorElement(null);
	}, [cancelScheduledHide, cancelScheduledShow, tooltip]);

	const scheduleHideIfInactive = useCallback(() => {
		if (isAnchorHoveredRef.current || isTooltipHoveredRef.current || isFocusWithinRef.current) {
			return;
		}

		cancelScheduledHide();

		if (hideDelay <= 0) {
			hideTip();
			return;
		}

		hideTimerRef.current = setTimeout(() => {
			hideTimerRef.current = null;
			if (isAnchorHoveredRef.current || isTooltipHoveredRef.current || isFocusWithinRef.current) {
				return;
			}
			hideTip();
		}, hideDelay);
	}, [cancelScheduledHide, hideDelay, hideTip]);

	useEffect(() => {
		return () => {
			cancelScheduledShow();
			cancelScheduledHide();
		};
	}, [cancelScheduledHide, cancelScheduledShow]);

	useEffect(() => {
		if (!hasContent || disabled || suspended) {
			isAnchorHoveredRef.current = false;
			isTooltipHoveredRef.current = false;
			isFocusWithinRef.current = false;
			hideTip();
		}
	}, [disabled, hasContent, hideTip, suspended]);

	const showForCurrentTarget: MouseEventHandler<HTMLElement> = useCallback(
		event => {
			isAnchorHoveredRef.current = true;
			cancelScheduledHide();
			cancelScheduledShow();

			const anchor = event.currentTarget;
			const showNow = () => {
				showTimerRef.current = null;
				if (!isAnchorHoveredRef.current || suspended) {
					return;
				}
				tooltip.setAnchorElement(anchor);
				tooltip.show();
			};

			if (showDelay <= 0) {
				showNow();
				return;
			}

			showTimerRef.current = setTimeout(showNow, showDelay);
		},
		[cancelScheduledHide, cancelScheduledShow, showDelay, suspended, tooltip]
	);

	const showForFocusedTarget: FocusEventHandler<HTMLElement> = useCallback(
		event => {
			isFocusWithinRef.current = true;
			cancelScheduledHide();
			cancelScheduledShow();

			if (Date.now() < suppressFocusUntilRef.current) {
				return;
			}

			tooltip.setAnchorElement(event.target as HTMLElement);
			tooltip.show();
		},
		[cancelScheduledHide, cancelScheduledShow, tooltip]
	);

	const hideOnBlurCapture: FocusEventHandler<HTMLElement> = useCallback(
		event => {
			const nextTarget = event.relatedTarget as Node | null;
			if (nextTarget && event.currentTarget.contains(nextTarget)) {
				return;
			}
			isFocusWithinRef.current = false;
			scheduleHideIfInactive();
		},
		[scheduleHideIfInactive]
	);

	const hideForPointerInteraction = useCallback(() => {
		// Pointer activation moves focus before click. Prevent that focus event
		// from reopening a tooltip that has just been dismissed.
		suppressFocusUntilRef.current = Date.now() + 100;
		hideTip();
	}, [hideTip]);

	const hideAfterMouseLeave: MouseEventHandler<HTMLElement> = useCallback(() => {
		isAnchorHoveredRef.current = false;
		cancelScheduledShow();
		scheduleHideIfInactive();
	}, [cancelScheduledShow, scheduleHideIfInactive]);

	const keepTipVisible = useCallback(() => {
		isTooltipHoveredRef.current = true;
		cancelScheduledHide();
	}, [cancelScheduledHide]);

	const hideAfterTooltipLeave = useCallback(() => {
		isTooltipHoveredRef.current = false;
		scheduleHideIfInactive();
	}, [scheduleHideIfInactive]);

	const hideOnEscape: KeyboardEventHandler<HTMLElement> = useCallback(
		event => {
			if (event.key === 'Escape') {
				hideTip();
			}
		},
		[hideTip]
	);

	if (!hasContent || disabled) {
		// oxlint-disable-next-line react/jsx-no-useless-fragment
		return <>{children}</>;
	}

	return (
		<>
			<Wrapper
				className={wrapperClassName}
				onMouseEnter={suspended ? undefined : showForCurrentTarget}
				onMouseLeave={suspended ? undefined : hideAfterMouseLeave}
				onFocusCapture={suspended ? undefined : showForFocusedTarget}
				onBlurCapture={suspended ? undefined : hideOnBlurCapture}
				onPointerDownCapture={suspended ? undefined : hideForPointerInteraction}
				onClickCapture={suspended ? undefined : hideTip}
				onKeyDownCapture={suspended ? undefined : hideOnEscape}
			>
				{children}
			</Wrapper>
			{suspended ? null : (
				<Tooltip
					store={tooltip}
					gutter={gutter}
					overflowPadding={overflowPadding}
					portal
					onMouseEnter={keepTipVisible}
					onMouseLeave={hideAfterTooltipLeave}
					className={`rounded-box bg-base-100 text-base-content border-base-300 pointer-events-auto z-1000 max-w-xs border px-3 py-2 text-xs/4 whitespace-pre-line shadow-xl ${tooltipClassName}`}
				>
					{content}
				</Tooltip>
			)}
		</>
	);
}
