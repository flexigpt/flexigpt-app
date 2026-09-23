import type { ReactNode } from 'react';
import { FiAlertTriangle, FiCheckCircle, FiCode } from 'react-icons/fi';

export interface WorkflowStarter {
	title: string;
	description: string;
	workflowID: string;
	agentName: string;
	icon: ReactNode;
}

/**
 * These names are stable built-in Agent logical names. Their local Artifact
 * refs are resolved at runtime from the Agent catalog, so home cards do not
 * depend on generated or installation-specific Artifact IDs.
 */
export const workflowStarters: WorkflowStarter[] = [
	{
		title: 'Develop a Feature',
		description: 'Plan, implement, and verify a focused change using the repository or files you provide.',
		workflowID: 'develop-feature',
		agentName: 'spec-driven-dev',
		icon: <FiCode size={24} />,
	},
	{
		title: 'Review Code',
		description: 'Review a diff, changed files, or repository area for correctness, risk, and missing tests.',
		workflowID: 'code-review',
		agentName: 'reviewing-code',
		icon: <FiCheckCircle size={24} />,
	},
	{
		title: 'Investigate a Bug',
		description: 'Work from logs, errors, tests, code, and configuration to find the likely cause and next steps.',
		workflowID: 'bug-investigation',
		agentName: 'bug-investigator',
		icon: <FiAlertTriangle size={24} />,
	},
];
