import { useCallback, useEffect, useRef, useState } from 'react';

import type { ParsedInteractiveDiff } from '@/components/markdown/diff_apply_model';
import { DiffApplyActions, getDiffBlockPresentation } from '@/components/markdown/diff_apply_actions';
import { useDiffApplyController } from '@/components/markdown/diff_apply_controller';
import { DiffApplyModal } from '@/components/markdown/diff_apply_modal';
import { parseInteractiveDiff } from '@/components/markdown/diff_apply_model';
import { useMessageDiffScope, useMessageDiffSnapshot } from '@/components/markdown/message_diff_scope';

interface DiffApplyControlProps {
	language: string;
	diffText: string;
	isBusy: boolean;
	candidatePaths?: string[];
	candidatePathsReady?: boolean;
	workspaceRoots?: string[];
	messageDiffID?: string;
}

interface DiffApplySessionProps {
	parsed: ParsedInteractiveDiff;
	candidatePaths?: string[];
	workspaceRoots?: string[];
	openInitially: boolean;
	consumeInitialReview: (parsed: ParsedInteractiveDiff) => boolean;
}

function DiffApplySession({
	parsed,
	candidatePaths,
	workspaceRoots,
	openInitially,
	consumeInitialReview,
}: DiffApplySessionProps) {
	const controller = useDiffApplyController(parsed, candidatePaths, workspaceRoots);
	const [isDetailsOpen, setIsDetailsOpen] = useState(openInitially);
	const { reviewAll } = controller;

	useEffect(() => {
		let cancelled = false;

		queueMicrotask(() => {
			if (!cancelled && consumeInitialReview(parsed)) {
				void reviewAll();
			}
		});

		return () => {
			cancelled = true;
		};
	}, [consumeInitialReview, parsed, reviewAll]);

	const presentation = getDiffBlockPresentation(controller);
	const busy = controller.busy !== null;
	const total = controller.files.length;

	return (
		<>
			<DiffApplyActions
				className="app-text-code font-mono"
				onReview={() => {
					void controller.reviewAll();
				}}
				onDetails={() => {
					setIsDetailsOpen(true);
				}}
				onApply={() => {
					void controller.applyReady();
				}}
				reviewDisabled={busy || total === 0}
				applyDisabled={busy || controller.submittableFileCount === 0}
				reviewing={controller.busy?.kind === 'review'}
				applying={controller.busy?.kind === 'apply'}
				detailsLabel={presentation.label}
				detailsTone={presentation.tone}
				applyLabel={total === 1 ? 'Apply' : `Apply ${controller.submittableFileCount}/${total}`}
			/>

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

function StandaloneDiffApplyControl({
	language,
	diffText,
	candidatePaths,
	candidatePathsReady = true,
	workspaceRoots,
}: DiffApplyControlProps) {
	const [session, setSession] = useState<{
		language: string;
		diffText: string;
		parsed: ParsedInteractiveDiff;
		openInitially: boolean;
	} | null>(null);

	const pendingInitialReviewRef = useRef<ParsedInteractiveDiff | null>(null);
	const consumeInitialReview = useCallback((parsed: ParsedInteractiveDiff): boolean => {
		if (pendingInitialReviewRef.current !== parsed) {
			return false;
		}
		pendingInitialReviewRef.current = null;
		return true;
	}, []);

	if (!session || session.language !== language || session.diffText !== diffText) {
		const initialize = (review: boolean): void => {
			const parsed = parseInteractiveDiff(diffText, language);
			pendingInitialReviewRef.current = review ? parsed : null;
			setSession({
				language,
				diffText,
				parsed,
				openInitially: !review,
			});
		};

		return (
			<DiffApplyActions
				className="app-text-code font-mono"
				onReview={() => {
					initialize(true);
				}}
				onDetails={() => {
					initialize(false);
				}}
				onApply={() => {}}
				reviewDisabled={!candidatePathsReady}
				detailsDisabled={!candidatePathsReady}
				applyDisabled
				reviewTitle={
					candidatePathsReady
						? 'Parse this patch and run a backend dry-run review. No files are changed.'
						: 'Preparing trusted target path suggestions for this patch.'
				}
			/>
		);
	}

	return (
		<DiffApplySession
			key={`${session.language}\u0000${session.diffText}`}
			parsed={session.parsed}
			candidatePaths={candidatePaths}
			workspaceRoots={workspaceRoots}
			openInitially={session.openInitially}
			consumeInitialReview={consumeInitialReview}
		/>
	);
}

function MessageOwnedDiffApplyControl({ messageDiffID }: { messageDiffID: string }) {
	const store = useMessageDiffScope();
	const snapshot = useMessageDiffSnapshot(store);

	if (!store) {
		return null;
	}

	const block = snapshot.blocks.find(candidate => candidate.id === messageDiffID);
	const controller = store.getControllerForUI(messageDiffID);
	const rawController = snapshot.controllers.get(messageDiffID);
	const total = block?.parsed.files.length ?? 0;
	const eligible = controller?.submittableFileCount ?? 0;
	const presentation = controller
		? getDiffBlockPresentation(controller)
		: { label: 'Details / targets', tone: 'neutral' as const };

	const operation = snapshot.operation;
	const active = operation?.activeBlockID === messageDiffID;
	const disabled = !snapshot.targetsReady || snapshot.isBusy || Boolean(operation);

	return (
		<DiffApplyActions
			className="app-text-code font-mono"
			onReview={() => {
				void store.reviewBlock(messageDiffID);
			}}
			onDetails={() => {
				store.openDetails(messageDiffID);
			}}
			onApply={() => {
				void store.applyBlock(messageDiffID);
			}}
			reviewDisabled={disabled || total === 0}
			detailsDisabled={!snapshot.targetsReady}
			applyDisabled={disabled || eligible === 0}
			reviewing={active && operation?.kind.startsWith('review')}
			applying={active && operation?.kind.startsWith('apply')}
			detailsLabel={presentation.label}
			detailsTone={presentation.tone}
			applyLabel={total === 1 ? 'Apply' : `Apply ${eligible}/${total}`}
			applyTitle={
				rawController?.files.some(file => file.targetPath)
					? 'Submit eligible unfinished changes. Explicit targets can permit submission without review.'
					: 'Submit eligible unfinished changes with successful backend reviews.'
			}
		/>
	);
}

export function DiffApplyControl(props: DiffApplyControlProps) {
	const store = useMessageDiffScope();

	if (props.isBusy) {
		return null;
	}

	if (store && props.messageDiffID) {
		return <MessageOwnedDiffApplyControl messageDiffID={props.messageDiffID} />;
	}

	return <StandaloneDiffApplyControl {...props} />;
}
