import type { ReactNode } from 'react';

import { FiArrowRight, FiBookOpen, FiCompass, FiHome, FiLayers, FiMessageSquare, FiTarget } from 'react-icons/fi';

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
		description: 'Add a provider key, choose a built-in preset, and send your first message.',
		to: '/docs/#getting-started',
		icon: <FiCompass size={18} />,
	},
	{
		title: 'Core Concepts',
		description: 'Understand providers, presets, tools, skills, attachments, and context.',
		to: '/docs/#core-concepts',
		icon: <FiLayers size={18} />,
	},
	{
		title: 'App Tour',
		description: 'See where everything lives and how the chat workflow fits together.',
		to: '/docs/#app-tour-and-chat-workflow',
		icon: <FiBookOpen size={18} />,
	},
	{
		title: 'Better Results',
		description: 'Improve output quality with better prompts, context, tools, and presets.',
		to: '/docs/#getting-better-results',
		icon: <FiTarget size={18} />,
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
				<div className="mt-2 flex max-w-xl flex-col items-center text-center">
					<div className="flex items-center gap-4">
						<img src="/icon.png" alt="FlexiGPT Icon" width={48} height={48} />
						<h1 className="text-xl font-bold md:text-2xl">FlexiGPT</h1>
					</div>

					<p className="text-base-content/70 mt-2 max-w-2xl text-sm leading-relaxed">
						Local-first desktop workspace for multi-provider LLM chats, with assistants, model presets, templates,
						attachments, tools and skills.
					</p>
				</div>

				<div className="mt-10 flex w-full flex-1 flex-col items-center gap-10">
					<PrimaryActionCard
						title="Open Chats"
						description="Start a new conversation or continue a saved local thread."
						to="/chats/"
						icon={<FiMessageSquare size={24} />}
					/>

					<section className="mt-8 w-full pb-12 xl:mt-24 xl:pb-24">
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
						</div>

						<div className="mx-auto mt-6 grid w-full max-w-5xl grid-cols-2 gap-6 lg:grid-cols-4">
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
