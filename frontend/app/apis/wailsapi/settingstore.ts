import type {
	AppTheme,
	AuthKey,
	AuthKeyMeta,
	AuthKeyName,
	AuthKeyType,
	DebugLogLevel,
	DebugSettings,
	SettingsSchema,
	ThemeType,
} from '@/spec/setting';
import { DEFAULT_DEBUG_SETTINGS } from '@/spec/setting';

import type { ISettingStoreAPI } from '@/apis/interface';
import { GetAuthKey, GetSettings, SetAppTheme, SetDebugSettings } from '@/apis/wailsjs/go/main/SettingStoreWrapper';
import { type spec as wailsSpec } from '@/apis/wailsjs/go/models';

/**
 * @public
 */
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
		};
		await SetDebugSettings(r as any);
	}

	async getAuthKey(type: AuthKeyType, keyName: AuthKeyName): Promise<AuthKey> {
		const r = {
			Type: type,
			KeyName: keyName,
		};
		const resp = await GetAuthKey(r as wailsSpec.GetAuthKeyRequest);
		return { secret: resp.Body?.secret ?? '', sha256: resp.Body?.sha256 ?? '', nonEmpty: resp.Body?.nonEmpty ?? false };
	}

	async getSettings(forceFetch?: boolean): Promise<SettingsSchema> {
		const r: wailsSpec.GetSettingsRequest = {
			ForceFetch: !!forceFetch,
		};
		const resp = await GetSettings(r);
		const debug = resp.Body?.debug;
		return {
			appTheme: {
				type: resp.Body?.appTheme.type as ThemeType,
				name: resp.Body?.appTheme.name ?? '',
			},
			debug: {
				logLLMReqResp: debug?.logLLMReqResp ?? DEFAULT_DEBUG_SETTINGS.logLLMReqResp,
				disableContentStripping: debug?.disableContentStripping ?? DEFAULT_DEBUG_SETTINGS.disableContentStripping,
				logLevel: (debug?.logLevel as DebugLogLevel) ?? DEFAULT_DEBUG_SETTINGS.logLevel,
			},
			authKeys: resp.Body?.authKeys as AuthKeyMeta[],
		};
	}
}
