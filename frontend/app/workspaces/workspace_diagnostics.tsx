import type { WorkspaceDiagnostic } from '@/spec/workspace';

import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';

import { getDiagnosticTone } from '@/workspaces/lib/workspace_utils';

interface WorkspaceDiagnosticsProps {
	diagnostics?: WorkspaceDiagnostic[];
	emptyMessage?: string;
}

function getLocationLabel(diagnostic: WorkspaceDiagnostic): string | undefined {
	const location = diagnostic.location;
	if (!location) {
		return undefined;
	}

	const path = location.subresourceLocator ?? location.locator;
	const position =
		location.line !== undefined ? `:${location.line}${location.column !== undefined ? `:${location.column}` : ''}` : '';

	return path ? `${path}${position}` : undefined;
}

export function WorkspaceDiagnostics({
	diagnostics = [],
	emptyMessage = 'No diagnostics reported.',
}: WorkspaceDiagnosticsProps) {
	if (diagnostics.length === 0) {
		return <div className="text-base-content/70 text-sm">{emptyMessage}</div>;
	}

	return (
		<div className="space-y-2">
			{diagnostics.map((diagnostic, index) => {
				const location = getLocationLabel(diagnostic);

				return (
					<div
						key={`${diagnostic.code}:${diagnostic.message}:${index}`}
						className="border-base-content/10 rounded-2xl border p-3"
					>
						<div className="flex flex-wrap items-center gap-2">
							<StatusBadge tone={getDiagnosticTone(diagnostic)}>{diagnostic.severity}</StatusBadge>
							<MetadataPill label="Code">{diagnostic.code}</MetadataPill>
						</div>
						<div className="mt-2 text-sm whitespace-pre-wrap">{diagnostic.message}</div>
						{location ? <div className="text-base-content/60 mt-2 font-mono text-xs break-all">{location}</div> : null}
					</div>
				);
			})}
		</div>
	);
}
