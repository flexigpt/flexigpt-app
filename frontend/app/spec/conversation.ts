import type { ArtifactRef } from '@/spec/artifact';
import type { Attachment } from '@/spec/attachment';
import type {
	InferenceError,
	InferenceUsage,
	InputUnion,
	ModelParam,
	OutputUnion,
	ReasoningContent,
	RoleEnum,
	Status,
	ToolChoice,
	UIToolCall,
	UIToolOutput,
	URLCitation,
} from '@/spec/inference';
import type { MCPAppModelContextUpdate, MCPConversationContext, MCPProviderToolMapping } from '@/spec/mcp';
import type { ModelPresetRef } from '@/spec/modelpreset';
import type { ToolSelection, ToolSelectionIssue, ToolStoreChoice } from '@/spec/tool';
import type { WorkspaceConversationSelection, WorkspaceConversationUsage } from '@/spec/workspace';

/** Keep in sync with Go's ConversationSchemaVersion. */
export const CONVERSATION_SCHEMA_VERSION = 'v1.0.0';

export interface StoreConversationMessage {
	id: string;
	createdAt: string;
	role: RoleEnum;
	status: Status;

	modelParam?: ModelParam;
	modelPresetRef?: ModelPresetRef;
	inputs?: InputUnion[];
	outputs?: OutputUnion[];

	toolChoices?: ToolChoice[];

	toolSelections?: ToolSelection[];
	mcpContext?: MCPConversationContext;
	mcpToolMappings?: MCPProviderToolMapping[];
	mcpAppContextUpdates?: MCPAppModelContextUpdate[];
	attachments?: Attachment[];
	enabledSkillRefs?: ArtifactRef[];
	activeSkillRefs?: ArtifactRef[];

	workspaceSelection?: WorkspaceConversationSelection;
	workspaceUsage?: WorkspaceConversationUsage;

	usage?: InferenceUsage;
	error?: InferenceError;

	meta?: Record<string, any>;
	debugDetails?: any;
}

interface BaseConversation<TMessage> {
	schemaVersion: string;
	id: string;
	title: string;
	createdAt: string;
	modifiedAt: string;
	messages: TMessage[];
	meta?: Record<string, any>;
}

interface UIConversationMessageDetails {
	// UI-only, derived from outputs (we'll derive in helpers)
	uiContent: string;
	uiDebugDetails?: string;
	uiReasoningContents?: ReasoningContent[];
	uiToolCalls?: UIToolCall[];
	uiToolOutputs?: UIToolOutput[];
	uiToolChoices?: ToolStoreChoice[];
	uiToolSelectionIssues?: ToolSelectionIssue[];
	uiCitations?: URLCitation[];
}

export type ConversationMessage = StoreConversationMessage & UIConversationMessageDetails;
export type StoreConversation = BaseConversation<StoreConversationMessage>;
export type Conversation = BaseConversation<ConversationMessage>;

export interface ConversationSearchItem {
	id: string;
	title: string;
	modifiedAt?: string;
}

export interface RestorableConversationContext {
	modelPresetRef?: ModelPresetRef;
	modelParam?: ModelParam;
	toolChoices: ToolStoreChoice[];
	toolSelectionIssues?: ToolSelectionIssue[];
	mcpContext?: MCPConversationContext;
	mcpAppContextUpdates?: MCPAppModelContextUpdate[];
	webSearchChoices: ToolStoreChoice[];
	enabledSkillRefs: ArtifactRef[];
	activeSkillRefs: ArtifactRef[];
	workspaceSelection?: WorkspaceConversationSelection;
}
