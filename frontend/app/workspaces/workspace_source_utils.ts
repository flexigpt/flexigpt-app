import type { WorkspaceSourceFormat } from '@/spec/workspace';
import { WorkspaceSourceFormat as WorkspaceSourceFormatValue } from '@/spec/workspace';

export function workspaceSourceExtension(format: WorkspaceSourceFormat): string {
	return format === WorkspaceSourceFormatValue.WorkspaceSourceFormatJSON ? 'json' : 'yaml';
}
