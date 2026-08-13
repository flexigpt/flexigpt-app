import type { JSONRawString } from '@/lib/jsonschema_utils';

export type ArtifactRootID = string;
export type ArtifactSourceID = string;
export type ArtifactCollectionID = string;
export type ArtifactID = string;

/**
 * Artifact kinds and Source kinds are registry-extensible backend identifiers.
 * They intentionally remain strings rather than frontend enums.
 */
export type ArtifactKind = string;
type ArtifactCollectionKind = string;
export type ArtifactSourceKind = string;
type ArtifactAttachmentRole = string;
export type ArtifactLocator = string;
export type ArtifactDigest = string;
type ArtifactSourceGeneration = string;

/**
 * Write-only adapter configuration. It is never returned by Source APIs.
 */
type ArtifactSourceConfig = JSONRawString;

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

export interface ArtifactAddress extends ArtifactRef {
	collectionID: ArtifactCollectionID;
	kind: ArtifactKind;
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

export interface ArtifactCollection {
	id: ArtifactCollectionID;
	rootID: ArtifactRootID;
	kind: ArtifactCollectionKind;
	displayName: string;
	description?: string;
	enabled: boolean;
	revision: number;
	createdAt: Date;
	modifiedAt: Date;
	retiredAt?: Date;
}

export interface ArtifactCollectionAttachment {
	rootID: ArtifactRootID;
	collectionID: ArtifactCollectionID;
	sourceID: ArtifactSourceID;
	role: ArtifactAttachmentRole;
	enabled: boolean;
	revision: number;
	createdAt: Date;
	modifiedAt: Date;
}

export interface ArtifactRecord {
	id: ArtifactID;
	rootID: ArtifactRootID;
	collectionID: ArtifactCollectionID;
	binding: ArtifactSourceBinding;
	kind: ArtifactKind;
	name: string;
	enabled: boolean;
	adoption: ArtifactAdoptionMode;
	resolvedDefinition?: ArtifactDigest;
	state: ArtifactState;
	diagnostics?: ArtifactDiagnostic[];
	revision: number;
	createdAt: Date;
	modifiedAt: Date;
}

export interface CreateArtifactRootBody {
	id: ArtifactRootID;
	displayName: string;
	description?: string;
}

export interface UpdateArtifactRootBody {
	expectedRevision: number;
	displayName: string;
	description?: string;
}

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
	id: ArtifactSourceID;
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

export interface ManagedSourcePackageFile {
	locator: ArtifactLocator;
	content: Uint8Array;
}

export interface PublishManagedSourcePackageBody {
	expectedSourceRevision: number;
	directory: ArtifactLocator;
	expectedGeneration?: ArtifactSourceGeneration;
	files: ManagedSourcePackageFile[];
}

export interface RemoveManagedSourcePackageBody {
	expectedSourceRevision: number;
	directory: ArtifactLocator;
	expectedGeneration: ArtifactSourceGeneration;
}

export interface ManagedSourceState {
	generation: ArtifactSourceGeneration;
	source: ArtifactSourceSummary;
}

export interface ManagedSourcePackageResult {
	generation: ArtifactSourceGeneration;
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
