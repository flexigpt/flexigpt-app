import { type KeyboardEvent as ReactKeyboardEvent, useCallback, useRef } from 'react';

type BoolOrGetter = boolean | (() => boolean);
const resolveBool = (v: BoolOrGetter) => (typeof v === 'function' ? v() : v);

interface EnterSubmitConfig {
	// can be a boolean or a getter (if computing busy state is cheaper lazily)
	isBusy: BoolOrGetter;
	// whether a submit is allowed (e.g., non-empty text)
	canSubmit: () => boolean;
	// if provided, used to insert a soft break on Shift+Enter
	insertSoftBreak?: () => void;
	// if provided, used instead of formRef.requestSubmit()
	onSubmitRequest?: () => void;
}

export function useEnterSubmit({ isBusy, canSubmit, insertSoftBreak, onSubmitRequest }: EnterSubmitConfig) {
	const formRef = useRef<HTMLFormElement>(null);

	const onKeyDown = useCallback(
		(e: ReactKeyboardEvent<HTMLDivElement>) => {
			if (e.isDefaultPrevented()) return;

			const native = e.nativeEvent;
			const isComposing = (native as KeyboardEvent & { isComposing?: boolean }).isComposing;

			// Shift+Enter => soft break
			if (e.key === 'Enter' && e.shiftKey && !isComposing) {
				insertSoftBreak?.();
				e.preventDefault();
				return;
			}

			// Enter => submit
			if (e.key === 'Enter' && !e.shiftKey && !isComposing) {
				const busy = resolveBool(isBusy);
				if (!busy && canSubmit()) {
					if (onSubmitRequest) {
						onSubmitRequest();
					} else {
						formRef.current?.requestSubmit();
					}
				}
				e.preventDefault();
			}
		},
		[isBusy, canSubmit, insertSoftBreak, onSubmitRequest]
	);

	return { formRef, onKeyDown };
}
