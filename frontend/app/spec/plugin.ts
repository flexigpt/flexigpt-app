import type {
	ArtifactKind,
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceID,
	ArtifactState,
	CapabilityOccurrence,
	CapabilityResolutionStatus,
	CapabilityTarget,
	StoreArtifact,
} from '@/spec/artifact';

export enum PluginMembershipMode {
	SingleType = 'single-type',
	MixedType = 'mixed-type',
	LegacyUnconstrained = 'legacy-unconstrained',
}

export enum PluginMemberForm {
	Named = 'named',
	Contained = 'contained',
	Selector = 'selector',
}

export interface PluginMembershipPolicy {
	mode: PluginMembershipMode;
	allowedTypes?: ArtifactKind[];
	allowedForms?: PluginMemberForm[];
}

interface PluginDeclarationLocator {
	kind?: string;
	path?: string;
	url?: string;
	integrity?: string;
	repository?: string;
	revision?: string;
	manager?: string;
	package?: string;
	version?: string;
	registry?: string;
	command?: string;
}

interface PluginMemberReference {
	type: ArtifactKind;
	name: string;
	insert?: string;
	locator?: PluginDeclarationLocator;
	scope?: string;
	server?: string;
}

interface PluginMemberView {
	type: ArtifactKind;
	name?: string;
	insert?: string;
	locator?: PluginDeclarationLocator;
	server?: string;
	contained: boolean;
	selector: boolean;
}

export interface PluginView {
	artifact: StoreArtifact;
	name: string;
	displayName: string;
	description?: string;
	members: PluginMemberReference[];
	entries: PluginMemberView[];
	editable: boolean;
	deletable: boolean;
	baseline: boolean;
	membership: PluginMembershipPolicy;
}

/**
 * Exact frontend projection of generated `plugin.ListItem`.
 *
 * Plugin list endpoints intentionally omit member declarations and Artifact
 * details. Fetch `PluginView` only for editing, membership inspection, or
 * capability resolution.
 */
export interface PluginListItem {
	ref: ArtifactRef;
	sourceID: ArtifactSourceID;
	name: string;
	displayName: string;
	description?: string;
	state: ArtifactState;
	enabled: boolean;
	revision: number;
	memberCount: number;
	builtIn: boolean;
	editable: boolean;
	deletable: boolean;
	baseline: boolean;
}

export function pluginListItemFromPluginView(plugin: PluginView, builtIn: boolean): PluginListItem {
	return {
		ref: {
			rootID: plugin.artifact.rootID,
			artifactID: plugin.artifact.id,
		},
		sourceID: plugin.artifact.binding.sourceID,
		name: plugin.name,
		displayName: plugin.displayName,
		description: plugin.description,
		state: plugin.artifact.state,
		enabled: plugin.artifact.enabled,
		revision: plugin.artifact.revision,
		memberCount: plugin.members.length,
		builtIn,
		editable: plugin.editable,
		deletable: plugin.deletable,
		baseline: plugin.baseline,
	};
}

export interface ArtifactPluginMembershipView {
	plugin: ArtifactRef;
	pluginName: string;
	pluginRevision: number;
	memberIndex: number;
	member: PluginMemberReference;
	status: CapabilityResolutionStatus;
	resolvedArtifact?: ArtifactRef;
	resolvedToArtifact: boolean;
	code?: string;
	message?: string;
}

export interface PluginCapabilityPlan {
	plugin: PluginView;
	occurrences: CapabilityOccurrence[];
	complete: boolean;
}

export interface PluginDirectSelectorMatch {
	artifact: ArtifactRef;
	status: CapabilityResolutionStatus;
	target?: CapabilityTarget;
	code?: string;
	message?: string;
}

export interface PluginDirectMember {
	index: number;
	entry: PluginMemberView;
	status: CapabilityResolutionStatus;
	target?: CapabilityTarget;
	selectorMatches?: PluginDirectSelectorMatch[];
	code?: string;
	message?: string;
}

export interface PluginDirectMembership {
	plugin: PluginView;
	members: PluginDirectMember[];
	complete: boolean;
}

export interface CreatePluginRequest {
	rootID: ArtifactRootID;
	sourceID?: ArtifactSourceID;
	name: string;
	displayName?: string;
	description?: string;
	membershipPolicy?: PluginMembershipPolicy;
}

export interface UpdatePluginRequest {
	plugin: ArtifactRef;
	expectedRevision: number;
	description?: string;
	displayName?: string;
}

export interface DeletePluginRequest {
	plugin: ArtifactRef;
	expectedRevision: number;
}

export interface AddPluginMemberRequest {
	plugin: ArtifactRef;
	expectedRevision: number;
	member: PluginMemberReference;
}

export interface AddPluginArtifactMemberRequest {
	plugin: ArtifactRef;
	expectedRevision: number;
	artifact: ArtifactRef;
}

export interface RemovePluginMemberRequest {
	plugin: ArtifactRef;
	expectedRevision: number;
	index: number;
}
