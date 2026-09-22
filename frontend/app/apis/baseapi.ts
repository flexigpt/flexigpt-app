// oxlint-disable import/no-mutable-exports
import { IS_WAILS_PLATFORM } from '@/lib/features';
import { setFrontendErrorLogger } from '@/lib/frontend_error_reporter';

import type {
	IAgentStoreAPI,
	IAggregateAPI,
	IAttachmentsDropAPI,
	IBackendAPI,
	IConversationStoreAPI,
	ILogger,
	IMCPAggregateAPI,
	IMCPRuntimeAPI,
	IMCPStoreAPI,
	IModelPresetStoreAPI,
	ISettingStoreAPI,
	ISkillAggregateAPI,
	ISkillRuntimeAPI,
	ISkillStoreAPI,
	IToolRuntimeAPI,
	IToolStoreAPI,
	IWorkspaceRuntimeAPI,
	IWorkspaceStoreAPI,
} from '@/apis/interface';
import { AgentManagementAPI } from '@/apis/agent_management';
import { MCPManagementAPI } from '@/apis/mcp_management';
import { SkillManagementAPI } from '@/apis/skill_management';
// oxlint-disable-next-line import/no-namespace
import * as wailsImpl from '@/apis/wailsapi';
import { WailsAgentStoreAPI } from '@/apis/wailsapi/agent_store';
import { WailsMCPAggregateAPI } from '@/apis/wailsapi/mcp_aggregate';
import { WailsMCPRuntimeAPI } from '@/apis/wailsapi/mcp_runtime';
import { WailsMCPStoreAPI } from '@/apis/wailsapi/mcp_store';
import { WailsSkillAggregateAPI } from '@/apis/wailsapi/skill_aggregate';
import { WailsSkillRuntimeAPI } from '@/apis/wailsapi/skill_runtime';
import { WailsSkillStoreAPI } from '@/apis/wailsapi/skill_store';
import { WailsWorkspaceRuntimeAPI } from '@/apis/wailsapi/workspace_runtime';
import { WailsWorkspaceStoreAPI } from '@/apis/wailsapi/workspace_store';
import { WorkspaceManagementAPI } from '@/apis/workspace_management';

export let log: ILogger;

export let agentStoreAPI: IAgentStoreAPI;
export let agentManagementAPI: AgentManagementAPI;
export let attachmentsDropAPI: IAttachmentsDropAPI;
export let backendAPI: IBackendAPI;
export let conversationStoreAPI: IConversationStoreAPI;
export let aggregateAPI: IAggregateAPI;
export let settingstoreAPI: ISettingStoreAPI;
export let modelPresetStoreAPI: IModelPresetStoreAPI;

export let toolStoreAPI: IToolStoreAPI;
export let toolRuntimeAPI: IToolRuntimeAPI;

let skillStoreAPI: ISkillStoreAPI;
let skillAggregateAPI: ISkillAggregateAPI;
let skillRuntimeAPI: ISkillRuntimeAPI;
export let skillManagementAPI: SkillManagementAPI;

let mcpStoreAPI: IMCPStoreAPI;
let mcpAggregateAPI: IMCPAggregateAPI;
let mcpRuntimeAPI: IMCPRuntimeAPI;
export let mcpManagementAPI: MCPManagementAPI;

let workspaceStoreAPI: IWorkspaceStoreAPI;
let workspaceRuntimeAPI: IWorkspaceRuntimeAPI;
export let workspaceManagementAPI: WorkspaceManagementAPI;

// Conditional initialization
if (IS_WAILS_PLATFORM) {
	// Initialize with Wails implementations
	log = new wailsImpl.WailsLogger();
	setFrontendErrorLogger(log);

	attachmentsDropAPI = new wailsImpl.WailsAttachmentsDropAPI();
	backendAPI = new wailsImpl.WailsBackendAPI();
	conversationStoreAPI = new wailsImpl.WailsConversationStoreAPI();
	aggregateAPI = new wailsImpl.WailsAggregateAPI();
	settingstoreAPI = new wailsImpl.WailsSettingStoreAPI();
	modelPresetStoreAPI = new wailsImpl.WailsModelPresetStoreAPI();
	toolStoreAPI = new wailsImpl.WailsToolStoreAPI();
	toolRuntimeAPI = new wailsImpl.WailsToolRuntimeAPI();

	agentStoreAPI = new WailsAgentStoreAPI();
	agentManagementAPI = new AgentManagementAPI(agentStoreAPI, toolStoreAPI, modelPresetStoreAPI);

	skillStoreAPI = new WailsSkillStoreAPI();
	skillAggregateAPI = new WailsSkillAggregateAPI();
	skillRuntimeAPI = new WailsSkillRuntimeAPI();
	skillManagementAPI = new SkillManagementAPI(
		skillRuntimeAPI,
		skillStoreAPI,
		skillAggregateAPI,
		toolStoreAPI,
		modelPresetStoreAPI
	);

	mcpStoreAPI = new WailsMCPStoreAPI();
	mcpRuntimeAPI = new WailsMCPRuntimeAPI();
	mcpAggregateAPI = new WailsMCPAggregateAPI();
	mcpManagementAPI = new MCPManagementAPI(
		mcpStoreAPI,
		mcpAggregateAPI,
		mcpRuntimeAPI,
		toolStoreAPI,
		modelPresetStoreAPI
	);

	workspaceStoreAPI = new WailsWorkspaceStoreAPI();
	workspaceRuntimeAPI = new WailsWorkspaceRuntimeAPI();
	workspaceManagementAPI = new WorkspaceManagementAPI(
		workspaceStoreAPI,
		workspaceRuntimeAPI,
		toolStoreAPI,
		modelPresetStoreAPI
	);
} else {
	// Error for unsupported platforms
	throw new Error('Unsupported platform');
}
