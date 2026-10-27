import type { ReactNode } from 'react';
import { useId, useRef, useState } from 'react';
import { FiChevronRight, FiGitPullRequest, FiX } from 'react-icons/fi';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import type { DropdownItem } from '@/components/dropdown';
import type { DiffApplyController, DiffApplyFileView } from '@/components/markdown/diff_apply_controller';
import type { DiffApplyOutcome } from '@/components/markdown/diff_apply_model';
import { Dropdown } from '@/components/dropdown';
import {
	getInteractiveDiffTargetPath,
	isNewInteractiveDiffFile,
	MAX_INTERACTIVE_HUNK_PROBES,
} from '@/components/markdown/diff_apply_model';
import { renderDiagnosticsPanel, uniqueDiagnostics } from '@/components/markdown/diff_diagnostic';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';

interface DiffApplyModalProps {
	isOpen: boolean;
	onClose: () => void;
	controller: DiffApplyController;
}

interface LazyPanelProps {
	summary: ReactNode;
	className?: string;
	renderContent: () => ReactNode;
	defaultOpen?: boolean;
	onOpen?: () => void;
}

const DISPLAY_CANDIDATE_LIMIT = 24;
const DISPLAY_HUNK_PAGE_SIZE = 32;

function LazyPanel({ summary, className = '', renderContent, defaultOpen = false, onOpen }: LazyPanelProps) {
	const [isOpen, setIsOpen] = useState(defaultOpen);
	const contentID = useId();

	return (
		<div className={`border-base-300 bg-base-100 overflow-hidden rounded-lg border ${className}`}>
			<button
				type="button"
				className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left font-mono text-xs font-semibold"
				aria-expanded={isOpen}
				aria-controls={contentID}
				onClick={() => {
					if (!isOpen) {
						onOpen?.();
					}
					setIsOpen(previous => !previous);
				}}
			>
				<span>{summary}</span>
				<FiChevronRight
					size={12}
					className={`text-base-content/50 transition-transform ${isOpen ? 'rotate-90' : ''}`}
				/>
			</button>

			{isOpen ? (
				<div id={contentID} className="border-base-300 border-t px-3 py-2">
					{renderContent()}
				</div>
			) : null}
		</div>
	);
}

function statusLabel(status: DiffApplyFileView['status']): string {
	switch (status) {
		case 'ready':
			return 'ready';
		case 'needs-info':
			return 'needs info';
		case 'blocked':
			return 'blocked';
		case 'applied':
			return 'applied';
		case 'already-applied':
			return 'already applied';
		case 'partial':
			return 'partially applied';
		default:
			return 'not reviewed';
	}
}

function statusClassName(status: DiffApplyFileView['status']): string {
	switch (status) {
		case 'ready':
		case 'applied':
			return 'badge-success';
		case 'already-applied':
			return 'badge-info';
		case 'needs-info':
		case 'partial':
			return 'badge-warning';
		case 'blocked':
			return 'badge-error';
		default:
			return 'badge-ghost';
	}
}

function OutcomeNotice({ label, outcome }: { label: string; outcome?: DiffApplyOutcome }) {
	if (!outcome) {
		return null;
	}

	const isProblem = outcome.status === 'blocked' || outcome.status === 'needs-info';

	return (
		<p className={`text-xs/5 whitespace-pre-wrap ${isProblem ? 'text-error' : 'text-base-content/70'}`}>
			<span className="font-semibold">{label}: </span>
			{outcome.message}
		</p>
	);
}

function HunkPanel({ view, controller }: { view: DiffApplyFileView; controller: DiffApplyController }) {
	const { file } = view;
	const inputPrefix = useId();
	const [page, setPage] = useState(0);
	const pageCount = Math.ceil(file.hunks.length / DISPLAY_HUNK_PAGE_SIZE);
	const visibleHunks = file.hunks.slice(page * DISPLAY_HUNK_PAGE_SIZE, (page + 1) * DISPLAY_HUNK_PAGE_SIZE);
	const checkedCount = Object.keys(view.hunkOutcomes).length;
	const readyCount = file.hunks.filter(hunk => view.hunkOutcomes[hunk.id]?.status === 'ready').length;
	const unavailableCount = file.hunks.filter(hunk => {
		const status = view.hunkOutcomes[hunk.id]?.status;
		return status === 'blocked' || status === 'needs-info';
	}).length;
	const selectedCount = file.hunks.filter(
		hunk =>
			view.selectedHunkIDs.includes(hunk.id) && (!!view.targetPath || view.hunkOutcomes[hunk.id]?.status === 'ready')
	).length;
	const canReviewHunks = view.canTryHunks && file.hunks.length <= MAX_INTERACTIVE_HUNK_PROBES && !controller.busy;

	return (
		<div className="space-y-3">
			<div className="flex flex-wrap items-center justify-between gap-2">
				<p className="text-base-content/60 min-w-0 flex-1 text-xs">
					Inspect source or optionally review hunks. An explicit target allows direct submission.
				</p>

				<button
					type="button"
					className="btn btn-xs btn-outline"
					disabled={!canReviewHunks}
					onClick={() => {
						void controller.reviewHunks(file.id);
					}}
				>
					Review hunks
				</button>

				<button
					type="button"
					className="btn btn-xs btn-primary"
					disabled={!!controller.busy || selectedCount === 0}
					title="Submit selected source hunks. An explicit target does not require a prior review."
					onClick={() => {
						void controller.applySelectedHunks(file.id);
					}}
				>
					Apply selected hunks ({selectedCount})
				</button>
			</div>

			{pageCount > 1 ? (
				<div className="flex items-center justify-between gap-2 text-xs">
					<button
						type="button"
						className="btn btn-xs btn-ghost"
						disabled={page === 0}
						onClick={() => {
							setPage(current => Math.max(0, current - 1));
						}}
					>
						Previous hunks
					</button>
					<span>
						{page + 1} / {pageCount}
					</span>
					<button
						type="button"
						className="btn btn-xs btn-ghost"
						disabled={page + 1 >= pageCount}
						onClick={() => {
							setPage(current => Math.min(pageCount - 1, current + 1));
						}}
					>
						Next hunks
					</button>
				</div>
			) : null}

			{file.hunks.length > MAX_INTERACTIVE_HUNK_PROBES ? (
				<div className="text-warning text-xs">
					This file has {file.hunks.length} hunks. Individual hunk review is limited to {MAX_INTERACTIVE_HUNK_PROBES}.
				</div>
			) : null}

			{checkedCount > 0 ? (
				<div className="text-base-content/70 text-xs">
					{checkedCount}/{file.hunks.length} reviewed, {readyCount} applicable, {unavailableCount} unavailable.
				</div>
			) : null}

			{file.hunks.length > 0 ? (
				<div className="space-y-2">
					{visibleHunks.map((hunk, pageIndex) => {
						const index = page * DISPLAY_HUNK_PAGE_SIZE + pageIndex;
						const outcome = view.hunkOutcomes[hunk.id];
						const canSelect = !!view.targetPath || outcome?.status === 'ready';
						const hunkCardClassName = outcome?.status === 'blocked' ? 'border-error/30 bg-error/10' : 'border-base-300';

						return (
							<div key={hunk.id} className={`rounded-lg border p-3 ${hunkCardClassName}`}>
								{/* oxlint-disable-next-line jsx-a11y/label-has-associated-control */}
								<label htmlFor={`${inputPrefix}-${index}`} className="flex items-start gap-2 text-xs">
									<input
										id={`${inputPrefix}-${index}`}
										type="checkbox"
										className="checkbox checkbox-xs mt-0.5"
										checked={view.selectedHunkIDs.includes(hunk.id)}
										disabled={!canSelect || !!controller.busy}
										onChange={event => {
											controller.setHunkSelected(file.id, hunk.id, event.target.checked);
										}}
									/>

									<span className="min-w-0">
										<span className="font-semibold">Hunk {index + 1}</span>
										<span className="text-base-content/60 ml-2">{outcome?.status ?? 'not reviewed'}</span>
										<code className="mt-1 block break-all">{hunk.header}</code>
									</span>
								</label>
								<LazyPanel
									className="mt-2"
									summary="View hunk source"
									renderContent={() => (
										<pre
											aria-label={`Hunk ${index + 1} source`}
											className="app-bg-code app-text-code max-h-64 overflow-auto rounded-sm p-2 text-xs whitespace-pre"
										>
											<code>{hunk.sourceText}</code>
										</pre>
									)}
								/>
								{outcome ? (
									<LazyPanel
										className="mt-2"
										summary="Review notes"
										renderContent={() => <OutcomeNotice label={`Hunk ${index + 1}`} outcome={outcome} />}
									/>
								) : null}
								{outcome?.diagnostics.length ? (
									<div className="mt-2">
										{renderDiagnosticsPanel({
											title: `Hunk ${index + 1} diagnostics`,
											diagnostics: outcome.diagnostics,
										})}
									</div>
								) : null}
							</div>
						);
					})}
				</div>
			) : null}
		</div>
	);
}

function FileCard({ view, controller }: { view: DiffApplyFileView; controller: DiffApplyController }) {
	const { file } = view;
	const autoHunkReviewFingerprintRef = useRef<string | null>(null);
	const [targetPickerVersion, setTargetPickerVersion] = useState(0);
	const inputID = useId();
	const displayPath =
		view.targetPath ||
		view.applyOutcome?.resolvedTargetPath ||
		view.reviewOutcome?.resolvedTargetPath ||
		getInteractiveDiffTargetPath(file) ||
		'Target path unresolved';
	const isActive = controller.busy?.activeFileIDs.includes(file.id) ?? false;
	const candidates = view.candidates.slice(0, DISPLAY_CANDIDATE_LIMIT);
	const diagnostics = uniqueDiagnostics([
		...file.diagnostics,
		...(view.reviewOutcome?.diagnostics ?? []),
		...(view.applyOutcome?.diagnostics ?? []),
	]);
	const candidateDropdownItems = candidates.reduce<Record<string, DropdownItem>>((items, candidate) => {
		items[candidate] = { isEnabled: true };
		return items;
	}, {});
	const selectedCandidate = candidates.includes(view.pathInput) ? view.pathInput : '';
	const hasSelectedCandidate = selectedCandidate.length > 0;
	const manualTargetSummary = hasSelectedCandidate
		? 'Enter a different target path'
		: view.targetPath
			? 'Edit target path'
			: 'Enter a target path manually';

	const showHunks = file.hunks.length > 0;

	const shouldStartInitialHunkReview =
		view.canTryHunks &&
		file.hunks.length <= MAX_INTERACTIVE_HUNK_PROBES &&
		Object.keys(view.hunkOutcomes).length === 0 &&
		view.status !== 'applied' &&
		view.status !== 'already-applied' &&
		!controller.busy;

	const handleFirstHunkInspection = (): void => {
		if (autoHunkReviewFingerprintRef.current === view.fingerprint || !shouldStartInitialHunkReview) {
			return;
		}

		autoHunkReviewFingerprintRef.current = view.fingerprint;
		void controller.reviewHunks(file.id);
	};

	return (
		<section className="border-base-300 bg-base-100 rounded-xl border p-4 shadow-sm" aria-busy={isActive}>
			<div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
				<div className="min-w-0 flex-1">
					<div className="flex flex-wrap items-center gap-2">
						<span className={`badge badge-outline badge-sm ${statusClassName(view.status)}`}>
							{statusLabel(view.status)}
						</span>
						<span className="badge badge-ghost badge-sm">{file.kind}</span>
						<span className="text-base-content/50 font-mono text-xs">{file.id}</span>
					</div>

					<div className="mt-2 font-mono text-sm font-semibold break-all">{displayPath}</div>

					<div className="text-base-content/60 mt-1 text-xs">
						{file.hunks.length} detected hunk{file.hunks.length === 1 ? '' : 's'}
					</div>

					{file.oldPath && file.newPath && file.oldPath !== file.newPath ? (
						<div className="text-base-content/60 mt-2 space-y-1 font-mono text-xs break-all">
							<div>Old: {file.oldPath}</div>
							<div>New: {file.newPath}</div>
						</div>
					) : null}

					{view.reviewOutcome?.resolvedTargetPath ? (
						<div className="text-base-content/60 mt-2 font-mono text-xs break-all">
							Backend target: {view.reviewOutcome.resolvedTargetPath}
						</div>
					) : null}

					{isActive ? (
						<output className="text-info mt-3 flex items-center gap-2 text-xs">
							<span className="loading loading-spinner loading-xs" />
							Reviewing or applying this file.
						</output>
					) : null}
				</div>

				<div className="flex shrink-0 flex-wrap gap-2">
					<button
						type="button"
						className="btn btn-xs btn-outline"
						disabled={!!controller.busy}
						onClick={() => {
							void controller.reviewFile(file.id);
						}}
					>
						{view.status === 'pending' ? 'Review file' : 'Review again'}
					</button>

					<button
						type="button"
						className="btn btn-xs btn-primary"
						disabled={!!controller.busy || !view.canApply}
						title="Submit with an explicit target or a successful backend review."
						onClick={() => {
							void controller.applyFile(file.id);
						}}
					>
						{view.targetPath ? 'Apply to target' : 'Apply file'}
					</button>
				</div>
			</div>

			{showHunks ? (
				<LazyPanel
					className="mt-3"
					summary="Inspect and review hunks"
					onOpen={handleFirstHunkInspection}
					renderContent={() => <HunkPanel view={view} controller={controller} />}
				/>
			) : null}

			{view.status === 'pending' && !controller.busy ? (
				<p className="text-base-content/70 mt-3 text-xs">
					{view.targetPath
						? 'Explicit target set. Apply directly, or review first if useful.'
						: 'Not reviewed for the current options. Review this file or choose an explicit target.'}
				</p>
			) : null}

			<div className="mt-4 space-y-2">
				<div className="flex items-center justify-between gap-2">
					<span className="text-xs font-semibold">Suggested target paths</span>
					<span className="text-base-content/50 text-[11px]">Ranked</span>
				</div>

				<Dropdown<string>
					key={`${file.id}\u0000${targetPickerVersion}`}
					dropdownItems={candidateDropdownItems}
					selectedKey={selectedCandidate}
					onChange={candidate => {
						setTargetPickerVersion(version => version + 1);
						controller.setTargetPath(file.id, candidate);
					}}
					filterDisabled={false}
					title={`Suggested target paths for ${displayPath}`}
					placeholderLabel={
						candidates.length > 0 ? 'Choose a suggested target path' : 'No target suggestions were found'
					}
					orderedKeys={candidates}
					inlineMenu
					defaultOpen={false}
					maxMenuHeight={220}
					disabled={!!controller.busy}
				/>

				<div className="text-base-content/60 text-xs">
					{hasSelectedCandidate
						? 'The selected path is sent as an explicit backend target.'
						: view.targetPath
							? 'A manually entered target path is sent to the backend.'
							: candidates.length > 0
								? 'Suggestions are guesses ranked by file and directory match. The backend validates the chosen path.'
								: isNewInteractiveDiffFile(file)
									? 'No destination was derived for this new file. Enter one manually or let the backend request it.'
									: 'No target was derived. The backend can still resolve the path during review.'}
				</div>
			</div>

			<LazyPanel
				className="mt-3"
				summary={manualTargetSummary}
				defaultOpen={!!view.pathInput && !hasSelectedCandidate}
				renderContent={() => (
					<div>
						<label className="mb-1 block text-xs font-semibold" htmlFor={inputID}>
							Target path override
						</label>
						<input
							id={inputID}
							className="input input-sm w-full font-mono text-xs"
							value={view.pathInput}
							disabled={!!controller.busy}
							placeholder="Explicit target path; blank uses backend resolution"
							spellCheck={false}
							onChange={event => {
								controller.setTargetPath(file.id, event.target.value);
							}}
							onKeyDown={event => {
								if (event.key === 'Enter' && !event.nativeEvent.isComposing && !controller.busy) {
									event.preventDefault();
									event.stopPropagation();
									void controller.reviewFile(file.id);
								}
							}}
						/>
						<button
							type="button"
							className="btn btn-xs btn-outline mt-2 font-mono"
							disabled={!!controller.busy}
							title="Review this file with the current target input. No changes are written."
							onClick={() => {
								void controller.reviewFile(file.id);
							}}
						>
							Review target
						</button>
						<div className="text-base-content/60 mt-1 text-xs">
							Editing clears previous results. This explicit override can be applied directly; review is optional. Leave
							it blank to use backend target resolution.
						</div>
					</div>
				)}
			/>

			<LazyPanel
				className="mt-3"
				summary="View file patch"
				renderContent={() => (
					<pre
						aria-label="File patch source"
						className="app-bg-code app-text-code max-h-80 overflow-auto rounded-sm p-3 text-xs whitespace-pre"
					>
						<code>{file.sourceText || 'No source text is available for this file section.'}</code>
					</pre>
				)}
			/>

			{view.reviewOutcome || view.applyOutcome ? (
				<LazyPanel
					className="mt-3"
					summary="Review / apply notes"
					renderContent={() => (
						<div className="space-y-2">
							<OutcomeNotice label="Review" outcome={view.reviewOutcome} />
							<OutcomeNotice label="Apply" outcome={view.applyOutcome} />
						</div>
					)}
				/>
			) : null}

			{diagnostics.length > 0 ? (
				<div className="mt-3">
					{renderDiagnosticsPanel({
						title: 'File details',
						description: 'Parser notes and backend diagnostics for this file section.',
						diagnostics,
					})}
				</div>
			) : null}
		</section>
	);
}

function DiffApplyModalContent({ controller }: { controller: DiffApplyController }) {
	const { requestClose } = useModalDialogController();

	const busyLabel = controller.busy
		? controller.busy.stopping
			? 'Stopping after in-flight requests finish. Already submitted writes cannot be undone.'
			: `${controller.busy.kind.replaceAll('-', ' ')}: ${controller.busy.completed}/${controller.busy.total}`
		: '';

	return (
		<>
			<div className="modal-box bg-base-100 flex max-h-[calc(100dvh-2rem)] w-11/12 max-w-5xl flex-col overflow-hidden rounded-2xl p-0 shadow-2xl">
				<div className="border-base-300 border-b px-4 py-3 sm:px-5">
					<div className="flex items-start justify-between gap-4">
						<div className="min-w-0 flex-1">
							<h3 className="flex items-center gap-2 font-mono text-base font-semibold">
								<FiGitPullRequest size={16} className="shrink-0" />
								<span>Patch details</span>
							</h3>

							<div className="text-base-content/60 mt-1 flex flex-wrap gap-x-2 gap-y-1 text-xs">
								<span>
									{controller.files.length} file section{controller.files.length === 1 ? '' : 's'}
								</span>
							</div>
						</div>

						<button
							type="button"
							className="btn btn-ghost btn-sm btn-circle"
							onClick={() => {
								requestClose();
							}}
							aria-label="Close diff details"
						>
							<FiX size={14} />
						</button>
					</div>

					<label className="mt-3 flex items-center gap-2 text-sm" title="Passed directly to the backend fuzzy applier.">
						<input
							type="checkbox"
							className="checkbox checkbox-xs"
							checked={controller.strict}
							disabled={!!controller.busy}
							onChange={event => {
								controller.setStrict(event.target.checked);
							}}
						/>
						Strict backend matching
					</label>

					<div className="text-base-content/60 mt-1 text-xs">
						Changing this option clears previous backend review results.
					</div>
				</div>

				<ModalActions className="bg-base-100">
					<button
						type="button"
						className="btn btn-sm"
						onClick={() => {
							requestClose();
						}}
					>
						Close
					</button>

					<button
						type="button"
						className="btn btn-sm"
						disabled={!!controller.busy || controller.files.length === 0}
						onClick={() => {
							void controller.reviewAll();
						}}
					>
						{controller.hasReviewResult ? 'Review again' : 'Review all'}
					</button>

					<button
						type="button"
						className="btn btn-sm btn-primary"
						disabled={!!controller.busy || controller.submittableFileCount === 0}
						title="Submit files with explicit targets or successful backend reviews."
						onClick={() => {
							void controller.applyReady();
						}}
					>
						Apply files ({controller.submittableFileCount})
					</button>
				</ModalActions>

				{controller.busy ? (
					<div className="border-base-300 flex items-center justify-between gap-3 border-t px-5 py-2 text-xs">
						<div className="min-w-0">
							<output className="flex items-center gap-2" aria-live="polite">
								<span className="loading loading-spinner loading-xs" />
								<span>{busyLabel}</span>
							</output>
							<div className="text-base-content/60 mt-1">
								Review requests are bounded. Write requests are serialized.
							</div>
						</div>

						<button
							type="button"
							className="btn btn-xs"
							disabled={controller.busy.stopping}
							onClick={controller.cancel}
						>
							Stop remaining work
						</button>
					</div>
				) : (
					<div className="border-base-300 text-base-content/60 border-t px-5 py-2 text-xs">
						Backend responses determine review and apply outcomes.
					</div>
				)}

				<div className="min-h-0 flex-1 space-y-3 overflow-y-auto overscroll-contain p-4 sm:px-5">
					{controller.files.length === 0 ? (
						<div className="border-base-300 rounded-xl border p-4 text-sm">
							No file sections could be separated within the interactive UI budget.
						</div>
					) : null}

					{controller.files.map(view => (
						<FileCard key={view.file.id} view={view} controller={controller} />
					))}

					{renderDiagnosticsPanel({
						title: 'Patch notes',
						description: 'Non-blocking parser notes. The backend determines request outcomes.',
						diagnostics: controller.patchDiagnostics,
					})}
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</>
	);
}

export function DiffApplyModal({ isOpen, onClose, controller }: DiffApplyModalProps) {
	if (!isOpen) {
		return null;
	}

	return (
		<ModalDialog isOpen={isOpen} onClose={onClose} data-disable-chat-shortcuts="true">
			<DiffApplyModalContent controller={controller} />
		</ModalDialog>
	);
}
