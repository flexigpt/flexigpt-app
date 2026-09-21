import type { ArtifactRef } from '@/spec/artifact';
import type { StoreArtifact, StoreArtifactDefinition } from '@/spec/artifact_store';

export interface MappedTarget {
	provider: string;
	identifier: string;
	type: string;
	name: string;
	builtin: boolean;
}

export interface CapabilityOccurrence {
	path: string;
	kind: string;
	type: string;
	name?: string;
	status: string;
	required: boolean;
	scope?: string;
	artifact?: ArtifactRef;
	mapped?: MappedTarget;
	overrides?: Record<string, number[]>;
	use?: Record<string, number[]>;
	code?: string;
	message?: string;
}

export interface CapabilityPlan {
	rootArtifact?: ArtifactRef;
	rootMapped?: MappedTarget;
	rootType: string;
	rootName: string;
	occurrences: CapabilityOccurrence[];
	complete: boolean;
}

export interface ResolutionIssue {
	code: string;
	message: string;
}

export interface ResolvedSelectorMatch {
	artifact: ArtifactRef;
	status: string;
	resolved?: ResolvedEntry;
	issue?: ResolutionIssue;
}

export interface ResolvedSelector {
	type: string;
	base: string;
	matches: ResolvedSelectorMatch[];
}

export interface ResolvedRelationship {
	declared: unknown;
	form: string;
	status: string;
	required: boolean;
	scope?: string;
	overrides?: Record<string, number[]>;
	use?: Record<string, number[]>;
	resolved?: ResolvedEntry;
	selector?: ResolvedSelector;
	issue?: ResolutionIssue;
}

export interface ResolvedWorkspace {
	members: ResolvedEntry[];
	memberResults: ResolvedRelationship[];
}

export interface ResolvedMCP {
	policy?: ResolvedEntry;
	policyResult?: ResolvedRelationship;
	policyRequired: boolean;
}

export interface ResolvedLoop {
	bodyResult?: ResolvedRelationship;
	body?: ResolvedEntry;
	maxIterations?: number;
	until?: unknown;
}

export interface ResolvedWorkflowNode {
	id: string;
	join: string;
	member?: ResolvedEntry;
	memberResult?: ResolvedRelationship;
}

export interface ResolvedWorkflowEdge {
	from: string;
	to: string;
	match?: unknown;
}

export interface ResolvedWorkflow {
	start: string[];
	nodes: ResolvedWorkflowNode[];
	edges: ResolvedWorkflowEdge[];
}

export interface ResolvedEntry {
	type: string;
	DeclarationOrigin?: StoreArtifact;
	artifact?: StoreArtifact;
	definition?: StoreArtifactDefinition;
	mapped?: MappedTarget;
	members?: ResolvedEntry[];
	memberResults?: ResolvedRelationship[];
	allowedTools?: ResolvedEntry[];
	allowedToolResults?: ResolvedRelationship[];
	loop?: ResolvedEntry;
	loopResult?: ResolvedRelationship;
	workflow?: ResolvedEntry;
	workflowResult?: ResolvedRelationship;
	loopState?: ResolvedLoop;
	workflowState?: ResolvedWorkflow;
	workspace?: ResolvedWorkspace;
	mcp?: ResolvedMCP;
}
