import { useState } from 'react';
import { FiChevronRight } from 'react-icons/fi';

import type { ApplyUnifiedDiffDiagnostic, ApplyUnifiedDiffDiagnosticLevel } from '@/spec/unified_diff';

import {
	formatDiagnosticLevelCount,
	getDiagnosticDisplay,
	renderDiagnosticEntry,
} from '@/components/markdown/diagnostic_utils';

export function CollapsedDiagnosticGroup({
	label,
	diagnostics,
	level,
	keyPrefix,
}: {
	label: string;
	diagnostics: ApplyUnifiedDiffDiagnostic[];
	level: ApplyUnifiedDiffDiagnosticLevel;
	keyPrefix: string;
}) {
	const [isOpen, setIsOpen] = useState(false);
	if (diagnostics.length === 0) {
		return null;
	}

	const display = getDiagnosticDisplay(level);

	return (
		<details
			className="group border-base-300 bg-base-200/40 overflow-hidden rounded-lg border"
			open={isOpen}
			onToggle={event => {
				setIsOpen(event.currentTarget.open);
			}}
		>
			<summary className="flex cursor-pointer list-none items-center justify-between gap-3 px-3 py-2 text-xs font-semibold">
				<span className="inline-flex min-w-0 items-center gap-2">
					<span className={display.textClassName}>{display.icon}</span>
					<span>{label}</span>
				</span>
				<span className="text-base-content/50 inline-flex flex-wrap items-center gap-1 font-normal">
					<FiChevronRight size={11} className="transition group-open:rotate-90" />
					<span className={`badge badge-outline badge-xs ${display.badgeClassName}`}>
						{formatDiagnosticLevelCount(level, diagnostics.length)}
					</span>
				</span>
			</summary>

			{isOpen ? (
				<div className="border-base-300 space-y-2 border-t px-3 py-2">
					{diagnostics.map((diagnostic, index) => renderDiagnosticEntry(diagnostic, index, keyPrefix))}
				</div>
			) : null}
		</details>
	);
}
