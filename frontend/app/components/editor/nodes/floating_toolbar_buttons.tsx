import { KEYS } from 'platejs';
import { useEditorReadOnly } from 'platejs/react';

import { MarkToolbarButton } from '@/components/editor/nodes/mark_toolbar_button';
import { ToolbarGroup } from '@/components/editor/nodes/toolbar';

function useModAndShiftSymbols() {
	const isMac = typeof navigator !== 'undefined' && /(Mac|iPhone|iPod|iPad)/i.test(navigator.platform || '');

	return {
		mod: isMac ? '⌘' : 'Ctrl',
		shift: isMac ? '⇧' : 'Shift',
	};
}

export function FloatingToolbarButtons() {
	const readOnly = useEditorReadOnly();
	const { mod, shift } = useModAndShiftSymbols();

	if (readOnly) return null;

	return (
		<ToolbarGroup>
			<MarkToolbarButton nodeType={KEYS.bold} tooltip={`Bold (${mod}+B)`}>
				<span className="font-bold">B</span>
			</MarkToolbarButton>

			<MarkToolbarButton nodeType={KEYS.italic} tooltip={`Italic (${mod}+I)`}>
				<span className="italic">I</span>
			</MarkToolbarButton>

			<MarkToolbarButton nodeType={KEYS.underline} tooltip={`Underline (${mod}+U)`}>
				<span className="underline underline-offset-2">U</span>
			</MarkToolbarButton>

			<MarkToolbarButton nodeType={KEYS.strikethrough} tooltip={`Strikethrough (${mod}+${shift}+M)`}>
				<span className="line-through">S</span>
			</MarkToolbarButton>

			{/* New: Subscript */}
			<MarkToolbarButton nodeType={KEYS.sub} tooltip={`Subscript (${mod}+,)`}>
				<span aria-hidden>x₂</span>
			</MarkToolbarButton>

			{/* New: Superscript */}
			<MarkToolbarButton nodeType={KEYS.sup} tooltip={`Superscript (${mod}+.)`}>
				<span aria-hidden>x²</span>
			</MarkToolbarButton>

			{/* New: Highlight */}
			<MarkToolbarButton
				// If your highlight plugin key differs, replace KEYS.highlight with your plugin key.
				nodeType={KEYS.highlight}
				tooltip={`Highlight (${mod}+${shift}+H)`}
			>
				<span className="bg-warning/30 rounded px-1 leading-none">H</span>
			</MarkToolbarButton>

			<MarkToolbarButton nodeType={KEYS.code} tooltip={`Code (${mod}+E)`}>
				<span className="font-mono text-[0.9em]">{'</>'}</span>
			</MarkToolbarButton>
		</ToolbarGroup>
	);
}
