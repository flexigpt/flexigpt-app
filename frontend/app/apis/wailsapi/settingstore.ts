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
import { enumFromWails, requireWailsBody } from '@/apis/wailsapi/transport';
import { GetAuthKey, GetSettings, SetAppTheme, SetDebugSettings } from '@/apis/wailsjs/go/main/SettingStoreWrapper';
import type { spec as wailsSpec } from '@/apis/wailsjs/go/models';

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
		return { secret: body.secret, sha256: body.sha256, nonEmpty: body.nonEmpty };
	}

	async getSettings(forceFetch?: boolean): Promise<SettingsSchema> {
		const r: wailsSpec.GetSettingsRequest = {
			ForceFetch: !!forceFetch,
		};
		const resp = await GetSettings(r);
		const body = requireWailsBody(resp.Body, 'GetSettings');
		const debug = body.debug;
		return {
			appTheme: {
				type: enumFromWails(body.appTheme.type, ThemeType, 'settings.appTheme.type'),
				name: body.appTheme.name,
			},
			debug: {
				logLLMReqResp: debug?.logLLMReqResp ?? DEFAULT_DEBUG_SETTINGS.logLLMReqResp,
				disableContentStripping: debug?.disableContentStripping ?? DEFAULT_DEBUG_SETTINGS.disableContentStripping,
				logLevel: debug?.logLevel
					? enumFromWails(debug.logLevel, DebugLogLevel, 'settings.debug.logLevel')
					: DEFAULT_DEBUG_SETTINGS.logLevel,
			},
			authKeys: body.authKeys as AuthKeyMeta[],
		};
	}
}
