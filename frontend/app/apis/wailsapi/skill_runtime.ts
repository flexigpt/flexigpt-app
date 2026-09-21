import type {
	InvokeSkillToolResponse,
	RuntimeSkillDefinition,
	RuntimeSkillRecord,
	RuntimeSkillRenderResult,
	RuntimeSkillSession,
	RuntimeSkillSessionOptions,
} from '@/spec/skill';
import type { RuntimeSkillListFilter, RuntimeSkillPromptFilter } from '@/spec/skill_store';

import type { JSONRawString } from '@/lib/jsonschema_utils';

import type { ISkillRuntimeAPI } from '@/apis/interface';
import {
	requiredObject,
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
		const response = await CreateSkillSession({
			Body: options,
		} as Parameters<typeof CreateSkillSession>[0]);

		return requiredObject<RuntimeSkillSession>(response.Body, 'CreateSkillSession');
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

		const response = await GetSkillsPrompt(request as Parameters<typeof GetSkillsPrompt>[0]);
		const body = requiredObject<{ prompt: unknown }>(response.Body, 'GetSkillsPrompt');

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

		const response = await ListSkills(request as Parameters<typeof ListSkills>[0]);
		const body = requiredObject<{ skills?: RuntimeSkillRecord[] }>(response.Body, 'ListSkills');

		return wailsObjectArrayOrEmpty<RuntimeSkillRecord>(body.skills, 'ListSkills.skills');
	}

	async renderSkill(
		definition: RuntimeSkillDefinition,
		args?: Record<string, string>
	): Promise<RuntimeSkillRenderResult> {
		const response = await RenderSkill({
			Body: {
				definition,
				arguments: args,
			},
		} as Parameters<typeof RenderSkill>[0]);

		return requiredObject<RuntimeSkillRenderResult>(response.Body, 'RenderSkill');
	}

	async invokeSkillTool(sessionID: string, toolName: string, args?: JSONRawString): Promise<InvokeSkillToolResponse> {
		const response = await InvokeSkillTool({
			Body: {
				sessionID: requireNonBlankString(sessionID, 'sessionID'),
				toolName: requireNonBlankString(toolName, 'toolName'),
				args,
			},
		} as Parameters<typeof InvokeSkillTool>[0]);

		return requiredObject<InvokeSkillToolResponse>(response.Body, 'InvokeSkillTool');
	}
}
