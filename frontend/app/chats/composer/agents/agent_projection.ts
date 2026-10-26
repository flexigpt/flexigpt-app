import type { AgentCapabilityOccurrence, AgentResolution, AgentView } from '@/spec/agent';
import type { ArtifactRef, CapabilityTarget } from '@/spec/artifact';
import type { SessionActiveSkillOverride } from '@/spec/session';
import { AgentSkillUseMode, isAgentSkillUseAsInstructions } from '@/spec/agent';
import { CapabilityResolutionStatus, CapabilityTargetForm } from '@/spec/artifact';
import { SessionActivityOverrideMode } from '@/spec/session_base';

import { sessionArtifactKey, sessionCapabilityKey } from '@/chats/session/session_capability_key';

export interface AgentToolProjection {
	occurrence: AgentCapabilityOccurrence;
	target: CapabilityTarget;
	sessionKey: string;
	autoExecute: boolean;
}

export interface AgentSkillProjection {
	occurrence: AgentCapabilityOccurrence;
	ref: ArtifactRef;
	key: string;
	displayName: string;
	active: boolean;
}

export interface AgentMCPProjection {
	occurrence: AgentCapabilityOccurrence;
	ref: ArtifactRef;
	key: string;
	displayName: string;
}

export interface AgentInstructionSourceDeclaration {
	occurrencePath: string;
	kind: 'skill' | 'text';
	artifact: ArtifactRef;
	displayName: string;
}

export interface AgentSkillActivityChange {
	override?: SessionActiveSkillOverride;
	releasedOverrideKey?: string;
}

function availableTarget(occurrence: AgentCapabilityOccurrence): CapabilityTarget | undefined {
	if (occurrence.status !== CapabilityResolutionStatus.Available || !occurrence.target) {
		return undefined;
	}

	return occurrence.target;
}

function artifactTarget(occurrence: AgentCapabilityOccurrence): ArtifactRef | undefined {
	const target = availableTarget(occurrence);

	if (target?.form !== CapabilityTargetForm.Artifact || !target.artifact) {
		return undefined;
	}

	return target.artifact;
}

function occurrenceDisplayName(occurrence: AgentCapabilityOccurrence, fallback: string): string {
	return occurrence.name?.trim() || fallback;
}

export function projectAgentToolCapabilities(resolution: AgentResolution | null): AgentToolProjection[] {
	const projections: AgentToolProjection[] = [];

	for (const occurrence of resolution?.capabilities.occurrences ?? []) {
		const target = occurrence.type === 'tool' ? availableTarget(occurrence) : undefined;
		if (!target) {
			continue;
		}

		projections.push({
			occurrence,
			target,
			sessionKey: sessionCapabilityKey(target),
			autoExecute: occurrence.autoExecute === true,
		});
	}

	return projections;
}

export function projectAgentSkillCapabilities(resolution: AgentResolution | null): AgentSkillProjection[] {
	const projections: AgentSkillProjection[] = [];

	for (const occurrence of resolution?.capabilities.occurrences ?? []) {
		if (occurrence.type !== 'skill' || isAgentSkillUseAsInstructions(occurrence.skillUseMode)) {
			continue;
		}

		const ref = artifactTarget(occurrence);
		if (!ref) {
			continue;
		}

		projections.push({
			occurrence,
			ref,
			key: sessionArtifactKey(ref),
			displayName: occurrenceDisplayName(occurrence, ref.artifactID),
			active: occurrence.skillUseMode === AgentSkillUseMode.Active,
		});
	}

	return projections;
}

export function projectAgentMCPCapabilities(resolution: AgentResolution | null): AgentMCPProjection[] {
	const projections: AgentMCPProjection[] = [];

	for (const occurrence of resolution?.capabilities.occurrences ?? []) {
		if (occurrence.type !== 'mcp') {
			continue;
		}

		const ref = artifactTarget(occurrence);
		if (!ref) {
			continue;
		}

		projections.push({
			occurrence,
			ref,
			key: sessionArtifactKey(ref),
			displayName: occurrenceDisplayName(occurrence, ref.artifactID),
		});
	}

	return projections;
}

export function projectAgentInstructionSources(
	resolution: AgentResolution,
	instructionTextArtifacts: ArtifactRef[]
): AgentInstructionSourceDeclaration[] {
	const instructionTextKeys = new Set(
		instructionTextArtifacts.map(s => {
			return sessionArtifactKey(s);
		})
	);
	const output: AgentInstructionSourceDeclaration[] = [];

	for (const occurrence of resolution.capabilities.occurrences) {
		const ref = artifactTarget(occurrence);
		if (!ref) {
			continue;
		}

		if (occurrence.type === 'skill' && isAgentSkillUseAsInstructions(occurrence.skillUseMode)) {
			output.push({
				occurrencePath: occurrence.path,
				kind: 'skill',
				artifact: ref,
				displayName: occurrenceDisplayName(occurrence, ref.artifactID),
			});
			continue;
		}

		if (occurrence.type === 'text' && instructionTextKeys.has(sessionArtifactKey(ref))) {
			output.push({
				occurrencePath: occurrence.path,
				kind: 'text',
				artifact: ref,
				displayName: occurrenceDisplayName(occurrence, ref.artifactID),
			});
		}
	}

	return output;
}

export function effectiveAgentSkillActive(
	projection: AgentSkillProjection,
	override: SessionActiveSkillOverride | undefined
): boolean {
	switch (override?.mode) {
		case SessionActivityOverrideMode.Activate:
			return true;
		case SessionActivityOverrideMode.Deactivate:
			return false;
		default:
			return projection.active;
	}
}

export function agentSkillActivityChange(
	projection: AgentSkillProjection,
	nextActive: boolean
): AgentSkillActivityChange {
	if (nextActive === projection.active) {
		return {
			releasedOverrideKey: projection.key,
		};
	}

	return {
		override: {
			key: projection.key,
			mode: nextActive ? SessionActivityOverrideMode.Activate : SessionActivityOverrideMode.Deactivate,
			...(nextActive
				? {
						value: {
							rootID: projection.ref.rootID,
							artifactID: projection.ref.artifactID,
						},
					}
				: {}),
		},
	};
}

export function sameAgentDefinition(
	left: Pick<AgentView, 'ref' | 'revision' | 'definitionDigest'> | undefined,
	right: Pick<AgentView, 'ref' | 'revision' | 'definitionDigest'> | undefined
): boolean {
	return (
		left?.ref.rootID === right?.ref.rootID &&
		left?.ref.artifactID === right?.ref.artifactID &&
		left?.revision === right?.revision &&
		left?.definitionDigest === right?.definitionDigest
	);
}
