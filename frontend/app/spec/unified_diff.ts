/**
 * @public
 */
export enum ApplyUnifiedDiffStatus {
	Applicable = 'applicable',
	Applied = 'applied',
	AlreadyApplied = 'already_applied',
	NeedsInfo = 'needs_info',
	Conflict = 'conflict',
	Error = 'error',
}

/**
 * @public
 */
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

/**
 * @public
 */
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
	diagnostics?: string[];
	hunks: number;
	appliedHunks: number;
	alreadyAppliedHunks: number;
	addedLines: number;
	deletedLines: number;
}

/**
 * @public
 */
export interface ApplyUnifiedDiffSummary {
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
	diagnostics?: string[];
	summary: ApplyUnifiedDiffSummary;
	fileTargets?: ApplyUnifiedDiffFileTarget[];
	files?: ApplyUnifiedDiffFileOut[];
}
