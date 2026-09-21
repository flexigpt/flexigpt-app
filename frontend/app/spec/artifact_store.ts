import type {
	ArtifactDiagnostic,
	ArtifactDigest,
	ArtifactKind,
	ArtifactLocator,
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceID,
	ArtifactState,
	ArtifactStorageKey,
} from '@/spec/artifact';

export interface StoreArtifactBinding {
	sourceID: ArtifactSourceID;
	locator: ArtifactLocator;
	subresourceLocator?: ArtifactLocator;
}

export interface StoreArtifact {
	id: string;
	rootID: ArtifactRootID;
	binding: StoreArtifactBinding;
	kind: ArtifactKind;
	logicalName: string;
	logicalVersion?: string;
	resolvedDefinition?: ArtifactDigest;
	sourceContentDigest?: ArtifactDigest;
	state: ArtifactState;
	diagnostics?: ArtifactDiagnostic[];
	displayName: string;
	enabled: boolean;
	revision: number;
	createdAt: string;
	modifiedAt: string;
}

export interface StoreArtifactAddress {
	rootID: ArtifactRootID;
	artifactID: string;
	kind: ArtifactKind;
	logicalName: string;
}

export interface StoreArtifactDefinition {
	digest: ArtifactDigest;
	kind: ArtifactKind;
	schemaID: string;
	schemaVersion: string;
	logicalName: string;
	logicalVersion?: string;
	displayName?: string;
	description?: string;
	labels?: Record<string, string>;
	body: number[];
	dependencies?: StoreArtifactDefinitionSelector[];
}

export interface StoreArtifactDefinitionSelector {
	kind: ArtifactKind;
	logicalName?: string;
	versionConstraint?: string;
	labels?: Record<string, string>;
}

export interface StoreDirectoryRoot {
	root: ArtifactLocator;
	recursive?: boolean;
	includePatterns?: string[];
	excludePatterns?: string[];
}

export interface StoreDecoderHint {
	locator: ArtifactLocator;
	recursive?: boolean;
	decoderIDs: string[];
}

export interface StoreDiscoverySpec {
	explicitLocators?: ArtifactLocator[];
	directoryRoots?: StoreDirectoryRoot[];
	decoderHints?: StoreDecoderHint[];
	allowedDecoderIDs?: string[];
	expectedContentDigests?: Record<string, string>;
	authoritative?: boolean;
	maxCandidateBytes?: number;
	maxTotalBytes?: number;
	maxCandidates?: number;
	maxEntries?: number;
	maxDepth?: number;
}

export interface StoreArtifactSourceSummary {
	id: ArtifactSourceID;
	rootID: ArtifactRootID;
	rootStorageKey: ArtifactStorageKey;
	storageKey: ArtifactStorageKey;
	kind: string;
	displayName: string;
	enabled: boolean;
	discovery: StoreDiscoverySpec;
	revision: number;
	createdAt: string;
	modifiedAt: string;
	retiredAt?: string;
}

export interface StoreArtifactRoot {
	id: ArtifactRootID;
	storageKey: ArtifactStorageKey;
	displayName: string;
	description?: string;
	revision: number;
	createdAt: string;
	modifiedAt: string;
	retiredAt?: string;
}

export interface StoreArtifactRootDraft {
	id: ArtifactRootID;
	storageKey: ArtifactStorageKey;
	displayName: string;
	description?: string;
}

export interface StoreManagedPackageFile {
	locator: ArtifactLocator;
	content: number[];
}

export interface StoreArtifactResolution {
	artifact: ArtifactRef;
	definition?: StoreArtifactDefinition;
}
