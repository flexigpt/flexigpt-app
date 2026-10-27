import type { MCPResourceRef, MCPRuntimeServerID } from '@/spec/mcp';

import { mcpManagementAPI } from '@/apis/baseapi';

import { ManagementArtifactSource } from '@/components/managementui/management_artifact_source';

export type MCPResourceSourceFormat = 'json' | 'yaml';

function asRecord(value: unknown): Record<string, unknown> | undefined {
	return value !== null && typeof value === 'object' && !Array.isArray(value)
		? (value as Record<string, unknown>)
		: undefined;
}

function decodeBase64UTF8(value: string): string | undefined {
	if (typeof atob !== 'function') {
		return undefined;
	}

	try {
		const binary = atob(value);
		const bytes = Uint8Array.from(binary, character => character.charCodeAt(0));

		return new TextDecoder().decode(bytes);
	} catch {
		return undefined;
	}
}

function sourceFileName(resource: MCPResourceRef, format: MCPResourceSourceFormat): string {
	const candidate = resource.name || resource.displayName || resource.uri;
	const stem =
		candidate
			.trim()
			.toLowerCase()
			.replaceAll(/[^a-z0-9._-]+/g, '-')
			.replaceAll(/^-+|-+$/g, '')
			.slice(0, 96) || 'resource';

	return `${stem}.${format === 'yaml' ? 'yaml' : 'json'}`;
}

function getMCPResourceSourceFormat(mimeType: string | undefined): MCPResourceSourceFormat | undefined {
	const normalized = mimeType?.split(';', 1)[0]?.trim().toLowerCase();

	if (!normalized) {
		return undefined;
	}

	if (normalized === 'application/json' || normalized === 'text/json' || normalized.endsWith('+json')) {
		return 'json';
	}

	if (
		normalized === 'application/yaml' ||
		normalized === 'application/x-yaml' ||
		normalized === 'text/yaml' ||
		normalized === 'text/x-yaml' ||
		normalized.endsWith('+yaml')
	) {
		return 'yaml';
	}

	return undefined;
}

/**
 * MCP resource reads normally return `contents` with one text resource.
 * This accepts the known text and blob shapes without exposing the complete
 * runtime response as an internal JSON document.
 */
function extractMCPResourceText(response: unknown): string {
	const root = asRecord(response);
	const blocks = Array.isArray(root?.contents) ? root.contents : [response];
	const textParts: string[] = [];

	for (const block of blocks) {
		const value = asRecord(block);

		if (typeof value?.text === 'string') {
			textParts.push(value.text);
			continue;
		}

		if (typeof value?.blob === 'string') {
			const decoded = decodeBase64UTF8(value.blob);

			if (decoded !== undefined) {
				textParts.push(decoded);
			}
		}
	}

	if (textParts.length === 0) {
		throw new Error('This MCP resource did not return text content.');
	}

	return textParts.join('\n\n');
}

interface MCPResourceSourceProps {
	server?: MCPRuntimeServerID;
	resource: MCPResourceRef;
}

export function MCPResourceSource({ server, resource }: MCPResourceSourceProps) {
	const format = getMCPResourceSourceFormat(resource.mimeType);

	if (!server || !format) {
		return null;
	}

	return (
		<ManagementArtifactSource
			label={`resource ${format.toUpperCase()}`}
			load={async () => {
				const response = await mcpManagementAPI.readMCPResource(server, resource.uri);

				return {
					data: response,
					text: extractMCPResourceText(response),
					fileName: sourceFileName(resource, format),
					extension: format === 'yaml' ? 'yaml' : 'json',
				};
			}}
			note="This is the resource source returned by the MCP server."
		/>
	);
}
