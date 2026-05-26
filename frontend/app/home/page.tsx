import type { ReactNode } from 'react';

import {
	FiAlertTriangle,
	FiArrowRight,
	FiBookOpen,
	FiCode,
	FiFileText,
	FiHome,
	FiLayers,
	FiMessageSquare,
} from 'react-icons/fi';

import { Link } from 'react-router';

import { useTitleBarContent } from '@/hooks/use_title_bar';

import { PageFrame } from '@/components/page_frame';

type NavCardProps = {
	title: string;
	description: string;
	to: string;
	icon?: ReactNode;
};

const docsCards: NavCardProps[] = [
	{
		title: 'Getting Started',
		description: 'Connect a provider key, choose a workflow setup, and send your first useful message.',
		to: '/docs/#getting-started',
		icon: <FiMessageSquare size={18} />,
	},
	{
		title: 'Chats and Composer',
		description: 'Run repeatable workflows from chat tabs with attachments, prompts, tools, and model choices.',
		to: '/docs/#chats-composer-and-everyday-workflow',
		icon: <FiMessageSquare size={18} />,
	},
	{
		title: 'Context and Automation',
		description: 'Combine attachments, prompts, tools, skills, web search, and auto-execute flows.',
		to: '/docs/#attachments-tools-skills-and-prompts',
		icon: <FiLayers size={18} />,
	},
];

type WorkflowStarter = {
	title: string;
	description: string;
	workflowID: string;
	prompt: string;
	icon: ReactNode;
	assistantPresetBundleID?: string;
	assistantPresetSlug?: string;
	assistantPresetVersion?: string;
};

function buildWorkflowStarterHref(workflow: WorkflowStarter) {
	const params = new URLSearchParams({
		workflow: workflow.workflowID,
		draft: workflow.prompt,
	});

	if (workflow.assistantPresetBundleID) {
		params.set('assistantPresetBundleID', workflow.assistantPresetBundleID);
	}

	if (workflow.assistantPresetSlug) {
		params.set('assistantPresetSlug', workflow.assistantPresetSlug);
	}

	if (workflow.assistantPresetVersion) {
		params.set('assistantPresetVersion', workflow.assistantPresetVersion);
	}

	return `/chats?${params.toString()}`;
}

const SOFTWARE_ASSISTANTS_BUNDLE_ID = '019d676e-2533-7fdf-a0af-d3a571ab4f4f';
const CORE_ASSISTANTS_BUNDLE_ID = '019d2423-01b0-7f87-be39-fe02d844453a';

const workflowStarters: WorkflowStarter[] = [
	{
		title: 'Analyze File',
		description: 'Attach a file and get a guided explanation of purpose, or any other query.',
		workflowID: 'analyze-file',
		assistantPresetBundleID: CORE_ASSISTANTS_BUNDLE_ID,
		assistantPresetSlug: 'fs-exec',
		assistantPresetVersion: 'v1.0.0',
		prompt: [
			'<Attach the file or paste/replace this line with file contents, or replace this line with the full local path to inspect: `/absolute/path/to/file`>',
			'',
			'Analyze and explain this file. Cover purpose, main flows, important analysis and dependencies/references.',
		].join('\n'),
		icon: <FiFileText size={18} />,
	},
	{
		title: 'Code Review',
		description:
			'Review the code or diff for correctness, security, reliability, maintainability, and test gaps. Focus on concrete findings and narrow fixes.',
		workflowID: 'code-review',
		assistantPresetBundleID: SOFTWARE_ASSISTANTS_BUNDLE_ID,
		assistantPresetSlug: 'reviewing-code',
		assistantPresetVersion: 'v1.0.0',
		prompt: [
			'<Attach changed files, paste a diff, or replace this line with the repo path plus commit/branch/PR context: `/absolute/path/to/repo`>',
			'',
			'Review the code or diff for correctness, security, reliability, maintainability, and test gaps. Focus on concrete findings and narrow fixes.',
		].join('\n'),
		icon: <FiCode size={18} />,
	},
	{
		title: 'Bug Investigation',
		description:
			'Investigate a bug. Identify likely root cause, evidence, missing evidence, the smallest safe fix direction, and verification steps.',
		workflowID: 'bug-investigation',
		assistantPresetBundleID: CORE_ASSISTANTS_BUNDLE_ID,
		assistantPresetSlug: 'fs-exec',
		assistantPresetVersion: 'v1.0.0',
		prompt: [
			'<Paste the error, stack trace, failing test output, or replace this line with the relevant repo/file path: `/absolute/path/to/repo-or-file`>',
			'',
			'Investigate this bug. Identify likely root cause, evidence, missing evidence, the smallest safe fix direction, and verification steps.',
		].join('\n'),
		icon: <FiAlertTriangle size={18} />,
	},
	{
		title: 'Architecture Review',
		description:
			'Review the architecture. Identify boundaries, risks, coupling, unclear ownership, and the simplest viable improvements.',
		workflowID: 'architecture-review',
		assistantPresetBundleID: SOFTWARE_ASSISTANTS_BUNDLE_ID,
		assistantPresetSlug: 'designing-system-architecture',
		assistantPresetVersion: 'v1.0.0',
		prompt: [
			'<Attach the README/design docs, paste architecture notes, or replace this line with the repo/docs path to inspect: `/absolute/path/to/repo-or-docs`>',
			'',
			'Review the architecture. Identify boundaries, risks, coupling, unclear ownership, and the simplest viable improvements.',
		].join('\n'),
		icon: <FiLayers size={18} />,
	},
];

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

function WorkflowStarterCard({ workflow }: { workflow: WorkflowStarter }) {
	return (
		<Link to={buildWorkflowStarterHref(workflow)} className="group block h-full">
			<div className="bg-base-100 border-base-300/70 hover:border-primary/40 flex h-full flex-col rounded-2xl border shadow-md transition-all duration-200 hover:-translate-y-1 hover:shadow-xl">
				<div className="flex flex-col p-4">
					<div className="flex items-center justify-between p-0">
						<div className="flex items-center gap-2">
							<div className="bg-primary/10 text-primary rounded-xl p-2">{workflow.icon}</div>
							<h3 className="text-sm font-semibold">{workflow.title}</h3>
						</div>
						<FiArrowRight size={18} className="transition-transform group-hover:translate-x-1" />
					</div>
					<div className="text-base-content/70 mt-1 text-xs leading-relaxed">{workflow.description}</div>
				</div>
			</div>
		</Link>
	);
}

function DocsCard({ title, description, to }: NavCardProps) {
	return (
		<Link to={to} className="group block w-full">
			<div className="bg-base-100 border-base-300/70 hover:border-primary/40 flex h-full flex-col rounded-2xl border shadow-md transition-all duration-200 hover:-translate-y-1 hover:shadow-xl">
				<div className="flex flex-col p-4">
					<div className="flex items-center justify-between p-0">
						<h3 className="text-sm font-semibold">{title}</h3>
						<FiArrowRight size={18} className="transition-transform group-hover:translate-x-1" />
					</div>
					<div className="text-base-content/70 mt-1 text-xs leading-relaxed">{description}</div>
				</div>
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
			<div className="mx-auto flex h-full w-full max-w-6xl flex-col items-center px-4 py-8">
				<div className="flex max-w-5xl flex-col items-center text-center">
					<div className="flex items-center gap-4">
						<img src="/icon.png" alt="FlexiGPT Icon" width={48} height={48} />
						<h1 className="text-xl font-bold md:text-2xl">FlexiGPT</h1>
					</div>

					<p className="text-base-content/70 max-w-5xl text-sm leading-relaxed">
						Local-first AI workspace for reusable assistants/agents, prompts, tools, skills, model choices, attachments,
						and private local history across multiple LLM providers.
					</p>
				</div>

				<div className="mt-8 flex w-full flex-1 flex-col items-center justify-between pb-4 xl:mt-16">
					<PrimaryActionCard
						title="Open Chats Workspace"
						description="Start a new chat/workflow or continue a saved local thread."
						to="/chats/"
						icon={<FiMessageSquare size={24} />}
					/>

					<section className="mt-8 w-full">
						<div className="mx-auto max-w-3xl text-center">
							<h2 className="text-lg font-semibold">Start from workflow</h2>
							<p className="text-base-content/70 text-xs">
								Pick a starter to open Chats with a workflow assistant/agent loaded and a prefilled draft prompt.
							</p>
						</div>

						<div className="mx-auto mt-4 grid w-full max-w-5xl grid-cols-1 gap-4 sm:grid-cols-2">
							{workflowStarters.map(workflow => (
								<WorkflowStarterCard key={workflow.workflowID} workflow={workflow} />
							))}
						</div>
					</section>

					<section className="mt-8 w-full">
						<div className="mx-auto max-w-3xl text-center">
							<Link
								to="/docs/"
								className="inline-flex items-center gap-2 text-lg font-semibold transition-opacity hover:opacity-80"
							>
								<FiBookOpen size={24} />
								Documentation
								<div className="flex justify-end">
									<FiArrowRight size={24} className="transition-transform group-hover:translate-x-1" />
								</div>
							</Link>
							<p className="text-base-content/70 text-xs">
								Bundled guide for workflows, local context, automation, tools, skills, and architecture.
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
