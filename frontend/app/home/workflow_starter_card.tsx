import type { ReactNode } from 'react';

import { FiAlertTriangle, FiArrowRight, FiCode, FiFileText } from 'react-icons/fi';

import { Link } from 'react-router';

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

const SOFTWARE_ASSISTANTS_BUNDLE_ID = '019d676e-2533-7fdf-a0af-d3a571ab4f4f';
const CORE_ASSISTANTS_BUNDLE_ID = '019d2423-01b0-7f87-be39-fe02d844453a';

export const workflowStarters: WorkflowStarter[] = [
	{
		title: 'Analyze File',
		description: 'Attach a file and get a guided explanation of purpose, or any other query',
		workflowID: 'analyze-file',
		assistantPresetBundleID: CORE_ASSISTANTS_BUNDLE_ID,
		assistantPresetSlug: 'local-reader',
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
		description: 'Review the code or diff for correctness, security, reliability, maintainability, and test gaps',
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
			'Diagnose root cause, identify missing evidence, the smallest safe fix direction, and verification steps from logs, errors, stack traces, failing outputs, code, and config',
		workflowID: 'bug-investigation',
		assistantPresetBundleID: SOFTWARE_ASSISTANTS_BUNDLE_ID,
		assistantPresetSlug: 'bug-investigator',
		assistantPresetVersion: 'v1.0.0',
		prompt: [
			'<Paste the error, stack trace, failing test output, or replace this line with the relevant repo/file path: `/absolute/path/to/repo-or-file`>',
			'',
			'Investigate this bug. Identify likely root cause, evidence, missing evidence, the smallest safe fix direction, and verification steps.',
		].join('\n'),
		icon: <FiAlertTriangle size={18} />,
	},
];

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

export function WorkflowStarterCard({ workflow }: { workflow: WorkflowStarter }) {
	return (
		<Link to={buildWorkflowStarterHref(workflow)} className="group block h-full">
			<div className="bg-base-100 border-base-300/70 hover:border-primary/40 flex h-full items-center gap-3 rounded-2xl border p-4 shadow-md transition-all duration-200 hover:-translate-y-1 hover:shadow-xl">
				<div className="flex shrink-0 items-center justify-center">
					<div className="bg-primary/10 text-primary rounded-xl p-2">{workflow.icon}</div>
				</div>

				<div className="flex min-w-0 flex-1 flex-col">
					<div className="flex items-center justify-between gap-3">
						<h3 className="text-sm font-semibold">{workflow.title}</h3>
						<FiArrowRight size={18} className="shrink-0 transition-transform group-hover:translate-x-1" />
					</div>

					<div className="text-base-content/70 mt-1 text-xs leading-relaxed">{workflow.description}</div>
				</div>
			</div>
		</Link>
	);
}
