export enum ProviderSDKType {
	ProviderSDKTypeAnthropic = 'providerSDKTypeAnthropicMessages',
	ProviderSDKTypeOpenAIChatCompletions = 'providerSDKTypeOpenAIChatCompletions',
	ProviderSDKTypeOpenAIResponses = 'providerSDKTypeOpenAIResponses',
	ProviderSDKTypeGoogleGenerateContent = 'providerSDKTypeGoogleGenerateContent',
}

export interface ProviderSDKOption {
	adapter: string;
	sdkType: ProviderSDKType;
	displayName: string;
	defaultPath: string;
	defaultAPIKeyHeader: string;
	defaultHeaders: Record<string, string>;
}

const SDK_DISPLAY_NAME: Record<ProviderSDKType, string> = {
	[ProviderSDKType.ProviderSDKTypeAnthropic]: 'Anthropic Messages API',
	[ProviderSDKType.ProviderSDKTypeOpenAIChatCompletions]: 'OpenAI ChatCompletions API',
	[ProviderSDKType.ProviderSDKTypeOpenAIResponses]: 'OpenAI Responses API',
	[ProviderSDKType.ProviderSDKTypeGoogleGenerateContent]: 'Google GenAI API',
};

const SDK_DEFAULTS: Record<
	ProviderSDKType,
	{ chatPath: string; apiKeyHeaderKey: string; defaultHeaders: Record<string, string> }
> = {
	[ProviderSDKType.ProviderSDKTypeAnthropic]: {
		chatPath: '/v1/messages',
		apiKeyHeaderKey: 'x-api-key',
		defaultHeaders: {
			'Content-Type': 'application/json',
			'anthropic-version': '2023-06-01',
		},
	},
	[ProviderSDKType.ProviderSDKTypeOpenAIChatCompletions]: {
		chatPath: '/v1/chat/completions',
		apiKeyHeaderKey: 'Authorization',
		defaultHeaders: {
			'Content-Type': 'application/json',
		},
	},
	[ProviderSDKType.ProviderSDKTypeOpenAIResponses]: {
		chatPath: '/v1/responses',
		apiKeyHeaderKey: 'Authorization',
		defaultHeaders: {
			'Content-Type': 'application/json',
		},
	},
	[ProviderSDKType.ProviderSDKTypeGoogleGenerateContent]: {
		chatPath: '/',
		apiKeyHeaderKey: 'x-goog-api-key',
		defaultHeaders: {
			'Content-Type': 'application/json',
		},
	},
};

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
