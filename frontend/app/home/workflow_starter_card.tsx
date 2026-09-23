import { FiAlertCircle, FiArrowRight } from 'react-icons/fi';

import { Link } from 'react-router';

import type { AgentCatalogOption } from '@/apis/agent_management';

import type { WorkflowStarter } from '@/home/workflow_starter';

function buildWorkflowStarterHref(workflow: WorkflowStarter, agent: AgentCatalogOption) {
	const params = new URLSearchParams({
		workflow: workflow.workflowID,
		agentRootID: agent.ref.rootID,
		agentArtifactID: agent.ref.artifactID,
	});

	return `/chats?${params.toString()}`;
}

interface WorkflowStarterCardProps {
	workflow: WorkflowStarter;
	agent?: AgentCatalogOption;
	loading?: boolean;
}

function StarterCardContent({ workflow, message }: { workflow: WorkflowStarter; message?: string }) {
	return (
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

				{message ? (
					<div className="text-warning mt-2 flex items-start gap-1 text-xs">
						<FiAlertCircle size={12} className="mt-0.5 shrink-0" />
						<span>{message}</span>
					</div>
				) : null}
			</div>
		</div>
	);
}

export function WorkflowStarterCard({ workflow, agent, loading = false }: WorkflowStarterCardProps) {
	if (!agent?.isSelectable) {
		return (
			<Link to="/agents/" className="group block h-full">
				<StarterCardContent
					workflow={workflow}
					message={
						loading
							? 'Loading this starter...'
							: agent?.availabilityReason || 'This starter is unavailable. Open Agents to review or enable it.'
					}
				/>
			</Link>
		);
	}

	return (
		<Link to={buildWorkflowStarterHref(workflow, agent)} className="group block h-full">
			<StarterCardContent workflow={workflow} />
		</Link>
	);
}
