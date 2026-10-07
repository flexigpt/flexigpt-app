import type { AppTheme, DebugSettings, SettingsSchema } from '@/spec/setting';
import { DebugLogLevel, DEFAULT_DEBUG_SETTINGS, ThemeType } from '@/spec/setting';

import type { ISettingStoreAPI } from '@/apis/interface';
import type { setting as wailsSettingSpec } from '@/apis/wailsjs/go/models';
import {
	enumFromWails,
	optionalWailsBody,
	requiredWailsResponseBody,
	requireWailsBody,
	requireWailsBoolean,
	requireWailsString,
} from '@/apis/wailsapi/transport';
import { GetSettings, SetAppTheme, SetDebugSettings } from '@/apis/wailsjs/go/main/SettingStoreWrapper';

function booleanOrDefault(value: unknown, fallback: boolean, field: string): boolean {
	if (value === null || value === undefined) {
		return fallback;
	}

	return requireWailsBoolean(value, field);
}

export class WailsSettingStoreAPI implements ISettingStoreAPI {
	async setAppTheme(theme: AppTheme): Promise<void> {
		const r = {
			Body: {
				type: theme.type,
				name: theme.name,
			} as wailsSettingSpec.SetAppThemeRequestBody,
		};
		await SetAppTheme(r as wailsSettingSpec.SetAppThemeRequest);
	}

	async setDebugSettings(settings: DebugSettings): Promise<void> {
		const r = {
			Body: {
				logLLMReqResp: settings.logLLMReqResp,
				disableContentStripping: settings.disableContentStripping,
				logLevel: settings.logLevel,
			},
		} as wailsSettingSpec.SetDebugSettingsRequest;
		await SetDebugSettings(r);
	}

	async getSettings(forceFetch?: boolean): Promise<SettingsSchema> {
		const r: wailsSettingSpec.GetSettingsRequest = {
			ForceFetch: !!forceFetch,
		};
		const body = requiredWailsResponseBody<{
			appTheme: wailsSettingSpec.AppTheme;
			debug?: wailsSettingSpec.DebugSettings;
		}>(await GetSettings(r), 'GetSettings');
		const appTheme = requireWailsBody<wailsSettingSpec.AppTheme>(body.appTheme, 'GetSettings.appTheme');
		const debug = optionalWailsBody<wailsSettingSpec.DebugSettings>(body.debug, 'GetSettings.debug');

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
