import type { Attachment, UIAttachment } from '@/spec/attachment';
import type { UIToolOutput } from '@/spec/inference';
import type { SkillRef } from '@/spec/skill';
import type { ToolStoreChoice, UIToolStoreChoice } from '@/spec/tool';

export interface EditorExternalMessage {
	text: string;
	attachments?: Attachment[];
	toolChoices?: ToolStoreChoice[];
	toolOutputs?: UIToolOutput[];
	enabledSkillRefs?: SkillRef[];
	activeSkillRefs?: SkillRef[];
}

export interface EditorSubmitPayload {
	text: string;
	templateSystemPrompt?: string;
	attachedTools: UIToolStoreChoice[];
	attachments: UIAttachment[];
	toolOutputs: UIToolOutput[];
	finalToolChoices: ToolStoreChoice[];
	enabledSkillRefs?: SkillRef[];
	activeSkillRefs?: SkillRef[];
	skillSessionID?: string;
}
