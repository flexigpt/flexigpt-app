import { createContext } from 'react';

export interface MarkdownCodeRendererSettings {
	isBusy: boolean;
	hideMermaidCode: boolean;
	diffCandidatePaths?: string[];
	diffWorkspaceRoots?: string[];
	defaultCodeBlockExpanded: boolean;

	/**
	 * Incremented by the active message owner after an observed stream settles.
	 * Historical messages retain zero and are never auto-reviewed.
	 */
	autoReviewEpoch: number;
}

export const MarkdownCodeRendererContext = createContext<MarkdownCodeRendererSettings>({
	isBusy: false,
	hideMermaidCode: false,
	defaultCodeBlockExpanded: true,
	autoReviewEpoch: 0,
});
