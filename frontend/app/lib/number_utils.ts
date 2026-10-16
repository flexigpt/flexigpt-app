export function parseOptionalPositiveInteger(value: string): number | undefined {
	const trimmed = value.trim();
	if (!trimmed) {
		return undefined;
	}

	const numberValue = Number(trimmed);
	if (!Number.isFinite(numberValue) || !Number.isInteger(numberValue) || numberValue <= 0) {
		return Number.NaN;
	}

	return numberValue;
}
