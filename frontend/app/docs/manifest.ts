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
				summary:
					'Set up a provider key, open the chat workspace, choose a starting setup, and send your first message.',
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
					'Learn how the Chats page is put together, how the composer works, and how a normal conversation flows from setup to response.',
				body: chatsWorkflowBody,
			},
			{
				id: 'attachments-tools-skills-and-prompts',
				title: 'Attachments, Tools, Skills, and Prompts',
				summary:
					'Use context and automation intentionally: attachments, prompt templates, tools, web search, and skills all solve different problems.',
				body: contextAndToolsBody,
			},
			{
				id: 'presets-providers-and-settings',
				title: 'Presets, Providers, and Settings',
				summary:
					'See which admin page owns which reusable building block: assistant presets, prompts, tools, skills, model presets, and app settings.',
				body: presetsProvidersSettingsBody,
			},
			{
				id: 'privacy-storage-and-troubleshooting',
				title: 'Privacy, Storage, and Troubleshooting',
				summary:
					'Understand what stays local, what can be sent to providers, how debug settings affect visibility, and what to check when a workflow fails.',
				body: privacyAndTroubleshootingBody,
			},
		],
	},
	{
		id: 'architecture',
		title: 'Architecture',
		summary: '',
		sections: [
			{
				id: 'architecture-overview',
				title: 'Architecture Overview',
				summary:
					'See the app as a set of stable roles: frontend surfaces, Wails bridge, backend stores and runtimes, providers, and local execution.',
				body: architectureOverviewBody,
			},
			{
				id: 'frontend-roles-and-responsibilities',
				title: 'Frontend Roles and Responsibilities',
				summary:
					'Understand which frontend surfaces act as stitching layers, how the Chats workspace is composed, and what each major module group contributes.',
				body: frontendRolesBody,
			},
			{
				id: 'backend-roles-and-data-flow',
				title: 'Backend Roles and Data Flow',
				summary:
					'Understand the Wails app shell, wrapper boundaries, store packages, runtimes, built-ins, and the end-to-end request path.',
				body: backendRolesBody,
			},
		],
	},
];
