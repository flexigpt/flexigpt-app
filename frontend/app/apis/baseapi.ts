// oxlint-disable import/no-mutable-exports
import { IS_WAILS_PLATFORM } from '@/lib/features';
import { setFrontendErrorLogger } from '@/lib/frontend_error_reporter';

import type {
	IAgentStoreAPI,
	IAttachmentsDropAPI,
	IBackendAPI,
	ILogger,
	IMCPAggregateAPI,
	IMCPRuntimeAPI,
	IMCPStoreAPI,
	IModelAggregateAPI,
	IModelStoreAPI,
	ISessionRuntimeAPI,
	ISessionStoreAPI,
	ISettingStoreAPI,
	ISkillAggregateAPI,
	ISkillStoreAPI,
	ITextStoreAPI,
	IToolStoreAPI,
	IWorkspaceRuntimeAPI,
	IWorkspaceStoreAPI,
} from '@/apis/interface';
import { AgentManagementAPI } from '@/apis/agent_management';
import { MCPManagementAPI } from '@/apis/mcp_management';
import { ModelManagementAPI } from '@/apis/model_management';
import { SessionManagementAPI } from '@/apis/session_management';
import { SkillManagementAPI } from '@/apis/skill_management';
import { ToolManagementAPI } from '@/apis/tool_management';
import { WailsAgentStoreAPI } from '@/apis/wailsapi/agent_store';
import { WailsAttachmentsDropAPI } from '@/apis/wailsapi/attachmentsdrop';
import { WailsBackendAPI, WailsLogger } from '@/apis/wailsapi/backend';
import { WailsMCPAggregateAPI } from '@/apis/wailsapi/mcp_aggregate';
import { WailsMCPRuntimeAPI } from '@/apis/wailsapi/mcp_runtime';
import { WailsMCPStoreAPI } from '@/apis/wailsapi/mcp_store';
import { WailsModelAggregateAPI } from '@/apis/wailsapi/model_aggregate';
import { WailsModelStoreAPI } from '@/apis/wailsapi/model_store';
import { WailsSessionRuntimeAPI } from '@/apis/wailsapi/session_runtime';
import { WailsSessionStoreAPI } from '@/apis/wailsapi/session_store';
import { WailsSettingStoreAPI } from '@/apis/wailsapi/settingstore';
import { WailsSkillAggregateAPI } from '@/apis/wailsapi/skill_aggregate';
import { WailsSkillStoreAPI } from '@/apis/wailsapi/skill_store';
import { WailsTextStoreAPI } from '@/apis/wailsapi/text_store';
import { WailsToolStoreAPI } from '@/apis/wailsapi/tool_store';
import { WailsWorkspaceRuntimeAPI } from '@/apis/wailsapi/workspace_runtime';
import { WailsWorkspaceStoreAPI } from '@/apis/wailsapi/workspace_store';
import { WorkspaceManagementAPI } from '@/apis/workspace_management';

export let log: ILogger;

export let attachmentsDropAPI: IAttachmentsDropAPI;
export let backendAPI: IBackendAPI;
export let settingstoreAPI: ISettingStoreAPI;

let sessionRuntimeAPI: ISessionRuntimeAPI;
let sessionStoreAPI: ISessionStoreAPI;
export let sessionManagementAPI: SessionManagementAPI;

let modelStoreAPI: IModelStoreAPI;
let modelAggregateAPI: IModelAggregateAPI;
export let modelManagementAPI: ModelManagementAPI;

let toolStoreAPI: IToolStoreAPI;
export let toolManagementAPI: ToolManagementAPI;

let skillStoreAPI: ISkillStoreAPI;
let skillAggregateAPI: ISkillAggregateAPI;
export let skillManagementAPI: SkillManagementAPI;

let mcpStoreAPI: IMCPStoreAPI;
let mcpAggregateAPI: IMCPAggregateAPI;
let mcpRuntimeAPI: IMCPRuntimeAPI;
export let mcpManagementAPI: MCPManagementAPI;

let workspaceStoreAPI: IWorkspaceStoreAPI;
let workspaceRuntimeAPI: IWorkspaceRuntimeAPI;
export let workspaceManagementAPI: WorkspaceManagementAPI;

let agentStoreAPI: IAgentStoreAPI;
export let agentManagementAPI: AgentManagementAPI;

/** @public */
export let textStoreAPI: ITextStoreAPI;

if (IS_WAILS_PLATFORM) {
	log = new WailsLogger();
	setFrontendErrorLogger(log);

	attachmentsDropAPI = new WailsAttachmentsDropAPI();
	backendAPI = new WailsBackendAPI();
	settingstoreAPI = new WailsSettingStoreAPI();

	sessionRuntimeAPI = new WailsSessionRuntimeAPI();
	sessionStoreAPI = new WailsSessionStoreAPI();
	sessionManagementAPI = new SessionManagementAPI(sessionRuntimeAPI, sessionStoreAPI);

	modelStoreAPI = new WailsModelStoreAPI();
	modelAggregateAPI = new WailsModelAggregateAPI();
	modelManagementAPI = new ModelManagementAPI(modelStoreAPI, modelAggregateAPI);

	toolStoreAPI = new WailsToolStoreAPI();
	toolManagementAPI = new ToolManagementAPI(toolStoreAPI);

	skillStoreAPI = new WailsSkillStoreAPI();
	skillAggregateAPI = new WailsSkillAggregateAPI();
	skillManagementAPI = new SkillManagementAPI(
		skillStoreAPI,
		skillAggregateAPI,
		toolManagementAPI,
		modelManagementAPI,
		sessionManagementAPI
	);

	mcpStoreAPI = new WailsMCPStoreAPI();
	mcpRuntimeAPI = new WailsMCPRuntimeAPI();
	mcpAggregateAPI = new WailsMCPAggregateAPI();
	mcpManagementAPI = new MCPManagementAPI(
		mcpStoreAPI,
		mcpAggregateAPI,
		mcpRuntimeAPI,
		toolManagementAPI,
		modelManagementAPI
	);

	agentStoreAPI = new WailsAgentStoreAPI();
	textStoreAPI = new WailsTextStoreAPI();
	agentManagementAPI = new AgentManagementAPI(agentStoreAPI, toolManagementAPI, textStoreAPI);

	workspaceStoreAPI = new WailsWorkspaceStoreAPI();
	workspaceRuntimeAPI = new WailsWorkspaceRuntimeAPI();
	workspaceManagementAPI = new WorkspaceManagementAPI(
		workspaceStoreAPI,
		workspaceRuntimeAPI,
		toolManagementAPI,
		modelManagementAPI
	);
} else {
	throw new Error('Unsupported platform');
}
