import type {
	AppTheme,
	AuthKey,
	AuthKeyMeta,
	AuthKeyName,
	AuthKeyType,
	DebugSettings,
	SettingsSchema,
} from '@/spec/setting';
import { DebugLogLevel, DEFAULT_DEBUG_SETTINGS, ThemeType } from '@/spec/setting';

import type { ISettingStoreAPI } from '@/apis/interface';
import {
	enumFromWails,
	requireWailsBody,
	requireWailsBoolean,
	requireWailsString,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import { GetAuthKey, GetSettings, SetAppTheme, SetDebugSettings } from '@/apis/wailsjs/go/main/SettingStoreWrapper';
import type { spec as wailsSpec } from '@/apis/wailsjs/go/models';

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
			} as wailsSpec.SetAppThemeRequestBody,
		};
		await SetAppTheme(r as wailsSpec.SetAppThemeRequest);
	}

	async setDebugSettings(settings: DebugSettings): Promise<void> {
		const r = {
			Body: {
				logLLMReqResp: settings.logLLMReqResp,
				disableContentStripping: settings.disableContentStripping,
				logLevel: settings.logLevel,
			},
		} as wailsSpec.SetDebugSettingsRequest;
		await SetDebugSettings(r);
	}

	async getAuthKey(type: AuthKeyType, keyName: AuthKeyName): Promise<AuthKey> {
		const r = {
			Type: type,
			KeyName: keyName,
		};
		const resp = await GetAuthKey(r as wailsSpec.GetAuthKeyRequest);
		const body = requireWailsBody(resp.Body, 'GetAuthKey');
		return {
			secret: requireWailsString(body.secret, 'GetAuthKey.secret'),
			sha256: requireWailsString(body.sha256, 'GetAuthKey.sha256'),
			nonEmpty: requireWailsBoolean(body.nonEmpty, 'GetAuthKey.nonEmpty'),
		};
	}

	async getSettings(forceFetch?: boolean): Promise<SettingsSchema> {
		const r: wailsSpec.GetSettingsRequest = {
			ForceFetch: !!forceFetch,
		};
		const resp = await GetSettings(r);
		const body = requireWailsBody(resp.Body, 'GetSettings');
		const appTheme = requireWailsBody(body.appTheme, 'GetSettings.appTheme');
		const debug =
			body.debug === null || body.debug === undefined ? undefined : requireWailsBody(body.debug, 'GetSettings.debug');

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
			authKeys: wailsObjectArrayOrEmpty<AuthKeyMeta>(body.authKeys, 'GetSettings.authKeys'),
		};
	}
}
