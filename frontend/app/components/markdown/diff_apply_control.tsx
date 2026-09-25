import { useMemo, useState } from 'react';
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

function DiffApplySession({
	language,
	diffText,
	candidatePaths,
	workspaceRoots,
}: Omit<DiffApplyControlProps, 'isBusy'>) {
	const parsed = useMemo(() => parseInteractiveDiff(diffText, language), [diffText, language]);
	const controller = useDiffApplyController(parsed, candidatePaths, workspaceRoots);
	const [isDetailsOpen, setIsDetailsOpen] = useState(true);

	const isBusy = controller.busy !== null;
	const readyCount = controller.readyFileCount;
	const completedCount = controller.appliedFileCount + controller.alreadyAppliedFileCount;

	return (
		<>
			<div className="app-text-code flex min-w-0 flex-wrap items-center gap-1">
				<button
					type="button"
					className={getButtonClassName()}
					onClick={() => {
						setIsDetailsOpen(true);
					}}
					title="Open per-file diff details"
				>
					<FiInfo size={12} />
					Files {controller.files.length}
				</button>

				{isBusy ? (
					<button
						type="button"
						className={getButtonClassName('info')}
						disabled
						title="A diff operation is in progress."
					>
						<FiLoader size={12} className="animate-spin" />
						Working
					</button>
				) : (
					<>
						<button
							type="button"
							className={getButtonClassName()}
							disabled={controller.patchHasErrors || controller.files.length === 0}
							onClick={() => {
								void controller.checkAll();
							}}
							title="Dry-run each file independently"
						>
							Check all
						</button>

						{readyCount > 0 ? (
							<button
								type="button"
								className={getButtonClassName('success')}
								onClick={() => {
									void controller.applyReady();
								}}
								title="Rechecks and applies only files currently ready from a dry run."
							>
								<FiGitPullRequest size={12} />
								Apply ready {readyCount}
							</button>
						) : null}
					</>
				)}

				{controller.patchHasErrors || controller.blockedFileCount > 0 ? (
					<button
						type="button"
						className={getButtonClassName('error')}
						onClick={() => {
							setIsDetailsOpen(true);
						}}
						title="Open file-level errors and patch diagnostics"
					>
						<FiAlertTriangle size={12} />
						Blocked {controller.blockedFileCount || ''}
					</button>
				) : null}

				{controller.needsInfoFileCount > 0 ? (
					<button
						type="button"
						className={getButtonClassName('warning')}
						onClick={() => {
							setIsDetailsOpen(true);
						}}
						title="Some files need an absolute target path."
					>
						<FiInfo size={12} />
						Need path {controller.needsInfoFileCount}
					</button>
				) : null}

				{completedCount > 0 ? (
					<button
						type="button"
						className={getButtonClassName('success')}
						onClick={() => {
							setIsDetailsOpen(true);
						}}
						title="Open completed file details"
					>
						<FiCheckCircle size={12} />
						Done {completedCount}
					</button>
				) : null}

				{controller.partialFileCount > 0 ? (
					<button
						type="button"
						className={getButtonClassName('warning')}
						onClick={() => {
							setIsDetailsOpen(true);
						}}
						title="Some files were partially applied."
					>
						Partial {controller.partialFileCount}
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
	const [isActivated, setIsActivated] = useState(false);

	// The parent CodeBlock already performed cheap diff detection. Do not parse,
	// infer paths, or allocate controller state until the user asks to review it.
	if (isBusy) {
		return null;
	}

	if (!isActivated) {
		return (
			<button
				type="button"
				className={getButtonClassName()}
				onClick={() => {
					setIsActivated(true);
				}}
				title="Review and apply this diff"
			>
				<FiGitPullRequest size={12} />
				Review diff
			</button>
		);
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
