import type { AgentView } from '@/spec/agent';
import type { ArtifactRef, CapabilityPlan } from '@/spec/artifact';
import type { CollectionCapabilityPlan, CollectionListItem, CollectionView } from '@/spec/collection';
import type { MCPPolicyListItem, MCPServerListItem } from '@/spec/mcp';
import type { StoreSkillListItem } from '@/spec/skill';
import type { ToolStoreListItem } from '@/spec/tool';
import type { WorkspaceDirectoryListItem } from '@/spec/workspace';
import { ArtifactState } from '@/spec/artifact';

import {
	enumFromWails,
	optionalWailsString,
	requiredObject,
	requireWailsBoolean,
	requireWailsFiniteNumber,
	requireWailsString,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';

function artifactRefFromWails(value: unknown, field: string): ArtifactRef {
	const ref = requiredObject<Record<string, unknown>>(value, field);

	return {
		rootID: requireWailsString(ref.rootID, `${field}.rootID`),
		artifactID: requireWailsString(ref.artifactID, `${field}.artifactID`),
	};
}

function artifactListFields(value: Record<string, unknown>, field: string) {
	return {
		ref: artifactRefFromWails(value.ref, `${field}.ref`),
		name: requireWailsString(value.name, `${field}.name`),
		displayName: requireWailsString(value.displayName, `${field}.displayName`),
		description: optionalWailsString(value.description, `${field}.description`),
		state: enumFromWails(value.state, ArtifactState, `${field}.state`),
		enabled: requireWailsBoolean(value.enabled, `${field}.enabled`),
		revision: requireWailsFiniteNumber(value.revision, `${field}.revision`),
		definitionDigest: optionalWailsString(value.definitionDigest, `${field}.definitionDigest`),
		builtIn: requireWailsBoolean(value.builtIn, `${field}.builtIn`),
	};
}

export function collectionViewFromWails(value: unknown, operation: string): CollectionView {
	const view = requiredObject<CollectionView>(value, operation);

	return {
		...view,
		members: wailsObjectArrayOrEmpty<CollectionView['members'][number]>(view.members, `${operation}.members`),
		entries: wailsObjectArrayOrEmpty<CollectionView['entries'][number]>(view.entries, `${operation}.entries`),
	};
}

// oxlint-disable-next-line typescript/no-unnecessary-type-parameters
export function collectionResultFromWails<T extends { collection: CollectionView }>(
	value: unknown,
	operation: string
): T {
	const result = requiredObject<T>(value, operation);

	return {
		...result,
		collection: collectionViewFromWails(result.collection, `${operation}.collection`),
	} as T;
}

export function capabilityPlanFromWails(value: unknown, operation: string): CapabilityPlan {
	const plan = requiredObject<CapabilityPlan>(value, operation);

	return {
		...plan,
		occurrences: wailsObjectArrayOrEmpty<CapabilityPlan['occurrences'][number]>(
			plan.occurrences,
			`${operation}.occurrences`
		),
	};
}

export function collectionCapabilityPlanFromWails(value: unknown, operation: string): CollectionCapabilityPlan {
	const plan = requiredObject<CollectionCapabilityPlan>(value, operation);

	return {
		...plan,
		collection: collectionViewFromWails(plan.collection, `${operation}.collection`),
		occurrences: wailsObjectArrayOrEmpty<CollectionCapabilityPlan['occurrences'][number]>(
			plan.occurrences,
			`${operation}.occurrences`
		),
	};
}

export function collectionListItemFromWails(value: unknown, field: string): CollectionListItem {
	const item = requiredObject<Record<string, unknown>>(value, field);

	return {
		...artifactListFields(item, field),
		sourceID: requireWailsString(item.sourceID, `${field}.sourceID`),
		memberCount: requireWailsFiniteNumber(item.memberCount, `${field}.memberCount`),
		editable: requireWailsBoolean(item.editable, `${field}.editable`),
		deletable: requireWailsBoolean(item.deletable, `${field}.deletable`),
		baseline: requireWailsBoolean(item.baseline, `${field}.baseline`),
	};
}

export function agentViewFromWails(value: unknown, field: string): AgentView {
	const item = requiredObject<Record<string, unknown>>(value, field);

	return {
		...artifactListFields(item, field),
		managed: requireWailsBoolean(item.managed, `${field}.managed`),
	};
}

export function toolStoreListItemFromWails(value: unknown, field: string): ToolStoreListItem {
	const item = requiredObject<Record<string, unknown>>(value, field);

	return artifactListFields(item, field);
}

export function storeSkillListItemFromWails(value: unknown, field: string): StoreSkillListItem {
	const item = requiredObject<Record<string, unknown>>(value, field);

	return {
		...artifactListFields(item, field),
		managed: requireWailsBoolean(item.managed, `${field}.managed`),
	};
}

export function mcpServerListItemFromWails(value: unknown, field: string): MCPServerListItem {
	const item = requiredObject<Record<string, unknown>>(value, field);

	return artifactListFields(item, field);
}

export function mcpPolicyListItemFromWails(value: unknown, field: string): MCPPolicyListItem {
	const item = requiredObject<Record<string, unknown>>(value, field);

	return artifactListFields(item, field);
}

export function workspaceDirectoryListItemFromWails(value: unknown, field: string): WorkspaceDirectoryListItem {
	const item = requiredObject<Record<string, unknown>>(value, field);
	const rootID = requireWailsString(item.rootID, `${field}.rootID`);

	return {
		ref: {
			rootID: requireWailsString(
				requiredObject<Record<string, unknown>>(item.ref, `${field}.ref`).rootID,
				`${field}.ref.rootID`
			),
		},
		rootID,
		rootDisplayName: requireWailsString(item.rootDisplayName, `${field}.rootDisplayName`),
		enabled: requireWailsBoolean(item.enabled, `${field}.enabled`),
		directorySourceID: requireWailsString(item.directorySourceID, `${field}.directorySourceID`),
		directorySourceRevision: requireWailsFiniteNumber(item.directorySourceRevision, `${field}.directorySourceRevision`),
		policyID: requireWailsString(item.policyID, `${field}.policyID`),
		policyVersion: requireWailsString(item.policyVersion, `${field}.policyVersion`),
		policyDigest: requireWailsString(item.policyDigest, `${field}.policyDigest`),
	};
}
