import { useEffect, useRef, useState } from 'react';

/**
 * @public
 */
export type UseEventOptions<Ev extends Event = Event, D = unknown> = {
	// Target to attach the listener to (defaults to window if available)
	target?: EventTarget | null;

	// Native addEventListener options
	capture?: boolean;
	passive?: boolean;
	once?: boolean;

	// Optional event filtering: if returns false, event is ignored
	filter?: (ev: Ev) => boolean;

	// Optional handler called when the event passes the filter
	handler?: (ev: Ev) => void;

	// Enable keeping derived data (detail) in state
	// If true, detail is set from mapDetail(ev) or from (ev as CustomEvent).detail
	trackDetail?: boolean;

	// Custom derivation for detail when trackDetail is true
	mapDetail?: (ev: Ev) => D;

	// Optional "pulse" timeout in ms: sets pulse=true on event, then back to false after this many ms
	pulseMs?: number;
};

/**
 * @public
 */
export type UseEventResult<D = unknown> = {
	// True for pulseMs after each event (or always false if pulseMs not provided)
	pulse: boolean;

	// Last derived detail (only updated if trackDetail=true)
	detail: D | undefined;

	// Manually reset the pulse state to false
	resetPulse: () => void;
};

// Overload: native window events
export function useEvent<K extends keyof WindowEventMap>(
	name: K,
	options?: UseEventOptions<WindowEventMap[K]>
): UseEventResult;

// Overload: custom/string events
export function useEvent<D = unknown, Ev extends Event = CustomEvent<D>>(
	name: string,
	options?: UseEventOptions<Ev, D>
): UseEventResult<D>;

export function useEvent(name: string, options: UseEventOptions<Event, any> = {}): UseEventResult<any> {
	const { target, capture, passive, once, filter, handler, trackDetail = false, mapDetail, pulseMs } = options;

	const handlerRef = useRef(handler);
	const filterRef = useRef(filter);
	const mapDetailRef = useRef(mapDetail);
	const trackDetailRef = useRef(trackDetail);
	const pulseMsRef = useRef(pulseMs);

	useEffect(() => {
		handlerRef.current = handler;
	}, [handler]);

	useEffect(() => {
		filterRef.current = filter;
	}, [filter]);

	useEffect(() => {
		mapDetailRef.current = mapDetail;
	}, [mapDetail]);

	useEffect(() => {
		trackDetailRef.current = trackDetail;
	}, [trackDetail]);

	useEffect(() => {
		pulseMsRef.current = pulseMs;
	}, [pulseMs]);

	const [pulse, setPulse] = useState(false);
	const [detail, setDetail] = useState<any>(undefined);
	const timeoutRef = useRef<number | null>(null);

	const resetPulse = () => {
		if (timeoutRef.current) {
			window.clearTimeout(timeoutRef.current);
			timeoutRef.current = null;
		}
		setPulse(false);
	};

	useEffect(() => {
		const tgt = target ?? (typeof window !== 'undefined' ? window : undefined);
		if (!tgt || !('addEventListener' in tgt)) return;

		const listener = (ev: Event) => {
			if (filterRef.current && !filterRef.current(ev)) return;

			if (trackDetailRef.current) {
				if (mapDetailRef.current) {
					setDetail(mapDetailRef.current(ev));
				} else if ('detail' in ev) {
					setDetail((ev as CustomEvent).detail);
				} else {
					setDetail(undefined);
				}
			}

			if (typeof handlerRef.current === 'function') {
				handlerRef.current(ev);
			}

			const ms = pulseMsRef.current;
			if (typeof ms === 'number' && ms >= 0) {
				setPulse(true);
				if (timeoutRef.current) {
					window.clearTimeout(timeoutRef.current);
				}
				timeoutRef.current = window.setTimeout(() => {
					setPulse(false);
					timeoutRef.current = null;
				}, ms);
			}
		};

		tgt.addEventListener(name, listener, { capture, passive, once });
		return () => {
			tgt.removeEventListener(name, listener, capture);
		};
	}, [name, target, capture, passive, once]);

	// Ensure pending pulse timeouts are cleared on unmount
	useEffect(() => {
		return () => {
			if (timeoutRef.current) {
				window.clearTimeout(timeoutRef.current);
			}
		};
	}, []);

	return { pulse, detail, resetPulse };
}

// Typed CustomEvent dispatcher
export function emitCustomEvent<Detail = unknown>(
	name: string,
	detail?: Detail,
	target?: EventTarget | null,
	init?: Omit<CustomEventInit<Detail>, 'detail'>
): boolean {
	const tgt = target ?? (typeof window !== 'undefined' ? window : undefined);
	if (!tgt) return false;
	const ev = new CustomEvent<Detail>(name, { ...(init ?? {}), detail });
	return tgt.dispatchEvent(ev);
}
