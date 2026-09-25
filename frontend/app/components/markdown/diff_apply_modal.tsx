import type { ReactNode } from 'react';
import { useId, useState } from 'react';
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
}

const DISPLAY_CANDIDATE_LIMIT = 24;

function LazyPanel({ summary, className = '', renderContent }: LazyPanelProps) {
	const [isOpen, setIsOpen] = useState(false);
	const contentID = useId();

	return (
		<div className={`border-base-300 bg-base-100 overflow-hidden rounded-lg border ${className}`}>
			<button
				type="button"
				className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-xs font-semibold"
				aria-expanded={isOpen}
				aria-controls={contentID}
				onClick={() => {
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

function cardClassName(view: DiffApplyFileView): string {
	if (view.status === 'blocked') {
		return 'border-error/40 bg-error/5';
	}

	if (view.status === 'needs-info' || view.status === 'partial') {
		return 'border-warning/40 bg-warning/5';
	}

	return 'border-base-300 bg-base-100';
}

function OutcomeNotice({ label, outcome }: { label: string; outcome?: DiffApplyOutcome }) {
	if (!outcome) {
		return null;
	}

	const isProblem = outcome.status === 'blocked' || outcome.status === 'needs-info';

	return (
		<div
			className={`mt-3 rounded-lg border px-3 py-2 text-xs/5 whitespace-pre-wrap ${
				isProblem ? 'border-error/30 bg-error/5 text-error' : 'border-base-300 bg-base-200/40 text-base-content/80'
			}`}
			role={isProblem ? 'alert' : 'status'}
		>
			<span className="font-semibold">{label}: </span>
			{outcome.message}
		</div>
	);
}

function HunkPanel({ view, controller }: { view: DiffApplyFileView; controller: DiffApplyController }) {
	const { file } = view;
	const checkedCount = Object.keys(view.hunkOutcomes).length;
	const readyCount = file.hunks.filter(hunk => view.hunkOutcomes[hunk.id]?.status === 'ready').length;
	const unavailableCount = file.hunks.filter(hunk => {
		const status = view.hunkOutcomes[hunk.id]?.status;
		return status === 'blocked' || status === 'needs-info';
	}).length;
	const selectedCount = view.selectedHunkIDs.length;
	const allHunksReviewed = checkedCount === file.hunks.length && checkedCount > 0;
	const canReviewHunks = view.canTryHunks && file.hunks.length <= MAX_INTERACTIVE_HUNK_PROBES && !controller.busy;

	return (
		<div className="space-y-3">
			<div className="flex flex-wrap items-center justify-between gap-2">
				<p className="text-base-content/60 min-w-0 flex-1 text-xs">
					File-level review did not pass. Each hunk can be sent to the backend independently.
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
			</div>

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

			{checkedCount > 0 ? (
				<div className="space-y-2">
					{file.hunks.slice(0, MAX_INTERACTIVE_HUNK_PROBES).map((hunk, index) => {
						const outcome = view.hunkOutcomes[hunk.id];
						const isReady = outcome?.status === 'ready';
						const isProblem = outcome?.status === 'blocked' || outcome?.status === 'needs-info';

						return (
							<div
								key={hunk.id}
								className={`rounded-lg border p-3 ${isProblem ? 'border-error/30 bg-error/5' : 'border-base-300'}`}
							>
								{/* oxlint-disable-next-line jsx-a11y/label-has-associated-control */}
								<label className="flex items-start gap-2 text-xs">
									<input
										type="checkbox"
										className="checkbox checkbox-xs mt-0.5"
										checked={view.selectedHunkIDs.includes(hunk.id)}
										disabled={!isReady || !!controller.busy}
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

								<OutcomeNotice label={`Hunk ${index + 1}`} outcome={outcome} />

								{outcome?.diagnostics.length ? (
									<div className="mt-2">
										{renderDiagnosticsPanel({
											title: `Hunk ${index + 1} diagnostics`,
											diagnostics: outcome.diagnostics,
										})}
									</div>
								) : null}

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
							</div>
						);
					})}
				</div>
			) : null}

			<OutcomeNotice label="Selected hunks" outcome={view.applyOutcome} />

			<button
				type="button"
				className="btn btn-sm btn-primary"
				disabled={!!controller.busy || !allHunksReviewed || selectedCount === 0}
				title="Apply only backend-reviewed ready hunks. This does not submit the other hunks."
				onClick={() => {
					void controller.applySelectedHunks(file.id);
				}}
			>
				Apply selected hunks ({selectedCount})
			</button>
		</div>
	);
}

function FileCard({ view, controller }: { view: DiffApplyFileView; controller: DiffApplyController }) {
	const { file } = view;
	const inputID = useId();
	const displayPath = getInteractiveDiffTargetPath(file) || 'Target path unresolved';
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

	const showHunks = file.hunks.length > 0 && (view.canTryHunks || Object.keys(view.hunkOutcomes).length > 0);

	return (
		<section className={`rounded-xl border p-4 shadow-sm ${cardClassName(view)}`} aria-busy={isActive}>
			<div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
				<div className="min-w-0 flex-1">
					<div className="flex flex-wrap items-center gap-2">
						<span className={`badge badge-outline badge-sm ${statusClassName(view.status)}`}>
							{statusLabel(view.status)}
						</span>
						<span className="badge badge-ghost badge-sm">{file.kind}</span>
						<span className="text-base-content/50 text-xs">{file.id}</span>
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
						Review file
					</button>

					<button
						type="button"
						className="btn btn-xs btn-primary"
						disabled={!!controller.busy || !view.canApply}
						title="Apply after a successful file-level backend review."
						onClick={() => {
							void controller.applyFile(file.id);
						}}
					>
						Apply file
					</button>
				</div>
			</div>

			<OutcomeNotice label="Review" outcome={view.reviewOutcome} />
			<OutcomeNotice label="Apply" outcome={view.applyOutcome} />

			<label className="mt-4 mb-1 block text-xs font-semibold" htmlFor={inputID}>
				Optional target file path
			</label>

			<input
				id={inputID}
				className="input input-sm w-full font-mono text-xs"
				value={view.pathInput}
				disabled={!!controller.busy}
				placeholder="Leave blank to let the backend resolve the target"
				spellCheck={false}
				onChange={event => {
					controller.setTargetPath(file.id, event.target.value);
				}}
			/>

			<div className="text-base-content/60 mt-1 text-xs">
				{!view.targetPath
					? isNewInteractiveDiffFile(file)
						? 'For a new file, enter a target path or let the backend request one during review.'
						: 'The backend resolves and validates target paths from patch headers, workspace roots, or candidates.'
					: 'Changing this target clears old review and apply results for this file.'}
			</div>

			{candidates.length > 0 ? (
				<div className="mt-2">
					<Dropdown<string>
						dropdownItems={candidateDropdownItems}
						selectedKey={selectedCandidate}
						onChange={candidate => {
							controller.setTargetPath(file.id, candidate);
						}}
						filterDisabled={false}
						title={`Suggested target paths for ${displayPath}`}
						placeholderLabel="Choose a suggested target path"
						orderedKeys={candidates}
						inlineMenu
						maxMenuHeight={220}
						disabled={!!controller.busy}
					/>
				</div>
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

			{showHunks ? (
				<LazyPanel
					className="mt-3"
					summary="Try individual hunks"
					renderContent={() => <HunkPanel view={view} controller={controller} />}
				/>
			) : null}

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
							<h3 className="flex items-center gap-2 text-base font-semibold">
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

				<div className="min-h-0 flex-1 space-y-3 overflow-y-auto overscroll-contain p-4 sm:px-5">
					{renderDiagnosticsPanel({
						title: 'Patch notes',
						description: 'These are non-blocking UI parser notes. Backend review remains authoritative.',
						diagnostics: controller.patchDiagnostics,
					})}

					{controller.files.length === 0 ? (
						<div className="border-base-300 rounded-xl border p-4 text-sm">
							No file sections could be separated within the interactive UI budget.
						</div>
					) : null}

					{controller.files.map(view => (
						<FileCard key={view.file.id} view={view} controller={controller} />
					))}
				</div>

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
						disabled={!!controller.busy || controller.readyFileCount === 0}
						title="Apply only file sections whose latest backend review returned Applicable."
						onClick={() => {
							void controller.applyReady();
						}}
					>
						Apply ready ({controller.readyFileCount})
					</button>
				</ModalActions>
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
