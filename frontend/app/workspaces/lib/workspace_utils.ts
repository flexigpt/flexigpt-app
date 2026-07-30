import type { ArtifactDiagnostic, ArtifactKind, ArtifactState } from '@/spec/artifact';
import {
	ArtifactDiagnosticSeverity,
	ArtifactOccurrenceState as ArtifactOccurrenceStateValue,
	ArtifactState as ArtifactStateValue,
} from '@/spec/artifact';
import type {
	WorkspaceArtifactView,
	WorkspaceCatalogView,
	WorkspaceContextView,
	WorkspaceOccurrenceView,
	WorkspaceSkillView,
	WorkspaceSuppressionView,
	WorkspaceView,
} from '@/spec/workspace';

import type { StatusTone } from '@/components/managementui/management_class_consts';

export interface WorkspaceCatalogData {
	catalog: WorkspaceCatalogView;
	contexts: WorkspaceContextView[];
	skills: WorkspaceSkillView[];
	suppressions: WorkspaceSuppressionView[];
	contextLoadError?: string;
	skillLoadError?: string;
	suppressionLoadError?: string;
}

export const WORKSPACE_DEFAULT_CONTEXT_FILES = ['AGENTS.md', 'CLAUDE.md'];
export const WORKSPACE_DEFAULT_SKILL_ROOTS = ['.skills/**/SKILL.md'];

export const WORKSPACE_CONTEXT_ARTIFACT_KIND = 'workspace.context';
export const WORKSPACE_SKILL_ARTIFACT_KIND = 'agent.skill';

function normalizeRequiredArray<T>(value: T[] | null | undefined): T[] {
	return Array.isArray(value) ? value : [];
}

function normalizeOptionalArray<T>(value: T[] | null | undefined): T[] | undefined {
	return Array.isArray(value) ? value : undefined;
}

/**
 * Normalizes collection fields from older or malformed API responses.
 *
 * The API contract requires these fields to be arrays. The Go API also
 * initializes them to empty slices, but this protects the UI when connected
 * to a still-running older backend that serializes nil slices as null.
 */
export function normalizeWorkspaceCatalog(catalog: WorkspaceCatalogView): WorkspaceCatalogView {
	return {
		...catalog,
		diagnostics: normalizeOptionalArray(catalog.diagnostics),
		resources: normalizeRequiredArray(catalog.resources),
		groups: normalizeRequiredArray(catalog.groups).map(group =>
			Object.assign(group, {
				resources: normalizeRequiredArray(group.resources),
				unrecorded: normalizeRequiredArray(group.unrecorded),
			})
		),
		occurrences: normalizeRequiredArray(catalog.occurrences),
		validOccurrences: normalizeRequiredArray(catalog.validOccurrences),
		invalidOccurrences: normalizeRequiredArray(catalog.invalidOccurrences),
		missingOccurrences: normalizeRequiredArray(catalog.missingOccurrences),
		unrecordedOccurrences: normalizeRequiredArray(catalog.unrecordedOccurrences),
		unresolvedArtifacts: normalizeRequiredArray(catalog.unresolvedArtifacts),
	};
}

export function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim()) {
		return error.message;
	}
	return fallback;
}

function cleanFilesystemPath(rawPath: string): string {
	const value = rawPath.trim().replaceAll('\\', '/');
	const drive = /^[A-Za-z]:\//.exec(value)?.[0];
	const isAbsolute = Boolean(drive) || value.startsWith('/');
	const prefix = drive ?? (value.startsWith('/') ? '/' : '');
	const remainder = drive ? value.slice(drive.length) : value.startsWith('/') ? value.slice(1) : value;
	const segments: string[] = [];

	for (const segment of remainder.split('/')) {
		if (!segment || segment === '.') {
			continue;
		}
		if (segment === '..') {
			if (segments.length > 0) {
				segments.pop();
			} else if (!isAbsolute) {
				segments.push(segment);
			}
			continue;
		}
		segments.push(segment);
	}

	const joined = segments.join('/');
	if (!joined) {
		return prefix || '.';
	}
	return `${prefix}${joined}`;
}

function isAbsoluteFilesystemPath(value: string): boolean {
	return value.startsWith('/') || /^[A-Za-z]:[\\/]/.test(value);
}

function normalizeWorkspaceLocator(value: string, allowRoot = false): string {
	const normalized = cleanFilesystemPath(value);

	if (
		normalized === '.' ||
		normalized.startsWith('/') ||
		/^[A-Za-z]:\//.test(normalized) ||
		normalized.split('/').includes('..')
	) {
		if (normalized === '.' && allowRoot) {
			return normalized;
		}
		throw new Error('Use a path inside the workspace folder.');
	}

	if (!normalized || normalized.includes(':')) {
		throw new Error('Workspace paths must be non-empty relative paths.');
	}

	return normalized;
}

export function workspacePathToLocator(rootPath: string, value: string, allowRoot = false): string {
	if (!isAbsoluteFilesystemPath(value.trim())) {
		return normalizeWorkspaceLocator(value, allowRoot);
	}

	const root = cleanFilesystemPath(rootPath);
	const candidate = cleanFilesystemPath(value);
	const caseInsensitive = /^[A-Za-z]:\//.test(root);
	const comparableRoot = caseInsensitive ? root.toLowerCase() : root;
	const comparableCandidate = caseInsensitive ? candidate.toLowerCase() : candidate;

	if (comparableCandidate === comparableRoot) {
		if (allowRoot) {
			return '.';
		}
		throw new Error('Choose a file or folder inside the workspace, not the workspace folder itself.');
	}

	const prefix = comparableRoot === '/' ? '/' : `${comparableRoot}/`;
	if (!comparableCandidate.startsWith(prefix)) {
		throw new Error('Selected paths must be inside the workspace folder.');
	}

	return normalizeWorkspaceLocator(candidate.slice(root.length + (root === '/' ? 0 : 1)), allowRoot);
}

export function workspaceLocatorToPath(rootPath: string | undefined, locator: string): string {
	if (!rootPath) {
		return locator;
	}

	const separator = rootPath.includes('\\') ? '\\' : '/';
	const root = rootPath.replace(/[\\/]+$/, '');
	const relative = locator.replaceAll('/', separator);
	return root ? `${root}${separator}${relative}` : `${separator}${relative}`;
}

export function sortWorkspaces(workspaces: WorkspaceView[]): WorkspaceView[] {
	return [...workspaces].toSorted((left, right) => {
		if (left.enabled !== right.enabled) {
			return left.enabled ? -1 : 1;
		}

		const displayNameOrder = left.displayName.localeCompare(right.displayName, undefined, {
			sensitivity: 'base',
		});
		return displayNameOrder !== 0
			? displayNameOrder
			: left.workspace.collectionID.localeCompare(right.workspace.collectionID);
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
		workspace.mode,
		workspace.primaryPath,
		workspace.workspace.rootID,
		workspace.workspace.collectionID,
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
		haystackParts.push(attachment.path, attachment.sourceDisplayName, attachment.sourceKind, attachment.role);
	}

	const haystack = haystackParts.filter(Boolean).join('\n').toLowerCase();

	return haystack.includes(query);
}

export function getWorkspaceArtifacts(catalog: WorkspaceCatalogView): WorkspaceArtifactView[] {
	const artifacts = new Map<string, WorkspaceArtifactView>();

	for (const resource of catalog.resources) {
		artifacts.set(resource.artifact.artifact.artifactID, resource.artifact);
	}

	for (const artifact of catalog.unresolvedArtifacts) {
		artifacts.set(artifact.artifact.artifactID, artifact);
	}

	return [...artifacts.values()].toSorted((left, right) => {
		const kindOrder = left.kind.localeCompare(right.kind);
		if (kindOrder !== 0) {
			return kindOrder;
		}

		const nameOrder = left.name.localeCompare(right.name, undefined, {
			sensitivity: 'base',
		});
		if (nameOrder !== 0) {
			return nameOrder;
		}

		return left.artifact.artifactID.localeCompare(right.artifact.artifactID);
	});
}

export function replaceWorkspaceArtifact(
	data: WorkspaceCatalogData,
	nextArtifact: WorkspaceArtifactView
): WorkspaceCatalogData {
	const artifactID = nextArtifact.artifact.artifactID;
	const replace = (artifact: WorkspaceArtifactView): WorkspaceArtifactView =>
		artifact.artifact.artifactID === artifactID ? nextArtifact : artifact;

	return {
		...data,
		catalog: {
			...data.catalog,
			resources: data.catalog.resources.map(resource =>
				resource.artifact.artifact.artifactID === artifactID
					? {
							...resource,
							artifact: nextArtifact,
							definitionDigest: nextArtifact.resolvedDefinition ?? resource.definitionDigest,
						}
					: resource
			),
			groups: data.catalog.groups.map(group => ({
				...group,
				resources: group.resources.map(resource =>
					resource.artifact.artifact.artifactID === artifactID
						? {
								...resource,
								artifact: nextArtifact,
								definitionDigest: nextArtifact.resolvedDefinition ?? resource.definitionDigest,
							}
						: resource
				),
			})),
			unresolvedArtifacts: data.catalog.unresolvedArtifacts.map(replace),
		},
		contexts: data.contexts.map(context =>
			context.artifact.artifactID === artifactID
				? {
						...context,
						recordRevision: nextArtifact.revision,
						definitionDigest: nextArtifact.resolvedDefinition ?? context.definitionDigest,
						enabled: nextArtifact.enabled,
						state: nextArtifact.state,
						runtimeDisabled: nextArtifact.runtimeDisabled,
						diagnostics: nextArtifact.diagnostics,
					}
				: context
		),
		skills: data.skills.map(skill =>
			skill.artifact.artifactID === artifactID
				? {
						...skill,
						recordRevision: nextArtifact.revision,
						definitionDigest: nextArtifact.resolvedDefinition ?? skill.definitionDigest,
						state: nextArtifact.state,
						runtimeDisabled: nextArtifact.runtimeDisabled,
						diagnostics: nextArtifact.diagnostics,
						skill: {
							...skill.skill,
							isEnabled: nextArtifact.enabled,
						},
					}
				: skill
		),
	};
}

export function removeWorkspaceArtifact(data: WorkspaceCatalogData, artifactID: string): WorkspaceCatalogData {
	return {
		...data,
		catalog: {
			...data.catalog,
			resources: data.catalog.resources.filter(resource => resource.artifact.artifact.artifactID !== artifactID),
			groups: data.catalog.groups.map(group => ({
				...group,
				resources: group.resources.filter(resource => resource.artifact.artifact.artifactID !== artifactID),
			})),
			unresolvedArtifacts: data.catalog.unresolvedArtifacts.filter(
				artifact => artifact.artifact.artifactID !== artifactID
			),
			unresolvedArtifactCount: Math.max(
				0,
				data.catalog.unresolvedArtifactCount -
					(data.catalog.unresolvedArtifacts.some(artifact => artifact.artifact.artifactID === artifactID) ? 1 : 0)
			),
		},
		contexts: data.contexts.filter(context => context.artifact.artifactID !== artifactID),
		skills: data.skills.filter(skill => skill.artifact.artifactID !== artifactID),
	};
}

export function getArtifactStateTone(state: ArtifactState): StatusTone {
	switch (state) {
		case ArtifactStateValue.Available:
			return 'success';
		case ArtifactStateValue.Missing:
			return 'warning';
		case ArtifactStateValue.Invalid:
		case ArtifactStateValue.Incompatible:
			return 'error';
		default:
			return 'neutral';
	}
}

export function getOccurrenceStateTone(occurrence: WorkspaceOccurrenceView): StatusTone {
	switch (occurrence.state) {
		case ArtifactOccurrenceStateValue.Valid:
			return 'success';
		case ArtifactOccurrenceStateValue.Missing:
			return 'warning';
		case ArtifactOccurrenceStateValue.Invalid:
			return 'error';
		default:
			return 'neutral';
	}
}

export function getDiagnosticTone(diagnostic: ArtifactDiagnostic): StatusTone {
	switch (diagnostic.severity) {
		case ArtifactDiagnosticSeverity.Error:
			return 'error';
		case ArtifactDiagnosticSeverity.Warning:
			return 'warning';
		case ArtifactDiagnosticSeverity.Info:
			return 'info';
		default:
			return 'neutral';
	}
}

export function getArtifactKindLabel(kind: ArtifactKind): string {
	switch (kind) {
		case WORKSPACE_CONTEXT_ARTIFACT_KIND:
			return 'Context';
		case WORKSPACE_SKILL_ARTIFACT_KIND:
			return 'Skill';
		default:
			return kind;
	}
}

export function collectWorkspaceDiagnostics(data: WorkspaceCatalogData): ArtifactDiagnostic[] {
	const diagnostics: ArtifactDiagnostic[] = [];
	const seen = new Set<string>();

	const add = (items?: ArtifactDiagnostic[]) => {
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
		add(resource.artifact.diagnostics);
	}

	for (const artifact of data.catalog.unresolvedArtifacts) {
		add(artifact.diagnostics);
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

export function workspaceArtifactMatchesSearch(artifact: WorkspaceArtifactView, rawQuery: string): boolean {
	const query = rawQuery.trim().toLowerCase();
	if (!query) {
		return true;
	}

	return [
		artifact.name,
		artifact.kind,
		artifact.state,
		artifact.locator,
		artifact.sourceID,
		artifact.subresourceLocator,
		artifact.adoption,
		artifact.resolvedDefinition,
		artifact.artifact.artifactID,
		...(artifact.diagnostics ?? []).flatMap(diagnostic => [diagnostic.code, diagnostic.message]),
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
