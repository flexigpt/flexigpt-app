import type { MouseEvent } from 'react';
import { createContext } from 'react';

interface MarkdownPresentationSettings {
	align: string;
	hideH1Title: boolean;
	onLinkClick?: (href: string, event: MouseEvent<HTMLAnchorElement>) => boolean;
}

export const MarkdownPresentationContext = createContext<MarkdownPresentationSettings>({
	align: 'left',
	hideH1Title: false,
});
