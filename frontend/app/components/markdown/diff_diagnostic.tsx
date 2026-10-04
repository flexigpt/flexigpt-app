import { FiX } from 'react-icons/fi';

import type { ApplyUnifiedDiffDiagnostic } from '@/spec/unified_diff';
import { ApplyUnifiedDiffDiagnosticLevel } from '@/spec/unified_diff';

import { CollapsedDiagnosticGroup } from '@/components/markdown/collapsed_diagnostic_group';
import {
	formatDiagnosticLevelCount,
	getDiagnosticDisplay,
	renderDiagnosticEntry,
} from '@/components/markdown/diagnostic_utils';

interface DiagnosticSeverityCounts {
	total: number;
	error: number;
	warning: number;
	info: number;
}

interface DiagnosticPanelOptions {
	title: string;
	description?: string;
	diagnostics: ApplyUnifiedDiffDiagnostic[];
	className?: string;
	footer?: string;
}

function getHighestDiagnosticLevel(
	diagnostics: ApplyUnifiedDiffDiagnostic[]
): ApplyUnifiedDiffDiagnosticLevel | undefined {
	if (diagnostics.some(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Error)) {
		return ApplyUnifiedDiffDiagnosticLevel.Error;
	}

	if (diagnostics.some(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Warning)) {
		return ApplyUnifiedDiffDiagnosticLevel.Warning;
	}

	if (diagnostics.some(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Info)) {
		return ApplyUnifiedDiffDiagnosticLevel.Info;
	}

	return undefined;
}

function getDiagnosticSeverityCounts(diagnostics: ApplyUnifiedDiffDiagnostic[]): DiagnosticSeverityCounts {
	const counts: DiagnosticSeverityCounts = {
		total: diagnostics.length,
		error: 0,
		warning: 0,
		info: 0,
	};

	for (const diagnostic of diagnostics) {
		switch (diagnostic.level) {
			case ApplyUnifiedDiffDiagnosticLevel.Error:
				counts.error += 1;
				break;
			case ApplyUnifiedDiffDiagnosticLevel.Warning:
				counts.warning += 1;
				break;

			default:
				counts.info += 1;
				break;
		}
	}

	return counts;
}

export function uniqueDiagnostics(
	values: Array<ApplyUnifiedDiffDiagnostic | undefined | null>
): ApplyUnifiedDiffDiagnostic[] {
	const out: ApplyUnifiedDiffDiagnostic[] = [];
	const seen = new Set<string>();

	for (const value of values) {
		if (!value) {
			continue;
		}

		const message = value.message.trim();
		if (!message) {
			continue;
		}

		const level = value.level ?? ApplyUnifiedDiffDiagnosticLevel.Info;
		const code = value.code?.trim() ?? '';
		const key = `${level}\u0000${code}\u0000${message.replaceAll('\\', '/').replaceAll(/\/+/g, '/')}`;

		if (seen.has(key)) {
			continue;
		}

		seen.add(key);
		out.push({
			level,
			code: code || undefined,
			message,
		});
	}

	return out;
}

function renderDiagnosticSeveritySummary(diagnostics: ApplyUnifiedDiffDiagnostic[]) {
	const counts = getDiagnosticSeverityCounts(diagnostics);
	if (counts.total === 0) {
		return null;
	}

	return (
		<div className="flex flex-wrap gap-1.5 text-[11px]">
			{counts.error > 0 ? (
				<span className="badge badge-outline badge-error">
					{counts.error} error{counts.error === 1 ? '' : 's'}
				</span>
			) : null}
			{counts.warning > 0 ? (
				<span className="badge badge-outline badge-warning">
					{counts.warning} warning{counts.warning === 1 ? '' : 's'}
				</span>
			) : null}
			{counts.info > 0 ? (
				<span className="badge badge-outline badge-info">
					{counts.info} info{counts.info === 1 ? '' : 's'}
				</span>
			) : null}
		</div>
	);
}

export function renderDiagnosticsPanel({ title, description, diagnostics, className, footer }: DiagnosticPanelOptions) {
	if (diagnostics.length === 0) {
		return null;
	}

	const highest = getHighestDiagnosticLevel(diagnostics) ?? ApplyUnifiedDiffDiagnosticLevel.Info;
	const headerDisplay = getDiagnosticDisplay(highest);

	const errors = diagnostics.filter(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Error);
	const warnings = diagnostics.filter(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Warning);
	const info = diagnostics.filter(
		diagnostic =>
			diagnostic.level !== ApplyUnifiedDiffDiagnosticLevel.Error &&
			diagnostic.level !== ApplyUnifiedDiffDiagnosticLevel.Warning
	);

	return (
		<div className={`border-base-300 bg-base-100 rounded-xl border p-3 shadow-sm ${className ?? ''}`}>
			<div className="mb-2 flex items-start justify-between gap-3">
				<div className="min-w-0">
					<div className="flex items-center gap-2 text-sm font-semibold">
						<span className={headerDisplay.textClassName}>{headerDisplay.icon}</span>
						<span>{title}</span>
					</div>
					{description ? <div className="text-base-content/60 mt-1 text-xs">{description}</div> : null}
				</div>
				{renderDiagnosticSeveritySummary(diagnostics)}
			</div>

			<div className="space-y-3">
				{errors.length > 0 ? (
					<div className="space-y-2">
						<div className="text-error flex items-center gap-2 text-[11px] font-semibold tracking-wide uppercase">
							<FiX size={11} />
							<span>Errors</span>
							<span className="badge badge-outline badge-error badge-xs">
								{formatDiagnosticLevelCount(ApplyUnifiedDiffDiagnosticLevel.Error, errors.length)}
							</span>
						</div>
						<div className="space-y-2">
							{errors.map((diagnostic, index) => renderDiagnosticEntry(diagnostic, index, 'error'))}
						</div>
					</div>
				) : null}

				<CollapsedDiagnosticGroup
					label="Warnings"
					diagnostics={warnings}
					level={ApplyUnifiedDiffDiagnosticLevel.Warning}
					keyPrefix="warning"
				/>

				<CollapsedDiagnosticGroup
					label="Info"
					diagnostics={info}
					level={ApplyUnifiedDiffDiagnosticLevel.Info}
					keyPrefix="info"
				/>

				{footer ? (
					<div className="border-base-300 bg-base-200/40 text-base-content/60 rounded-lg border px-3 py-2 text-xs">
						{footer}
					</div>
				) : null}
			</div>
		</div>
	);
}
