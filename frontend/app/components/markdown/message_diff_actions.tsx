import { useId } from 'react';
import { FiAlertTriangle, FiChevronDown, FiLoader, FiSquare } from 'react-icons/fi';

import { Popover, PopoverDisclosure, usePopoverStore, useStoreState } from '@ariakit/react';

import type { MessageDiffRow } from '@/components/markdown/message_diff_scope';
import {
	DiffApplyActions,
	getDiffActionButtonClassName,
	getDiffFilePresentation,
} from '@/components/markdown/diff_apply_actions';
import { getInteractiveDiffTargetPath, isDiffApplyOutcomeComplete } from '@/components/markdown/diff_apply_model';
import {
	getMessageDiffRows,
	getMessageDiffSummary,
	useMessageDiffScope,
	useMessageDiffSnapshot,
} from '@/components/markdown/message_diff_scope';

function getRowPath(row: MessageDiffRow): string {
	if (row.file.id === 'unsplit-section') {
		return 'Unresolved file sections';
	}

	return (
		row.view?.applyOutcome?.resolvedTargetPath ||
		row.view?.reviewOutcome?.resolvedTargetPath ||
		row.view?.targetPath ||
		getInteractiveDiffTargetPath(row.file) ||
		'Target path unresolved'
	);
}

function getFileName(path: string): string {
	return path.replaceAll('\\', '/').split('/').at(-1) || path;
}

export function MessageDiffActions() {
	const store = useMessageDiffScope();
	const snapshot = useMessageDiffSnapshot(store);
	const popover = usePopoverStore({ placement: 'top-end' });
	const open = useStoreState(popover, 'open');
	const headingID = useId();

	if (!store || snapshot.blocks.length === 0 || snapshot.isBusy) {
		return null;
	}

	const summary = getMessageDiffSummary(snapshot);
	const rows = getMessageDiffRows(snapshot);
	const operation = snapshot.operation;
	const progress = store.getProgress();
	const attention = summary.needsInfo + summary.blocked + summary.partial;
	const allComplete = summary.total > 0 && summary.completed === summary.total;
	const unavailable = !snapshot.targetsReady || Boolean(operation);
	const hasPatchNotes = snapshot.blocks.some(block => block.parsed.diagnostics.length > 0);

	const summaryText = [
		summary.unresolved > 0
			? `${summary.total} change entries; ${summary.unresolved} contain unresolved file sections`
			: `${summary.total} file change${summary.total === 1 ? '' : 's'} in this reply`,
		`${summary.eligible} can be submitted`,
		`${summary.completed} completed`,
		`${summary.ready} ready`,
		`${summary.needsInfo} need information`,
		`${summary.blocked} blocked`,
		`${summary.partial} partially applied`,
		`${summary.pending} not reviewed`,
	].join('\n');

	const applyLabel = allComplete
		? `Applied ${summary.completed}/${summary.total}`
		: `Apply ${summary.eligible}/${summary.total}`;

	const operationLabel = operation
		? operation.stopping
			? 'Stopping'
			: operation.kind === 'review-hunks'
				? 'Reviewing hunks'
				: operation.kind === 'apply-hunks'
					? 'Applying hunks'
					: operation.kind === 'review'
						? 'Reviewing'
						: 'Applying'
		: '';

	const pathOccurrences = new Map<string, number>();
	for (const row of rows) {
		const path = getRowPath(row);
		pathOccurrences.set(path, (pathOccurrences.get(path) ?? 0) + 1);
	}
	const seenPaths = new Map<string, number>();

	return (
		<div className="flex shrink-0 items-center" data-message-diff-actions>
			<fieldset
				className="border-base-300 flex min-w-0 items-center overflow-hidden rounded-full border"
				aria-label="Diff actions for this assistant message"
			>
				{operation ? (
					<>
						<output className="inline-flex h-6 items-center gap-1 px-2 text-[11px] tabular-nums" aria-live="polite">
							<FiLoader size={12} className="animate-spin" aria-hidden="true" />
							{operationLabel} {progress.completed}/{progress.total}
						</output>
						<button
							type="button"
							className={`${getDiffActionButtonClassName('warning')} rounded-none border-y-0 border-r-0`}
							disabled={operation.stopping}
							onClick={store.cancel}
							title="Stop remaining work. Already submitted writes can still finish and cannot be undone."
						>
							<FiSquare size={11} />
							Stop remaining
						</button>
					</>
				) : (
					<>
						<button
							type="button"
							className={`${getDiffActionButtonClassName()} rounded-none border-0`}
							disabled={unavailable || summary.total === 0}
							onClick={() => {
								void store.reviewAll();
							}}
							title={
								snapshot.targetsReady
									? 'Review every file change in this reply. No files are changed.'
									: 'Preparing trusted target path suggestions.'
							}
						>
							Review all
						</button>
						<button
							type="button"
							className={`${getDiffActionButtonClassName('success')} rounded-none border-y-0 border-r-0`}
							disabled={unavailable || summary.eligible === 0}
							onClick={() => {
								void store.applyAll();
							}}
							title={`${summaryText}\n\nSubmit eligible unfinished changes in message order. Explicit targets can permit submission without review.`}
						>
							{applyLabel}
						</button>
					</>
				)}

				<PopoverDisclosure
					store={popover}
					className={`${getDiffActionButtonClassName()} gap-1 rounded-none border-y-0 border-r-0 px-2`}
					aria-label={`Show file changes: ${summary.ready} ready, ${attention} need attention`}
					title={summaryText}
				>
					{attention > 0 ? (
						<FiAlertTriangle
							size={12}
							className={summary.blocked > 0 ? 'text-error' : 'text-warning'}
							aria-hidden="true"
						/>
					) : null}
					<FiChevronDown size={13} className={`transition-transform ${open ? 'rotate-180' : ''}`} aria-hidden="true" />
				</PopoverDisclosure>
			</fieldset>

			{open ? (
				<Popover
					store={popover}
					portal
					gutter={8}
					overflowPadding={8}
					autoFocusOnShow
					aria-labelledby={headingID}
					data-disable-chat-shortcuts="true"
					className="bg-base-100 text-base-content border-base-300 z-50 flex max-h-[min(70dvh,32rem)] w-152 max-w-[calc(100vw-1rem)] flex-col overflow-hidden rounded-xl border shadow-xl focus-visible:outline-none"
				>
					<div className="border-base-300 shrink-0 border-b px-3 py-2">
						<h3 id={headingID} className="text-xs font-semibold">
							File changes in this reply
						</h3>
						<p className="text-base-content/60 mt-1 text-[11px]">
							Review is a backend dry run. Details opens the existing patch modal.
						</p>
						{snapshot.notice ? (
							<output className="text-error mt-1 block text-xs" aria-live="polite">
								{snapshot.notice}
							</output>
						) : null}
					</div>

					<div className="min-h-0 overflow-y-auto overscroll-contain p-1">
						{rows.map(row => {
							const path = getRowPath(row);
							const occurrence = (seenPaths.get(path) ?? 0) + 1;
							seenPaths.set(path, occurrence);

							const repeated = (pathOccurrences.get(path) ?? 0) > 1;
							const view = row.view;
							const status = view?.status ?? 'pending';
							const presentation = getDiffFilePresentation(status, Boolean(view?.targetPath));
							const nativeBusy = snapshot.controllers.get(row.blockID)?.busy;
							const active =
								operation?.activeBlockID === row.blockID &&
								(nativeBusy?.activeFileIDs.includes(row.file.id) || operation.activeFileID === row.file.id);
							const reviewing = active && operation?.kind.startsWith('review');
							const applying = active && operation?.kind.startsWith('apply');
							const outcome = view?.applyOutcome ?? view?.reviewOutcome;
							const completed = isDiffApplyOutcomeComplete(status);

							return (
								<div
									key={`${row.blockID}:${row.file.id}`}
									className="border-base-300/60 flex min-w-0 flex-col gap-2 rounded-lg border-b p-2 last:border-b-0 sm:flex-row sm:items-center"
									aria-busy={active}
								>
									<div className="min-w-0 flex-1" title={path}>
										<div className="flex min-w-0 items-center gap-2">
											<span className="min-w-0 truncate font-mono text-xs font-medium">{getFileName(path)}</span>
											{row.file.kind !== 'modify' ? (
												<span className="text-base-content/60 shrink-0 text-[10px]">{row.file.kind}</span>
											) : null}
											{repeated ? (
												<span className="text-base-content/60 shrink-0 text-[10px]">Change {occurrence}</span>
											) : null}
										</div>
										{path !== getFileName(path) ? (
											<div className="text-base-content/60 mt-0.5 truncate font-mono text-[10px]">{path}</div>
										) : null}
										{row.file.kind === 'rename' && row.file.newPath ? (
											<div className="text-base-content/60 mt-0.5 truncate font-mono text-[10px]">
												New path: {row.file.newPath}
											</div>
										) : null}
									</div>

									<DiffApplyActions
										className="shrink-0"
										onReview={() => {
											void store.reviewFile(row.blockID, row.file.id);
										}}
										onDetails={() => {
											store.openDetails(row.blockID, row.file.id);
										}}
										onApply={() => {
											void store.applyFile(row.blockID, row.file.id);
										}}
										reviewDisabled={unavailable}
										detailsDisabled={!snapshot.targetsReady}
										applyDisabled={unavailable || !view?.canApply || completed}
										reviewing={reviewing}
										applying={applying}
										detailsLabel={presentation.label}
										detailsTone={presentation.tone}
										detailsTitle={
											outcome?.message
												? `${outcome.message}\nOpen patch details for this file change.`
												: 'Open patch details for this file change.'
										}
									/>
								</div>
							);
						})}

						{rows.length === 0 ? (
							<div className="text-base-content/70 px-2 py-3 text-xs">
								No file sections could be separated. Patch notes remain available below.
							</div>
						) : null}

						{hasPatchNotes
							? snapshot.blocks
									.filter(block => block.parsed.diagnostics.length > 0)
									.map(block => (
										<button
											key={`notes:${block.id}`}
											type="button"
											className="text-base-content/60 hover:bg-base-200 block w-full rounded-lg p-2 text-left text-[11px]"
											disabled={!snapshot.targetsReady}
											onClick={() => {
												store.openDetails(block.id);
											}}
										>
											{snapshot.blocks.length > 1 ? `Patch ${snapshot.blocks.indexOf(block) + 1} notes` : 'Patch notes'}
										</button>
									))
							: null}
					</div>
				</Popover>
			) : null}
		</div>
	);
}
