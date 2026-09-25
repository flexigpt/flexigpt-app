import type { ReactNode } from 'react';
import { useState } from 'react';
import { FiChevronRight, FiGitPullRequest, FiX } from 'react-icons/fi';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import type { DiffApplyController, DiffApplyFileView } from '@/components/markdown/diff_apply_controller';
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

interface LazyDetailsProps {
	summary: ReactNode;
	className?: string;
	renderContent: () => ReactNode;
}

const DISPLAY_CANDIDATE_LIMIT = 24;

function LazyDetails({ summary, className = '', renderContent }: LazyDetailsProps) {
	const [isOpen, setIsOpen] = useState(false);

	return (
		<details
			className={`group border-base-300 bg-base-100 overflow-hidden rounded-lg border ${className}`}
			onToggle={event => {
				setIsOpen(event.currentTarget.open);
			}}
		>
			<summary className="flex cursor-pointer list-none items-center justify-between gap-3 px-3 py-2 text-xs font-semibold">
				<span>{summary}</span>
				<FiChevronRight size={12} className="text-base-content/50 transition-transform group-open:rotate-90" />
			</summary>

			{isOpen ? <div className="border-base-300 border-t px-3 py-2">{renderContent()}</div> : null}
		</details>
	);
}

function getStatusLabel(status: DiffApplyFileView['status']): string {
	switch (status) {
		case 'ready':
			return 'ready';
		case 'needs-info':
			return 'needs path';
		case 'blocked':
			return 'blocked';
		case 'applied':
			return 'applied';
		case 'already-applied':
			return 'already applied';
		case 'partial':
			return 'partially applied';
		default:
			return 'not checked';
	}
}

function getStatusClassName(status: DiffApplyFileView['status']): string {
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

function getCardClassName(view: DiffApplyFileView): string {
	if (view.status === 'blocked') {
		return 'border-error/40 bg-error/5';
	}

	if (view.status === 'needs-info' || view.status === 'partial') {
		return 'border-warning/40 bg-warning/5';
	}

	return 'border-base-300 bg-base-100';
}

function OutcomeNotice({ label, outcome }: { label: string; outcome?: DiffApplyFileView['fullOutcome'] }) {
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
	const blockedCount = file.hunks.filter(hunk => {
		const status = view.hunkOutcomes[hunk.id]?.status;
		return status === 'blocked' || status === 'needs-info';
	}).length;
	const selectedCount = view.selectedHunkIDs.length;
	const allHunksChecked = checkedCount === file.hunks.length && checkedCount > 0;
	const canCheckHunks =
		view.canCheck && file.canApplyPartial && file.hunks.length <= MAX_INTERACTIVE_HUNK_PROBES && !controller.busy;

	return (
		<div className="space-y-3">
			<div className="flex flex-wrap items-center justify-between gap-2">
				<div>
					<div className="text-sm font-semibold">Apply selected hunks</div>
					<div className="text-base-content/60 mt-1 text-xs">
						Each hunk is checked without writing. The selected combined patch is dry-run again before any write.
					</div>
				</div>

				<button
					type="button"
					className="btn btn-xs btn-outline"
					disabled={!canCheckHunks}
					onClick={() => {
						void controller.checkHunks(file.id);
					}}
				>
					Check hunks
				</button>
			</div>

			{!file.canApplyPartial ? (
				<div className="text-base-content/60 text-xs">
					Partial application is available only for valid, non-overlapping text modifications without file-operation
					metadata.
				</div>
			) : null}

			{file.hunks.length > MAX_INTERACTIVE_HUNK_PROBES ? (
				<div className="text-warning text-xs">
					This file has {file.hunks.length} hunks. Per-hunk checking is limited to {MAX_INTERACTIVE_HUNK_PROBES} hunks.
				</div>
			) : null}

			{checkedCount > 0 ? (
				<div className="text-base-content/70 text-xs">
					{checkedCount}/{file.hunks.length} checked, {readyCount} applicable, {blockedCount} unavailable.
				</div>
			) : null}

			<div className="space-y-2">
				{file.hunks.map((hunk, index) => {
					const outcome = view.hunkOutcomes[hunk.id];
					const isReady = outcome?.status === 'ready';
					const isProblem = outcome?.status === 'blocked' || outcome?.status === 'needs-info';

					return (
						<div
							key={hunk.id}
							className={`rounded-lg border p-3 ${isProblem ? 'border-error/30 bg-error/5' : 'border-base-300'}`}
						>
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
								<span className="font-semibold">Hunk {index + 1}</span>
								<span className="min-w-0">
									<span className="text-base-content/60 ml-2">{outcome?.status ?? 'not checked'}</span>
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

							<LazyDetails
								className="mt-2"
								summary="View hunk source"
								renderContent={() => (
									<pre className="app-bg-code max-h-64 overflow-auto rounded-sm p-2 text-xs">
										<code>{[hunk.header, ...hunk.lines].join('\n')}</code>
									</pre>
								)}
							/>
						</div>
					);
				})}
			</div>

			<OutcomeNotice label="Selected hunks" outcome={view.partialOutcome} />

			<button
				type="button"
				className="btn btn-sm btn-primary"
				disabled={!!controller.busy || !allHunksChecked || selectedCount === 0}
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
	const inputID = `interactive-diff-target-${file.id}`;
	const isActive = controller.busy?.activeFileIDs.includes(file.id) ?? false;
	const displayPath = getInteractiveDiffTargetPath(file) || 'Target path required';
	const candidates = view.candidates.slice(0, DISPLAY_CANDIDATE_LIMIT);
	const fileDiagnostics = uniqueDiagnostics([
		...file.diagnostics,
		...(view.fullOutcome?.diagnostics ?? []),
		...(view.partialOutcome?.diagnostics ?? []),
	]);
	const canUsePartialHunks = file.hunks.length > 1;

	return (
		<section className={`rounded-xl border p-4 shadow-sm ${getCardClassName(view)}`} aria-busy={isActive}>
			<div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
				<div className="min-w-0 flex-1">
					<div className="flex flex-wrap items-center gap-2">
						<span className={`badge badge-outline badge-sm ${getStatusClassName(view.status)}`}>
							{getStatusLabel(view.status)}
						</span>
						<span className="badge badge-ghost badge-sm">{file.kind}</span>
						<span className="text-base-content/50 text-xs">{file.id}</span>
					</div>

					<div className="mt-2 font-mono text-sm font-semibold break-all">{displayPath}</div>

					<div className="text-base-content/60 mt-1 text-xs">
						{file.hunks.length} hunk{file.hunks.length === 1 ? '' : 's'}, +{file.addedLines}/-{file.deletedLines}
					</div>

					{file.oldPath && file.newPath && file.oldPath !== file.newPath ? (
						<div className="text-base-content/60 mt-2 space-y-1 font-mono text-xs break-all">
							<div>Old: {file.oldPath}</div>
							<div>New: {file.newPath}</div>
						</div>
					) : null}

					{isActive ? (
						<output className="text-info mt-3 flex items-center gap-2 text-xs">
							<span className="loading loading-spinner loading-xs" />
							Checking or applying this file.
						</output>
					) : null}
				</div>

				<div className="flex shrink-0 flex-wrap gap-2">
					<button
						type="button"
						className="btn btn-xs btn-outline"
						disabled={!!controller.busy || !view.canCheck}
						onClick={() => {
							void controller.checkFile(file.id);
						}}
					>
						Dry run
					</button>

					<button
						type="button"
						className="btn btn-xs btn-primary"
						disabled={!!controller.busy || !view.canCheck}
						title="Runs a fresh dry run immediately before applying this file."
						onClick={() => {
							void controller.applyFile(file.id);
						}}
					>
						Apply file
					</button>
				</div>
			</div>

			<OutcomeNotice label="File" outcome={view.fullOutcome} />

			<label className="mt-4 mb-1 block text-xs font-semibold" htmlFor={inputID}>
				Absolute target file path
			</label>

			<input
				id={inputID}
				className={`input input-sm w-full font-mono text-xs ${
					view.hasInvalidPathInput ? 'input-error' : !view.targetPath ? 'input-warning' : ''
				}`}
				value={view.pathInput}
				disabled={!!controller.busy}
				placeholder="Enter an absolute local target path"
				spellCheck={false}
				onChange={event => {
					controller.setTargetPath(file.id, event.target.value);
				}}
			/>

			<div className="text-base-content/60 mt-1 text-xs">
				{view.hasInvalidPathInput
					? 'Target paths must be absolute and cannot contain dot segments.'
					: !view.targetPath
						? isNewInteractiveDiffFile(file)
							? 'A new file needs a workspace-supported or manually entered absolute target path.'
							: 'Choose or enter an absolute target path before checking this file.'
						: 'Changing this path invalidates prior checks for this file.'}
			</div>

			{candidates.length > 0 ? (
				<select
					className="select select-sm mt-2 w-full font-mono text-xs"
					value=""
					disabled={!!controller.busy}
					aria-label={`Supported target paths for ${displayPath}`}
					onChange={event => {
						if (event.target.value) {
							controller.setTargetPath(file.id, event.target.value);
						}
					}}
				>
					<option value="">Choose a supported target path</option>
					{candidates.map(candidate => (
						<option key={candidate} value={candidate}>
							{candidate}
						</option>
					))}
				</select>
			) : null}

			{fileDiagnostics.length > 0 ? (
				<div className="mt-3">
					{renderDiagnosticsPanel({
						title: 'File diagnostics',
						description: 'These diagnostics apply only to this file section.',
						diagnostics: fileDiagnostics,
					})}
				</div>
			) : null}

			{canUsePartialHunks ? (
				<LazyDetails
					className="mt-3"
					summary="Review individual hunks"
					renderContent={() => <HunkPanel view={view} controller={controller} />}
				/>
			) : null}

			<LazyDetails
				className="mt-3"
				summary="View file patch"
				renderContent={() => (
					<pre className="app-bg-code max-h-80 overflow-auto rounded-sm p-3 text-xs">
						<code>{file.sourceText}</code>
					</pre>
				)}
			/>
		</section>
	);
}

function DiffApplyModalContent({ controller }: { controller: DiffApplyController }) {
	const { requestClose } = useModalDialogController();

	const completedCount = controller.appliedFileCount + controller.alreadyAppliedFileCount;

	const busyLabel = controller.busy
		? controller.busy.stopping
			? 'Stopping after in-flight requests finish. An already-submitted write cannot be undone.'
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
								<span>Apply unified diff</span>
							</h3>

							<div className="text-base-content/60 mt-1 flex flex-wrap gap-x-2 gap-y-1 text-xs">
								<span>{controller.files.length} file sections</span>
								<span>{controller.readyFileCount} ready</span>
								{controller.needsInfoFileCount > 0 ? <span>{controller.needsInfoFileCount} need paths</span> : null}
								{controller.blockedFileCount > 0 ? <span>{controller.blockedFileCount} blocked</span> : null}
								{completedCount > 0 ? <span>{completedCount} complete</span> : null}
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

					<label
						className="mt-3 flex items-center gap-2 text-sm"
						title="Strict matching disables backend fuzzy matching."
					>
						<input
							type="checkbox"
							className="checkbox checkbox-xs"
							checked={controller.strict}
							disabled={!!controller.busy}
							onChange={event => {
								controller.setStrict(event.target.checked);
							}}
						/>
						Strict matching
					</label>

					<div className="text-base-content/60 mt-1 text-xs">
						Changing matching mode invalidates previous dry-run results.
					</div>
				</div>

				<div className="min-h-0 flex-1 space-y-3 overflow-y-auto overscroll-contain p-4 sm:px-5">
					{renderDiagnosticsPanel({
						title: 'Patch diagnostics',
						description: 'These are parser-level diagnostics for the entire patch.',
						diagnostics: controller.patchDiagnostics,
					})}

					{controller.files.length === 0 ? (
						<div className="border-base-300 rounded-xl border p-4 text-sm">
							No interactive file sections could be extracted from this diff.
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
								Dry runs are bounded. Writes are serialized and rechecked immediately before apply.
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
						Files apply independently. A failure in one file does not roll back files already applied.
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
						disabled={!!controller.busy || controller.patchHasErrors || controller.files.length === 0}
						onClick={() => {
							void controller.checkAll();
						}}
					>
						Dry run all
					</button>

					<button
						type="button"
						className="btn btn-sm btn-primary"
						disabled={!!controller.busy || controller.patchHasErrors || controller.readyFileCount === 0}
						title="Only files with a current successful dry run are selected. Each is dry-run again immediately before writing."
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
