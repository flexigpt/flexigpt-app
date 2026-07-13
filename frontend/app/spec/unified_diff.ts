export enum ApplyUnifiedDiffDiagnosticLevel {
	Info = 'info',
	Warning = 'warning',
	Error = 'error',
}

export interface ApplyUnifiedDiffDiagnostic {
	level: ApplyUnifiedDiffDiagnosticLevel;
	code?: string;
	message: string;
}

export enum ApplyUnifiedDiffStatus {
	Applicable = 'applicable',
	Applied = 'applied',
	AlreadyApplied = 'already_applied',
	NeedsInfo = 'needs_info',
	Conflict = 'conflict',
	Error = 'error',
}

export interface ApplyUnifiedDiffFileTarget {
	fileKey?: string;
	oldPath?: string;
	newPath?: string;
	targetPath: string;
}

export interface ApplyUnifiedDiffArgs {
	diffText: string;
	dryRun?: boolean;
	strict?: boolean;
	fileTargets?: ApplyUnifiedDiffFileTarget[];
	candidatePaths?: string[];
}

export interface ApplyUnifiedDiffFileOut {
	ok: boolean;
	fileKey: string;
	oldPath?: string;
	newPath?: string;
	targetPath?: string;
	resolvedPath?: string;
	status: ApplyUnifiedDiffStatus;
	message?: string;
	candidatePaths?: string[];
	diagnostics?: ApplyUnifiedDiffDiagnostic[];
	hunks: number;
	appliedHunks: number;
	alreadyAppliedHunks: number;
	addedLines: number;
	deletedLines: number;
}

interface ApplyUnifiedDiffSummary {
	files: number;
	hunks: number;
	appliedHunks: number;
	alreadyAppliedHunks: number;
	addedLines: number;
	deletedLines: number;
}

export interface ApplyUnifiedDiffOut {
	ok: boolean;
	dryRun: boolean;
	status: ApplyUnifiedDiffStatus;
	message?: string;
	diagnostics?: ApplyUnifiedDiffDiagnostic[];
	summary: ApplyUnifiedDiffSummary;
	fileTargets?: ApplyUnifiedDiffFileTarget[];
	files?: ApplyUnifiedDiffFileOut[];
}
