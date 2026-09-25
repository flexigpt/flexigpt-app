import { useMemo, useState } from 'react';
import { FiGitPullRequest, FiInfo, FiLoader } from 'react-icons/fi';

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

	return `inline-flex h-6 items-center gap-1 rounded-full border border-base-300 px-2 text-[11px] font-medium leading-none whitespace-nowrap transition-colors hover:opacity-60 disabled:cursor-not-allowed disabled:opacity-50 ${toneClassName}`;
}

interface DiffApplySessionProps {
	language: string;
	diffText: string;
	candidatePaths?: string[];
	workspaceRoots?: string[];
}

function DiffApplySession({ language, diffText, candidatePaths, workspaceRoots }: DiffApplySessionProps) {
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
	const canInspect = controller.hasReviewResult || controller.patchDiagnostics.length > 0;
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
	if (isBusy) {
		return null;
	}

	return (
		<DiffApplySession
			language={language}
			diffText={diffText}
			candidatePaths={candidatePaths}
			workspaceRoots={workspaceRoots}
		/>
	);
}
