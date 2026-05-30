import { IS_WAILS_PLATFORM } from '@/lib/features';
import { setFrontendErrorLogger } from '@/lib/frontend_error_reporter';

import type {
	IAggregateAPI,
	IAssistantPresetStoreAPI,
	IAttachmentsDropAPI,
	IBackendAPI,
	IConversationStoreAPI,
	ILogger,
	IMCPAPI,
	IModelPresetStoreAPI,
	IPromptStoreAPI,
	ISettingStoreAPI,
	ISkillStoreAPI,
	IToolRuntimeAPI,
	IToolStoreAPI,
} from '@/apis/interface';
import * as wailsImpl from '@/apis/wailsapi';

export let log: ILogger;

export let attachmentsDropAPI: IAttachmentsDropAPI;
export let backendAPI: IBackendAPI;
export let conversationStoreAPI: IConversationStoreAPI;
export let aggregateAPI: IAggregateAPI;
export let settingstoreAPI: ISettingStoreAPI;
export let modelPresetStoreAPI: IModelPresetStoreAPI;

/**
 * @public
 */
export let mcpAPI: IMCPAPI;

export let promptStoreAPI: IPromptStoreAPI;
export let toolStoreAPI: IToolStoreAPI;
export let toolRuntimeAPI: IToolRuntimeAPI;
export let skillStoreAPI: ISkillStoreAPI;
export let assistantPresetStoreAPI: IAssistantPresetStoreAPI;

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
	mcpAPI = new wailsImpl.WailsMCPAPI();
	promptStoreAPI = new wailsImpl.WailsPromptStoreAPI();
	toolStoreAPI = new wailsImpl.WailsToolStoreAPI();
	toolRuntimeAPI = new wailsImpl.WailsToolRuntimeAPI();
	skillStoreAPI = new wailsImpl.WailsSkillStoreAPI();
	assistantPresetStoreAPI = new wailsImpl.WailsAssistantPresetStoreAPI();
} else {
	// Error for unsupported platforms
	throw new Error('Unsupported platform');
}
