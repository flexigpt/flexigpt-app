export function focusTextInputAtEnd(el: HTMLInputElement | HTMLTextAreaElement | null) {
	if (!el) return;
	el.focus({ preventScroll: true });
	const end = el.value.length;
	try {
		el.setSelectionRange(end, end);
	} catch {
		// ok.
	}
}
