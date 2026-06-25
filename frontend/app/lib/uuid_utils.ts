import { parse as uuidParse, v7 as uuidv7, validate as uuidValidate, version as uuidVersion } from 'uuid';

/**
 * Raised whenever the supplied string is not a valid RFC-4122 UUIDv7.
 */
class UUIDv7Error extends Error {
	constructor(message: string) {
		super(message);
		this.name = 'UUIDv7Error';
	}
}

function uuidValidateV7(id: string): boolean {
	return uuidValidate(id) && uuidVersion(id) === 7;
}

export function extractTimeFromUUIDv7Str(id: string): Date {
	if (!uuidValidateV7(id)) {
		throw new UUIDv7Error(`invalid UUID: ${id}`);
	}
	const uuidBytes = uuidParse(id);

	/* 48-bit timestamp, big-endian:
     byte[0] .. byte[5]    ( milliseconds since 1970-01-01T00:00:00Z )
  */
	const tsMs =
		(BigInt(uuidBytes[0]) << 40n) |
		(BigInt(uuidBytes[1]) << 32n) |
		(BigInt(uuidBytes[2]) << 24n) |
		(BigInt(uuidBytes[3]) << 16n) |
		(BigInt(uuidBytes[4]) << 8n) |
		BigInt(uuidBytes[5]);

	// 48-bit value ( ≤ 2^48-1 ) is well within JS’ safe integer range (2^53-1).
	return new Date(Number(tsMs)); // Date is always in UTC internally.
}

export function getUUIDv7(): string {
	return uuidv7();
}

export function ensureMakeID(explicit?: string): string {
	const trimmed = explicit?.trim();
	if (trimmed) {
		return trimmed;
	}
	// Strong uniqueness across tabs/windows.
	if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
		return crypto.randomUUID();
	}
	return `${Date.now()}-${Math.random().toString(36).slice(2, 11)}`;
}
