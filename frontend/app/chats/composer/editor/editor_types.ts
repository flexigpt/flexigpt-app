import type { ArtifactRef } from '@/spec/artifact';
import type { Attachment, UIAttachment } from '@/spec/attachment';
import type { UIToolOutput } from '@/spec/inference';
import type { MCPAppModelContextUpdate, MCPConversationContext } from '@/spec/mcp';
import type { SessionArtifactSelection } from '@/spec/session';
import type { SkillRef } from '@/spec/skill';
import type { ToolSelectionIssue, ToolStoreChoice, UIToolStoreChoice } from '@/spec/tool';
import type { WorkspaceConversationSelection, WorkspaceRuntimePlan } from '@/spec/workspace';

export interface EditorExternalMessage {
	text: string;
	attachments?: Attachment[];
	toolChoices?: ToolStoreChoice[];
	toolSelectionIssues?: ToolSelectionIssue[];
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
	sessionInstructionText?: string;
	includeModelDefault?: boolean;
	agentRef?: ArtifactRef;
	agentSelection?: SessionArtifactSelection;
	modelSelectionSource?: 'session' | 'agent';
	attachedTools: UIToolStoreChoice[];
	attachments: UIAttachment[];
	toolOutputs: UIToolOutput[];
	finalToolChoices: ToolStoreChoice[];
	mcpContext?: MCPConversationContext;
	mcpAppContextUpdates?: MCPAppModelContextUpdate[];
	enabledSkillRefs?: SkillRef[];
	enabledSkillSelections?: SessionArtifactSelection[];
	activeSkillRefs?: SkillRef[];
	activeSkillSelections?: SessionArtifactSelection[];
	skillSessionID?: string;
	workspaceSelection?: WorkspaceConversationSelection;
	workspacePlan?: WorkspaceRuntimePlan;
}

export interface AssistantTurnFinishedPayload {
	loadedRunnableToolCallCount: number;
}
