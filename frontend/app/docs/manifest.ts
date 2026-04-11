import gettingStartedBody from '@/docs/content/01-getting-started.md?raw';
import coreConceptsBody from '@/docs/content/02-core-concepts.md?raw';
import appTourAndChatWorkflowBody from '@/docs/content/03-app-tour-and-chat-workflow.md?raw';
import gettingBetterResultsBody from '@/docs/content/04-getting-better-results.md?raw';
import privacyStorageAndUsageBody from '@/docs/content/05-privacy-storage-and-usage.md?raw';

type DocsSection = {
	id: string;
	title: string;
	summary: string;
	body: string;
};

export const docsSections: DocsSection[] = [
	{
		id: 'getting-started',
		title: 'Getting Started',
		summary: 'Set up a provider key, open the chat workspace, choose a starting setup, and send your first message.',
		body: gettingStartedBody,
	},
	{
		id: 'core-concepts',
		title: 'Core Concepts',
		summary:
			'Understand providers, model presets, assistant presets, prompts, tools and tool auto-execute, skills, attachments, and how request context is built.',
		body: coreConceptsBody,
	},
	{
		id: 'app-tour-and-chat-workflow',
		title: 'App Tour and Chat Workflow',
		summary:
			'Learn where each page lives in the app, how the composer works, how rich responses render, and what a normal day-to-day chat workflow looks like.',
		body: appTourAndChatWorkflowBody,
	},
	{
		id: 'getting-better-results',
		title: 'Getting Better Results',
		summary: 'Use presets, context controls, attachments, prompts, tools, and advanced parameters more effectively.',
		body: gettingBetterResultsBody,
	},
	{
		id: 'privacy-storage-and-usage',
		title: 'Privacy, Storage, and Usage',
		summary:
			'See what stays local, what gets sent to providers, how logs behave, and what to watch for with attachments and history.',
		body: privacyStorageAndUsageBody,
	},
];
