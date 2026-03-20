import type { PromptTemplate, PromptTemplateRef } from '@/spec/prompt';

export function buildPromptTemplateSeriesKey(bundleID: string, templateSlug: string): string {
	return `${bundleID}/${templateSlug}`;
}

export function buildPromptTemplateRefKey(identity: {
	bundleID: string;
	templateSlug: string;
	templateVersion: string;
}): string {
	return `${buildPromptTemplateSeriesKey(identity.bundleID, identity.templateSlug)}@${identity.templateVersion}`;
}

export function getPromptTemplateRef(
	bundleID: string,
	template: Pick<PromptTemplate, 'slug' | 'version'>
): PromptTemplateRef {
	return {
		bundleID,
		templateSlug: template.slug,
		templateVersion: template.version,
	};
}
