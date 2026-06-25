import type { ReactNode } from 'react';

import { FiAlertTriangle, FiArrowRight, FiCheckCircle, FiCode } from 'react-icons/fi';

import { Link } from 'react-router';

interface WorkflowStarter {
	title: string;
	description: string;
	workflowID: string;
	draft?: string;
	icon: ReactNode;
	assistantPresetBundleID?: string;
	assistantPresetSlug?: string;
	assistantPresetVersion?: string;
}

const SOFTWARE_ASSISTANTS_BUNDLE_ID = '019d676e-2533-7fdf-a0af-d3a571ab4f4f';

export const workflowStarters: WorkflowStarter[] = [
	{
		title: 'Develop a Feature',
		description: 'Implement a bounded code change from a repo path or attached files, using spec driven development',
		workflowID: 'develop-feature',
		assistantPresetBundleID: SOFTWARE_ASSISTANTS_BUNDLE_ID,
		assistantPresetSlug: 'spec-driven-dev',
		assistantPresetVersion: 'v1.0.0',
		icon: <FiCode size={24} />,
	},
	{
		title: 'Review Code',
		description: 'Review the code or diff for correctness, security, reliability, maintainability, and test gaps',
		workflowID: 'code-review',
		assistantPresetBundleID: SOFTWARE_ASSISTANTS_BUNDLE_ID,
		assistantPresetSlug: 'reviewing-code',
		assistantPresetVersion: 'v1.0.0',

		icon: <FiCheckCircle size={24} />,
	},
	{
		title: 'Investigate a Bug',
		description:
			'Diagnose root cause and the smallest safe fix direction, from logs, errors, stack traces, failing outputs, code, and config',
		workflowID: 'bug-investigation',
		assistantPresetBundleID: SOFTWARE_ASSISTANTS_BUNDLE_ID,
		assistantPresetSlug: 'bug-investigator',
		assistantPresetVersion: 'v1.0.0',
		icon: <FiAlertTriangle size={24} />,
	},
];

function buildWorkflowStarterHref(workflow: WorkflowStarter) {
	const params = new URLSearchParams({
		workflow: workflow.workflowID,
	});
	if (workflow.draft?.trim()) {
		params.set('draft', workflow.draft);
	}
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

					<div className="text-base-content/70 mt-1 text-xs/relaxed">{workflow.description}</div>
				</div>
			</div>
		</Link>
	);
}
