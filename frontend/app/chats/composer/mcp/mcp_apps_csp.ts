interface MCPAppUIResourceCSP {
	connectDomains?: string[];
	resourceDomains?: string[];
	frameDomains?: string[];
	baseUriDomains?: string[];
}

interface MCPAppUIResourcePermissions {
	camera?: Record<string, never>;
	microphone?: Record<string, never>;
	geolocation?: Record<string, never>;
	clipboardWrite?: Record<string, never>;
}

export interface MCPAppUIResourceMeta {
	csp?: MCPAppUIResourceCSP;
	permissions?: MCPAppUIResourcePermissions;
	domain?: string;
	prefersBorder?: boolean;
}

function normalizeSource(raw: string): string | undefined {
	const value = raw.trim();
	if (!value) return undefined;

	try {
		const url = new URL(value);

		switch (url.protocol) {
			case 'https:':
			case 'wss:':
				return url.origin;
			case 'http:':
			case 'ws:':
				if (url.hostname === 'localhost' || url.hostname === '127.0.0.1' || url.hostname === '::1') {
					return url.origin;
				}
				return undefined;
			default:
				return undefined;
		}
	} catch {
		return undefined;
	}
}

function normalizeSourceList(values?: string[]): string[] {
	if (!values || values.length === 0) return [];

	const seen = new Set<string>();
	const out: string[] = [];

	for (const value of values) {
		const normalized = normalizeSource(value);
		if (!normalized || seen.has(normalized)) continue;
		seen.add(normalized);
		out.push(normalized);
	}

	return out;
}

function joinSources(values: string[]): string {
	return values.length > 0 ? values.join(' ') : '';
}

export function getMCPAppUIResourceMeta(value: unknown): MCPAppUIResourceMeta | undefined {
	if (!value || typeof value !== 'object' || Array.isArray(value)) return undefined;

	const root = value as Record<string, unknown>;
	const meta = root._meta;

	if (!meta || typeof meta !== 'object' || Array.isArray(meta)) return undefined;

	const metaRecord = meta as Record<string, unknown>;
	const ui = metaRecord.ui;

	if (!ui || typeof ui !== 'object' || Array.isArray(ui)) return undefined;

	return ui as MCPAppUIResourceMeta;
}

export function buildMCPAppCSP(meta?: MCPAppUIResourceMeta): string {
	const declared = meta?.csp;

	const connectDomains = normalizeSourceList(declared?.connectDomains);
	const resourceDomains = normalizeSourceList(declared?.resourceDomains);
	const frameDomains = normalizeSourceList(declared?.frameDomains);
	const baseUriDomains = normalizeSourceList(declared?.baseUriDomains);

	const resourceSources = joinSources(resourceDomains);
	const connectSources = joinSources(connectDomains);
	const frameSources = joinSources(frameDomains);
	const baseUriSources = joinSources(baseUriDomains);

	return [
		"default-src 'none'",
		`script-src 'self' 'unsafe-inline'${resourceSources ? ` ${resourceSources}` : ''}`,
		`style-src 'self' 'unsafe-inline'${resourceSources ? ` ${resourceSources}` : ''}`,
		`img-src 'self' data: blob:${resourceSources ? ` ${resourceSources}` : ''}`,
		`media-src 'self' data: blob:${resourceSources ? ` ${resourceSources}` : ''}`,
		`font-src 'self' data:${resourceSources ? ` ${resourceSources}` : ''}`,
		`connect-src ${connectSources || "'none'"}`,
		`frame-src ${frameSources || "'none'"}`,
		"object-src 'none'",
		`base-uri ${baseUriSources || "'self'"}`,
		"form-action 'none'",
	].join('; ');
}

export function buildMCPAppAllowAttribute(meta?: MCPAppUIResourceMeta): string | undefined {
	const permissions = meta?.permissions;
	if (!permissions) return undefined;

	const allow: string[] = [];

	if (permissions.camera) allow.push('camera');
	if (permissions.microphone) allow.push('microphone');
	if (permissions.geolocation) allow.push('geolocation');
	if (permissions.clipboardWrite) allow.push('clipboard-write');

	return allow.length > 0 ? allow.join('; ') : undefined;
}
