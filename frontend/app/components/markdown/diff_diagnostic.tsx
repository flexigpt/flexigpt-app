import { FiAlertTriangle, FiChevronRight, FiInfo, FiX } from 'react-icons/fi';

import type { ApplyUnifiedDiffDiagnostic, ApplyUnifiedDiffOut } from '@/spec/unified_diff';
import { ApplyUnifiedDiffDiagnosticLevel } from '@/spec/unified_diff';

export type HeaderButtonTone = 'neutral' | 'success' | 'warning' | 'error' | 'info';

export interface DiagnosticSeverityCounts {
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

export function getHighestDiagnosticLevel(
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

export function getDiagnosticSeverityCounts(diagnostics: ApplyUnifiedDiffDiagnostic[]): DiagnosticSeverityCounts {
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

export function formatDiagnosticsTitle(diagnostics: ApplyUnifiedDiffDiagnostic[]): string {
	const counts = getDiagnosticSeverityCounts(diagnostics);
	if (counts.total === 0) {
		return '';
	}

	const summary = [
		counts.error > 0 ? `${counts.error} error${counts.error === 1 ? '' : 's'}` : undefined,
		counts.warning > 0 ? `${counts.warning} warning${counts.warning === 1 ? '' : 's'}` : undefined,
		counts.info > 0 ? `${counts.info} info` : undefined,
	]
		.filter(Boolean)
		.join(', ');

	const messages = diagnostics
		.slice(0, 6)
		.map(diagnostic => `${diagnostic.level}${diagnostic.code ? ` ${diagnostic.code}` : ''}: ${diagnostic.message}`);

	if (diagnostics.length > 6) {
		messages.push(`+${diagnostics.length - 6} more`);
	}

	return [`Patch diagnostics: ${summary}`, ...messages].join('\n');
}

export function getDiagnosticToneFromCounts(counts: DiagnosticSeverityCounts): HeaderButtonTone {
	if (counts.error > 0) {
		return 'error';
	}
	if (counts.warning > 0) {
		return 'warning';
	}
	if (counts.info > 0) {
		return 'info';
	}
	return 'neutral';
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

export function collectPatchLevelDiagnostics(output?: ApplyUnifiedDiffOut): ApplyUnifiedDiffDiagnostic[] {
	return uniqueDiagnostics(output?.diagnostics ?? []);
}

export function collectFileLevelDiagnostics(output?: ApplyUnifiedDiffOut): ApplyUnifiedDiffDiagnostic[] {
	return uniqueDiagnostics((output?.files ?? []).flatMap(file => file.diagnostics ?? []));
}

function uniqueStringsFromDiagnostics(values: Array<ApplyUnifiedDiffDiagnostic | undefined | null>): string[] {
	const out: string[] = [];
	const seen = new Set<string>();

	for (const value of values) {
		const trimmed = value?.message.trim();
		if (!trimmed) {
			continue;
		}

		const key = trimmed.replaceAll('\\', '/').replaceAll(/\/+/g, '/');

		if (seen.has(key)) {
			continue;
		}

		seen.add(key);
		out.push(trimmed);
	}

	return out;
}

export function collectOutputDiagnostics(output?: ApplyUnifiedDiffOut): string[] {
	return uniqueStringsFromDiagnostics([
		...collectPatchLevelDiagnostics(output),
		...collectFileLevelDiagnostics(output),
	]);
}

function getDiagnosticDisplay(level: ApplyUnifiedDiffDiagnosticLevel) {
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

function renderDiagnosticEntry(diagnostic: ApplyUnifiedDiffDiagnostic, index: number, keyPrefix: string) {
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

function formatDiagnosticLevelCount(level: ApplyUnifiedDiffDiagnosticLevel, count: number): string {
	switch (level) {
		case ApplyUnifiedDiffDiagnosticLevel.Error:
			return `${count} error${count === 1 ? '' : 's'}`;
		case ApplyUnifiedDiffDiagnosticLevel.Warning:
			return `${count} warning${count === 1 ? '' : 's'}`;
		default:
			return `${count} info`;
	}
}

function renderCollapsedDiagnosticGroup({
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
	if (diagnostics.length === 0) {
		return null;
	}

	const display = getDiagnosticDisplay(level);

	return (
		<details className="group border-base-300 bg-base-200/40 overflow-hidden rounded-lg border">
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

			<div className="border-base-300 space-y-2 border-t px-3 py-2">
				{diagnostics.map((diagnostic, index) => renderDiagnosticEntry(diagnostic, index, keyPrefix))}
			</div>
		</details>
	);
}

export function renderDiagnosticSeveritySummary(diagnostics: ApplyUnifiedDiffDiagnostic[]) {
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

				{renderCollapsedDiagnosticGroup({
					label: 'Warnings',
					diagnostics: warnings,
					level: ApplyUnifiedDiffDiagnosticLevel.Warning,
					keyPrefix: 'warning',
				})}

				{renderCollapsedDiagnosticGroup({
					label: 'Info',
					diagnostics: info,
					level: ApplyUnifiedDiffDiagnosticLevel.Info,
					keyPrefix: 'info',
				})}

				{footer ? (
					<div className="border-base-300 bg-base-200/40 text-base-content/60 rounded-lg border px-3 py-2 text-xs">
						{footer}
					</div>
				) : null}
			</div>
		</div>
	);
}
