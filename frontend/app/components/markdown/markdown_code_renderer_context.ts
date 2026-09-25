import { createContext } from 'react';

export interface MarkdownCodeRendererSettings {
	isBusy: boolean;
	hideMermaidCode: boolean;
	diffCandidatePaths?: string[];
	diffWorkspaceRoots?: string[];
	defaultCodeBlockExpanded: boolean;
}

export const MarkdownCodeRendererContext = createContext<MarkdownCodeRendererSettings>({
	isBusy: false,
	hideMermaidCode: false,
	defaultCodeBlockExpanded: true,
});
