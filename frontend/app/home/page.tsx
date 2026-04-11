import type { ReactNode } from 'react';

import { FiBookOpen, FiHome, FiMessageSquare } from 'react-icons/fi';

import { Link } from 'react-router';

import { useTitleBarContent } from '@/hooks/use_title_bar';

import { PageFrame } from '@/components/page_frame';

type HomeCardProps = {
	title: string;
	description: string;
	to: string;
	icon: ReactNode;
};

function HomeCard({ title, description, to, icon }: HomeCardProps) {
	return (
		<Link to={to} className="group block w-full max-w-80">
			<div className="bg-base-100 flex h-40 flex-col justify-between rounded-2xl p-4 text-center shadow-lg transition-transform hover:scale-105">
				<div>
					<div className="mb-4 flex items-center justify-center gap-2">
						{icon}
						<h3 className="text-2xl font-semibold">{title}</h3>
					</div>
					<p>{description}</p>
				</div>
				<h3 className="mt-1 text-2xl font-semibold">
					<span className="ml-4 inline-block transform transition-transform group-hover:translate-x-1">-&gt;</span>
				</h3>
			</div>
		</Link>
	);
}

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
			<div className="flex h-full w-full flex-col items-center px-4 py-8">
				<div className="mt-4 flex flex-row items-center">
					<img src="/icon.png" alt="FlexiGPT Icon" width={64} height={64} />
					<h1 className="m-8 text-2xl font-bold">FlexiGPT</h1>
				</div>

				<div className="flex min-h-0 w-full flex-1 flex-col items-center justify-center">
					<HomeCard
						title="Chat with AI"
						description="Interact with LLMs and get assistance"
						to="/chats/"
						icon={<FiMessageSquare size={20} />}
					/>
				</div>

				<div className="w-full pb-16 xl:pb-32">
					<div className="mx-auto grid w-full max-w-3xl grid-cols-1 justify-items-center gap-6 md:grid-cols-2">
						<HomeCard
							title="Using LLMs"
							description="Introduction to concepts, common terms, workflows"
							to="/docs/#introduction-to-using-llms"
							icon={<FiBookOpen size={20} />}
						/>
						<HomeCard
							title="How To"
							description="Setup and usage docs"
							to="/docs/#app-setup-and-docs-guide"
							icon={<FiBookOpen size={20} />}
						/>
					</div>
				</div>
			</div>
		</PageFrame>
	);
}
