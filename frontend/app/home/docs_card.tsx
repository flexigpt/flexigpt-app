import { FiArrowRight, FiLayers, FiMessageSquare } from 'react-icons/fi';

import { Link } from 'react-router';

import type { NavCardProps } from '@/home/nav_card';

export const docsCards: NavCardProps[] = [
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

export function DocsCard({ title, description, to }: NavCardProps) {
	return (
		<Link to={to} className="group block w-full">
			<div className="bg-base-100 border-base-300/70 hover:border-primary/40 flex h-full flex-col rounded-2xl border shadow-md transition-all duration-200 hover:-translate-y-1 hover:shadow-xl">
				<div className="flex flex-col p-4">
					<div className="flex items-center justify-between p-0">
						<h3 className="text-sm font-semibold">{title}</h3>
						<FiArrowRight size={18} className="transition-transform group-hover:translate-x-1" />
					</div>
					<div className="text-base-content/70 mt-1 text-xs/relaxed">{description}</div>
				</div>
			</div>
		</Link>
	);
}
