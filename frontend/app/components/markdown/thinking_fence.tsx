import type { ReactNode } from 'react';
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';

interface ThinkingFenceProps {
	/** Summary row content (label, spinner, etc) */
	detailsSummary: ReactNode;

	/** Plain text body. If `children` is provided, `text` is ignored. */
	text?: string;
	children?: ReactNode;

	/** Uncontrolled initial open state */
	defaultOpen?: boolean;

	/** Controlled open state (optional) */
	open?: boolean;
	onOpenChange?: (open: boolean) => void;

	/** Non-streaming mode max-height (original behavior) */
	maxHeightClass?: string; // e.g. 'max-h-[50vh]' or 'max-h-60'

	/**
	 * Streaming mode:
	 * - expands from ~1 line up to maxRows lines
	 * - then becomes internally scrollable and autoscrolls to bottom (if autoScroll=true)
	 */
	streaming?: boolean;
	maxRows?: number; // defaults to 3 when streaming=true
	autoScroll?: boolean; // defaults to true when streaming=true
}

const BOTTOM_EPSILON_PX = 1;

function isAtBottom(el: HTMLElement) {
	return el.scrollHeight - el.scrollTop - el.clientHeight <= BOTTOM_EPSILON_PX;
}

export function ThinkingFence({
	detailsSummary,
	text,
	children,
	defaultOpen = false,
	open,
	onOpenChange,
	maxHeightClass = 'max-h-[50vh]',
	streaming = false,
	maxRows,
	autoScroll,
}: ThinkingFenceProps) {
	const isControlled = open !== undefined;
	const [internalOpen, setInternalOpen] = useState(defaultOpen);
	const isOpen = isControlled ? open : internalOpen;

	const bodyRef = useRef<HTMLDivElement | null>(null);
	const [streamHeightPx, setStreamHeightPx] = useState<number | undefined>(undefined);
	const autoScrollPinnedRef = useRef(true);
	const previousStreamingRef = useRef(streaming);

	const effectiveMaxRows = streaming ? (typeof maxRows === 'number' && maxRows > 0 ? maxRows : 3) : undefined;
	const effectiveAutoScroll = streaming ? (autoScroll ?? true) : false;
	const hasExplicitChildren = children !== undefined && children !== null;

	// A new streaming session starts pinned to bottom by default.
	// If the user scrolls up, we stop autoscrolling until they manually
	// reach the absolute bottom again.
	useEffect(() => {
		if (!streaming) {
			autoScrollPinnedRef.current = true;
			previousStreamingRef.current = false;
			return;
		}

		if (!previousStreamingRef.current) {
			autoScrollPinnedRef.current = true;
		}

		previousStreamingRef.current = true;
	}, [streaming]);

	useEffect(() => {
		if (!streaming || !isOpen || !effectiveAutoScroll) {
			return;
		}

		const el = bodyRef.current;
		if (!el) {
			return;
		}

		const handleScroll = () => {
			autoScrollPinnedRef.current = isAtBottom(el);
		};

		el.addEventListener('scroll', handleScroll, { passive: true });
		handleScroll();
		return () => {
			el.removeEventListener('scroll', handleScroll);
		};
	}, [streaming, isOpen, effectiveAutoScroll]);

	const bodyContent = useMemo(() => {
		if (hasExplicitChildren) {
			return children;
		}
		return text ?? '';
	}, [children, hasExplicitChildren, text]);

	// Streaming behavior: height grows from ~1 line to maxRows, then scroll.
	useLayoutEffect(() => {
		if (!streaming) {
			return;
		}
		if (!isOpen) {
			return;
		}

		const el = bodyRef.current;
		if (!el) {
			return;
		}

		// Measure line height + padding so “rows” feels consistent.
		const cs = window.getComputedStyle(el);
		const lineHeight = Number(cs.lineHeight || '16') || 16;
		const padTop = Number(cs.paddingTop || '0') || 0;
		const padBottom = Number(cs.paddingBottom || '0') || 0;

		const oneRow = lineHeight + padTop + padBottom;
		const maxH = (effectiveMaxRows ?? 3) * lineHeight + padTop + padBottom;

		// scrollHeight includes padding, good.
		const desired = Math.min(Math.max(el.scrollHeight, oneRow), maxH);
		setStreamHeightPx(desired);

		if (effectiveAutoScroll && autoScrollPinnedRef.current) {
			// Stay pinned only while the user has not scrolled away from bottom.
			el.scrollTop = el.scrollHeight;
		}
	}, [bodyContent, streaming, effectiveMaxRows, effectiveAutoScroll, isOpen]);

	return (
		<details
			open={isOpen}
			onToggle={e => {
				const next = (e.currentTarget as HTMLDetailsElement).open;
				if (!isControlled) {
					setInternalOpen(next);
				}
				onOpenChange?.(next);
			}}
			className="group bg-base-200/60 m-0 overflow-hidden px-2 py-1 shadow-none"
		>
			<summary
				// Hide default disclosure marker (we render our own chevron)
				className="flex cursor-pointer list-none items-center justify-between gap-2 px-2 py-1 text-xs transition-colors select-none"
				style={{
					// Safari/Chrome marker hiding (best-effort)
					WebkitAppearance: 'none',
				}}
			>
				<div className="min-w-0">{detailsSummary}</div>
				<svg
					className="ml-2 size-3 shrink-0 transition-transform group-open:rotate-90"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
				</svg>
			</summary>

			{/* body */}
			<div
				ref={bodyRef}
				className={[
					'overflow-y-auto px-2 py-1 text-xs wrap-break-word whitespace-pre-wrap',
					!streaming ? maxHeightClass : '',
				]
					.filter(Boolean)
					.join(' ')}
				style={
					streaming
						? {
								height: streamHeightPx,
								maxHeight: undefined, // height is controlled via streamHeightPx
							}
						: undefined
				}
			>
				{hasExplicitChildren ? children : (text ?? '')}
			</div>
		</details>
	);
}
