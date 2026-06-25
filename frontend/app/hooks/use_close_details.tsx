import type { RefObject } from 'react';
import { useEffect } from 'react';

interface useCloseDetailsProps {
	detailsRef: RefObject<HTMLDetailsElement | null>;
	events?: (keyof DocumentEventMap)[];
	onClose?: () => void;
}

export function useCloseDetails({
	detailsRef,
	events = ['mousedown'], // default is 'mousedown'
	onClose,
}: useCloseDetailsProps) {
	useEffect(() => {
		function handleClickOutside(event: Event) {
			if (detailsRef.current && !detailsRef.current.contains(event.target as Node)) {
				// Force-close the <details>
				detailsRef.current.open = false;
				// Call the optional onClose callback
				if (onClose) {
					onClose();
				}
			}
		}

		// Add all desired events
		events.forEach(eventName => {
			document.addEventListener(eventName, handleClickOutside);
		});

		return () => {
			// Clean up all added event listeners
			events.forEach(eventName => {
				document.removeEventListener(eventName, handleClickOutside);
			});
		};
	}, [detailsRef, events, onClose]);
}
