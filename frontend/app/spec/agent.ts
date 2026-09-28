import type {
	ArtifactDigest,
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceID,
	ArtifactState,
	MappedTarget,
} from '@/spec/artifact';
import type { CollectionView } from '@/spec/collection';
import type { MCPHTTPAuthMode, MCPInputKind, MCPTransportType } from '@/spec/mcp';

export enum AgentImportIssueSeverity {
	Error = 'error',
	Confirmation = 'confirmation',
	Warning = 'warning',
	Information = 'information',
}

export enum AgentTextInsert {
	Instructions = 'instructions',
	Warning = 'warning',
	Information = 'information',
	UserMessage = 'user-message',
}

export enum AgentImportRelationshipStatus {
	Available = 'available',
	Unavailable = 'unavailable',
	Ambiguous = 'ambiguous',
}

export enum AgentSkillUseMode {
	Available = 'available',
	Active = 'active',
	Instructions = 'instructions',
}

export interface AgentView {
	ref: ArtifactRef;
	name: string;
	displayName: string;
	description?: string;
	state: ArtifactState;
	enabled: boolean;
	revision: number;
	definitionDigest?: ArtifactDigest;
	builtIn: boolean;
	managed: boolean;
}

export interface AgentCapabilityOccurrence {
	path: string;
	type: string;
	name?: string;
	status: string;
	required: boolean;
	artifact?: ArtifactRef;
	mapped?: MappedTarget;
	autoExecute?: boolean;
	includeSystemPrompt?: boolean;
	skillUseMode?: AgentSkillUseMode;
	code?: string;
	message?: string;
}

export interface AgentCapabilityPlan {
	occurrences: AgentCapabilityOccurrence[];
	complete: boolean;
}

export interface AgentResolution {
	agent: AgentView;
	capabilities: AgentCapabilityPlan;
}

export interface AgentTextMaterialization {
	artifact: ArtifactRef;
	artifactRevision: number;
	definitionDigest: ArtifactDigest;
	name: string;
	insert: AgentTextInsert;
	mediaType?: string;
	content: string;
	locator: string;
	builtIn: boolean;
}

export interface ListAgentsRequest {
	rootID: ArtifactRootID;
	logicalNames?: string[];
	collection?: ArtifactRef;
	includeBuiltin?: boolean;
	enabled?: boolean;
}

interface AgentImportIssue {
	code: string;
	severity: AgentImportIssueSeverity;
	path?: string;
	message: string;
}

export interface AgentImportDestination {
	rootID: ArtifactRootID;
	rootDisplayName?: string;
	sourceID: ArtifactSourceID;
	collection: ArtifactRef;

	collectionRevision: number;
	collectionName: string;
	collectionDisplayName: string;
	baseline: boolean;
	enabled: boolean;
}

export interface AgentImportPreviewRequest {
	/**
	 * Transient selected input path. The backend accepts `.json`, `.yaml`,
	 * and `.yml`, selecting the parser from this extension.
	 */
	path: string;
	collection: ArtifactRef;
	expectedCollectionRevision: number;
	expectedSourceDigest?: ArtifactDigest;
}

interface AgentImportArtifactPreview {
	occurrencePath: string;
	type: string;
	name: string;
	logicalVersion?: string;
	definitionDigest: ArtifactDigest;
}

interface AgentImportRelationship {
	path: string;
	type: string;
	name: string;
	scope?: string;
	status: AgentImportRelationshipStatus;

	artifact?: ArtifactRef;
	mapped?: MappedTarget;

	code?: string;
	message?: string;
}

interface AgentImportConflict {
	code: string;
	path?: string;
	message: string;
}

interface AgentRestoredMembership {
	collection: ArtifactRef;
	path: string;
	message: string;
}

interface AgentMCPSetupInput {
	name: string;
	kind: MCPInputKind;
	label?: string;
	description?: string;
	required: boolean;
	clientSecretRequired: boolean;
}

export interface AgentMCPSetupDescriptor {
	occurrencePath: string;
	name: string;
	artifact?: ArtifactRef;

	transport?: MCPTransportType;
	command?: string;
	url?: string;
	authMode?: MCPHTTPAuthMode;

	inputs?: AgentMCPSetupInput[];
}

export interface AgentImportPreview {
	prepared?: string;
	preparedFingerprint?: ArtifactDigest;
	expiresAt?: string;

	sourceDigest?: ArtifactDigest;
	definitionDigest?: ArtifactDigest;

	/**
	 * Canonical YAML output for either JSON or YAML input.
	 */
	normalizedYAML?: string;

	agent?: AgentImportArtifactPreview;
	destination: AgentImportDestination;
	projectedArtifacts?: AgentImportArtifactPreview[];
	relationships?: AgentImportRelationship[];
	conflicts?: AgentImportConflict[];
	restoredMemberships?: AgentRestoredMembership[];
	mcpSetupDescriptors?: AgentMCPSetupDescriptor[];

	canImport: boolean;
	requiresConfirmation: boolean;
	requiredConfirmationCodes?: string[];
	issues?: AgentImportIssue[];
}

export interface AgentImportCommitRequest {
	prepared: string;
	preparedFingerprint: ArtifactDigest;
	acceptedConfirmationCodes?: string[];
}

export interface AgentImportCommitResult {
	agent: AgentView;
	collection: CollectionView;
	restoredMemberships?: AgentRestoredMembership[];
	mcpSetupDescriptors?: AgentMCPSetupDescriptor[];
	preparedFingerprint: ArtifactDigest;
}

export interface AgentExportResult {
	type: string;
	name: string;
	mediaType: string;
	suggestedFileName: string;
	content: string;
	contentDigest: ArtifactDigest;
	definitionDigest: ArtifactDigest;
	artifactRevision: number;
	builtIn: boolean;
	managed: boolean;
}
