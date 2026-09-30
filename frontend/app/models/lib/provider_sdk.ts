import type { ProviderSDKType } from '@/spec/inference';
import { ProviderSDKType as ProviderSDKTypeValue, SDK_DEFAULTS, SDK_DISPLAY_NAME } from '@/spec/inference';

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
		sdkType: ProviderSDKTypeValue.ProviderSDKTypeAnthropic,
		displayName: SDK_DISPLAY_NAME[ProviderSDKTypeValue.ProviderSDKTypeAnthropic],
		defaultPath: SDK_DEFAULTS[ProviderSDKTypeValue.ProviderSDKTypeAnthropic].chatPath,
		defaultAPIKeyHeader: SDK_DEFAULTS[ProviderSDKTypeValue.ProviderSDKTypeAnthropic].apiKeyHeaderKey,
		defaultHeaders: SDK_DEFAULTS[ProviderSDKTypeValue.ProviderSDKTypeAnthropic].defaultHeaders,
	},
	{
		adapter: 'openai.chatCompletions',
		sdkType: ProviderSDKTypeValue.ProviderSDKTypeOpenAIChatCompletions,
		displayName: SDK_DISPLAY_NAME[ProviderSDKTypeValue.ProviderSDKTypeOpenAIChatCompletions],
		defaultPath: SDK_DEFAULTS[ProviderSDKTypeValue.ProviderSDKTypeOpenAIChatCompletions].chatPath,
		defaultAPIKeyHeader: SDK_DEFAULTS[ProviderSDKTypeValue.ProviderSDKTypeOpenAIChatCompletions].apiKeyHeaderKey,
		defaultHeaders: SDK_DEFAULTS[ProviderSDKTypeValue.ProviderSDKTypeOpenAIChatCompletions].defaultHeaders,
	},
	{
		adapter: 'openai.responses',
		sdkType: ProviderSDKTypeValue.ProviderSDKTypeOpenAIResponses,
		displayName: SDK_DISPLAY_NAME[ProviderSDKTypeValue.ProviderSDKTypeOpenAIResponses],
		defaultPath: SDK_DEFAULTS[ProviderSDKTypeValue.ProviderSDKTypeOpenAIResponses].chatPath,
		defaultAPIKeyHeader: SDK_DEFAULTS[ProviderSDKTypeValue.ProviderSDKTypeOpenAIResponses].apiKeyHeaderKey,
		defaultHeaders: SDK_DEFAULTS[ProviderSDKTypeValue.ProviderSDKTypeOpenAIResponses].defaultHeaders,
	},
	{
		adapter: 'google.generateContent',
		sdkType: ProviderSDKTypeValue.ProviderSDKTypeGoogleGenerateContent,
		displayName: SDK_DISPLAY_NAME[ProviderSDKTypeValue.ProviderSDKTypeGoogleGenerateContent],
		defaultPath: SDK_DEFAULTS[ProviderSDKTypeValue.ProviderSDKTypeGoogleGenerateContent].chatPath,
		defaultAPIKeyHeader: SDK_DEFAULTS[ProviderSDKTypeValue.ProviderSDKTypeGoogleGenerateContent].apiKeyHeaderKey,
		defaultHeaders: SDK_DEFAULTS[ProviderSDKTypeValue.ProviderSDKTypeGoogleGenerateContent].defaultHeaders,
	},
];

export function getProviderSDKOptions(): readonly ProviderSDKOption[] {
	return PROVIDER_SDK_OPTIONS;
}

export function getProviderSDKOption(adapter: string): ProviderSDKOption | undefined {
	return PROVIDER_SDK_OPTIONS.find(option => option.adapter === adapter);
}

export function getProviderSDKType(adapter: string): ProviderSDKType {
	return getProviderSDKOption(adapter)?.sdkType ?? ProviderSDKTypeValue.ProviderSDKTypeOpenAIResponses;
}
