export function utf8TextToBase64(value: string): string {
	const bytes = new TextEncoder().encode(value);
	let binary = '';

	for (let offset = 0; offset < bytes.length; offset += 0x8000) {
		binary += String.fromCodePoint(...bytes.subarray(offset, offset + 0x8000));
	}

	return globalThis.btoa(binary);
}
