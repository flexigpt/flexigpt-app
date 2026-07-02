import gettingStartedBody from '@/docs/content/01-getting-started.md?raw';
import conceptsAndOwnershipBody from '@/docs/content/02-concepts-and-ownership.md?raw';
import chatWorkspaceBody from '@/docs/content/03-chat-workspace.md?raw';
import composerContextBody from '@/docs/content/04-composer-context.md?raw';
import reusableCatalogsBody from '@/docs/content/05-reusable-catalogs.md?raw';
import providersAndModelsBody from '@/docs/content/06-providers-and-models.md?raw';
import privacyDataTroubleshootingBody from '@/docs/content/07-privacy-data-and-troubleshooting.md?raw';
import everydayRecipesBody from '@/docs/content/08-everyday-recipes.md?raw';
import setupRecipesBody from '@/docs/content/09-setup-recipes.md?raw';
import mcpServersBody from '@/docs/content/10-mcp-servers.md?raw';
import architectureOverviewBody from '@/docs/content/11-architecture-overview.md?raw';
import backendRolesBody from '@/docs/content/12-backend-roles-and-responsibilities.md?raw';
import frontendRolesBody from '@/docs/content/13-frontend-roles-and-responsibilities.md?raw';
import chatsWorkspaceComposerDesignBody from '@/docs/content/14-chats-workspace-and-composer-design.md?raw';
import unifiedDiffApplyBody from '@/docs/content/15-unified-diff-apply.md?raw';
import localLLMSetupBody from '@/docs/content/16-local-llm-setup.md?raw';

interface DocsSection {
	id: string;
	title: string;
	summary: string;
	body: string;
}

interface DocsCategory {
	id: string;
	title: string;
	summary: string;
	sections: DocsSection[];
}

export const docsCategories: DocsCategory[] = [
	{
		id: 'start-here',
		title: 'Start Here',
		summary: 'Learn the basic workflow, vocabulary, and Chats workspace before configuring advanced features.',
		sections: [
			{
				id: 'getting-started',
				title: 'Getting Started',
				summary:
					'Connect a provider key, choose a model or assistant preset, add useful context, and send a first request.',
				body: gettingStartedBody,
			},
			{
				id: 'concepts-and-ownership',
				title: 'Concepts and Ownership',
				summary: 'Understand the main FlexiGPT terms and which page owns each reusable building block.',
				body: conceptsAndOwnershipBody,
			},
			{
				id: 'chat-workspace',
				title: 'Chat Workspace',
				summary: 'Use tabs, search, the conversation timeline, model controls, assistant presets, and send/edit flows.',
				body: chatWorkspaceBody,
			},
		],
	},
	{
		id: 'context-and-catalogs',
		title: 'Context and Catalogs',
		summary:
			'Use context inside Chats, then maintain reusable MCP servers, prompts, tools, skills, models, and assistant presets on their own pages.',
		sections: [
			{
				id: 'composer-context',
				title: 'Composer Context',
				summary:
					'Attach files, folders, URLs, prompt templates, tools, skills, and web search to the message you are composing.',
				body: composerContextBody,
			},
			{
				id: 'mcp-servers',
				title: 'MCP Servers',
				summary:
					'Configure Model context protocol server bundles, transport, auth, trust, and discovery before selecting the active MCP context in Chats.',
				body: mcpServersBody,
			},
			{
				id: 'reusable-catalogs',
				title: 'Reusable Catalogs',
				summary:
					'Manage assistant presets, prompt templates, tools, skills, model presets, and settings outside the chat flow.',
				body: reusableCatalogsBody,
			},
		],
	},
	{
		id: 'setup-safety-help',
		title: 'Setup, Safety, and Help',
		summary:
			'Configure providers and local endpoints, understand the trust boundary, manage local data, and troubleshoot common issues.',
		sections: [
			{
				id: 'providers-and-models',
				title: 'Providers and Models',
				summary:
					'Set up hosted providers, OpenRouter, custom compatible endpoints, and built-in local or self-hosted runtimes.',
				body: providersAndModelsBody,
			},
			{
				id: 'privacy-data-and-troubleshooting',
				title: 'Privacy, Data, and Troubleshooting',
				summary:
					'Review local storage, provider request boundaries, logs, backup/reset behavior, and troubleshooting checks.',
				body: privacyDataTroubleshootingBody,
			},
			{
				id: 'local-llm-setup',
				title: 'Local LLM Setup',
				summary:
					'Use built-in local runtime presets, fork providers before models, adjust endpoints and capabilities, and test local-only inference safely.',
				body: localLLMSetupBody,
			},
		],
	},
	{
		id: 'recipes',
		title: 'Recipes',
		summary: 'Use outcome-based recipes for work, and setup recipes for configuring reusable workflows.',
		sections: [
			{
				id: 'everyday-recipes',
				title: 'Everyday Recipes',
				summary:
					'Run common workflows such as feature development, code review, bug investigation, documentation, research, and model comparison.',
				body: everydayRecipesBody,
			},
			{
				id: 'unified-diff-apply',
				title: 'Unified Diff Apply',
				summary: 'Review a patch in a code block, run a dry run, and apply the approved local file changes safely.',
				body: unifiedDiffApplyBody,
			},
			{
				id: 'setup-recipes',
				title: 'Setup Recipes',
				summary:
					'Set up OpenRouter, local models, assistant presets, prompt templates, tool-assisted workflows, and skill-backed workflows.',
				body: setupRecipesBody,
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
