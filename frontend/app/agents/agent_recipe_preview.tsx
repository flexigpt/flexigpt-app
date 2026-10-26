import { useCallback } from 'react';
import { FiAlertCircle, FiCheck } from 'react-icons/fi';

import type { AgentResolution } from '@/spec/agent';
import { CapabilityResolutionStatus } from '@/spec/artifact';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { agentManagementAPI } from '@/apis/baseapi';

interface AgentRecipePreviewProps {
	resolution: AgentResolution | null;
}

export function AgentRecipePreview({ resolution }: AgentRecipePreviewProps) {
	const loadRecipe = useCallback(
		async (_signal: AbortSignal) => {
			if (!resolution) {
				return null;
			}
			return agentManagementAPI.prepareAgentStarterFromResolution(resolution);
		},
		[resolution]
	);

	const {
		data: recipe,
		error,
		isLoading,
	} = useAsyncResource(loadRecipe, {
		initialData: null,
	});

	if (!resolution) {
		return null;
	}

	if (isLoading && !recipe) {
		return <div className="text-base-content/70 text-sm">Preparing Composer starter recipe...</div>;
	}

	if (error) {
		return (
			<div className="alert alert-warning rounded-2xl text-sm">
				<FiAlertCircle size={15} />
				<span>{error instanceof Error ? error.message : 'Composer starter preparation failed.'}</span>
			</div>
		);
	}

	if (!recipe) {
		return null;
	}

	const diagnostics = recipe.issues.filter(
		item =>
			item.code !== 'agent_mcp_requires_connection' &&
			!recipe.resolution.capabilities.occurrences.some(
				occurrence => occurrence.path === item.path && occurrence.status !== CapabilityResolutionStatus.Available
			)
	);

	return (
		<div className="space-y-3">
			<div className={`alert ${recipe.canApply ? 'alert-success' : 'alert-warning'} rounded-2xl text-sm`}>
				{recipe.canApply ? <FiCheck size={15} /> : <FiAlertCircle size={15} />}
				<span>
					{recipe.canApply
						? 'Composer starter preview is ready. Final Run preparation resolves declarations again.'
						: 'The current Composer cannot prepare this Agent as a starter.'}
				</span>
			</div>

			{diagnostics.length > 0 ? (
				<div className="space-y-2">
					<div className="text-sm font-semibold">Starter projection diagnostics</div>

					{diagnostics.map(item => (
						<div
							key={`${item.severity}:${item.code}:${item.path ?? ''}`}
							className={`alert ${item.severity === 'error' ? 'alert-error' : 'alert-warning'} rounded-2xl text-sm`}
						>
							<FiAlertCircle size={15} />
							<div>
								<div>{item.message}</div>
							</div>
						</div>
					))}
				</div>
			) : null}
		</div>
	);
}
