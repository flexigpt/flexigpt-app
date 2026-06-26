import { FiLayers, FiMessageSquare } from 'react-icons/fi';

import type { NavCardProps } from '@/home/nav_card';

export const docsStarters: NavCardProps[] = [
	{
		title: 'Getting Started',
		description:
			'Connect a provider key, choose a model or assistant preset, add context, and send your first request.',
		to: '/docs/?doc=getting-started',
		icon: <FiMessageSquare size={18} />,
	},
	{
		title: 'Chat Workspace',
		description: 'Use tabs, search, the timeline, model controls, assistant presets, and send/edit flows.',
		to: '/docs/?doc=chat-workspace',
		icon: <FiMessageSquare size={18} />,
	},
	{
		title: 'Composer Context',
		description: 'Attach files, folders, URLs, prompt templates, tools, skills, and web search to the current message.',
		to: '/docs/?doc=composer-context',
		icon: <FiLayers size={18} />,
	},
];
