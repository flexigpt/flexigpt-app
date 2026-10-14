import type {
	SessionCommandMeta,
	SessionCommandReceipt,
	SessionRevision,
	SessionRun,
	SessionRunSnapshot,
	SessionSnapshot,
} from '@/spec/session';
import { SessionRunState } from '@/spec/session';

import { ensureMakeID, getUUIDv7 } from '@/lib/uuid_utils';

export function newSessionCommandID(): string {
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

export function isTerminalSessionRun(run: SessionRun | undefined): boolean {
	if (!run) {
		return true;
	}

	return (
		run.state === SessionRunState.Completed ||
		run.state === SessionRunState.Failed ||
		run.state === SessionRunState.Cancelled
	);
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
		[...snapshot.runs].toSorted((left, right) => right.updatedAt.localeCompare(left.updatedAt))[0]
	);
}

export function selectedRunSnapshot(
	snapshot: SessionSnapshot,
	preferredRunID?: string
): SessionRunSnapshot | undefined {
	const run = selectedSessionRun(snapshot, preferredRunID);

	if (!run) {
		return undefined;
	}

	return {
		sessionID: snapshot.session.id,
		sessionRevision: snapshot.session.revision,
		run,
	};
}
