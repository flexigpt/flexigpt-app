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
		summary: 'Set up a runnable model, understand the chat workflow, and send a focused first request.',
		sections: [
			{
				id: 'getting-started',
				title: 'Getting Started',
				summary: 'Configure a provider and model, open Chats, add focused context, and send a first request.',
				body: gettingStartedBody,
			},
			{
				id: 'concepts-and-ownership',
				title: 'How FlexiGPT Fits Together',
				summary: 'Learn the main terms, resolution states, and which page owns each part of a workflow.',
				body: conceptsAndOwnershipBody,
			},
			{
				id: 'chat-workspace',
				title: 'Chat Workspace',
				summary:
					'Use tabs, saved conversations, Agent starters, runtime controls, message review, and edit/resend flows.',
				body: chatWorkspaceBody,
			},
		],
	},
	{
		id: 'context-and-setup',
		title: 'Context and Reusable Setup',
		summary:
			'Choose context for the next Run, then maintain the reusable Agents, Workspaces, Skills, tools, and MCP servers it uses.',
		sections: [
			{
				id: 'composer-context',
				title: 'Composer Context',
				summary: 'Choose attachments, Workspaces, tools, Skills, MCP context, and provider-compatible web search.',
				body: composerContextBody,
			},
			{
				id: 'agents',
				title: 'Agents',
				summary:
					'Apply reusable Agent recipes, inspect their resolved contributions, and manage imported Agent declarations.',
				body: agentsBody,
			},
			{
				id: 'workspaces',
				title: 'Workspaces',
				summary: 'Use repository and folder setup to select reusable project context, Skills, and MCP servers.',
				body: workspacesBody,
			},
			{
				id: 'mcp-servers',
				title: 'MCP Servers',
				summary: 'Configure MCP transports, installation values, authorization, policies, discovery, and MCP Apps.',
				body: mcpServersBody,
			},
			{
				id: 'reusable-catalogs',
				title: 'Manage Reusable Setup',
				summary:
					'Use this page as a short ownership map for catalog management pages without repeating their detailed guides.',
				body: reusableCatalogsBody,
			},
		],
	},
	{
		id: 'providers-and-safety',
		title: 'Providers, Safety, and Help',
		summary:
			'Configure provider endpoints and models, use local runtimes, understand request boundaries, and troubleshoot.',
		sections: [
			{
				id: 'providers-and-models',
				title: 'Providers and Models',
				summary: 'Configure provider compatibility, credentials, models, defaults, and custom compatible endpoints.',
				body: providersAndModelsBody,
			},
			{
				id: 'privacy-data-and-troubleshooting',
				title: 'Privacy, Data, and Troubleshooting',
				summary: 'Understand local data, outbound request context, secrets, logs, exports, and common checks.',
				body: privacyDataTroubleshootingBody,
			},
			{
				id: 'local-llm-setup',
				title: 'Local LLM Setup',
				summary: 'Connect a local or self-hosted endpoint by copying or adding a provider, then configuring a model.',
				body: localLLMSetupBody,
			},
		],
	},
	{
		id: 'recipes',
		title: 'Everyday Tasks',
		summary:
			'Use focused starting points for development, review, investigation, writing, research, and reviewed local changes.',
		sections: [
			{
				id: 'everyday-recipes',
				title: 'Everyday Recipes',
				summary: 'Use outcome-focused prompts while keeping model, context, and execution choices intentional.',
				body: everydayRecipesBody,
			},
			{
				id: 'unified-diff-apply',
				title: 'Apply a Suggested Change',
				summary: 'Dry-run a suggested patch, resolve target paths or conflicts, and apply reviewed local changes.',
				body: unifiedDiffApplyBody,
			},
			{
				id: 'setup-recipes',
				title: 'Setup Recipes',
				summary: 'Use short checklists for common setup tasks, with links to the page that owns the full procedure.',
				body: setupRecipesBody,
			},
		],
	},
];
