import type { ReactNode } from 'react';

import { FiAlertTriangle, FiCheckCircle, FiCode } from 'react-icons/fi';

const SOFTWARE_ASSISTANTS_BUNDLE_ID = '019d676e-2533-7fdf-a0af-d3a571ab4f4f';

export interface WorkflowStarter {
	title: string;
	description: string;
	workflowID: string;
	draft?: string;
	icon: ReactNode;
	assistantPresetBundleID?: string;
	assistantPresetSlug?: string;
	assistantPresetVersion?: string;
}

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
