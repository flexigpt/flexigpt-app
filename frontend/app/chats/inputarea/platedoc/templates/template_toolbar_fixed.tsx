import { FiCheck, FiEdit2, FiMaximize2, FiUpload, FiX } from 'react-icons/fi';

import { PromptRoleEnum } from '@/spec/prompt';

import { type SelectedTemplateForRun } from '@/chats/inputarea/platedoc/templates/template_processing';

export function TemplateFixedToolbar(props: {
	selection: SelectedTemplateForRun;
	flashing?: boolean;
	onOpenModal: () => void;
	onRemove: () => void;
	onFlatten: () => void;
	onSetAsSystemPrompt: () => void;
}) {
	const { selection, flashing, onOpenModal, onRemove, onFlatten, onSetAsSystemPrompt } = props;

	const templateName =
		selection.template.displayName || selection.templateSlug || selection.template.slug || 'Template';

	const hasSystemBlock = selection.blocks.some(
		b => [PromptRoleEnum.Developer, PromptRoleEnum.System].includes(b.role) && b.content.trim() !== ''
	);

	return (
		<div
			className={`flex w-full items-center gap-3 border-b px-2 py-0 font-mono text-xs ${
				flashing ? 'ring-error rounded-t-2xl ring ring-offset-0' : ''
			}`}
			data-template-toolbar
		>
			<span title={templateName}>{templateName.length > 32 ? templateName.slice(0, 32) + '...' : templateName}</span>

			<div
				className={`flex items-center gap-1 rounded-2xl px-2 py-1 ${selection.variablesSchema.length > 0 ? (selection.requiredCount > 0 ? 'bg-warning text-warning-content' : 'bg-success text-success-content') : 'bg-base-200'}`}
			>
				{selection.variablesSchema.length > 0 ? (
					<>
						<span
							className="flex items-center gap-1 px-2 py-0"
							title={selection.requiredCount > 0 ? `pending ${selection.requiredCount} vars` : 'All vars provided'}
						>
							{selection.variablesSchema.length} var{selection.variablesSchema.length === 1 ? '' : 's'}:{' '}
							{`pending ${selection.requiredCount}`}
						</span>
					</>
				) : (
					<span className="flex items-center gap-1 px-2 py-0" title="No vars configured">
						0 vars <FiX />
					</span>
				)}
			</div>

			<div
				className={`flex items-center gap-1 rounded-2xl px-2 py-1 ${hasSystemBlock ? 'bg-success text-success-content' : 'bg-base-200'}`}
			>
				{hasSystemBlock ? (
					<span className="flex items-center gap-1 px-2 py-0" title="Template contains a system/dev prompt block">
						system <FiCheck />
					</span>
				) : (
					<span className="flex items-center gap-1 px-2 py-0" title="Template contains a system/dev prompt block">
						system <FiX />
					</span>
				)}
			</div>
			<div className="grow" />
			<div className="flex items-center gap-2 px-2 py-1">
				<button
					type="button"
					className="btn btn-ghost btn-sm shrink-0 px-2 py-0 shadow-none"
					onClick={onOpenModal}
					title="Edit in template editor"
				>
					<FiEdit2 size={14} />
				</button>
				<button
					type="button"
					className="btn btn-ghost btn-sm shrink-0 px-2 py-0 shadow-none"
					onClick={onSetAsSystemPrompt}
					title="Set system prompt for chat"
					aria-label="Set system prompt for chat"
					disabled={!hasSystemBlock}
				>
					<FiUpload size={14} />
				</button>
				{/* Expand to plain text */}
				<button
					type="button"
					className="btn btn-ghost btn-sm shrink-0 px-2 py-0 shadow-none"
					onClick={onFlatten}
					title="Convert chips into plain text"
					aria-label="Convert template chips into plain text"
				>
					<FiMaximize2 size={14} />
				</button>

				{/* Remove template */}
				<button
					type="button"
					className="btn btn-ghost btn-sm text-error shrink-0 px-2 py-0"
					onClick={onRemove}
					title="Remove template"
					aria-label="Remove template"
				>
					<FiX size={14} />
				</button>
			</div>
		</div>
	);
}
