// oxlint-disable import/no-mutable-exports
import { IS_WAILS_PLATFORM } from '@/lib/features';
import { setFrontendErrorLogger } from '@/lib/frontend_error_reporter';

import type {
	IAggregateAPI,
	IArtifactStoreAPI,
	IAssistantPresetStoreAPI,
	IAttachmentsDropAPI,
	IBackendAPI,
	IConversationStoreAPI,
	ILogger,
	IMCPAPI,
	IModelPresetStoreAPI,
	ISettingStoreAPI,
	ISkillAggregateAPI,
	ISkillRuntimeAPI,
	ISkillStoreAPI,
	IToolRuntimeAPI,
	IToolStoreAPI,
	IWorkspaceAPI,
} from '@/apis/interface';
import { SkillManagementAPI } from '@/apis/skill_management';
// oxlint-disable-next-line import/no-namespace
import * as wailsImpl from '@/apis/wailsapi';
import { WailsMCPArtifactAPI } from '@/apis/wailsapi/mcp_artifact';
import { WailsSkillAggregateAPI } from '@/apis/wailsapi/skill_aggregate';
import { WailsSkillRuntimeAPI } from '@/apis/wailsapi/skill_runtime';
import { WailsSkillStoreAPI } from '@/apis/wailsapi/skill_store';

export let log: ILogger;

export let attachmentsDropAPI: IAttachmentsDropAPI;
export let backendAPI: IBackendAPI;
export let conversationStoreAPI: IConversationStoreAPI;
export let aggregateAPI: IAggregateAPI;
export let settingstoreAPI: ISettingStoreAPI;
export let modelPresetStoreAPI: IModelPresetStoreAPI;

export let mcpAPI: IMCPAPI;

export let toolStoreAPI: IToolStoreAPI;
export let toolRuntimeAPI: IToolRuntimeAPI;
let skillStoreAPI: ISkillStoreAPI;
let skillAggregateAPI: ISkillAggregateAPI;
let skillRuntimeAPI: ISkillRuntimeAPI;
export let assistantPresetStoreAPI: IAssistantPresetStoreAPI;
export let skillManagementAPI: SkillManagementAPI;
export let artifactStoreAPI: IArtifactStoreAPI;
export let workspaceAPI: IWorkspaceAPI;

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
	mcpAPI = new WailsMCPArtifactAPI();
	toolStoreAPI = new wailsImpl.WailsToolStoreAPI();
	toolRuntimeAPI = new wailsImpl.WailsToolRuntimeAPI();
	assistantPresetStoreAPI = new wailsImpl.WailsAssistantPresetStoreAPI();
	artifactStoreAPI = new wailsImpl.WailsArtifactStoreAPI();
	workspaceAPI = new wailsImpl.WailsWorkspaceAPI();
	skillStoreAPI = new WailsSkillStoreAPI();
	skillAggregateAPI = new WailsSkillAggregateAPI();
	skillRuntimeAPI = new WailsSkillRuntimeAPI();
	skillManagementAPI = new SkillManagementAPI(skillStoreAPI, skillAggregateAPI, skillRuntimeAPI, artifactStoreAPI);
} else {
	// Error for unsupported platforms
	throw new Error('Unsupported platform');
}
