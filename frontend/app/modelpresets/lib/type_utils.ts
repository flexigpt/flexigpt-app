import type { OutputParam, ReasoningParam } from '@/spec/inference';

export function outputParamsEqual(a?: OutputParam, b?: OutputParam): boolean {
	if (!a && !b) return true;
	if (!a || !b) return false;

	if (a.verbosity !== b.verbosity) return false;

	const aFormat = a.format;
	const bFormat = b.format;
	if (!!aFormat !== !!bFormat) return false;
	if (!aFormat && !bFormat) return true;
	if (!aFormat || !bFormat) return false;

	return (
		aFormat.kind === bFormat.kind &&
		JSON.stringify(aFormat.jsonSchemaParam ?? null) === JSON.stringify(bFormat.jsonSchemaParam ?? null)
	);
}

export function reasoningEqual(a?: ReasoningParam, b?: ReasoningParam): boolean {
	if (!a && !b) return true;
	if (!a || !b) return false;
	return a.type === b.type && a.level === b.level && a.tokens === b.tokens && a.summaryStyle === b.summaryStyle;
}
