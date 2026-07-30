// oxlint-disable typescript/no-misused-spread
import type {
	InvokeSkillToolResponse,
	ListSkillsRequest,
	ProvidedSkill,
	PutSkillArtifactPayload,
	RenderProvidedSkillResponse,
	RenderSkillResponse,
	RuntimeSkillFilter,
	RuntimeSkillListItem,
	Skill,
	SkillBundle,
	SkillListItem,
	SkillRef,
	SkillSession,
	SkillType,
} from '@/spec/skill';
import type { WorkspaceRef } from '@/spec/workspace';

import type { JSONRawString } from '@/lib/jsonschema_utils';

import type { ISkillStoreAPI } from '@/apis/interface';
import { requireWailsBody, toFrontendDate } from '@/apis/wailsapi/transport';
import {
	CloseSkillSession,
	CreateSkillSession,
	DeleteSkill,
	DeleteSkillBundle,
	GetSkill,
	GetSkillsPrompt,
	InvokeSkillTool,
	ListProvidedSkills,
	ListRuntimeSkills,
	ListSkillBundles,
	ListSkills,
	PatchSkill,
	PatchSkillBundle,
	PutSkill,
	PutSkillArtifact,
	PutSkillBundle,
	RenderProvidedSkill,
	RenderSkill,
} from '@/apis/wailsjs/go/main/SkillStoreWrapper';
import type { skillruntime, spec } from '@/apis/wailsjs/go/models';

function optionalFrontendDate(value: unknown, field: string): Date | undefined {
	if (value === undefined || value === null || value === '') {
		return undefined;
	}

	return toFrontendDate(value, field);
}

function skillFromWails(skill: spec.Skill): Skill {
	const presence = skill.presence
		? {
				...skill.presence,
				lastCheckedAt: optionalFrontendDate(skill.presence.lastCheckedAt, 'skill.presence.lastCheckedAt'),
				lastSeenAt: optionalFrontendDate(skill.presence.lastSeenAt, 'skill.presence.lastSeenAt'),
				missingSince: optionalFrontendDate(skill.presence.missingSince, 'skill.presence.missingSince'),
			}
		: undefined;

	return {
		...skill,
		...(presence === undefined ? {} : { presence }),
		createdAt: toFrontendDate(skill.createdAt, 'skill.createdAt'),
		modifiedAt: toFrontendDate(skill.modifiedAt, 'skill.modifiedAt'),
	} as Skill;
}

function skillBundleFromWails(bundle: spec.SkillBundle): SkillBundle {
	return {
		...bundle,
		createdAt: toFrontendDate(bundle.createdAt, 'skillBundle.createdAt'),
		modifiedAt: toFrontendDate(bundle.modifiedAt, 'skillBundle.modifiedAt'),
		softDeletedAt: optionalFrontendDate(bundle.softDeletedAt, 'skillBundle.softDeletedAt'),
	} as SkillBundle;
}

function skillListItemFromWails(item: spec.SkillListItem): SkillListItem {
	return {
		...item,
		skillDefinition: skillFromWails(item.skillDefinition),
	} as SkillListItem;
}

function providedSkillFromWails(skill: skillruntime.Skill): ProvidedSkill {
	return {
		...skill,
		createdAt: toFrontendDate(skill.createdAt, 'providedSkill.createdAt'),
		modifiedAt: toFrontendDate(skill.modifiedAt, 'providedSkill.modifiedAt'),
	} as ProvidedSkill;
}

function renderedProvidedSkillFromWails(skill: skillruntime.RenderedSkill): RenderProvidedSkillResponse {
	return {
		...skill,
		skill: providedSkillFromWails(skill.skill),
	} as RenderProvidedSkillResponse;
}

export class WailsSkillStoreAPI implements ISkillStoreAPI {
	async listSkillBundles(
		bundleIDs?: string[],
		includeDisabled?: boolean,
		pageSize?: number,
		pageToken?: string
	): Promise<{ skillBundles: SkillBundle[]; nextPageToken?: string }> {
		const req = {
			BundleIDs: bundleIDs ?? [],
			IncludeDisabled: includeDisabled ?? false,
			PageSize: pageSize ?? 0,
			PageToken: pageToken ?? '',
		};
		const resp = await ListSkillBundles(req as spec.ListSkillBundlesRequest);
		const body = requireWailsBody(resp.Body, 'ListSkillBundles');

		return {
			skillBundles: (body.skillBundles ?? []).map(s => {
				return skillBundleFromWails(s);
			}),
			nextPageToken: body.nextPageToken ?? undefined,
		};
	}

	async putSkillBundle(
		bundleID: string,
		slug: string,
		displayName: string,
		isEnabled: boolean,
		description?: string
	): Promise<void> {
		const req = {
			BundleID: bundleID,
			Body: {
				slug,
				displayName,
				isEnabled,
				description,
			} as spec.PutSkillBundleRequestBody,
		};
		await PutSkillBundle(req as spec.PutSkillBundleRequest);
	}

	async patchSkillBundle(bundleID: string, isEnabled: boolean): Promise<void> {
		const req = {
			BundleID: bundleID,
			Body: {
				isEnabled,
			},
		};
		await PatchSkillBundle(req as spec.PatchSkillBundleRequest);
	}

	async deleteSkillBundle(bundleID: string): Promise<void> {
		const req: spec.DeleteSkillBundleRequest = { BundleID: bundleID };
		await DeleteSkillBundle(req);
	}

	async listSkills(req: ListSkillsRequest): Promise<{ skillListItems: SkillListItem[]; nextPageToken?: string }> {
		const inReq = {
			BundleIDs: req.bundleIDs ?? [],
			Types: req.types ?? [],
			Inserts: req.inserts ?? [],
			Tags: req.tags ?? [],
			IncludeDisabled: req.includeDisabled ?? false,
			IncludeMissing: req.includeMissing ?? false,
			RecommendedPageSize: req.recommendedPageSize ?? 0,
			PageToken: req.pageToken ?? '',
		};
		const resp = await ListSkills(inReq as spec.ListSkillsRequest);
		const body = requireWailsBody(resp.Body, 'ListSkills');

		return {
			skillListItems: (body.skillListItems ?? []).map(s => {
				return skillListItemFromWails(s);
			}),
			nextPageToken: body.nextPageToken ?? undefined,
		};
	}

	async putSkill(
		bundleID: string,
		skillSlug: string,
		skillType: SkillType,
		location: string,
		name: string,
		isEnabled: boolean,
		displayName?: string,
		description?: string,
		tags?: string[]
	): Promise<void> {
		const req = {
			BundleID: bundleID,
			SkillSlug: skillSlug,
			Body: {
				skillType,
				location,
				name,
				isEnabled,
				displayName,
				description,
				tags,
			} as spec.PutSkillRequestBody,
		};
		await PutSkill(req as spec.PutSkillRequest);
	}

	async putSkillArtifact(bundleID: string, skillSlug: string, payload: PutSkillArtifactPayload): Promise<Skill> {
		const req = {
			BundleID: bundleID,
			SkillSlug: skillSlug,
			Body: payload as spec.PutSkillArtifactRequestBody,
		};
		const resp = await PutSkillArtifact(req as spec.PutSkillArtifactRequest);
		const body = requireWailsBody(resp.Body, 'PutSkillArtifact');

		return skillFromWails(body.skill);
	}

	async patchSkill(
		bundleID: string,
		skillSlug: string,
		isEnabled?: boolean,
		location?: string,
		displayName?: string,
		description?: string,
		tags?: string[]
	): Promise<void> {
		const req = {
			BundleID: bundleID,
			SkillSlug: skillSlug,
			Body: {
				isEnabled: isEnabled,
				location: location,
				displayName: displayName,
				description: description,
				tags: tags,
			} as spec.PatchSkillRequestBody,
		};
		await PatchSkill(req as spec.PatchSkillRequest);
	}

	async deleteSkill(bundleID: string, skillSlug: string): Promise<void> {
		const req: spec.DeleteSkillRequest = { BundleID: bundleID, SkillSlug: skillSlug };
		await DeleteSkill(req);
	}

	async getSkill(bundleID: string, skillSlug: string, includeDisabled: boolean): Promise<Skill | undefined> {
		const req: spec.GetSkillRequest = { BundleID: bundleID, SkillSlug: skillSlug, IncludeDisabled: includeDisabled };
		const resp = await GetSkill(req);

		return resp.Body ? skillFromWails(resp.Body) : undefined;
	}

	async getSkillsPrompt(filter?: RuntimeSkillFilter): Promise<string> {
		const req = {
			Body: { filter: filter as spec.RuntimeSkillFilter } as spec.GetSkillsPromptRequestBody,
		} as spec.GetSkillsPromptRequest;
		const resp = await GetSkillsPrompt(req);

		return requireWailsBody(resp.Body, 'GetSkillsPrompt').prompt;
	}

	async createSkillSession(
		closeSessionID?: string,
		maxActivePerSession?: number,
		allowSkillRefs?: SkillRef[],
		activeSkillRefs?: SkillRef[],
		workspace?: WorkspaceRef
	): Promise<SkillSession> {
		const req = {
			Body: {
				closeSessionID: closeSessionID,
				maxActivePerSession: maxActivePerSession,
				allowSkillRefs: allowSkillRefs,
				activeSkillRefs: activeSkillRefs,
				workspace: workspace,
			} as spec.CreateSkillSessionRequestBody,
		} as spec.CreateSkillSessionRequest;

		const resp = await CreateSkillSession(req);
		const body = requireWailsBody(resp.Body, 'CreateSkillSession');

		return {
			sessionID: body.sessionID,
			activeSkillRefs: (body.activeSkillRefs ?? []) as SkillRef[],
		};
	}

	async closeSkillSession(sessionID: string): Promise<void> {
		const req: spec.CloseSkillSessionRequest = { SessionID: sessionID };
		await CloseSkillSession(req);
	}

	async listRuntimeSkills(filter?: RuntimeSkillFilter): Promise<RuntimeSkillListItem[]> {
		const req = {
			Body: { filter: filter as spec.RuntimeSkillFilter } as spec.ListRuntimeSkillsRequestBody,
		} as spec.ListRuntimeSkillsRequest;

		const resp = await ListRuntimeSkills(req);
		const body = requireWailsBody(resp.Body, 'ListRuntimeSkills');

		return (body.skills ?? []) as RuntimeSkillListItem[];
	}

	async invokeSkillTool(sessionID: string, toolName: string, args?: JSONRawString): Promise<InvokeSkillToolResponse> {
		const req = {
			Body: { sessionID: sessionID, toolName: toolName, args: args } as spec.InvokeSkillToolRequestBody,
		} as spec.InvokeSkillToolRequest;

		const resp = await InvokeSkillTool(req);
		return requireWailsBody(resp.Body, 'InvokeSkillTool') as InvokeSkillToolResponse;
	}

	async renderSkill(
		ref: SkillRef,
		args?: Record<string, string>,
		workspace?: WorkspaceRef
	): Promise<RenderSkillResponse> {
		const req = {
			Body: { skillRef: ref, arguments: args, workspace: workspace } as spec.RenderSkillRequestBody,
		} as spec.RenderSkillRequest;

		const resp = await RenderSkill(req);

		return requireWailsBody(resp.Body, 'RenderSkill') as RenderSkillResponse;
	}

	async listProvidedSkills(workspace?: WorkspaceRef): Promise<ProvidedSkill[]> {
		const response = await ListProvidedSkills({
			workspace,
		} as Parameters<typeof ListProvidedSkills>[0]);

		if (!response.Body) {
			throw new Error('ListProvidedSkills returned an empty response body.');
		}

		return (response.Body.skills ?? []).map(s => {
			return providedSkillFromWails(s);
		});
	}

	async renderProvidedSkill(
		ref: SkillRef,
		args?: Record<string, string>,
		workspace?: WorkspaceRef
	): Promise<RenderProvidedSkillResponse> {
		const response = await RenderProvidedSkill({
			Body: {
				workspace,
				ref,
				arguments: args,
			},
		} as Parameters<typeof RenderProvidedSkill>[0]);

		if (!response.Body) {
			throw new Error('RenderProvidedSkill returned an empty response body.');
		}

		return renderedProvidedSkillFromWails(response.Body);
	}
}
