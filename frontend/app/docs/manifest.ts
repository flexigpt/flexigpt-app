import gettingStartedBody from '@/docs/content/01-getting-started.md?raw';
import coreConceptsBody from '@/docs/content/02-core-concepts.md?raw';
import chatsWorkflowBody from '@/docs/content/03-chats-composer-and-everyday-workflow.md?raw';
import contextAndToolsBody from '@/docs/content/04-attachments-tools-skills-prompts.md?raw';
import presetsProvidersSettingsBody from '@/docs/content/05-presets-providers-settings.md?raw';
import storageDataControlTroubleshootingBody from '@/docs/content/06-storage-data-control-and-troubleshooting.md?raw';
import securityPrivacyTrustModelBody from '@/docs/content/07-security-privacy-and-trust-model.md?raw';
import recipesAndStarterWorkflowsBody from '@/docs/content/08-recipes-and-starter-workflows.md?raw';
import providerAndLocalModelSetupBody from '@/docs/content/09-provider-and-local-model-setup.md?raw';
import architectureOverviewBody from '@/docs/content/11-architecture-overview.md?raw';
import backendRolesBody from '@/docs/content/12-backend-roles-and-responsibilities.md?raw';
import frontendRolesBody from '@/docs/content/13-frontend-roles-and-responsibilities.md?raw';
import chatsWorkspaceComposerDesignBody from '@/docs/content/14-chats-workspace-and-composer-design.md?raw';

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
		summary:
			'Get started with FlexiGPT, connect providers, start useful workflows, and understand local-first data handling.',
		sections: [
			{
				id: 'getting-started',
				title: 'Getting Started',
				summary:
					'Connect a provider key, learn the Chats workspace layout, pick a starting setup, and send the first useful request.',
				body: gettingStartedBody,
			},
			{
				id: 'core-concepts',
				title: 'Core Concepts',
				summary:
					'Understand providers, model presets, assistant presets, prompts, tools, skills, attachments, and how request context is assembled.',
				body: coreConceptsBody,
			},
			{
				id: 'chats-composer-and-everyday-workflow',
				title: 'Chats, Composer, and Everyday Workflow',
				summary: 'Learn how the Chats workspace combines tabs, search, the conversation timeline, and the composer.',
				body: chatsWorkflowBody,
			},
			{
				id: 'attachments-tools-skills-and-prompts',
				title: 'Attachments, Tools, Skills, and Prompts',
				summary:
					'Choose when to use source material, reusable prompts, callable tools, workflow modes, and provider-dependent web search.',
				body: contextAndToolsBody,
			},
			{
				id: 'presets-providers-and-settings',
				title: 'Presets, Providers, and Settings',
				summary: 'See which page owns assistant presets, provider and model setup, keys, and debug settings.',
				body: presetsProvidersSettingsBody,
			},
			{
				id: 'storage-data-control-and-troubleshooting',
				title: 'Storage, Data Control, and Troubleshooting',
				summary:
					'Find local data categories, storage locations, backup/reset/export options, and troubleshooting checks.',
				body: storageDataControlTroubleshootingBody,
			},
			{
				id: 'security-privacy-and-trust-model',
				title: 'Security, Privacy, and Trust Model',
				summary:
					'Review trust boundaries for provider requests, keys, attachments, URL fetching, tools, logs, deletion, and safe sends.',
				body: securityPrivacyTrustModelBody,
			},
			{
				id: 'recipes-and-starter-workflows',
				title: 'Recipes and Starter Workflows',
				summary:
					'Outcome-based recipes for code review, docs, research, local files, model comparison, and safe tool use.',
				body: recipesAndStarterWorkflowsBody,
			},
			{
				id: 'provider-and-local-model-setup',
				title: 'Provider and Local Model Setup',
				summary: 'Setup notes for OpenRouter, compatible endpoints, Ollama-style local servers, and llama.cpp.',
				body: providerAndLocalModelSetupBody,
			},
		],
	},
	{
		id: 'architecture',
		title: 'Architecture',
		summary:
			'Reference pages for the system-level view, frontend and backend ownership, and the detailed chats workspace design.',
		sections: [
			{
				id: 'architecture-overview',
				title: 'Architecture Overview',
				summary:
					'See FlexiGPT as a local-first desktop system: frontend surfaces, Wails boundary, backend domains, local storage, and external provider/runtime edges.',
				body: architectureOverviewBody,
			},
			{
				id: 'backend-roles-and-responsibilities',
				title: 'Backend Roles and Responsibilities',
				summary: 'See which backend domains own storage, search, execution, and runtime concerns.',
				body: backendRolesBody,
			},
			{
				id: 'frontend-roles-and-responsibilities',
				title: 'Frontend Roles and Responsibilities',
				summary: 'See how the frontend owns surfaces, workspace orchestration, and the typed boundary to the backend.',
				body: frontendRolesBody,
			},
			{
				id: 'chats-workspace-and-composer-design',
				title: 'Chats Workspace and Composer Design',
				summary: 'See the detailed design of the main workspace, its tab model, and the composer internals.',
				body: chatsWorkspaceComposerDesignBody,
			},
		],
	},
];
