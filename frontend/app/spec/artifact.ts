export type ArtifactRootID = string;
export type ArtifactSourceID = string;
type ArtifactID = string;
type ArtifactStorageKey = string;

/**
 * Artifact kinds and Source kinds are registry-extensible backend identifiers.
 * They intentionally remain strings rather than frontend enums.
 */
export type ArtifactKind = string;

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

enum ArtifactDiagnosticSeverity {
	Error = 'error',
	Warning = 'warning',
	Info = 'info',
}

export interface ArtifactRef {
	rootID: ArtifactRootID;
	artifactID: ArtifactID;
}

export interface ArtifactAddress extends ArtifactRef {
	kind: ArtifactKind;
	logicalName: string;
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

export interface ArtifactSourceBinding {
	sourceID: ArtifactSourceID;
	locator: ArtifactLocator;
	subresourceLocator?: ArtifactLocator;
}

export interface StoreArtifact {
	id: string;
	rootID: ArtifactRootID;
	binding: ArtifactSourceBinding;
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

interface StoreDirectoryRoot {
	root: ArtifactLocator;
	recursive?: boolean;
	includePatterns?: string[];
	excludePatterns?: string[];
}

interface StoreDecoderHint {
	locator: ArtifactLocator;
	recursive?: boolean;
	decoderIDs: string[];
}

interface StoreDiscoverySpec {
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

export interface StoreManagedPackageFile {
	locator: ArtifactLocator;
	content: number[];
}

export enum CapabilityTargetForm {
	Artifact = 'artifact',
	Direct = 'direct',
}

export enum CapabilityTargetProvenance {
	CurrentRoot = 'currentRoot',
	BuiltinScope = 'builtinScope',
	CompositionSource = 'compositionSource',
	DirectCapability = 'directCapability',
}

/**
 * Exact frontend projection of generated `composition.CapabilityTarget`.
 *
 * Artifact targets preserve the source-backed Artifact identity. Direct
 * targets are runtime-owned and must not be converted into fabricated
 * ArtifactRefs.
 */
export interface CapabilityTarget {
	form: CapabilityTargetForm;
	type: ArtifactKind;
	name: string;
	provenance: CapabilityTargetProvenance;
	artifact?: ArtifactRef;
	providerIdentity?: string;
	providerLocalID?: string;
	evidence?: ArtifactDigest;
}

export enum CapabilityResolutionStatus {
	Available = 'available',
	Unavailable = 'unavailable',
	Ambiguous = 'ambiguous',
}

export interface CapabilityOccurrence {
	path: string;
	kind: string;
	type: string;
	name?: string;
	status: CapabilityResolutionStatus;
	required: boolean;
	scope?: string;
	target?: CapabilityTarget;
	overrides?: Record<string, number[]>;
	use?: Record<string, number[]>;
	code?: string;
	message?: string;
}

export interface CapabilityPlan {
	rootArtifact?: ArtifactRef;
	rootTarget?: CapabilityTarget;
	rootType: string;
	rootName: string;
	occurrences: CapabilityOccurrence[];
	complete: boolean;
}
