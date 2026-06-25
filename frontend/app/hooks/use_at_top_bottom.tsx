import type { RefObject } from 'react';
import { useCallback, useEffect, useRef, useState } from 'react';

interface UseAtTopBottomResult {
	isAtBottom: boolean;
	isAtTop: boolean;
	isScrollable: boolean;
	checkScroll: () => void;
}

export function useAtTopBottom(ref: RefObject<HTMLElement | null>, offset = 0): UseAtTopBottomResult {
	const [isAtBottom, setIsAtBottom] = useState(false);
	const [isAtTop, setIsAtTop] = useState(false);
	const [isScrollable, setIsScrollable] = useState(false);

	// keep the handler in a ref so both the scroll listener and the
	// ResizeObserver can call the *same* function
	const handleScrollRef = useRef(() => {});

	// The actual scroll check logic, memoized for consumer use
	const checkScroll = useCallback(() => {
		const current = ref.current;
		if (!current) {
			return;
		}
		const { scrollTop, scrollHeight, clientHeight } = current;
		setIsScrollable(scrollHeight > clientHeight);
		setIsAtBottom(scrollTop + clientHeight >= scrollHeight - offset - 10);
		setIsAtTop(scrollTop <= 10);
	}, [ref, offset]);

	useEffect(() => {
		const current = ref.current;
		if (!current) {
			return;
		}

		// Save the latest checkScroll in the ref for ResizeObserver
		handleScrollRef.current = checkScroll;

		let throttleTimeout: NodeJS.Timeout | null = null;
		const throttledScroll = () => {
			if (throttleTimeout === null) {
				throttleTimeout = setTimeout(() => {
					checkScroll();
					throttleTimeout = null;
				}, 100);
			}
		};

		current.addEventListener('scroll', throttledScroll, { passive: true });

		const resizeObserver = new ResizeObserver(() => {
			// no throttling here – this already fires at a sane rate
			handleScrollRef.current();
		});
		resizeObserver.observe(current);

		// run once on mount
		checkScroll();

		return () => {
			current.removeEventListener('scroll', throttledScroll);
			resizeObserver.disconnect();
			if (throttleTimeout) {
				clearTimeout(throttleTimeout);
			}
		};
	}, [ref, offset, checkScroll]);

	return { isAtBottom, isAtTop, isScrollable, checkScroll };
}
