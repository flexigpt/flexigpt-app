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
import agentsBody from '@/docs/content/11-agents.md?raw';
import workspacesBody from '@/docs/content/12-workspaces.md?raw';
import unifiedDiffApplyBody from '@/docs/content/13-unified-diff-apply.md?raw';
import localLLMSetupBody from '@/docs/content/14-local-llm-setup.md?raw';

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
		summary: 'Learn the main workflow and get a useful first response before adding more context or setup.',
		sections: [
			{
				id: 'getting-started',
				title: 'Getting Started',
				summary: 'Connect a provider, choose a model or Agent, add useful context, and send a first request.',
				body: gettingStartedBody,
			},
			{
				id: 'concepts-and-ownership',
				title: 'How FlexiGPT Fits Together',
				summary: 'Learn the main terms and where to go when you want to change part of your setup.',
				body: conceptsAndOwnershipBody,
			},
			{
				id: 'chat-workspace',
				title: 'Chat Workspace',
				summary: 'Use tabs, search, the conversation timeline, Agent starters, model controls, and send or edit flows.',
				body: chatWorkspaceBody,
			},
		],
	},
	{
		id: 'context-and-setup',
		title: 'Context and Reusable Setup',
		summary:
			'Use context in Chats, then maintain reusable Agents, Workspaces, connected services, tools, Skills, and models.',
		sections: [
			{
				id: 'composer-context',
				title: 'Composer Context',
				summary: 'Attach files, folders, URLs, Workspaces, tools, Skills, connected services, and web search.',
				body: composerContextBody,
			},
			{
				id: 'agents',
				title: 'Agents',
				summary:
					'Use reusable Agent files as editable starting points for development, review, writing, research, and planning.',
				body: agentsBody,
			},
			{
				id: 'workspaces',
				title: 'Workspaces',
				summary: 'Use repository and folder setup to bring useful local project context into Chats.',
				body: workspacesBody,
			},
			{
				id: 'mcp-servers',
				title: 'MCP Servers',
				summary: 'Set up connected services, authorization, local values, discovery, and safe use in Chats.',
				body: mcpServersBody,
			},
			{
				id: 'reusable-catalogs',
				title: 'Reusable Setup',
				summary:
					'Choose the right page for Agents, Workspaces, tools, Skills, connected services, models, and settings.',
				body: reusableCatalogsBody,
			},
		],
	},
	{
		id: 'providers-and-safety',
		title: 'Providers, Safety, and Help',
		summary: 'Configure providers and local endpoints, understand privacy choices, and troubleshoot common problems.',
		sections: [
			{
				id: 'providers-and-models',
				title: 'Providers and Models',
				summary: 'Set up hosted providers, compatible endpoints, and local or self-hosted model servers.',
				body: providersAndModelsBody,
			},
			{
				id: 'privacy-data-and-troubleshooting',
				title: 'Privacy, Data, and Troubleshooting',
				summary: 'Review local storage, provider request boundaries, logs, backup, reset, and common checks.',
				body: privacyDataTroubleshootingBody,
			},
			{
				id: 'local-llm-setup',
				title: 'Local LLM Setup',
				summary: 'Set up local model servers, provider forks, model presets, and local-only working habits.',
				body: localLLMSetupBody,
			},
		],
	},
	{
		id: 'recipes',
		title: 'Everyday Tasks',
		summary:
			'Use practical starting points for development, review, investigation, research, writing, and safe local changes.',
		sections: [
			{
				id: 'everyday-recipes',
				title: 'Everyday Recipes',
				summary: 'Develop features, review code, investigate bugs, write docs, research topics, and compare models.',
				body: everydayRecipesBody,
			},
			{
				id: 'unified-diff-apply',
				title: 'Apply a Suggested Change',
				summary: 'Review a suggested patch, test it first, and apply approved local file changes safely.',
				body: unifiedDiffApplyBody,
			},
			{
				id: 'setup-recipes',
				title: 'Setup Recipes',
				summary: 'Set up providers, local models, Agents, Skills, Workspaces, tools, and connected services.',
				body: setupRecipesBody,
			},
		],
	},
];
