import type {
	WorkspaceCatalogView,
	WorkspaceContextView,
	WorkspaceDiagnostic,
	WorkspaceDiscovery,
	WorkspaceOccurrenceView,
	WorkspaceRecordView,
	WorkspaceSkillView,
	WorkspaceView,
} from '@/spec/workspace';
import {
	WorkspaceArtifactKind,
	WorkspaceDiagnosticSeverity,
	WorkspaceOccurrenceState,
	WorkspaceRecordMode,
	WorkspaceRecordState,
} from '@/spec/workspace';

import type { StatusTone } from '@/components/managementui/management_class_consts';

export interface WorkspaceCatalogData {
	catalog: WorkspaceCatalogView;
	contexts: WorkspaceContextView[];
	skills: WorkspaceSkillView[];
	contextLoadError?: string;
	skillLoadError?: string;
}

export function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim()) {
		return error.message;
	}
	return fallback;
}

export function cloneWorkspaceDiscovery(discovery?: WorkspaceDiscovery): WorkspaceDiscovery {
	return {
		includeReadme: discovery?.includeReadme ?? true,
		additionalLocators: [...(discovery?.additionalLocators ?? [])],
		additionalRoots: (discovery?.additionalRoots ?? []).map(root => ({
			root: root.root,
			recursive: root.recursive,
			includePatterns: root.includePatterns ? [...root.includePatterns] : undefined,
		})),
	};
}

export function sortWorkspaces(workspaces: WorkspaceView[]): WorkspaceView[] {
	return [...workspaces].toSorted((left, right) => {
		if (left.enabled !== right.enabled) {
			return left.enabled ? -1 : 1;
		}

		return left.displayName.localeCompare(right.displayName, undefined, {
			sensitivity: 'base',
		});
	});
}

export function workspaceMatchesSearch(workspace: WorkspaceView, rawQuery: string): boolean {
	const query = rawQuery.trim().toLowerCase();
	if (!query) {
		return true;
	}

	const haystackParts = [
		workspace.displayName,
		workspace.description,
		workspace.rootID,
		workspace.mode,
		workspace.primarySourceID,
	];

	for (const locator of workspace.discovery.additionalLocators ?? []) {
		haystackParts.push(locator);
	}

	for (const root of workspace.discovery.additionalRoots ?? []) {
		haystackParts.push(root.root);

		for (const includePattern of root.includePatterns ?? []) {
			haystackParts.push(includePattern);
		}
	}

	for (const attachment of workspace.attachments) {
		haystackParts.push(attachment.sourceID, attachment.role, String(attachment.priority));
	}

	const haystack = haystackParts.filter(Boolean).join('\n').toLowerCase();

	return haystack.includes(query);
}

export function getWorkspaceRecords(catalog: WorkspaceCatalogView): WorkspaceRecordView[] {
	const records = new Map<string, WorkspaceRecordView>();

	for (const resource of catalog.resources) {
		records.set(resource.record.id, resource.record);
	}

	for (const record of catalog.unresolvedRecords) {
		records.set(record.id, record);
	}

	return [...records.values()].toSorted((left, right) => {
		const kindOrder = left.kind.localeCompare(right.kind);
		if (kindOrder !== 0) {
			return kindOrder;
		}
		return left.name.localeCompare(right.name, undefined, { sensitivity: 'base' });
	});
}

export function replaceWorkspaceRecord(
	data: WorkspaceCatalogData,
	nextRecord: WorkspaceRecordView
): WorkspaceCatalogData {
	const replace = (record: WorkspaceRecordView): WorkspaceRecordView =>
		record.id === nextRecord.id ? nextRecord : record;

	return {
		...data,
		catalog: {
			...data.catalog,
			resources: data.catalog.resources.map(resource =>
				resource.record.id === nextRecord.id
					? {
							...resource,
							record: nextRecord,
							definitionDigest: nextRecord.resolvedDefinition ?? resource.definitionDigest,
						}
					: resource
			),
			groups: data.catalog.groups.map(group => ({
				...group,
				resources: group.resources.map(resource =>
					resource.record.id === nextRecord.id
						? {
								...resource,
								record: nextRecord,
								definitionDigest: nextRecord.resolvedDefinition ?? resource.definitionDigest,
							}
						: resource
				),
			})),
			unresolvedRecords: data.catalog.unresolvedRecords.map(replace),
		},
		contexts: data.contexts.map(context =>
			context.recordID === nextRecord.id
				? {
						...context,
						recordRevision: nextRecord.revision,
						definitionDigest: nextRecord.resolvedDefinition ?? context.definitionDigest,
						enabled: nextRecord.enabled,
						state: nextRecord.state,
						runtimeAllowed: nextRecord.runtimeAllowed,
						diagnostics: nextRecord.diagnostics,
					}
				: context
		),
		skills: data.skills.map(skill =>
			skill.recordID === nextRecord.id
				? {
						...skill,
						recordRevision: nextRecord.revision,
						definitionDigest: nextRecord.resolvedDefinition ?? skill.definitionDigest,
						state: nextRecord.state,
						runtimeAllowed: nextRecord.runtimeAllowed,
						diagnostics: nextRecord.diagnostics,
						skill: {
							...skill.skill,
							isEnabled: nextRecord.enabled,
						},
					}
				: skill
		),
	};
}

export function removeWorkspaceRecord(data: WorkspaceCatalogData, recordID: string): WorkspaceCatalogData {
	return {
		...data,
		catalog: {
			...data.catalog,
			resources: data.catalog.resources.filter(resource => resource.record.id !== recordID),
			groups: data.catalog.groups.map(group => ({
				...group,
				resources: group.resources.filter(resource => resource.record.id !== recordID),
			})),
			unresolvedRecords: data.catalog.unresolvedRecords.filter(record => record.id !== recordID),
			unresolvedRecordCount: Math.max(
				0,
				data.catalog.unresolvedRecordCount -
					(data.catalog.unresolvedRecords.some(record => record.id === recordID) ? 1 : 0)
			),
		},
		contexts: data.contexts.filter(context => context.recordID !== recordID),
		skills: data.skills.filter(skill => skill.recordID !== recordID),
	};
}

export function getRecordStateTone(state: WorkspaceRecordState): StatusTone {
	switch (state) {
		case WorkspaceRecordState.Available:
			return 'success';
		case WorkspaceRecordState.Stale:
		case WorkspaceRecordState.Missing:
			return 'warning';
		case WorkspaceRecordState.Invalid:
		case WorkspaceRecordState.Incompatible:
			return 'error';
		default:
			return 'neutral';
	}
}

export function getOccurrenceStateTone(occurrence: WorkspaceOccurrenceView): StatusTone {
	switch (occurrence.state) {
		case WorkspaceOccurrenceState.Valid:
			return 'success';
		case WorkspaceOccurrenceState.Missing:
			return 'warning';
		case WorkspaceOccurrenceState.Invalid:
			return 'error';
		default:
			return 'neutral';
	}
}

export function getDiagnosticTone(diagnostic: WorkspaceDiagnostic): StatusTone {
	switch (diagnostic.severity) {
		case WorkspaceDiagnosticSeverity.Error:
			return 'error';
		case WorkspaceDiagnosticSeverity.Warning:
			return 'warning';
		case WorkspaceDiagnosticSeverity.Info:
			return 'info';
		default:
			return 'neutral';
	}
}

export function getArtifactKindLabel(kind: WorkspaceArtifactKind): string {
	switch (kind) {
		case WorkspaceArtifactKind.Context:
			return 'Context';
		case WorkspaceArtifactKind.Skill:
			return 'Skill';
		case WorkspaceArtifactKind.Definition:
			return 'Workspace definition';
		default:
			return kind;
	}
}

export function getRecordModeLabel(mode: WorkspaceRecordMode): string {
	return mode === WorkspaceRecordMode.Pinned ? 'Pinned' : 'Following';
}

export function collectWorkspaceDiagnostics(data: WorkspaceCatalogData): WorkspaceDiagnostic[] {
	const diagnostics: WorkspaceDiagnostic[] = [];
	const seen = new Set<string>();

	const add = (items?: WorkspaceDiagnostic[]) => {
		for (const diagnostic of items ?? []) {
			const key = [
				diagnostic.severity,
				diagnostic.code,
				diagnostic.message,
				diagnostic.location?.locator,
				diagnostic.location?.subresourceLocator,
				diagnostic.location?.line,
				diagnostic.location?.column,
			].join(':');

			if (!seen.has(key)) {
				seen.add(key);
				diagnostics.push(diagnostic);
			}
		}
	};

	add(data.catalog.diagnostics);

	for (const resource of data.catalog.resources) {
		add(resource.diagnostics);
		add(resource.record.diagnostics);
	}

	for (const record of data.catalog.unresolvedRecords) {
		add(record.diagnostics);
	}

	for (const occurrence of data.catalog.occurrences) {
		add(occurrence.diagnostics);
	}

	for (const context of data.contexts) {
		add(context.diagnostics);
	}

	for (const skill of data.skills) {
		add(skill.diagnostics);
	}

	return diagnostics;
}

export function workspaceRecordMatchesSearch(record: WorkspaceRecordView, rawQuery: string): boolean {
	const query = rawQuery.trim().toLowerCase();
	if (!query) {
		return true;
	}

	return [
		record.name,
		record.id,
		record.kind,
		record.state,
		record.mode,
		record.sourceID,
		record.locator,
		record.subresourceLocator,
		record.pinnedDefinition,
		record.resolvedDefinition,
		...(record.diagnostics ?? []).flatMap(diagnostic => [diagnostic.code, diagnostic.message]),
	]
		.filter(Boolean)
		.join('\n')
		.toLowerCase()
		.includes(query);
}

export function formatByteCount(bytes: number): string {
	if (!Number.isFinite(bytes) || bytes <= 0) {
		return '0 B';
	}

	if (bytes < 1024) {
		return `${bytes} B`;
	}

	if (bytes < 1024 * 1024) {
		return `${(bytes / 1024).toFixed(1)} KiB`;
	}

	return `${(bytes / (1024 * 1024)).toFixed(1)} MiB`;
}
