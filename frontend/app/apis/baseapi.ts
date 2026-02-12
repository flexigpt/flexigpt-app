import { IS_WAILS_PLATFORM } from '@/lib/features';

import type {
	IBackendAPI,
	IConversationStoreAPI,
	ILogger,
	IModelPresetStoreAPI,
	IPromptStoreAPI,
	IProviderSetAPI,
	ISettingStoreAPI,
	ISkillStoreAPI,
	IToolStoreAPI,
} from '@/apis/interface';
import * as wailsImpl from '@/apis/wailsapi';

export let log: ILogger;
export let backendAPI: IBackendAPI;
export let conversationStoreAPI: IConversationStoreAPI;
export let providerSetAPI: IProviderSetAPI;
export let settingstoreAPI: ISettingStoreAPI;
export let modelPresetStoreAPI: IModelPresetStoreAPI;
export let promptStoreAPI: IPromptStoreAPI;
export let toolStoreAPI: IToolStoreAPI;
export let skillStoreAPI: ISkillStoreAPI;

// Conditional initialization
if (IS_WAILS_PLATFORM) {
	// Initialize with Wails implementations
	log = new wailsImpl.WailsLogger();
	backendAPI = new wailsImpl.WailsBackendAPI();
	conversationStoreAPI = new wailsImpl.WailsConversationStoreAPI();
	providerSetAPI = new wailsImpl.WailsProviderSetAPI();
	settingstoreAPI = new wailsImpl.WailsSettingStoreAPI();
	modelPresetStoreAPI = new wailsImpl.WailsModelPresetStoreAPI();
	promptStoreAPI = new wailsImpl.WailsPromptStoreAPI();
	toolStoreAPI = new wailsImpl.WailsToolStoreAPI();
	skillStoreAPI = new wailsImpl.WailsSkillStoreAPI();
} else {
	// Error for unsupported platforms
	throw new Error('Unsupported platform');
}
