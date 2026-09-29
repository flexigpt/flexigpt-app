export function Base64EncodeUTF8(str: string): string {
	// 1. Turn the JS string into a Uint8Array of UTF-8 bytes
	const utf8Bytes = new TextEncoder().encode(str);

	// 2. Convert those bytes to a "binary" string that btoa understands
	let binaryString = '';
	for (let i = 0; i < utf8Bytes.length; i++) {
		binaryString += String.fromCodePoint(utf8Bytes[i]);
	}

	// 3. btoa() on that binary string
	return btoa(binaryString);
}

export function GenerateRandomString(length: number, lowercase = false): string {
	const chars = 'ABCDEFGHJKLMNPQRSTUVWXY3456789'; // Exclude similar characters
	let result = '';
	for (let i = 0; i < length; i++) {
		result += chars.charAt(Math.floor(Math.random() * chars.length));
	}
	return lowercase ? result.toLowerCase() : result;
}
