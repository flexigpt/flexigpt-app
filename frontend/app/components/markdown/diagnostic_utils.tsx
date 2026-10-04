import { FiAlertTriangle, FiInfo, FiX } from 'react-icons/fi';

import type { ApplyUnifiedDiffDiagnostic } from '@/spec/unified_diff';
import { ApplyUnifiedDiffDiagnosticLevel } from '@/spec/unified_diff';

export function getDiagnosticDisplay(level: ApplyUnifiedDiffDiagnosticLevel) {
	switch (level) {
		case ApplyUnifiedDiffDiagnosticLevel.Error:
			return {
				label: 'Error',
				badgeClassName: 'badge-error',
				textClassName: 'text-error',
				icon: <FiX size={12} />,
			};
		case ApplyUnifiedDiffDiagnosticLevel.Warning:
			return {
				label: 'Warning',
				badgeClassName: 'badge-warning',
				textClassName: 'text-warning',
				icon: <FiAlertTriangle size={12} />,
			};
		default:
			return {
				label: 'Info',
				badgeClassName: 'badge-info',
				textClassName: 'text-info',
				icon: <FiInfo size={12} />,
			};
	}
}

export function formatDiagnosticLevelCount(level: ApplyUnifiedDiffDiagnosticLevel, count: number): string {
	switch (level) {
		case ApplyUnifiedDiffDiagnosticLevel.Error:
			return `${count} error${count === 1 ? '' : 's'}`;
		case ApplyUnifiedDiffDiagnosticLevel.Warning:
			return `${count} warning${count === 1 ? '' : 's'}`;
		default:
			return `${count} info`;
	}
}

export function renderDiagnosticEntry(diagnostic: ApplyUnifiedDiffDiagnostic, index: number, keyPrefix: string) {
	const display = getDiagnosticDisplay(diagnostic.level);

	return (
		<div
			key={`${keyPrefix}-${diagnostic.level}-${diagnostic.code ?? 'nocode'}-${diagnostic.message}-${index}`}
			className="border-base-300 bg-base-200/40 text-base-content/75 rounded-lg border px-3 py-2 text-xs"
		>
			<div className="flex items-start gap-2">
				<div className={`mt-0.5 shrink-0 ${display.textClassName}`}>{display.icon}</div>
				<div className="min-w-0 flex-1">
					<div className="flex flex-wrap items-center gap-2">
						<span className={`badge badge-outline badge-xs ${display.badgeClassName}`}>{display.label}</span>
						{diagnostic.code ? <span className="badge badge-ghost badge-xs font-mono">{diagnostic.code}</span> : null}
					</div>
					<div className="mt-1 leading-5 whitespace-pre-wrap">{diagnostic.message}</div>
				</div>
			</div>
		</div>
	);
}
