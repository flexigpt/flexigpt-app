export const DEFAULT_SEMVER = 'v1.0.0';

type SemverIdentifier = number | string;

type ParsedSemver = {
	hasV: boolean;
	major: number;
	minor: number;
	patch: number;
	prerelease: SemverIdentifier[];
};

// Supports:
// - 1.2.3
// - v1.2.3
// - 1.2.3-alpha.1
// - v1.2.3+build.5
// - 1.2.3-alpha.1+build.5
//
// Build metadata is accepted but ignored for precedence comparison,
// which matches semver rules.
const SEMVER_REGEX =
	/^(v)?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$/i;

function parseSemver(version?: string): ParsedSemver | null {
	const raw = (version ?? '').trim();
	const match = raw.match(SEMVER_REGEX);

	if (!match) return null;

	const prerelease = (match[5] ?? '')
		.split('.')
		.filter(Boolean)
		.map(part => (/^\d+$/.test(part) ? Number(part) : part));

	return {
		hasV: Boolean(match[1]),
		major: Number(match[2]),
		minor: Number(match[3]),
		patch: Number(match[4]),
		prerelease,
	};
}

function compareSemverIdentifier(left: SemverIdentifier, right: SemverIdentifier): number {
	const leftIsNumber = typeof left === 'number';
	const rightIsNumber = typeof right === 'number';

	if (leftIsNumber && rightIsNumber) {
		return left - right;
	}

	// Numeric identifiers always have lower precedence than non-numeric ones.
	if (leftIsNumber) return -1;
	if (rightIsNumber) return 1;

	const leftString = left;
	const rightString = right;

	if (leftString < rightString) return -1;
	if (leftString > rightString) return 1;
	return 0;
}

function compareSemver(left: ParsedSemver, right: ParsedSemver): number {
	if (left.major !== right.major) return left.major - right.major;
	if (left.minor !== right.minor) return left.minor - right.minor;
	if (left.patch !== right.patch) return left.patch - right.patch;

	const leftPrerelease = left.prerelease;
	const rightPrerelease = right.prerelease;

	if (leftPrerelease.length === 0 && rightPrerelease.length === 0) return 0;
	if (leftPrerelease.length === 0) return 1;
	if (rightPrerelease.length === 0) return -1;

	const len = Math.max(leftPrerelease.length, rightPrerelease.length);

	for (let i = 0; i < len; i += 1) {
		const leftPart = leftPrerelease[i];
		const rightPart = rightPrerelease[i];

		if (leftPart === undefined) return -1;
		if (rightPart === undefined) return 1;

		const byPart = compareSemverIdentifier(leftPart, rightPart);
		if (byPart !== 0) return byPart;
	}

	return 0;
}

function formatSuggestedVersion(hasV: boolean, major: number, minor: number): string {
	return `${hasV ? 'v' : ''}${major}.${minor}.0`;
}

function getMaxSemverVersion(versions: ParsedSemver[]): ParsedSemver | undefined {
	return versions.reduce<ParsedSemver | undefined>((max, current) => {
		if (!max) return current;
		return compareSemver(current, max) > 0 ? current : max;
	}, undefined);
}

export function isSemverVersion(v: string): boolean {
	return parseSemver(v) !== null;
}

export function compareVersionStrings(left: string, right: string): number {
	const leftSemver = parseSemver(left);
	const rightSemver = parseSemver(right);

	if (leftSemver && rightSemver) {
		return compareSemver(leftSemver, rightSemver);
	}

	return left.localeCompare(right, undefined, {
		numeric: true,
		sensitivity: 'base',
	});
}

/**
 * @public
 */
export function validateVersion(version: string): string | undefined {
	const trimmed = version.trim();

	if (!trimmed) return 'Version is required.';
	if (trimmed.length > 64) return 'Version must be at most 64 characters.';
	if (!/^[a-zA-Z0-9.+-]+$/.test(trimmed)) {
		return 'Version may only contain letters, numbers, "-", ".", and "+".';
	}

	return undefined;
}

/**
 * If `current` is semver-like (`v1.2.3`, `1.2.3`, `1.2.3-alpha`, `1.2.3+build`),
 * returns the next stable minor (`v1.3.0` / `1.3.0`).
 *
 * IMPORTANT:
 * If `existingVersions` is provided, the suggestion is computed from the same major as `current`
 * and uses the greater of:
 *   - current minor
 *   - max existing minor for that major
 *
 * This prevents suggesting a version that already exists, while also avoiding
 * going backwards if `current` is already newer than the existing set.
 *
 * If `current` is not semver, it falls back to the max semver found in `existingVersions`.
 * If none can be parsed, it returns `DEFAULT_SEMVER`.
 */
export function suggestNextMinorVersion(
	current?: string,
	existingVersions?: string[]
): { suggested: string; isSemver: boolean } {
	const cur = parseSemver(current);

	const parsedExisting = (existingVersions ?? [])
		.map(version => parseSemver(version))
		.filter((parsed): parsed is ParsedSemver => parsed !== null);

	// If current isn't semver, optionally fall back to max(existing) if possible.
	if (!cur) {
		const maxExisting = getMaxSemverVersion(parsedExisting);

		if (!maxExisting) {
			return { suggested: DEFAULT_SEMVER, isSemver: false };
		}

		// Prefer "v" if any highest-precedence matching version uses it.
		const hasV = parsedExisting.some(version => version.hasV && compareSemver(version, maxExisting) === 0);

		return {
			suggested: formatSuggestedVersion(hasV, maxExisting.major, maxExisting.minor + 1),
			isSemver: true,
		};
	}

	const parsedExistingSameMajor = parsedExisting.filter(version => version.major === cur.major);

	// Prefer "v" prefix if current has it; otherwise adopt it if the existing set uses it.
	const hasV = cur.hasV || parsedExistingSameMajor.some(version => version.hasV);

	// Use whichever is newer: current minor or max existing minor for the same major.
	const highestMinor = parsedExistingSameMajor.reduce(
		(maxMinor, version) => Math.max(maxMinor, version.minor),
		cur.minor
	);

	return {
		suggested: formatSuggestedVersion(hasV, cur.major, highestMinor + 1),
		isSemver: true,
	};
}
