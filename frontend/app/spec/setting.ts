export const CustomThemeSystem = 'system';
export const CustomThemeLight = 'applight';
export const CustomThemeDark = 'appdark';

const DAISYUI_LIGHT_THEMES = [
	'acid',
	'autumn',
	'bumblebee',
	'caramellatte',
	'cmyk',
	'corporate',
	'cupcake',
	'cyberpunk',
	'emerald',
	'fantasy',
	'garden',
	'lemonade',
	'light',
	'lofi',
	'nord',
	'pastel',
	'retro',
	'silk',
	'valentine',
	'winter',
	'wireframe',
];

const DAISYUI_DARK_THEMES = [
	'abyss',
	'aqua',
	'black',
	'business',
	'coffee',
	'dark',
	'dim',
	'dracula',
	'forest',
	'halloween',
	'luxury',
	'night',
	'sunset',
	'synthwave',
];

export const ALL_DARK_THEMES = [CustomThemeDark, ...DAISYUI_DARK_THEMES];

export const DAISYUI_BUILTIN_THEMES = [...DAISYUI_LIGHT_THEMES, ...DAISYUI_DARK_THEMES] as const;

export enum ThemeType {
	System = 'system',
	Light = 'light',
	Dark = 'dark',
	Other = 'other',
}
export interface AppTheme {
	type: ThemeType;
	name: string;
}

export enum DebugLogLevel {
	Debug = 'debug',
	Info = 'info',
	Warn = 'warn',
	Error = 'error',
}

export interface DebugSettings {
	logLLMReqResp: boolean;
	disableContentStripping: boolean;
	logLevel: DebugLogLevel;
}

export const DEFAULT_DEBUG_SETTINGS: DebugSettings = {
	logLLMReqResp: false,
	disableContentStripping: false,
	logLevel: DebugLogLevel.Info,
};

export type AuthKeyType = string;
export const AuthKeyTypeProvider = 'provider';

export type AuthKeyName = string;

export interface AuthKey {
	secret: string;
	sha256: string;
	nonEmpty: boolean;
}

export interface AuthKeyMeta {
	type: AuthKeyType;
	keyName: AuthKeyName;
	sha256: string;
	nonEmpty: boolean;
}

export interface SettingsSchema {
	appTheme: AppTheme;
	debug: DebugSettings;
	authKeys: AuthKeyMeta[];
}
