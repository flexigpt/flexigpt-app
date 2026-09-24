import { useCallback } from 'react';
import { FiAlertCircle, FiCheck, FiTool, FiZap } from 'react-icons/fi';

import type { AgentResolution } from '@/spec/agent';
import { ToolStoreChoiceType } from '@/spec/tool';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { agentManagementAPI } from '@/apis/baseapi';

import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';

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
		return (
			<div className="text-base-content/70 text-sm">Resolve the Agent declarations before preparing the recipe.</div>
		);
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

	const modelRelationship = recipe.resolution.capabilities.occurrences.find(
		occurrence => occurrence.type === 'model' && occurrence.status === 'available'
	);
	const modelLabel = modelRelationship?.mapped?.name || modelRelationship?.name;

	return (
		<div className="space-y-3">
			<div className={`alert ${recipe.canApply ? 'alert-success' : 'alert-warning'} rounded-2xl text-sm`}>
				{recipe.canApply ? <FiCheck size={15} /> : <FiAlertCircle size={15} />}
				<span>
					{recipe.canApply
						? 'This Agent can currently be projected into the supported Composer starter recipe.'
						: 'This Agent contains declarations the current Composer projection cannot safely materialize.'}
				</span>
			</div>

			<div className="grid gap-3 md:grid-cols-2">
				<ManagementItemCard
					title="Model"
					subtitle={recipe.modelPresetRef ? modelLabel || 'Selected model' : 'No Agent model selection'}
					metadata={
						recipe.includeModelSystemPrompt !== undefined ? (
							<MetadataPill label="Include model system prompt">
								{recipe.includeModelSystemPrompt ? 'Yes' : 'No'}
							</MetadataPill>
						) : null
					}
				/>

				<ManagementItemCard
					title="MCP"
					subtitle={
						recipe.mcpContext?.servers.length
							? `${recipe.mcpContext.servers.length} MCP server${recipe.mcpContext.servers.length === 1 ? '' : 's'}`
							: 'No MCP servers'
					}
					metadata={
						recipe.mcpContext?.servers.length ? (
							<MetadataPill label="Exposure">All enabled model-visible tools</MetadataPill>
						) : null
					}
				/>
			</div>

			<div className="grid gap-3 md:grid-cols-2">
				<ManagementItemCard
					title="Tools"
					subtitle={`${recipe.toolSelections.length} selected`}
					metadata={
						<>
							<FiTool size={14} />
							<MetadataPill label="Web search">
								{recipe.toolSelections.filter(item => item.choice.toolType === ToolStoreChoiceType.WebSearch).length}
							</MetadataPill>
						</>
					}
				/>

				<ManagementItemCard
					title="Skills"
					subtitle={`${recipe.enabledSkillRefs.length} enabled, ${recipe.activeSkillRefs.length} active`}
					metadata={
						<>
							<FiZap size={14} />
							<MetadataPill label="Instruction Skills">{recipe.instructionSkillRefs.length}</MetadataPill>
						</>
					}
				/>
			</div>

			{recipe.instructionText ? (
				<div className="border-base-content/10 rounded-2xl border p-3">
					<div className="mb-2 text-sm font-semibold">Rendered instruction Skill text</div>
					<pre className="bg-base-300 max-h-48 overflow-auto rounded-xl p-3 text-xs whitespace-pre-wrap">
						{recipe.instructionText}
					</pre>
				</div>
			) : null}

			{recipe.textArtifacts.length > 0 ? (
				<div className="alert alert-warning rounded-2xl text-sm">
					<FiAlertCircle size={15} />
					<span>
						{recipe.textArtifacts.length} text declaration{recipe.textArtifacts.length === 1 ? '' : 's'} cannot
						currently be added to this conversation starter.
					</span>
				</div>
			) : null}

			{recipe.issues.length > 0 ? (
				<div className="space-y-2">
					<div className="text-sm font-semibold">Starter projection diagnostics</div>

					{recipe.issues.map(item => (
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

			<StatusBadge tone={recipe.resolution.capabilities.complete ? 'success' : 'warning'}>
				{recipe.resolution.capabilities.complete ? 'Resolution complete' : 'Resolution partial'}
			</StatusBadge>
		</div>
	);
}
