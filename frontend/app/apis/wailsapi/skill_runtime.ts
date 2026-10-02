import type {
	InvokeSkillToolResponse,
	RuntimeSkillDefinition,
	RuntimeSkillListFilter,
	RuntimeSkillPromptFilter,
	RuntimeSkillRecord,
	RuntimeSkillRenderResult,
	RuntimeSkillSession,
	RuntimeSkillSessionOptions,
} from '@/spec/skill';

import type { JSONRawString } from '@/lib/jsonschema_utils';

import type { ISkillRuntimeAPI } from '@/apis/interface';
import {
	requiredObject,
	requiredRuntimeResponseBody,
	requireNonBlankString,
	requireWailsString,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	CloseSkillSession,
	CreateSkillSession,
	GetSkillsPrompt,
	InvokeSkillTool,
	ListSkills,
	RenderSkill,
} from '@/apis/wailsjs/go/main/SkillRuntimeWrapper';

export class WailsSkillRuntimeAPI implements ISkillRuntimeAPI {
	async createSkillSession(options: RuntimeSkillSessionOptions): Promise<RuntimeSkillSession> {
		const response = requiredObject<{ Body?: unknown }>(
			await CreateSkillSession({
				Body: options,
			} as Parameters<typeof CreateSkillSession>[0]),
			'CreateSkillSession response'
		);
		const body = requiredObject<Record<string, unknown>>(response.Body, 'CreateSkillSession');

		return {
			sessionID: requireNonBlankString(body.sessionID, 'CreateSkillSession.sessionID'),
			activeSkills: wailsObjectArrayOrEmpty<RuntimeSkillDefinition>(
				body.activeSkills,
				'CreateSkillSession.activeSkills'
			),
		};
	}

	async closeSkillSession(sessionID: string): Promise<void> {
		await CloseSkillSession({
			SessionID: requireNonBlankString(sessionID, 'sessionID'),
		} as Parameters<typeof CloseSkillSession>[0]);
	}

	async getSkillsPrompt(filter?: RuntimeSkillPromptFilter): Promise<string> {
		const request =
			filter === undefined
				? {}
				: {
						Body: {
							filter,
						},
					};

		const body = requiredRuntimeResponseBody<{ prompt: unknown }>(
			await GetSkillsPrompt(request as Parameters<typeof GetSkillsPrompt>[0]),
			'GetSkillsPrompt'
		);

		return requireWailsString(body.prompt, 'GetSkillsPrompt.prompt');
	}

	async listRuntimeSkills(filter?: RuntimeSkillListFilter): Promise<RuntimeSkillRecord[]> {
		const request =
			filter === undefined
				? {}
				: {
						Body: {
							filter,
						},
					};

		const body = requiredRuntimeResponseBody<{ skills?: unknown }>(
			await ListSkills(request as Parameters<typeof ListSkills>[0]),
			'ListSkills'
		);

		return wailsObjectArrayOrEmpty<RuntimeSkillRecord>(body.skills, 'ListSkills.skills');
	}

	async renderSkill(
		definition: RuntimeSkillDefinition,
		args?: Record<string, string>
	): Promise<RuntimeSkillRenderResult> {
		return requiredRuntimeResponseBody<RuntimeSkillRenderResult>(
			await RenderSkill({
				Body: {
					definition,
					arguments: args,
				},
			} as Parameters<typeof RenderSkill>[0]),
			'RenderSkill'
		);
	}

	async invokeSkillTool(sessionID: string, toolName: string, args?: JSONRawString): Promise<InvokeSkillToolResponse> {
		const response = await InvokeSkillTool({
			Body: {
				sessionID: requireNonBlankString(sessionID, 'sessionID'),
				toolName: requireNonBlankString(toolName, 'toolName'),
				args,
			},
		} as Parameters<typeof InvokeSkillTool>[0]);

		return requiredRuntimeResponseBody<InvokeSkillToolResponse>(response, 'InvokeSkillTool');
	}
}
