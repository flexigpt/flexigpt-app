import type {
	InvokeSkillToolResponse,
	RuntimeSkillDefinition,
	RuntimeSkillQuery,
	RuntimeSkillRecord,
	RuntimeSkillRenderResult,
	RuntimeSkillSession,
	RuntimeSkillSessionOptions,
} from '@/spec/skill';

import type { JSONRawString } from '@/lib/jsonschema_utils';

import type { ISkillRuntimeAPI } from '@/apis/interface';
import {
	omitUndefined,
	rawJSONToWails,
	requireNonBlankString,
	requireWailsBody,
	requireWailsString,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	CloseSkillSession,
	CreateSkillSession,
	GetSkillsPrompt,
	InvokeSkillTool,
	ListSkills,
	RemoveSkillCatalog,
	RenderSkill,
	SyncSkillCatalog,
} from '@/apis/wailsjs/go/main/SkillRuntimeWrapper';
import type { runtime as wailsRuntime } from '@/apis/wailsjs/go/models';

export class WailsSkillRuntimeAPI implements ISkillRuntimeAPI {
	async syncSkillCatalog(catalogID: string): Promise<void> {
		await SyncSkillCatalog({
			catalogID: requireNonBlankString(catalogID, 'catalogID'),
		} as wailsRuntime.SyncCatalogRequest);
	}

	async removeSkillCatalog(catalogID: string): Promise<void> {
		await RemoveSkillCatalog({
			catalogID: requireNonBlankString(catalogID, 'catalogID'),
		} as wailsRuntime.RemoveCatalogRequest);
	}

	async createSkillSession(options: RuntimeSkillSessionOptions): Promise<RuntimeSkillSession> {
		const body = omitUndefined({
			closeSessionID: options.closeSessionID,
			maxActivePerSession: options.maxActivePerSession,
			allowedSkills: options.allowedSkills,
			activeSkills: options.activeSkills,
		}) as wailsRuntime.CreateSkillSessionRequestBody;

		const response = await CreateSkillSession({
			Body: body,
		} as wailsRuntime.CreateSkillSessionRequest);
		const responseBody = requireWailsBody(response.Body, 'CreateSkillSession');

		return {
			sessionID: requireWailsString(responseBody.sessionID, 'CreateSkillSession.sessionID'),
			activeSkills: wailsObjectArrayOrEmpty(responseBody.activeSkills, 'CreateSkillSession.activeSkills'),
		};
	}

	async closeSkillSession(sessionID: string): Promise<void> {
		await CloseSkillSession({
			SessionID: requireNonBlankString(sessionID, 'sessionID'),
		} as wailsRuntime.CloseSkillSessionRequest);
	}

	async getSkillsPrompt(filter?: RuntimeSkillQuery): Promise<string> {
		const request =
			filter === undefined
				? {}
				: {
						Body: {
							filter: filter as wailsRuntime.SkillPromptFilter,
						},
					};

		const response = await GetSkillsPrompt(request as wailsRuntime.GetSkillsPromptRequest);
		const body = requireWailsBody(response.Body, 'GetSkillsPrompt');
		return requireWailsString(body.prompt, 'GetSkillsPrompt.prompt');
	}

	async listSkills(filter?: RuntimeSkillQuery): Promise<RuntimeSkillRecord[]> {
		const request =
			filter === undefined
				? {}
				: {
						Body: {
							filter: omitUndefined({
								...filter,
								inserts: filter.inserts,
							}) as wailsRuntime.SkillListFilter,
						},
					};

		const response = await ListSkills(request as wailsRuntime.ListSkillsRequest);
		const body = requireWailsBody(response.Body, 'ListSkills');

		return body.skills as RuntimeSkillRecord[];
	}

	async renderSkill(
		definition: RuntimeSkillDefinition,
		args?: Record<string, string>
	): Promise<RuntimeSkillRenderResult> {
		const response = await RenderSkill({
			Body: {
				definition: definition,
				arguments: args,
			},
		} as wailsRuntime.RenderSkillRequest);

		return response.Body as RuntimeSkillRenderResult;
	}

	async invokeSkillTool(sessionID: string, toolName: string, args?: JSONRawString): Promise<InvokeSkillToolResponse> {
		const response = await InvokeSkillTool({
			Body: omitUndefined({
				sessionID: requireNonBlankString(sessionID, 'sessionID'),
				toolName: requireNonBlankString(toolName, 'toolName'),
				args: args === undefined ? undefined : rawJSONToWails(args, 'skill tool arguments'),
			}) as wailsRuntime.InvokeSkillToolRequestBody,
		} as wailsRuntime.InvokeSkillToolRequest);

		return requireWailsBody(response.Body, 'InvokeSkillTool') as InvokeSkillToolResponse;
	}
}
