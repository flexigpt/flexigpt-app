import type { Attachment, UIAttachment } from '@/spec/attachment';
import type { UIToolOutput } from '@/spec/inference';
import type { MCPConversationContext } from '@/spec/mcp';
import type { SkillRef } from '@/spec/skill';
import type { ToolStoreChoice, UIToolStoreChoice } from '@/spec/tool';

export interface EditorExternalMessage {
	text: string;
	attachments?: Attachment[];
	toolChoices?: ToolStoreChoice[];
	mcpContext?: MCPConversationContext;
	toolOutputs?: UIToolOutput[];
	enabledSkillRefs?: SkillRef[];
	activeSkillRefs?: SkillRef[];
}

export interface EditorSubmitPayload {
	text: string;
	resolvedSystemPrompt?: string;
	templateSystemPrompt?: string;
	attachedTools: UIToolStoreChoice[];
	attachments: UIAttachment[];
	toolOutputs: UIToolOutput[];
	finalToolChoices: ToolStoreChoice[];
	mcpContext?: MCPConversationContext;
	enabledSkillRefs?: SkillRef[];
	activeSkillRefs?: SkillRef[];
	skillSessionID?: string;
}

export interface AssistantTurnFinishedPayload {
	loadedRunnableToolCallCount: number;
}
