import type { JSONRawString } from '@/lib/jsonschema_utils';

export type ArtifactRootID = string;
export type ArtifactSourceID = string;
type ArtifactCollectionID = string;
type ArtifactID = string;
export type ArtifactKind = string;
export type ArtifactSourceKind = string;
export type ArtifactLocator = string;
export type ArtifactDigest = string;

export enum ArtifactAdoptionMode {
	Observed = 'observed',
	Pinned = 'pinned',
}

export enum ArtifactState {
	Available = 'available',
	Missing = 'missing',
	Invalid = 'invalid',
	Incompatible = 'incompatible',
}

export enum ArtifactOccurrenceState {
	Valid = 'valid',
	Invalid = 'invalid',
	Missing = 'missing',
}

export enum ArtifactDiagnosticSeverity {
	Error = 'error',
	Warning = 'warning',
	Info = 'info',
}

export interface ArtifactRef {
	rootID: ArtifactRootID;
	artifactID: ArtifactID;
}

export interface ArtifactCollectionRef {
	rootID: ArtifactRootID;
	collectionID: ArtifactCollectionID;
}

export interface PurgeArtifactRootResult {
	rootID: ArtifactRootID;
}

export interface PurgeArtifactSourceResult {
	rootID: ArtifactRootID;
	sourceID: ArtifactSourceID;
}

interface ArtifactDiagnosticLocation {
	locator?: ArtifactLocator;
	subresourceLocator?: ArtifactLocator;
	line?: number;
	column?: number;
}

export interface ArtifactDiagnostic {
	severity: ArtifactDiagnosticSeverity;
	code: string;
	message: string;
	location?: ArtifactDiagnosticLocation;
}

export interface ArtifactRoot {
	id: ArtifactRootID;
	displayName: string;
	description?: string;
	revision: number;
	createdAt: Date;
	modifiedAt: Date;
	retiredAt?: Date;
}

export interface CreateArtifactRootBody {
	displayName: string;
	description?: string;
}

export interface UpdateArtifactRootBody {
	expectedRevision: number;
	displayName: string;
	description?: string;
}

/**
 * Raw JSON object passed to an Artifact Store source adapter.
 *
 * This is write-only. Source configuration is intentionally absent from
 * source summaries returned by the backend.
 */
type ArtifactSourceConfig = JSONRawString;

export interface ArtifactSourceSummary {
	id: ArtifactSourceID;
	rootID: ArtifactRootID;
	kind: ArtifactSourceKind;
	displayName: string;
	enabled: boolean;
	revision: number;
	createdAt: Date;
	modifiedAt: Date;
	retiredAt?: Date;
}

export interface CreateArtifactSourceBody {
	kind: ArtifactSourceKind;
	displayName: string;
	enabled: boolean;
	config: ArtifactSourceConfig;
}

export interface UpdateArtifactSourceBody {
	expectedRevision: number;
	displayName: string;
	enabled: boolean;

	/**
	 * Omitting config preserves the private existing source configuration.
	 * Providing config replaces it atomically after adapter normalization.
	 */
	config?: ArtifactSourceConfig;
}

interface ManagedPackageFile {
	locator: ArtifactLocator;
	content: Uint8Array;
}

export interface PublishManagedSourcePackageBody {
	expectedSourceRevision: number;
	directory: ArtifactLocator;
	expectedGeneration?: string;
	files: ManagedPackageFile[];
}

export interface ManagedSourceState {
	generation: string;
	source: ArtifactSourceSummary;
}

export interface ManagedSourcePackageResult {
	generation: string;
	source: ArtifactSourceSummary;
}

export interface ArtifactSourceBinding {
	sourceID: ArtifactSourceID;
	locator: ArtifactLocator;
	subresourceLocator?: ArtifactLocator;
	expectedKind: ArtifactKind;
}

export interface ArtifactDefinitionSelector {
	kind: ArtifactKind;
	logicalName?: string;
	versionConstraint?: string;
	labels?: Record<string, string>;
}

export interface ArtifactDefinitionView {
	digest: ArtifactDigest;
	kind: ArtifactKind;
	schemaID: string;
	schemaVersion: string;
	logicalName: string;
	logicalVersion?: string;
	displayName?: string;
	description?: string;
	labels?: Record<string, string>;
	body: JSONRawString;
	dependencies?: ArtifactDefinitionSelector[];
}
