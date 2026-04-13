import gettingStartedBody from '@/docs/content/01-getting-started.md?raw';
import coreConceptsBody from '@/docs/content/02-core-concepts.md?raw';
import chatsWorkflowBody from '@/docs/content/03-chats-composer-and-everyday-workflow.md?raw';
import contextAndToolsBody from '@/docs/content/04-attachments-tools-skills-prompts.md?raw';
import presetsProvidersSettingsBody from '@/docs/content/05-presets-providers-settings.md?raw';
import privacyAndTroubleshootingBody from '@/docs/content/06-privacy-storage-and-troubleshooting.md?raw';
import architectureOverviewBody from '@/docs/content/11-architecture-overview.md?raw';
import backendRolesBody from '@/docs/content/12-backend-roles-and-responsibilities.md?raw';
import backendHldBody from '@/docs/content/13-backend-hld.md?raw';
import frontendRolesBody from '@/docs/content/14-frontend-roles-and-responsibilities.md?raw';
import frontendHldBody from '@/docs/content/15-frontend-hld.md?raw';

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
		summary: 'Reference pages for how the app is organized and how the major domains fit together.',
		sections: [
			{
				id: 'architecture-overview',
				title: 'Architecture Overview',
				summary: 'Understand the app as a set of domains, and the external libraries that supply key capabilities.',
				body: architectureOverviewBody,
			},
			{
				id: 'backend-roles-and-responsibilities',
				title: 'Backend Roles and Responsibilities',
				summary: 'See which backend domains own what.',
				body: backendRolesBody,
			},
			{
				id: 'backend-hld',
				title: 'Backend HLD',
				summary: 'Look at the backend storage choices, module boundaries, and external dependencies.',
				body: backendHldBody,
			},
			{
				id: 'frontend-roles-and-responsibilities',
				title: 'Frontend Roles and Responsibilities',
				summary: 'See how the frontend owns surfaces, state coordination, and the typed boundary to the backend.',
				body: frontendRolesBody,
			},
			{
				id: 'frontend-hld',
				title: 'Frontend HLD',
				summary: 'Look at the frontend route model, docs packaging, rendering approach, and backend boundary.',
				body: frontendHldBody,
			},
		],
	},
];
