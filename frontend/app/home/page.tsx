import type { ReactNode } from 'react';

import { FiArrowRight, FiBookOpen, FiHome, FiLayers, FiMessageSquare } from 'react-icons/fi';

import { Link } from 'react-router';

import { useTitleBarContent } from '@/hooks/use_title_bar';

import { PageFrame } from '@/components/page_frame';

type NavCardProps = {
	title: string;
	description: string;
	to: string;
	icon?: ReactNode;
};

function PrimaryActionCard({ title, description, to, icon }: NavCardProps) {
	return (
		<Link to={to} className="group block w-full max-w-lg">
			<div className="bg-base-100 border-primary/20 ring-primary/10 flex min-h-48 flex-col justify-between rounded-3xl border p-6 shadow-xl ring-1 transition-all duration-200 hover:-translate-y-1 hover:shadow-2xl sm:p-8">
				<div className="flex items-start gap-4">
					<div className="bg-primary/10 text-primary rounded-2xl p-3">{icon}</div>

					<div className="min-w-0 flex-1 text-left">
						<h2 className="mt-2 text-2xl font-bold sm:text-3xl">{title}</h2>
						<p className="text-base-content/70 mt-3 text-sm leading-relaxed sm:text-base">{description}</p>
					</div>

					<div className="text-base-content/80 pt-1 transition-transform group-hover:translate-x-1">
						<FiArrowRight size={24} />
					</div>
				</div>
			</div>
		</Link>
	);
}

function DocsCard({ title, description, to }: NavCardProps) {
	return (
		<Link to={to} className="group block w-full">
			<div className="bg-base-100 border-base-300/60 flex h-40 flex-col justify-between rounded-2xl border p-4 text-left shadow-md transition-all duration-200 hover:-translate-y-1 hover:shadow-xl">
				<div>
					<div className="mb-3 flex items-center gap-3">
						<h3 className="text-lg font-semibold">{title}</h3>
					</div>

					<p className="text-base-content/70 text-sm leading-relaxed">{description}</p>
				</div>

				<div className="text-base-content/80 flex justify-end">
					<FiArrowRight size={18} className="transition-transform group-hover:translate-x-1" />
				</div>
			</div>
		</Link>
	);
}

const docsCards: NavCardProps[] = [
	{
		title: 'Getting Started',
		description: 'Connect a provider key, choose a starting setup, and send the first useful message.',
		to: '/docs/#getting-started',
		icon: <FiMessageSquare size={18} />,
	},
	{
		title: 'Chats and Composer',
		description: 'Learn how the Chats workspace is structured and how a conversation moves from setup to response.',
		to: '/docs/#chats-composer-and-everyday-workflow',
		icon: <FiMessageSquare size={18} />,
	},
	{
		title: 'Context and Automation',
		description: 'Use attachments, prompts, tools, skills, and auto-execute workflows.',
		to: '/docs/#attachments-tools-skills-and-prompts',
		icon: <FiLayers size={18} />,
	},
];

// eslint-disable-next-line no-restricted-exports
export default function HomePage() {
	useTitleBarContent(
		{
			center: (
				<div className="mx-auto flex items-center justify-center opacity-60">
					<FiHome size={16} />
				</div>
			),
		},
		[]
	);

	return (
		<PageFrame>
			<div className="mx-auto flex h-full w-full max-w-6xl flex-col items-center px-4 py-8">
				<div className="flex max-w-xl flex-col items-center text-center">
					<div className="flex items-center gap-4">
						<img src="/icon.png" alt="FlexiGPT Icon" width={48} height={48} />
						<h1 className="text-xl font-bold md:text-2xl">FlexiGPT</h1>
					</div>

					<p className="text-base-content/70 max-w-2xl text-sm leading-relaxed">
						Local-first desktop workspace for multi-provider LLM chats, with assistants, model presets, prompts,
						attachments, tools, and skills.
					</p>
				</div>

				<div className="mt-16 flex w-full flex-1 flex-col items-center justify-between xl:mt-24">
					<PrimaryActionCard
						title="Open Chats"
						description="Start a new conversation or continue a saved local thread."
						to="/chats/"
						icon={<FiMessageSquare size={24} />}
					/>

					<section className="w-full xl:pb-16">
						<div className="mx-auto max-w-3xl text-center">
							<Link
								to="/docs/"
								className="inline-flex items-center gap-2 text-xl font-semibold transition-opacity hover:opacity-80"
							>
								<FiBookOpen size={24} />
								Documentation
								<div className="flex justify-end">
									<FiArrowRight size={24} className="transition-transform group-hover:translate-x-1" />
								</div>
							</Link>
							<p className="text-base-content/70 mt-1 text-sm">
								Bundled workflow, context, automation, and architecture reference for FlexiGPT.
							</p>
						</div>

						<div className="mx-auto mt-4 grid w-full max-w-5xl grid-cols-1 gap-6 md:grid-cols-3">
							{docsCards.map(card => (
								<DocsCard key={card.to} title={card.title} description={card.description} to={card.to} />
							))}
						</div>
					</section>
				</div>
			</div>
		</PageFrame>
	);
}
