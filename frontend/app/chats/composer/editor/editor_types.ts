import type { Attachment, UIAttachment } from '@/spec/attachment';
import type { UIToolOutput } from '@/spec/inference';
import type { MCPAppModelContextUpdate, MCPConversationContext } from '@/spec/mcp_artifact';
import type { SkillRef } from '@/spec/skill';
import type { ToolStoreChoice, UIToolStoreChoice } from '@/spec/tool';
import type { WorkspaceConversationSelection } from '@/spec/workspace';

export interface EditorExternalMessage {
	text: string;
	attachments?: Attachment[];
	toolChoices?: ToolStoreChoice[];
	mcpContext?: MCPConversationContext;
	mcpAppContextUpdates?: MCPAppModelContextUpdate[];
	toolOutputs?: UIToolOutput[];
	enabledSkillRefs?: SkillRef[];
	activeSkillRefs?: SkillRef[];
	workspaceSelection?: WorkspaceConversationSelection;
}

export interface EditorSubmitPayload {
	text: string;
	resolvedSystemPrompt?: string;
	attachedTools: UIToolStoreChoice[];
	attachments: UIAttachment[];
	toolOutputs: UIToolOutput[];
	finalToolChoices: ToolStoreChoice[];
	mcpContext?: MCPConversationContext;
	mcpAppContextUpdates?: MCPAppModelContextUpdate[];
	enabledSkillRefs?: SkillRef[];
	activeSkillRefs?: SkillRef[];
	skillSessionID?: string;
	workspaceSelection?: WorkspaceConversationSelection;
}

export interface AssistantTurnFinishedPayload {
	loadedRunnableToolCallCount: number;
}
