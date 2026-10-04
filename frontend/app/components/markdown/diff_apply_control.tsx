import { useCallback, useEffect, useRef, useState } from 'react';
import { FiGitPullRequest, FiInfo, FiLoader } from 'react-icons/fi';

import type { ParsedInteractiveDiff } from '@/components/markdown/diff_apply_model';
import { useDiffApplyController } from '@/components/markdown/diff_apply_controller';
import { DiffApplyModal } from '@/components/markdown/diff_apply_modal';
import { parseInteractiveDiff } from '@/components/markdown/diff_apply_model';

interface DiffApplyControlProps {
	language: string;
	diffText: string;
	isBusy: boolean;
	candidatePaths?: string[];
	workspaceRoots?: string[];
}

type ButtonTone = 'neutral' | 'success' | 'warning' | 'error' | 'info';

function getButtonClassName(tone: ButtonTone = 'neutral'): string {
	const toneClassName = {
		neutral: 'app-text-code',
		success: 'text-success',
		warning: 'text-warning',
		error: 'text-error',
		info: 'text-info',
	}[tone];

	return `inline-flex h-6 items-center gap-1 rounded-full border border-base-300 px-2 font-mono text-[11px] font-medium leading-none whitespace-nowrap tabular-nums transition-colors hover:opacity-60 disabled:cursor-not-allowed disabled:opacity-50 ${toneClassName}`;
}

interface DiffApplySessionProps {
	parsed: ParsedInteractiveDiff;
	candidatePaths?: string[];
	workspaceRoots?: string[];
	consumeInitialReview: (parsed: ParsedInteractiveDiff) => boolean;
}

function DiffApplySession({ parsed, candidatePaths, workspaceRoots, consumeInitialReview }: DiffApplySessionProps) {
	// The explicitly created session owns parsed object identity.
	const controller = useDiffApplyController(parsed, candidatePaths, workspaceRoots);

	const [isDetailsOpen, setIsDetailsOpen] = useState(false);

	const { reviewAll } = controller;
	useEffect(() => {
		let cancelled = false;

		// Defer consumption until after effect setup/replay. A Strict Mode
		// cleanup must not consume an intent for a request it then cancels.
		queueMicrotask(() => {
			if (!cancelled && consumeInitialReview(parsed)) {
				void reviewAll();
			}
		});

		return () => {
			cancelled = true;
		};
	}, [consumeInitialReview, parsed, reviewAll]);

	const startReview = (): void => {
		/*
		 * reviewAll synchronously claims the controller operation slot before
		 * its first await. controller.busy is the sole reviewing-state source.
		 */
		void controller.reviewAll();
	};

	const controllerBusy = controller.busy !== null;
	const reviewInProgress = controller.busy?.kind === 'review';
	const applyInProgress = controller.busy?.kind === 'apply';
	const canInspect = controller.files.length > 0 || controller.patchDiagnostics.length > 0;
	const completedCount = controller.appliedFileCount + controller.alreadyAppliedFileCount;
	const detailsTone: ButtonTone =
		controller.blockedFileCount > 0
			? 'error'
			: controller.needsInfoFileCount > 0 || controller.partialFileCount > 0
				? 'warning'
				: controller.readyFileCount > 0 || completedCount > 0
					? 'success'
					: 'info';
	const detailsLabel =
		completedCount > 0
			? `${completedCount} completed`
			: controller.hasReviewResult
				? `${controller.reviewedFileCount} reviewed`
				: controller.files.length > 0
					? 'Details / targets'
					: 'Patch notes';
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

				{canInspect ? (
					<button
						type="button"
						className={getButtonClassName(detailsTone)}
						onClick={() => {
							setIsDetailsOpen(true);
						}}
						title="Inspect backend review, apply results, and parser notes."
					>
						<FiInfo size={12} />
						{detailsLabel}
					</button>
				) : null}

				{controller.readyFileCount > 0 || applyInProgress ? (
					<button
						type="button"
						className={getButtonClassName('success')}
						disabled={controllerBusy}
						onClick={() => {
							void controller.applyReady();
						}}
						title="Apply files whose latest backend review returned Applicable."
					>
						{applyInProgress ? <FiLoader size={12} className="animate-spin" /> : <FiGitPullRequest size={12} />}
						{applyInProgress ? 'Applying' : `Apply ${controller.readyFileCount}`}
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
}: DiffApplyControlProps) {
	const [session, setSession] = useState<{
		language: string;
		diffText: string;
		parsed: ParsedInteractiveDiff;
	} | null>(null);

	const pendingInitialReviewRef = useRef<ParsedInteractiveDiff | null>(null);
	const consumeInitialReview = useCallback((parsed: ParsedInteractiveDiff): boolean => {
		if (pendingInitialReviewRef.current !== parsed) {
			return false;
		}
		pendingInitialReviewRef.current = null;
		return true;
	}, []);

	if (isBusy) {
		return null;
	}

	if (!session || session.language !== language || session.diffText !== diffText) {
		return (
			<button
				type="button"
				className={getButtonClassName()}
				title="Parse this patch and run a backend dry-run review. No files are changed."
				onClick={() => {
					const parsed = parseInteractiveDiff(diffText, language);
					pendingInitialReviewRef.current = parsed;
					setSession({
						language,
						diffText,
						parsed,
					});
				}}
			>
				<FiGitPullRequest size={12} />
				Review
			</button>
		);
	}

	return (
		<DiffApplySession
			parsed={session.parsed}
			candidatePaths={candidatePaths}
			workspaceRoots={workspaceRoots}
			consumeInitialReview={consumeInitialReview}
		/>
	);
}
