import type { AppTheme, DebugSettings, SettingsSchema } from '@/spec/setting';
import { DebugLogLevel, DEFAULT_DEBUG_SETTINGS, ThemeType } from '@/spec/setting';

import type { ISettingStoreAPI } from '@/apis/interface';
import type {
	AppTheme as WailsAppTheme,
	DebugSettings as WailsDebugSettings,
	GetSettingsRequest as WailsGetSettingsRequest,
	SetAppThemeRequest as WailsSetAppThemeRequest,
	SetAppThemeRequestBody as WailsSetAppThemeRequestBody,
	SetDebugSettingsRequest as WailsSetDebugSettingsRequest,
} from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/internal/setting/models';
import {
	enumFromWails,
	enumToWails,
	optionalWailsBody,
	requiredWailsResponseBody,
	requireWailsBody,
	requireWailsBoolean,
	requireWailsString,
} from '@/apis/wailsapi/transport';
import {
	GetSettings,
	SetAppTheme,
	SetDebugSettings,
} from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/cmd/agentgo/settingstorewrapper';

function booleanOrDefault(value: unknown, fallback: boolean, field: string): boolean {
	if (value === null || value === undefined) {
		return fallback;
	}

	return requireWailsBoolean(value, field);
}

export class WailsSettingStoreAPI implements ISettingStoreAPI {
	async setAppTheme(theme: AppTheme): Promise<void> {
		const request: WailsSetAppThemeRequest = {
			Body: {
				type: enumToWails<ThemeType, WailsSetAppThemeRequestBody['type']>(
					theme.type,
					ThemeType,
					'SetAppTheme.Body.type'
				),
				name: theme.name,
			},
		};

		await SetAppTheme(request);
	}

	async setDebugSettings(settings: DebugSettings): Promise<void> {
		const request: WailsSetDebugSettingsRequest = {
			Body: {
				logLLMReqResp: settings.logLLMReqResp,
				disableContentStripping: settings.disableContentStripping,
				logLevel: enumToWails<DebugLogLevel, WailsDebugSettings['logLevel']>(
					settings.logLevel,
					DebugLogLevel,
					'SetDebugSettings.Body.logLevel'
				),
			},
		};

		await SetDebugSettings(request);
	}

	async getSettings(forceFetch?: boolean): Promise<SettingsSchema> {
		const request: WailsGetSettingsRequest = {
			ForceFetch: Boolean(forceFetch),
		};
		const body = requiredWailsResponseBody<{
			appTheme: WailsAppTheme;
			debug?: WailsDebugSettings | null;
		}>(await GetSettings(request), 'GetSettings');
		const appTheme = requireWailsBody<WailsAppTheme>(body.appTheme, 'GetSettings.appTheme');
		const debug = optionalWailsBody<WailsDebugSettings>(body.debug, 'GetSettings.debug');

		return {
			appTheme: {
				type: enumFromWails(appTheme.type, ThemeType, 'settings.appTheme.type'),
				name: requireWailsString(appTheme.name, 'settings.appTheme.name'),
			},
			debug: {
				logLLMReqResp: booleanOrDefault(
					debug?.logLLMReqResp,
					DEFAULT_DEBUG_SETTINGS.logLLMReqResp,
					'settings.debug.logLLMReqResp'
				),
				disableContentStripping: booleanOrDefault(
					debug?.disableContentStripping,
					DEFAULT_DEBUG_SETTINGS.disableContentStripping,
					'settings.debug.disableContentStripping'
				),
				logLevel:
					debug?.logLevel === null || debug?.logLevel === undefined
						? DEFAULT_DEBUG_SETTINGS.logLevel
						: enumFromWails(debug.logLevel, DebugLogLevel, 'settings.debug.logLevel'),
			},
		};
	}
}
