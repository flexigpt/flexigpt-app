import type { JSONRawString } from '@/lib/jsonschema_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';

export type ArtifactRootID = string;
export type ArtifactSourceID = string;
export type ArtifactCollectionID = string;
export type ArtifactID = string;
export type ArtifactStorageKey = string;

/**
 * Creates an opaque, storage-safe local key for a newly allocated Artifact
 * Store Root or Source. This is intentionally not a user-facing identifier,
 * bundle key, filesystem path, or portable package address.
 */
export function newArtifactStorageKey(): ArtifactStorageKey {
	return `s${getUUIDv7().replaceAll('-', '').toLowerCase()}`;
}

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
	storageKey: ArtifactStorageKey;
	displayName: string;
	description?: string;
	revision: number;
	createdAt: string;
	modifiedAt: string;
	retiredAt?: string;
}

export interface ArtifactCollection {
	id: ArtifactCollectionID;
	rootID: ArtifactRootID;
	kind: ArtifactCollectionKind;
	displayName: string;
	description?: string;
	enabled: boolean;
	revision: number;
	createdAt: string;
	modifiedAt: string;
	retiredAt?: string;
}

export interface ArtifactCollectionAttachment {
	rootID: ArtifactRootID;
	collectionID: ArtifactCollectionID;
	sourceID: ArtifactSourceID;
	role: ArtifactAttachmentRole;
	enabled: boolean;
	revision: number;
	createdAt: string;
	modifiedAt: string;
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
	createdAt: string;
	modifiedAt: string;
}

export interface CreateArtifactRootBody {
	id: ArtifactRootID;
	storageKey: ArtifactStorageKey;
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
	rootStorageKey: ArtifactStorageKey;
	storageKey: ArtifactStorageKey;
	kind: ArtifactSourceKind;
	displayName: string;
	enabled: boolean;
	revision: number;
	createdAt: string;
	modifiedAt: string;
	retiredAt?: string;
}

export interface ArtifactSourceDraft {
	id: ArtifactSourceID;
	storageKey: ArtifactStorageKey;
	kind: ArtifactSourceKind;
	displayName: string;
	enabled: boolean;
	config: JSONRawString;
}

export interface UpdateArtifactSourceBody {
	expectedRevision: number;
	displayName: string;
	enabled: boolean;

	/**
	 * Omitting config preserves the private existing source configuration.
	 * Providing config replaces it atomically after adapter normalization.
	 */
	config?: JSONRawString;
}

/**
 * Semantic identity of a managed package. This replaces physical managed
 * source directory addressing at the API boundary.
 */
export interface ManagedPackageAddress {
	kind: string;
	name: string;
	version: string;
}

export interface ManagedSourcePackageFile {
	locator: ArtifactLocator;
	content: Uint8Array;
}

export interface ArtifactSourceBinding {
	sourceID: ArtifactSourceID;
	locator: ArtifactLocator;
	subresourceLocator?: ArtifactLocator;
	expectedKind: ArtifactKind;
}

interface ArtifactDefinitionSelector {
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
