// oxlint-disable react/only-export-components
import { FiGitPullRequest, FiInfo, FiLoader } from 'react-icons/fi';

import type { DiffApplyController, DiffApplyFileView } from '@/components/markdown/diff_apply_controller';

export type DiffActionTone = 'neutral' | 'success' | 'warning' | 'error' | 'info';

export function getDiffActionButtonClassName(tone: DiffActionTone = 'neutral'): string {
	const toneClassName = {
		neutral: 'app-text-code',
		success: 'text-success',
		warning: 'text-warning',
		error: 'text-error',
		info: 'text-info',
	}[tone];

	return [
		'inline-flex h-6 shrink-0 items-center justify-center gap-1 rounded-full',
		'border border-base-300 px-2 text-[11px] font-medium leading-none',
		'whitespace-nowrap tabular-nums transition-colors hover:bg-base-300/40',
		'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2',
		'focus-visible:outline-primary disabled:cursor-not-allowed disabled:opacity-40',
		toneClassName,
	].join(' ');
}

export function getDiffFilePresentation(
	status: DiffApplyFileView['status'],
	hasExplicitTarget = false
): { label: string; tone: DiffActionTone } {
	switch (status) {
		case 'ready':
			return { label: 'Ready', tone: 'success' };
		case 'needs-info':
			return { label: 'Needs info', tone: 'warning' };
		case 'blocked':
			return { label: 'Blocked', tone: 'error' };
		case 'applied':
			return { label: 'Applied', tone: 'success' };
		case 'already-applied':
			return { label: 'Already present', tone: 'info' };
		case 'partial':
			return { label: 'Partially applied', tone: 'warning' };
		default:
			return hasExplicitTarget ? { label: 'Target set', tone: 'info' } : { label: 'Details / target', tone: 'neutral' };
	}
}

export function getDiffBlockPresentation(controller: DiffApplyController): {
	label: string;
	tone: DiffActionTone;
} {
	if (controller.files.length === 1) {
		const view = controller.files[0];
		return getDiffFilePresentation(view.status, Boolean(view.targetPath));
	}

	const completed = controller.appliedFileCount + controller.alreadyAppliedFileCount;
	const attention = controller.blockedFileCount + controller.needsInfoFileCount + controller.partialFileCount;
	const label =
		completed === controller.files.length && completed > 0
			? `${completed} completed`
			: attention > 0
				? `${controller.readyFileCount} ready · ${attention} need attention`
				: controller.readyFileCount > 0
					? `${controller.readyFileCount} ready`
					: controller.hasReviewResult
						? `${controller.reviewedFileCount} reviewed`
						: controller.files.length > 0
							? 'Details / targets'
							: 'Patch notes';

	const tone: DiffActionTone =
		controller.blockedFileCount > 0
			? 'error'
			: attention > 0
				? 'warning'
				: controller.readyFileCount > 0 || completed > 0
					? 'success'
					: 'neutral';

	return { label, tone };
}

interface DiffApplyActionsProps {
	onReview: () => void;
	onDetails?: () => void;
	onApply: () => void;
	reviewDisabled?: boolean;
	detailsDisabled?: boolean;
	applyDisabled?: boolean;
	reviewing?: boolean;
	applying?: boolean;
	detailsLabel?: string;
	detailsTone?: DiffActionTone;
	detailsTitle?: string;
	applyLabel?: string;
	reviewTitle?: string;
	applyTitle?: string;
	className?: string;
}

export function DiffApplyActions({
	onReview,
	onDetails,
	onApply,
	reviewDisabled = false,
	detailsDisabled = false,
	applyDisabled = false,
	reviewing = false,
	applying = false,
	detailsLabel = 'Details / targets',
	detailsTone = 'neutral',
	detailsTitle = 'Inspect targets, source hunks, backend results, and parser notes.',
	applyLabel = 'Apply',
	reviewTitle = 'Run a fresh backend dry-run review. No files are changed.',
	applyTitle = 'Submit with an explicit target or a successful backend review.',
	className = '',
}: DiffApplyActionsProps) {
	return (
		<div className={`flex min-w-0 flex-wrap items-center gap-1 ${className}`}>
			<button
				type="button"
				className={getDiffActionButtonClassName()}
				disabled={reviewDisabled}
				onClick={onReview}
				title={reviewTitle}
			>
				{reviewing ? <FiLoader size={12} className="animate-spin" /> : <FiGitPullRequest size={12} />}
				{reviewing ? 'Reviewing' : 'Review'}
			</button>

			{onDetails ? (
				<button
					type="button"
					className={getDiffActionButtonClassName(detailsTone)}
					disabled={detailsDisabled}
					onClick={onDetails}
					title={detailsTitle}
				>
					<FiInfo size={12} />
					{detailsLabel}
				</button>
			) : null}

			<button
				type="button"
				className={getDiffActionButtonClassName('success')}
				disabled={applyDisabled}
				onClick={onApply}
				title={applyTitle}
			>
				{applying ? <FiLoader size={12} className="animate-spin" /> : <FiGitPullRequest size={12} />}
				{applying ? 'Applying' : applyLabel}
			</button>
		</div>
	);
}
