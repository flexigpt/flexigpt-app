import { ProviderSDKType, SDK_DEFAULTS, SDK_DISPLAY_NAME } from '@/spec/inference';

export interface ProviderSDKOption {
	adapter: string;
	sdkType: ProviderSDKType;
	displayName: string;
	defaultPath: string;
	defaultAPIKeyHeader: string;
	defaultHeaders: Record<string, string>;
}

const PROVIDER_SDK_OPTIONS: ProviderSDKOption[] = [
	{
		adapter: 'anthropic.messages',
		sdkType: ProviderSDKType.ProviderSDKTypeAnthropic,
		displayName: SDK_DISPLAY_NAME[ProviderSDKType.ProviderSDKTypeAnthropic],
		defaultPath: SDK_DEFAULTS[ProviderSDKType.ProviderSDKTypeAnthropic].chatPath,
		defaultAPIKeyHeader: SDK_DEFAULTS[ProviderSDKType.ProviderSDKTypeAnthropic].apiKeyHeaderKey,
		defaultHeaders: SDK_DEFAULTS[ProviderSDKType.ProviderSDKTypeAnthropic].defaultHeaders,
	},
	{
		adapter: 'openai.chatCompletions',
		sdkType: ProviderSDKType.ProviderSDKTypeOpenAIChatCompletions,
		displayName: SDK_DISPLAY_NAME[ProviderSDKType.ProviderSDKTypeOpenAIChatCompletions],
		defaultPath: SDK_DEFAULTS[ProviderSDKType.ProviderSDKTypeOpenAIChatCompletions].chatPath,
		defaultAPIKeyHeader: SDK_DEFAULTS[ProviderSDKType.ProviderSDKTypeOpenAIChatCompletions].apiKeyHeaderKey,
		defaultHeaders: SDK_DEFAULTS[ProviderSDKType.ProviderSDKTypeOpenAIChatCompletions].defaultHeaders,
	},
	{
		adapter: 'openai.responses',
		sdkType: ProviderSDKType.ProviderSDKTypeOpenAIResponses,
		displayName: SDK_DISPLAY_NAME[ProviderSDKType.ProviderSDKTypeOpenAIResponses],
		defaultPath: SDK_DEFAULTS[ProviderSDKType.ProviderSDKTypeOpenAIResponses].chatPath,
		defaultAPIKeyHeader: SDK_DEFAULTS[ProviderSDKType.ProviderSDKTypeOpenAIResponses].apiKeyHeaderKey,
		defaultHeaders: SDK_DEFAULTS[ProviderSDKType.ProviderSDKTypeOpenAIResponses].defaultHeaders,
	},
	{
		adapter: 'google.generateContent',
		sdkType: ProviderSDKType.ProviderSDKTypeGoogleGenerateContent,
		displayName: SDK_DISPLAY_NAME[ProviderSDKType.ProviderSDKTypeGoogleGenerateContent],
		defaultPath: SDK_DEFAULTS[ProviderSDKType.ProviderSDKTypeGoogleGenerateContent].chatPath,
		defaultAPIKeyHeader: SDK_DEFAULTS[ProviderSDKType.ProviderSDKTypeGoogleGenerateContent].apiKeyHeaderKey,
		defaultHeaders: SDK_DEFAULTS[ProviderSDKType.ProviderSDKTypeGoogleGenerateContent].defaultHeaders,
	},
];

export function getProviderSDKOptions(): readonly ProviderSDKOption[] {
	return PROVIDER_SDK_OPTIONS;
}

export function getProviderSDKOption(adapter: string): ProviderSDKOption | undefined {
	return PROVIDER_SDK_OPTIONS.find(option => option.adapter === adapter);
}

export function getProviderSDKType(adapter: string): ProviderSDKType {
	return getProviderSDKOption(adapter)?.sdkType ?? ProviderSDKType.ProviderSDKTypeOpenAIResponses;
}
