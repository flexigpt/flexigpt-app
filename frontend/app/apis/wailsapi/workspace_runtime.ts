import type { ArtifactRef } from '@/spec/artifact';
import type {
	WorkspaceMCPServerLoadPlan,
	WorkspacePromptPlan,
	WorkspaceRuntimePlan,
	WorkspaceRuntimeSelection,
	WorkspaceSkillLoadPlan,
} from '@/spec/workspace';
import { WorkspaceInsertTarget, WorkspacePromptCompositionStatus } from '@/spec/workspace';

import type { IWorkspaceRuntimeAPI } from '@/apis/interface';
import { capabilityPlanFromWails } from '@/apis/wailsapi/list_item_projection';
import { enumFromWails, requiredObject, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
import {
	ComposeWorkspacePrompt,
	LoadWorkspaceMCPServers,
	LoadWorkspaceSkills,
	ResolveWorkspaceRuntimePlan,
} from '@/apis/wailsjs/go/main/WorkspaceRuntimeWrapper';

function projectWorkspacePromptPlan(value: unknown, operation: string): WorkspacePromptPlan {
	const plan = requiredObject<WorkspacePromptPlan>(value, operation);
	const contributions = wailsObjectArrayOrEmpty<WorkspacePromptPlan['contributions'][number]>(
		plan.contributions,
		`${operation}.contributions`
	);
	const decisions = wailsObjectArrayOrEmpty<WorkspacePromptPlan['decisions'][number]>(
		plan.decisions,
		`${operation}.decisions`
	);

	for (const contribution of contributions) {
		contribution.insert = enumFromWails(
			contribution.insert,
			WorkspaceInsertTarget,
			`${operation}.contributions.insert`
		) as WorkspaceInsertTarget;
	}
	for (const decision of decisions) {
		decision.status = enumFromWails(decision.status, WorkspacePromptCompositionStatus, `${operation}.decisions.status`);
	}

	return {
		...plan,
		contributions,
		decisions,
	};
}

function projectWorkspaceSkillLoadPlan(value: unknown, operation: string): WorkspaceSkillLoadPlan {
	const plan = requiredObject<WorkspaceSkillLoadPlan>(value, operation);
	const skills = wailsObjectArrayOrEmpty<WorkspaceSkillLoadPlan['skills'][number]>(plan.skills, `${operation}.skills`);

	for (const skill of skills) {
		if (skill.insert !== undefined) {
			skill.insert = enumFromWails(skill.insert, WorkspaceInsertTarget, `${operation}.skills.insert`);
		}
	}

	return {
		...plan,
		skills,
	};
}

function projectWorkspaceMCPServerLoadPlan(value: unknown, operation: string): WorkspaceMCPServerLoadPlan {
	const plan = requiredObject<WorkspaceMCPServerLoadPlan>(value, operation);

	return {
		...plan,
		servers: wailsObjectArrayOrEmpty<WorkspaceMCPServerLoadPlan['servers'][number]>(
			plan.servers,
			`${operation}.servers`
		),
	};
}

function projectWorkspaceRuntimePlan(value: unknown, operation: string): WorkspaceRuntimePlan {
	const plan = requiredObject<WorkspaceRuntimePlan>(value, operation);

	return {
		...plan,
		capabilities: capabilityPlanFromWails(plan.capabilities, `${operation}.capabilities`),
		prompt: projectWorkspacePromptPlan(plan.prompt, `${operation}.prompt`),
		skills: projectWorkspaceSkillLoadPlan(plan.skills, `${operation}.skills`),
		mcpServers: projectWorkspaceMCPServerLoadPlan(plan.mcpServers, `${operation}.mcpServers`),
	};
}

export class WailsWorkspaceRuntimeAPI implements IWorkspaceRuntimeAPI {
	async composeWorkspacePrompt(workspace: ArtifactRef, artifacts: ArtifactRef[]): Promise<WorkspacePromptPlan> {
		return projectWorkspacePromptPlan(
			await ComposeWorkspacePrompt(
				workspace as Parameters<typeof ComposeWorkspacePrompt>[0],
				artifacts as Parameters<typeof ComposeWorkspacePrompt>[1]
			),
			'ComposeWorkspacePrompt'
		);
	}

	async loadWorkspaceMCPServers(workspace: ArtifactRef, artifacts: ArtifactRef[]): Promise<WorkspaceMCPServerLoadPlan> {
		return projectWorkspaceMCPServerLoadPlan(
			await LoadWorkspaceMCPServers(
				workspace as Parameters<typeof LoadWorkspaceMCPServers>[0],
				artifacts as Parameters<typeof LoadWorkspaceMCPServers>[1]
			),
			'LoadWorkspaceMCPServers'
		);
	}

	async loadWorkspaceSkills(workspace: ArtifactRef, artifacts: ArtifactRef[]): Promise<WorkspaceSkillLoadPlan> {
		return projectWorkspaceSkillLoadPlan(
			await LoadWorkspaceSkills(
				workspace as Parameters<typeof LoadWorkspaceSkills>[0],
				artifacts as Parameters<typeof LoadWorkspaceSkills>[1]
			),
			'LoadWorkspaceSkills'
		);
	}

	async resolveWorkspaceRuntimePlan(
		workspace: ArtifactRef,
		selection: WorkspaceRuntimeSelection
	): Promise<WorkspaceRuntimePlan> {
		return projectWorkspaceRuntimePlan(
			await ResolveWorkspaceRuntimePlan(
				workspace as Parameters<typeof ResolveWorkspaceRuntimePlan>[0],
				selection as Parameters<typeof ResolveWorkspaceRuntimePlan>[1]
			),
			'ResolveWorkspaceRuntimePlan'
		);
	}
}
