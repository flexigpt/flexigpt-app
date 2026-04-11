import appSetupAndDocsGuideBody from '@/docs/content/app-setup-and-docs-guide.md?raw';
import gettingBetterResultsBody from '@/docs/content/getting-better-results.md?raw';
import introductionToUsingLLMsBody from '@/docs/content/introduction-to-using-llms.md?raw';
import privacyStorageAndUsageBody from '@/docs/content/privacy-storage-and-usage.md?raw';

type DocsSection = {
	id: string;
	title: string;
	summary: string;
	body: string;
};

export const docsSections: DocsSection[] = [
	{
		id: 'app-setup-and-docs-guide',
		title: 'App Setup and Docs Guide',
		summary: 'Set up a provider key, learn the app layout, and understand how built-in docs fit into the workflow.',
		body: appSetupAndDocsGuideBody,
	},
	{
		id: 'introduction-to-using-llms',
		title: 'Introduction to Using LLMs',
		summary: 'Learn the main LLM concepts, common terms, and what happens when you send a prompt.',
		body: introductionToUsingLLMsBody,
	},
	{
		id: 'getting-better-results',
		title: 'Getting Better Results',
		summary: 'Practical prompting and attachment tips for stronger outputs.',
		body: gettingBetterResultsBody,
	},
	{
		id: 'privacy-storage-and-usage',
		title: 'Privacy, Storage, and Usage Notes',
		summary: 'Review what stays local, what gets sent, and what to check before sending.',
		body: privacyStorageAndUsageBody,
	},
];
