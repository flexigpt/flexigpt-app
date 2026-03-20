import { emitCustomEvent, useEvent } from '@/hooks/use_event';

const TEMPLATE_FLASH_EVENT = 'tpl-toolbar:flash';

/**
 * Returns a boolean that becomes true for `durationMs` after each flash event.
 */
export function useTemplateFlashEvent(durationMs = 800, target?: EventTarget | null) {
	const { pulse } = useEvent<undefined, CustomEvent<void>>(TEMPLATE_FLASH_EVENT, {
		target,
		pulseMs: durationMs,
	});
	return pulse;
}
export function dispatchTemplateFlashEvent(target?: EventTarget | null) {
	emitCustomEvent(TEMPLATE_FLASH_EVENT, undefined, target);
}
