import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FiAlertTriangle, FiCheckCircle, FiGitPullRequest, FiInfo, FiLoader } from 'react-icons/fi';

import { useDiffApplyController } from '@/components/markdown/diff_apply_controller';
import { DiffApplyModal } from '@/components/markdown/diff_apply_modal';
import { parseInteractiveDiff } from '@/components/markdown/diff_apply_model';

interface DiffApplyControlProps {
	language: string;
	diffText: string;
	isBusy: boolean;
	candidatePaths?: string[];
	workspaceRoots?: string[];

	/**
	 * Incremented only after the owning active message observed a stream settle.
	 * Historical messages retain zero.
	 */
	autoReviewEpoch?: number;
}

function getButtonClassName(tone: 'neutral' | 'success' | 'warning' | 'error' | 'info' = 'neutral'): string {
	const toneClassName = {
		neutral: 'app-text-code',
		success: 'text-success',
		warning: 'text-warning',
		error: 'text-error',
		info: 'text-info',
	}[tone];

	return `inline-flex h-6 items-center gap-1 rounded-full border border-base-300 px-2 text-[11px] font-medium leading-none whitespace-nowrap transition-colors hover:opacity-60 disabled:cursor-not-allowed disabled:opacity-50 ${toneClassName}`;
}

interface DiffApplySessionProps {
	language: string;
	diffText: string;
	candidatePaths?: string[];
	workspaceRoots?: string[];
	autoReviewEpoch: number;
}

function DiffApplySession({
	language,
	diffText,
	candidatePaths,
	workspaceRoots,
	autoReviewEpoch,
}: DiffApplySessionProps) {
	/*
	 * This memoization is required for correctness, not merely performance.
	 *
	 * The controller intentionally compares parsed file object identity to
	 * reject stale requests. Re-parsing on every busy/state render creates new
	 * file objects and makes a just-started review appear stale.
	 */
	const parsed = useMemo(() => parseInteractiveDiff(diffText, language), [diffText, language]);
	const controller = useDiffApplyController(parsed, candidatePaths, workspaceRoots);

	const [isDetailsOpen, setIsDetailsOpen] = useState(false);
	const lastAutomaticEpochRef = useRef(autoReviewEpoch);

	const startReview = useCallback((): void => {
		/*
		 * reviewAll synchronously claims the controller operation slot before
		 * its first await. controller.busy is the sole reviewing-state source.
		 */
		void controller.reviewAll();
		// oxlint-disable-next-line react-hooks/exhaustive-deps
	}, [controller.reviewAll]);

	useEffect(() => {
		if (autoReviewEpoch <= 0 || autoReviewEpoch === lastAutomaticEpochRef.current) {
			return;
		}

		lastAutomaticEpochRef.current = autoReviewEpoch;
		startReview();
	}, [autoReviewEpoch, startReview]);

	const controllerBusy = controller.busy !== null;
	const reviewInProgress = controller.busy?.kind === 'review';
	const canInspect = controller.hasReviewResult || controller.patchDiagnostics.length > 0;
	const completedCount = controller.appliedFileCount + controller.alreadyAppliedFileCount;

	return (
		<>
			<div className="app-text-code flex min-w-0 flex-wrap items-center gap-1">
				<button
					type="button"
					className={getButtonClassName()}
					disabled={controller.files.length === 0 || controllerBusy}
					onClick={startReview}
					title="Run a fresh backend dry-run review for every file section."
				>
					{reviewInProgress ? <FiLoader size={12} className="animate-spin" /> : <FiGitPullRequest size={12} />}
					{reviewInProgress ? 'Reviewing' : 'Review'}
				</button>

				{controllerBusy ? (
					<output className="text-info inline-flex items-center gap-1 text-[11px]">
						<FiLoader size={12} className="animate-spin" />
						{reviewInProgress ? 'Reviewing' : controller.busy?.kind.replaceAll('-', ' ')}
					</output>
				) : null}

				{!controllerBusy && controller.hasReviewResult ? (
					<output className="text-info inline-flex items-center text-[11px]">
						{controller.readyFileCount > 0
							? `${controller.readyFileCount} applicable`
							: `${controller.reviewedFileCount} reviewed`}
					</output>
				) : null}

				{canInspect ? (
					<button
						type="button"
						className={getButtonClassName()}
						onClick={() => {
							setIsDetailsOpen(true);
						}}
						title="Inspect per-file review and apply details."
					>
						<FiInfo size={12} />
						Inspect details
					</button>
				) : null}

				{controller.readyFileCount > 0 ? (
					<button
						type="button"
						className={getButtonClassName('success')}
						disabled={controllerBusy}
						onClick={() => {
							void controller.applyReady();
						}}
						title="Apply files whose latest backend review returned Applicable."
					>
						<FiGitPullRequest size={12} />
						Apply {controller.readyFileCount}
					</button>
				) : null}

				{controller.blockedFileCount > 0 ? (
					<button
						type="button"
						className={getButtonClassName('error')}
						onClick={() => {
							setIsDetailsOpen(true);
						}}
						title="Open backend-reported file errors."
					>
						<FiAlertTriangle size={12} />
						Blocked {controller.blockedFileCount}
					</button>
				) : null}

				{controller.needsInfoFileCount > 0 ? (
					<button
						type="button"
						className={getButtonClassName('warning')}
						onClick={() => {
							setIsDetailsOpen(true);
						}}
						title="Open files that need a target path or backend information."
					>
						<FiInfo size={12} />
						Need info {controller.needsInfoFileCount}
					</button>
				) : null}

				{controller.partialFileCount > 0 ? (
					<button
						type="button"
						className={getButtonClassName('warning')}
						onClick={() => {
							setIsDetailsOpen(true);
						}}
						title="Open partially applied file sections."
					>
						Partial {controller.partialFileCount}
					</button>
				) : null}

				{completedCount > 0 ? (
					<button
						type="button"
						className={getButtonClassName('success')}
						onClick={() => {
							setIsDetailsOpen(true);
						}}
						title="Inspect completed file sections."
					>
						<FiCheckCircle size={12} />
						Done {completedCount}
					</button>
				) : null}
			</div>

			<DiffApplyModal
				isOpen={isDetailsOpen}
				onClose={() => {
					setIsDetailsOpen(false);
				}}
				controller={controller}
			/>
		</>
	);
}

export function DiffApplyControl({
	language,
	diffText,
	isBusy,
	candidatePaths,
	workspaceRoots,
	autoReviewEpoch = 0,
}: DiffApplyControlProps) {
	if (isBusy) {
		return null;
	}

	return (
		<DiffApplySession
			language={language}
			diffText={diffText}
			candidatePaths={candidatePaths}
			workspaceRoots={workspaceRoots}
			autoReviewEpoch={autoReviewEpoch}
		/>
	);
}
