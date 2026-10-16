import type { ArtifactRef, CapabilityTarget } from '@/spec/artifact';
import type { Attachment, UIAttachment } from '@/spec/attachment';
import type { UIToolOutput } from '@/spec/inference';
import type { SessionArtifactSelection } from '@/spec/session_base';
import type { MCPAppModelContextUpdate, MCPConversationContext } from '@/spec/session_mcp';
import type { SkillRef } from '@/spec/skill';
import type { ToolSelectionIssue, ToolStoreChoice, UIToolStoreChoice } from '@/spec/tool';
import type { WorkspaceConversationSelection, WorkspaceRuntimePlan } from '@/spec/workspace';

type AgentCapabilityExclusionKind = 'instruction' | 'tool' | 'skill' | 'mcp';

/**
 * Frontend-authored removal intent for an effective Agent baseline
 * contribution. It carries the backend-owned canonical target rather than a
 * display name or a transient Composer list identity.
 */
export interface AgentCapabilityExclusion {
	key: string;
	kind: AgentCapabilityExclusionKind;
	target: CapabilityTarget;
}

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
	agentCapabilityExclusions?: AgentCapabilityExclusion[];
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
	carryForwardToolResultIDs?: string[];
}

export interface AssistantTurnFinishedPayload {
	loadedRunnableToolCallCount: number;
}
