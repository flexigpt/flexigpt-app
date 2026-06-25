// Build the host context returned to the App view in response to `ui/initialize` or `initialize`.
export interface MCPAppInitializeResult {
	protocolVersion: '2026-01-26';
	hostCapabilities: {
		openLinks?: Record<string, never>;
		serverTools?: {
			listChanged?: boolean;
		};
		serverResources?: {
			listChanged?: boolean;
		};
		logging?: Record<string, never>;
		sandbox?: {
			permissions?: Record<string, unknown>;
			csp?: Record<string, unknown>;
		};
	};
	hostInfo: {
		name: string;
		version: string;
		platform: 'desktop';
	};
	hostContext: {
		theme: 'light' | 'dark';
		displayMode: 'inline';
		availableDisplayModes: Array<'inline'>;
		locale?: string;
		timeZone?: string;
		userAgent?: string;
		platform: 'desktop';
		containerDimensions?: {
			width?: number;
			height?: number;
			maxWidth?: number;
			maxHeight?: number;
		};
	};
}

function getPreferredTheme(): 'light' | 'dark' {
	if (typeof document !== 'undefined') {
		const attr = document.documentElement.dataset.theme;
		if (attr && /dark/i.test(attr)) return 'dark';
	}
	if (typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches) return 'dark';
	return 'light';
}

export function buildMCPAppHostContext(opts?: {
	width?: number;
	height?: number;
	allowToolCalls?: boolean;
}): MCPAppInitializeResult {
	const theme: 'light' | 'dark' = getPreferredTheme();
	return {
		protocolVersion: '2026-01-26',
		hostCapabilities: {
			openLinks: {},
			serverResources: {
				listChanged: true,
			},
			serverTools: opts?.allowToolCalls
				? {
						listChanged: true,
					}
				: undefined,
			logging: {},
			sandbox: {},
		},
		hostInfo: {
			name: 'FlexiGPT',
			version: 'dev',
			platform: 'desktop',
		},
		hostContext: {
			theme,
			displayMode: 'inline',
			availableDisplayModes: ['inline'],
			locale: navigator.language,
			timeZone: Intl.DateTimeFormat().resolvedOptions().timeZone,
			userAgent: 'FlexiGPT/dev',
			platform: 'desktop',
			containerDimensions: {
				width: opts?.width,
				height: opts?.height,
			},
		},
	};
}
