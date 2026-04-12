import gettingStartedBody from '@/docs/content/01-getting-started.md?raw';
import coreConceptsBody from '@/docs/content/02-core-concepts.md?raw';
import chatsWorkflowBody from '@/docs/content/03-chats-composer-and-everyday-workflow.md?raw';
import contextAndToolsBody from '@/docs/content/04-attachments-tools-skills-prompts.md?raw';
import presetsProvidersSettingsBody from '@/docs/content/05-presets-providers-settings.md?raw';
import privacyAndTroubleshootingBody from '@/docs/content/06-privacy-storage-and-troubleshooting.md?raw';
import architectureOverviewBody from '@/docs/content/11-architecture-overview.md?raw';
import frontendRolesBody from '@/docs/content/12-frontend-roles-and-responsibilities.md?raw';
import backendRolesBody from '@/docs/content/13-backend-roles-and-data-flow.md?raw';

type DocsSection = {
	id: string;
	title: string;
	summary: string;
	body: string;
};

type DocsCategory = {
	id: string;
	title: string;
	summary: string;
	sections: DocsSection[];
};

export const docsCategories: DocsCategory[] = [
	{
		id: 'user-guide',
		title: 'User Guide',
		summary: '',
		sections: [
			{
				id: 'getting-started',
				title: 'Getting Started',
				summary: 'Connect a provider key, open Chats, choose a starting setup, and send the first useful request.',
				body: gettingStartedBody,
			},
			{
				id: 'core-concepts',
				title: 'Core Concepts',
				summary:
					'Understand the main FlexiGPT layers: providers, model presets, assistant presets, prompts, tools, skills, attachments, and request context.',
				body: coreConceptsBody,
			},
			{
				id: 'chats-composer-and-everyday-workflow',
				title: 'Chats, Composer, and Everyday Workflow',
				summary:
					'Learn how the Chats workspace is structured and how a normal conversation moves from setup to response.',
				body: chatsWorkflowBody,
			},
			{
				id: 'attachments-tools-skills-and-prompts',
				title: 'Attachments, Tools, Skills, and Prompts',
				summary:
					'Choose the right helper for the job: source material, reusable prompts, tools, web search, and workflow modes.',
				body: contextAndToolsBody,
			},
			{
				id: 'presets-providers-and-settings',
				title: 'Presets, Providers, and Settings',
				summary:
					'See which page manages which reusable building block, from assistant presets to providers, keys, and debug settings.',
				body: presetsProvidersSettingsBody,
			},
			{
				id: 'privacy-storage-and-troubleshooting',
				title: 'Privacy, Storage, and Troubleshooting',
				summary: 'Understand what stays local, what can be sent to providers, and what to check when a workflow fails.',
				body: privacyAndTroubleshootingBody,
			},
		],
	},
	{
		id: 'architecture',
		title: 'Architecture',
		summary: 'Advanced reference for how the app is organized and how requests move through it.',
		sections: [
			{
				id: 'architecture-overview',
				title: 'Architecture Overview',
				summary:
					'See the app as a set of stable roles: frontend surfaces, desktop bridge, backend stores and runtimes, providers, and local execution.',
				body: architectureOverviewBody,
			},
			{
				id: 'frontend-roles-and-responsibilities',
				title: 'Frontend Roles and Responsibilities',
				summary:
					'Understand how the frontend coordinates the Chats workspace, docs, management pages, and typed app APIs.',
				body: frontendRolesBody,
			},
			{
				id: 'backend-roles-and-data-flow',
				title: 'Backend Roles and Data Flow',
				summary:
					'Understand local stores, request orchestration, tool and skill runtimes, and the end-to-end request path.',
				body: backendRolesBody,
			},
		],
	},
];
