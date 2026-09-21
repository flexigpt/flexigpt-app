import type { ArtifactRef } from '@/spec/artifact';
import type {
	WorkspaceMCPServerLoadPlan,
	WorkspacePromptPlan,
	WorkspaceRuntimePlan,
	WorkspaceRuntimeSelection,
	WorkspaceSkill,
	WorkspaceSkillLoadPlan,
} from '@/spec/workspace_store';

import type { IWorkspaceRuntimeAPI } from '@/apis/interface';
import { requiredObject, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
import {
	ComposeWorkspacePrompt,
	ListWorkspaceSkills,
	LoadWorkspaceMCPServers,
	LoadWorkspaceSkills,
	ResolveWorkspaceRuntimePlan,
} from '@/apis/wailsjs/go/main/WorkspaceRuntimeWrapper';

export class WailsWorkspaceRuntimeAPI implements IWorkspaceRuntimeAPI {
	async composeWorkspacePrompt(workspace: ArtifactRef, artifacts: ArtifactRef[]): Promise<WorkspacePromptPlan> {
		return requiredObject<WorkspacePromptPlan>(
			await ComposeWorkspacePrompt(
				workspace as Parameters<typeof ComposeWorkspacePrompt>[0],
				artifacts as Parameters<typeof ComposeWorkspacePrompt>[1]
			),
			'ComposeWorkspacePrompt'
		);
	}

	async listWorkspaceSkills(workspace: ArtifactRef): Promise<WorkspaceSkill[]> {
		return wailsObjectArrayOrEmpty<WorkspaceSkill>(
			await ListWorkspaceSkills(workspace as Parameters<typeof ListWorkspaceSkills>[0]),
			'ListWorkspaceSkills'
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
		return requiredObject<WorkspaceSkillLoadPlan>(
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
		return requiredObject<WorkspaceRuntimePlan>(
			await ResolveWorkspaceRuntimePlan(
				workspace as Parameters<typeof ResolveWorkspaceRuntimePlan>[0],
				selection as Parameters<typeof ResolveWorkspaceRuntimePlan>[1]
			),
			'ResolveWorkspaceRuntimePlan'
		);
	}
}
