// oxlint-disable typescript/no-unnecessary-type-parameters
import type { AgentView } from '@/spec/agent';
import type {
	ArtifactKind,
	ArtifactRef,
	CapabilityOccurrence,
	CapabilityPlan,
	CapabilityTarget,
} from '@/spec/artifact';
import type { MCPPolicyListItem, MCPServerListItem } from '@/spec/mcp';
import type {
	ArtifactPluginMembershipView,
	PluginCapabilityPlan,
	PluginDirectMember,
	PluginDirectMembership,
	PluginDirectSelectorMatch,
	PluginListItem,
	PluginMemberForm,
	PluginMembershipMode,
	PluginMembershipPolicy,
	PluginView,
} from '@/spec/plugin';
import type { StoreSkillListItem } from '@/spec/skill';
import type { ToolStoreListItem } from '@/spec/tool';
import type { WorkspaceDirectoryListItem } from '@/spec/workspace';
import {
	ArtifactState,
	CapabilityResolutionStatus,
	CapabilityTargetForm,
	CapabilityTargetProvenance,
} from '@/spec/artifact';
import {
	PluginMemberForm as PluginMemberFormValue,
	PluginMembershipMode as PluginMembershipModeValue,
} from '@/spec/plugin';

import {
	enumFromWails,
	optionalWailsString,
	requiredObject,
	requireWailsBoolean,
	requireWailsFiniteNumber,
	requireWailsString,
	wailsArrayOrEmpty,
	wailsObjectArrayOrEmpty,
	wailsRecordOrEmpty,
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

export function capabilityTargetFromWails(value: unknown, field: string): CapabilityTarget {
	const target = requiredObject<Record<string, unknown>>(value, field);
	const form = enumFromWails(target.form, CapabilityTargetForm, `${field}.form`);
	const provenance = enumFromWails(target.provenance, CapabilityTargetProvenance, `${field}.provenance`);

	const output: CapabilityTarget = {
		form,
		type: requireWailsString(target.type, `${field}.type`) as ArtifactKind,
		name: requireWailsString(target.name, `${field}.name`),
		provenance,
	};

	if (form === CapabilityTargetForm.Artifact) {
		output.artifact = artifactRefFromWails(target.artifact, `${field}.artifact`);
		return output;
	}

	output.providerIdentity = requireWailsString(target.providerIdentity, `${field}.providerIdentity`);
	output.providerLocalID = requireWailsString(target.providerLocalID, `${field}.providerLocalID`);
	output.evidence = requireWailsString(target.evidence, `${field}.evidence`);

	return output;
}

function capabilityOccurrenceFromWails(value: unknown, field: string): CapabilityOccurrence {
	const occurrence = requiredObject<Record<string, unknown>>(value, field);
	const target =
		occurrence.target === null || occurrence.target === undefined
			? undefined
			: capabilityTargetFromWails(occurrence.target, `${field}.target`);

	return {
		path: requireWailsString(occurrence.path, `${field}.path`),
		kind: requireWailsString(occurrence.kind, `${field}.kind`),
		type: requireWailsString(occurrence.type, `${field}.type`),
		name: optionalWailsString(occurrence.name, `${field}.name`),
		status: enumFromWails(occurrence.status, CapabilityResolutionStatus, `${field}.status`),
		required: requireWailsBoolean(occurrence.required, `${field}.required`),
		scope: optionalWailsString(occurrence.scope, `${field}.scope`),
		target,
		overrides:
			occurrence.overrides === null || occurrence.overrides === undefined
				? undefined
				: wailsRecordOrEmpty<number[]>(occurrence.overrides, `${field}.overrides`),
		use:
			occurrence.use === null || occurrence.use === undefined
				? undefined
				: wailsRecordOrEmpty<number[]>(occurrence.use, `${field}.use`),
		code: optionalWailsString(occurrence.code, `${field}.code`),
		message: optionalWailsString(occurrence.message, `${field}.message`),
	};
}

export function capabilityPlanFromWails(value: unknown, operation: string): CapabilityPlan {
	const plan = requiredObject<Record<string, unknown>>(value, operation);

	return {
		rootArtifact:
			plan.rootArtifact === null || plan.rootArtifact === undefined
				? undefined
				: artifactRefFromWails(plan.rootArtifact, `${operation}.rootArtifact`),
		rootTarget:
			plan.rootTarget === null || plan.rootTarget === undefined
				? undefined
				: capabilityTargetFromWails(plan.rootTarget, `${operation}.rootTarget`),
		rootType: requireWailsString(plan.rootType, `${operation}.rootType`),
		rootName: requireWailsString(plan.rootName, `${operation}.rootName`),
		occurrences: wailsObjectArrayOrEmpty(plan.occurrences, `${operation}.occurrences`).map((item, index) =>
			capabilityOccurrenceFromWails(item, `${operation}.occurrences[${index}]`)
		),
		complete: requireWailsBoolean(plan.complete, `${operation}.complete`),
	};
}

function pluginMembershipFromWails(value: unknown, field: string): PluginMembershipPolicy {
	const membership = requiredObject<Record<string, unknown>>(value, field);

	return {
		mode: enumFromWails(membership.mode, PluginMembershipModeValue, `${field}.mode`) as PluginMembershipMode,
		allowedTypes: wailsArrayOrEmpty<string>(membership.allowedTypes, `${field}.allowedTypes`) as ArtifactKind[],
		allowedForms: wailsArrayOrEmpty<string>(membership.allowedForms, `${field}.allowedForms`).map(
			(form, index) => enumFromWails(form, PluginMemberFormValue, `${field}.allowedForms[${index}]`) as PluginMemberForm
		),
	};
}

export function pluginViewFromWails(value: unknown, operation: string): PluginView {
	const view = requiredObject<Record<string, unknown>>(value, operation);

	return {
		artifact: requiredObject(view.artifact, `${operation}.artifact`),
		name: requireWailsString(view.name, `${operation}.name`),
		displayName: requireWailsString(view.displayName, `${operation}.displayName`),
		description: optionalWailsString(view.description, `${operation}.description`),
		members: wailsObjectArrayOrEmpty(view.members, `${operation}.members`),
		entries: wailsObjectArrayOrEmpty(view.entries, `${operation}.entries`),
		editable: requireWailsBoolean(view.editable, `${operation}.editable`),
		deletable: requireWailsBoolean(view.deletable, `${operation}.deletable`),
		baseline: requireWailsBoolean(view.baseline, `${operation}.baseline`),
		membership: pluginMembershipFromWails(view.membership, `${operation}.membership`),
	};
}

export function pluginResultFromWails<T extends { plugin: PluginView }>(value: unknown, operation: string): T {
	const result = requiredObject<T>(value, operation);

	return {
		...result,
		plugin: pluginViewFromWails(result.plugin, `${operation}.plugin`),
	} as T;
}

export function pluginCapabilityPlanFromWails(value: unknown, operation: string): PluginCapabilityPlan {
	const plan = requiredObject<Record<string, unknown>>(value, operation);

	return {
		plugin: pluginViewFromWails(plan.plugin, `${operation}.plugin`),
		occurrences: wailsObjectArrayOrEmpty(plan.occurrences, `${operation}.occurrences`).map((item, index) =>
			capabilityOccurrenceFromWails(item, `${operation}.occurrences[${index}]`)
		),
		complete: requireWailsBoolean(plan.complete, `${operation}.complete`),
	};
}

export function pluginDirectMembershipFromWails(value: unknown, operation: string): PluginDirectMembership {
	const membership = requiredObject<Record<string, unknown>>(value, operation);

	return {
		plugin: pluginViewFromWails(membership.plugin, `${operation}.plugin`),
		members: wailsObjectArrayOrEmpty(membership.members, `${operation}.members`).map((rawMember, index) => {
			const member = requiredObject<Record<string, unknown>>(rawMember, `${operation}.members[${index}]`);
			const target =
				member.target === null || member.target === undefined
					? undefined
					: capabilityTargetFromWails(member.target, `${operation}.members[${index}].target`);
			const selectorMatches =
				member.selectorMatches === null || member.selectorMatches === undefined
					? undefined
					: wailsObjectArrayOrEmpty(member.selectorMatches, `${operation}.members[${index}].selectorMatches`).map(
							(rawMatch, matchIndex): PluginDirectSelectorMatch => {
								const match = requiredObject<Record<string, unknown>>(
									rawMatch,
									`${operation}.members[${index}].selectorMatches[${matchIndex}]`
								);

								return {
									artifact: artifactRefFromWails(
										match.artifact,
										`${operation}.members[${index}].selectorMatches[${matchIndex}].artifact`
									),
									status: enumFromWails(
										match.status,
										CapabilityResolutionStatus,
										`${operation}.members[${index}].selectorMatches[${matchIndex}].status`
									),
									target:
										match.target === null || match.target === undefined
											? undefined
											: capabilityTargetFromWails(
													match.target,
													`${operation}.members[${index}].selectorMatches[${matchIndex}].target`
												),
									code: optionalWailsString(
										match.code,
										`${operation}.members[${index}].selectorMatches[${matchIndex}].code`
									),
									message: optionalWailsString(
										match.message,
										`${operation}.members[${index}].selectorMatches[${matchIndex}].message`
									),
								};
							}
						);

			return {
				index: requireWailsFiniteNumber(member.index, `${operation}.members[${index}].index`),
				entry: requiredObject(member.entry, `${operation}.members[${index}].entry`),
				status: enumFromWails(member.status, CapabilityResolutionStatus, `${operation}.members[${index}].status`),
				target,
				selectorMatches,
				code: optionalWailsString(member.code, `${operation}.members[${index}].code`),
				message: optionalWailsString(member.message, `${operation}.members[${index}].message`),
			} satisfies PluginDirectMember;
		}),
		complete: requireWailsBoolean(membership.complete, `${operation}.complete`),
	};
}

export function artifactPluginMembershipFromWails(value: unknown, operation: string): ArtifactPluginMembershipView {
	const membership = requiredObject<Record<string, unknown>>(value, operation);

	return {
		plugin: artifactRefFromWails(membership.plugin, `${operation}.plugin`),
		pluginName: requireWailsString(membership.pluginName, `${operation}.pluginName`),
		pluginRevision: requireWailsFiniteNumber(membership.pluginRevision, `${operation}.pluginRevision`),
		memberIndex: requireWailsFiniteNumber(membership.memberIndex, `${operation}.memberIndex`),
		member: requiredObject(membership.member, `${operation}.member`),
		status: enumFromWails(membership.status, CapabilityResolutionStatus, `${operation}.status`),
		resolvedArtifact:
			membership.resolvedArtifact === null || membership.resolvedArtifact === undefined
				? undefined
				: artifactRefFromWails(membership.resolvedArtifact, `${operation}.resolvedArtifact`),
		resolvedToArtifact: requireWailsBoolean(membership.resolvedToArtifact, `${operation}.resolvedToArtifact`),
		code: optionalWailsString(membership.code, `${operation}.code`),
		message: optionalWailsString(membership.message, `${operation}.message`),
	};
}

export function pluginListItemFromWails(value: unknown, field: string): PluginListItem {
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
