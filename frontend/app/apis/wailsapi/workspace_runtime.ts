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
import { enumFromWails, requiredObject } from '@/apis/wailsapi/transport';
import {
	ComposeWorkspacePrompt,
	LoadWorkspaceMCPServers,
	LoadWorkspaceSkills,
	ResolveWorkspaceRuntimePlan,
} from '@/apis/wailsjs/go/main/WorkspaceRuntimeWrapper';

function projectWorkspacePromptPlan(value: unknown, operation: string): WorkspacePromptPlan {
	const plan = requiredObject<WorkspacePromptPlan>(value, operation);

	for (const contribution of plan.contributions) {
		contribution.insert = enumFromWails(
			contribution.insert,
			WorkspaceInsertTarget,
			`${operation}.contributions.insert`
		) as WorkspaceInsertTarget;
	}
	for (const decision of plan.decisions) {
		decision.status = enumFromWails(decision.status, WorkspacePromptCompositionStatus, `${operation}.decisions.status`);
	}

	return plan;
}

function projectWorkspaceSkillLoadPlan(value: unknown, operation: string): WorkspaceSkillLoadPlan {
	const plan = requiredObject<WorkspaceSkillLoadPlan>(value, operation);

	for (const skill of plan.skills) {
		if (skill.insert !== undefined) {
			skill.insert = enumFromWails(skill.insert, WorkspaceInsertTarget, `${operation}.skills.insert`);
		}
	}

	return plan;
}

function projectWorkspaceRuntimePlan(value: unknown, operation: string): WorkspaceRuntimePlan {
	const plan = requiredObject<WorkspaceRuntimePlan>(value, operation);
	plan.prompt = projectWorkspacePromptPlan(plan.prompt, `${operation}.prompt`);
	plan.skills = projectWorkspaceSkillLoadPlan(plan.skills, `${operation}.skills`);
	return plan;
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
		return requiredObject<WorkspaceMCPServerLoadPlan>(
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
