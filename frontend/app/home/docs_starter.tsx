import { FiLayers, FiMessageSquare } from 'react-icons/fi';

import type { NavCardProps } from '@/home/nav_card';

export const docsStarters: NavCardProps[] = [
	{
		title: 'Getting Started',
		description: 'Connect a provider, choose a model or Agent, add useful context, and send your first request.',
		to: '/docs/?doc=getting-started',
		icon: <FiMessageSquare size={18} />,
	},
	{
		title: 'Agents',
		description:
			'Use reusable Agent files as starting points for development, review, investigation, writing, research, and planning.',
		to: '/docs/?doc=agents',
		icon: <FiMessageSquare size={18} />,
	},
	{
		title: 'Composer Context',
		description: 'Add files, Workspaces, tools, Skills, connected services, and web search to the current message.',
		to: '/docs/?doc=composer-context',
		icon: <FiLayers size={18} />,
	},
];
