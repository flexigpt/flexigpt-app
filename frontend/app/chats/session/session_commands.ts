import type { SessionRun, SessionSnapshot } from '@/spec/session';
import type { SessionCommandMeta, SessionCommandReceipt, SessionRevision } from '@/spec/session_base';
import { SessionRunState } from '@/spec/session_base';

import { ensureMakeID, getUUIDv7 } from '@/lib/uuid_utils';

function newSessionCommandID(): string {
	try {
		return getUUIDv7();
	} catch {
		return ensureMakeID();
	}
}

export function sessionCommandMeta(
	sessionRevision?: SessionRevision,
	runRevision?: SessionRevision
): SessionCommandMeta {
	return {
		id: newSessionCommandID(),
		...(sessionRevision === undefined ? {} : { expectedSessionRevision: sessionRevision }),
		...(runRevision === undefined ? {} : { expectedRunRevision: runRevision }),
	};
}

export function requireAcceptedSessionCommand(
	receipt: SessionCommandReceipt,
	operation: string
): SessionCommandReceipt {
	if (receipt.accepted) {
		return receipt;
	}

	throw new Error(receipt.message?.trim() || `${operation} was rejected by the Session Runtime.`);
}

function isTerminalSessionRun(run: SessionRun | undefined): boolean {
	if (!run) {
		return true;
	}

	return (
		run.state === SessionRunState.Completed ||
		run.state === SessionRunState.Failed ||
		run.state === SessionRunState.Cancelled
	);
}

function sessionRunTimestampMilliseconds(value: string | undefined): number {
	const parsed = Date.parse(value ?? '');
	return Number.isFinite(parsed) ? parsed : Number.NEGATIVE_INFINITY;
}

// Go serializes time.Time as RFC3339Nano, where the fractional part is
// optional. A lexical comparison can therefore rank "...:01Z" after
// "...:01.100Z", even though the latter is newer.
function compareSessionRunsNewestFirst(left: SessionRun, right: SessionRun): number {
	const leftUpdatedAt = sessionRunTimestampMilliseconds(left.updatedAt);
	const rightUpdatedAt = sessionRunTimestampMilliseconds(right.updatedAt);
	if (leftUpdatedAt !== rightUpdatedAt) {
		return rightUpdatedAt > leftUpdatedAt ? 1 : -1;
	}

	const leftCreatedAt = sessionRunTimestampMilliseconds(left.createdAt);
	const rightCreatedAt = sessionRunTimestampMilliseconds(right.createdAt);
	if (leftCreatedAt !== rightCreatedAt) {
		return rightCreatedAt > leftCreatedAt ? 1 : -1;
	}

	return right.id.localeCompare(left.id);
}

export function selectedSessionRun(snapshot: SessionSnapshot, preferredRunID?: string): SessionRun | undefined {
	if (preferredRunID) {
		const preferred = snapshot.runs.find(run => run.id === preferredRunID);
		if (preferred) {
			return preferred;
		}
	}

	return (
		snapshot.runs.find(run => !isTerminalSessionRun(run)) ??
		[...snapshot.runs].toSorted(compareSessionRunsNewestFirst)[0]
	);
}
