import { useEffect, useRef } from 'react';

import { MOD_LABEL } from '@/lib/features';

interface ShortcutChord {
	key: string; // e.g. 'k'
	ctrlOrMeta?: boolean; // true = requires Ctrl or Cmd
	alt?: boolean;
	shift?: boolean;
}

export enum ShortcutAction {
	// App / chat-level
	newChat = 'newChat',
	closeChat = 'closeChat',
	nextChat = 'nextChat',
	previousChat = 'previousChat',

	focusSearch = 'focusSearch',
	focusInput = 'focusInput',
	insertTemplate = 'insertTemplate',
	insertTool = 'insertTool',
	insertAttachment = 'insertAttachment',

	// Editor – marks
	editorBold = 'editorBold',
	editorItalic = 'editorItalic',
	editorUnderline = 'editorUnderline',
	editorStrikethrough = 'editorStrikethrough',
	editorHighlight = 'editorHighlight',
	editorCode = 'editorCode',

	// Editor – blocks
	editorHeading = 'editorHeading',
	editorBlockquote = 'editorBlockquote',

	// Editor – other
	editorEmoji = 'editorEmoji',
	editorFloatingToolbar = 'editorFloatingToolbar',
	editorUndo = 'editorUndo',
	editorRedo = 'editorRedo',
}

interface ShortcutKeyEvent {
	key: string;
	ctrlKey: boolean;
	metaKey: boolean;
	altKey: boolean;
	shiftKey: boolean;
}

export type ShortcutConfig = Partial<Record<ShortcutAction, ShortcutChord>>;

type ShortcutGroup = 'Chat' | 'Insert' | 'Editor';

/**
 * @public
 */
export interface ShortcutDisplayItem {
	action: ShortcutAction;
	/** Short, human-friendly action name */
	label: string;
	/**
	 * Formatted key combo or trigger description,
	 * e.g. "⌘ + Shift + 'P'" or `Type ":" in the editor`
	 */
	keys: string;
	/** Optional grouping for the UI */
	group: ShortcutGroup;
	/** For stable ordering */
	order: number;
}

// Default mappings; you can later load these from settings.
// NOTE: This contains only *app-level* shortcuts that we handle ourselves.
// Plate editor shortcuts are displayed, but handled internally by Plate.
export const defaultShortcutConfig: ShortcutConfig = {
	// Global-chat actions
	[ShortcutAction.newChat]: { key: 't', ctrlOrMeta: true, shift: false }, // Mod+T
	[ShortcutAction.closeChat]: { key: 'w', ctrlOrMeta: true, shift: false }, // Mod+W

	[ShortcutAction.previousChat]: { key: 'PageUp', ctrlOrMeta: true, shift: false },
	[ShortcutAction.nextChat]: { key: 'PageDown', ctrlOrMeta: true, shift: false },

	[ShortcutAction.focusSearch]: { key: 'f', ctrlOrMeta: true, shift: true }, // Mod+Shift+F
	[ShortcutAction.focusInput]: { key: 'i', ctrlOrMeta: true, shift: true }, // Mod+Shift+I

	// Editor/insert actions
	[ShortcutAction.insertTemplate]: { key: 'p', ctrlOrMeta: true, shift: true }, // Mod+Shift+P
	[ShortcutAction.insertAttachment]: { key: 'a', ctrlOrMeta: true, shift: true }, // Mod+Shift+A
	[ShortcutAction.insertTool]: { key: 'k', ctrlOrMeta: true, shift: true }, // Mod+Shift+k
};

type ConfigShortcutMeta = {
	label: string;
	group: ShortcutGroup;
	order: number;
	/** Backed by ShortcutConfig; key combo is configurable */
	source: 'config';
};

type StaticShortcutMeta = {
	label: string;
	group: ShortcutGroup;
	order: number;
	/** Fixed key combo / trigger, handled internally by Plate */
	source: 'static';
	keys: string;
};

type ShortcutMeta = ConfigShortcutMeta | StaticShortcutMeta;

/**
 * Metadata for all shortcuts we want to show in the UI.
 *
 * - `source: 'config'`  → app-level shortcuts, driven by `ShortcutConfig` and `useChatShortcuts`
 * - `source: 'static'` → Plate editor shortcuts (display only; Plate handles them)
 */
const ACTION_META: Record<ShortcutAction, ShortcutMeta> = {
	// CHAT / APP-LEVEL SHORTCUTS (config-backed)
	[ShortcutAction.newChat]: {
		label: 'New chat',
		group: 'Chat',
		order: 10,
		source: 'config',
	},
	[ShortcutAction.closeChat]: {
		label: 'Close chat',
		group: 'Chat',
		order: 20,
		source: 'config',
	},
	[ShortcutAction.nextChat]: {
		label: 'Next chat',
		group: 'Chat',
		order: 30,
		source: 'config',
	},
	[ShortcutAction.previousChat]: {
		label: 'Previous chat',
		group: 'Chat',
		order: 40,
		source: 'config',
	},
	[ShortcutAction.focusSearch]: {
		label: 'Focus search',
		group: 'Chat',
		order: 50,
		source: 'config',
	},
	[ShortcutAction.focusInput]: {
		label: 'Focus input',
		group: 'Chat',
		order: 60,
		source: 'config',
	},
	[ShortcutAction.insertTemplate]: {
		label: 'Insert template',
		group: 'Insert',
		order: 100,
		source: 'config',
	},
	[ShortcutAction.insertTool]: {
		label: 'Add tool',
		group: 'Insert',
		order: 110,
		source: 'config',
	},
	[ShortcutAction.insertAttachment]: {
		label: 'Attach context',
		group: 'Insert',
		order: 120,
		source: 'config',
	},

	// EDITOR – MARKS (Plate basic marks kit)
	// BoldPlugin (default: mod+b)
	[ShortcutAction.editorBold]: {
		label: 'Bold',
		group: 'Editor',
		order: 200,
		source: 'static',
		keys: `${MOD_LABEL} + 'B'`,
	},
	// ItalicPlugin (default: mod+i)
	[ShortcutAction.editorItalic]: {
		label: 'Italic',
		group: 'Editor',
		order: 210,
		source: 'static',
		keys: `${MOD_LABEL} + 'I'`,
	},
	// UnderlinePlugin (default: mod+u)
	[ShortcutAction.editorUnderline]: {
		label: 'Underline',
		group: 'Editor',
		order: 220,
		source: 'static',
		keys: `${MOD_LABEL} + 'U'`,
	},
	// StrikethroughPlugin.configure({ shortcuts: { toggle: { keys: 'mod+shift+x' } } })
	[ShortcutAction.editorStrikethrough]: {
		label: 'Strikethrough',
		group: 'Editor',
		order: 230,
		source: 'static',
		keys: `${MOD_LABEL} + Shift + 'X'`,
	},
	// HighlightPlugin.configure({ shortcuts: { toggle: { keys: 'mod+shift+h' } } })
	[ShortcutAction.editorHighlight]: {
		label: 'Highlight',
		group: 'Editor',
		order: 240,
		source: 'static',
		keys: `${MOD_LABEL} + Shift + 'H'`,
	},
	// CodePlugin.configure({ shortcuts: { toggle: { keys: 'mod+e' } } })
	[ShortcutAction.editorCode]: {
		label: 'Inline code',
		group: 'Editor',
		order: 250,
		source: 'static',
		keys: `${MOD_LABEL} + 'E'`,
	},

	// EDITOR – BLOCKS (Plate basic blocks kit)
	// H1Plugin.configure({ shortcuts: { toggle: { keys: 'mod+alt+1' } } })
	[ShortcutAction.editorHeading]: {
		label: 'Heading 1 to 6',
		group: 'Editor',
		order: 300,
		source: 'static',
		keys: `${MOD_LABEL} + Alt + '1 to 6'`,
	},
	// BlockquotePlugin.configure({ shortcuts: { toggle: { keys: 'mod+shift+period' } } })
	[ShortcutAction.editorBlockquote]: {
		label: 'Blockquote',
		group: 'Editor',
		order: 310,
		source: 'static',
		keys: `${MOD_LABEL} + Shift + '.'`,
	},

	// EDITOR – OTHER (Emoji, floating toolbar, undo/redo, etc.)
	// EmojiKit – type ":" to open emoji suggestions
	[ShortcutAction.editorEmoji]: {
		label: 'Emoji picker',
		group: 'Editor',
		order: 320,
		source: 'static',
		keys: `Type ":" in the editor`,
	},
	// FloatingToolbarKit – appears when you select text
	[ShortcutAction.editorFloatingToolbar]: {
		label: 'Floating toolbar',
		group: 'Editor',
		order: 330,
		source: 'static',
		keys: 'Select text',
	},
	// Undo / Redo (Plate/Slate defaults, plus your note: Ctrl/Cmd+Z/Y)
	[ShortcutAction.editorUndo]: {
		label: 'Undo',
		group: 'Editor',
		order: 340,
		source: 'static',
		keys: `${MOD_LABEL} + 'Z'`,
	},
	[ShortcutAction.editorRedo]: {
		label: 'Redo',
		group: 'Editor',
		order: 350,
		source: 'static',
		keys: `${MOD_LABEL} + 'Y'`,
	},
};

interface UseShortcutsOptions {
	config: ShortcutConfig;
	isBusy: boolean;
	handlers: Partial<Record<ShortcutAction, () => void>>;
}

function matchShortcut(event: ShortcutKeyEvent, shortcut?: ShortcutChord): boolean {
	if (!shortcut) return false;
	if (shortcut.ctrlOrMeta && !(event.ctrlKey || event.metaKey)) return false;
	if (!shortcut.ctrlOrMeta && (event.ctrlKey || event.metaKey)) return false;
	if (shortcut.alt !== undefined && shortcut.alt !== event.altKey) return false;
	if (shortcut.shift !== undefined && shortcut.shift !== event.shiftKey) return false;
	return event.key.toLowerCase() === shortcut.key.toLowerCase();
}

export function formatShortcut(shortcut?: ShortcutChord): string {
	if (!shortcut) return '';
	const parts: string[] = [];
	if (shortcut.ctrlOrMeta) parts.push(MOD_LABEL);
	if (shortcut.alt) parts.push('Alt');
	if (shortcut.shift) parts.push('Shift');
	parts.push(`'${shortcut.key}'`);
	return parts.join(' + ');
}

/**
 * Single global shortcuts hook. Attach once at the chat page level.
 * It uses the config map so you can swap mappings in the future.
 *
 * NOTE: This only handles actions that are actually present in `config`
 * (i.e. the app-level shortcuts). Plate editor shortcuts are **not**
 * wired through this hook; they are handled by Plate internally.
 */
export function useChatShortcuts({ config, isBusy, handlers }: UseShortcutsOptions) {
	const ref = useRef({ config, isBusy, handlers });

	useEffect(() => {
		ref.current = { config, isBusy, handlers };
	}, [config, isBusy, handlers]);

	useEffect(() => {
		const onKeyDown = (event: KeyboardEvent) => {
			if (event.defaultPrevented) return;

			// IME composition safety
			if (event.isComposing) return;

			// Allow specific widgets (rename inputs, etc.) to opt out of global shortcuts.
			// Important since we listen in capture phase.
			const target = event.target as HTMLElement | null;

			if (target?.closest?.('[data-disable-chat-shortcuts="true"]')) return;

			/**
			 * Prevent global app shortcuts from firing while the user is interacting
			 * with menus, listboxes, or dialogs. This avoids breaking keyboard navigation
			 * (e.g. Mod+W closing a tab while a menu is open, shortcuts firing inside modals).
			 *
			 * Note: We intentionally DO NOT block shortcuts inside the main Plate editor
			 * (contenteditable) because insert shortcuts are expected to work there.
			 */

			const inMenuOrDialog = !!target?.closest?.(
				[
					// Native dialogs (your modals)
					'dialog',
					'[role="dialog"]',
					// Ariakit menus
					'[role="menu"]',
					'[role="menuitem"]',
					// Other common popup widgets you use (EnumDropdownInline uses listbox/option)
					'[role="listbox"]',
					'[role="option"]',
				].join(', ')
			);
			if (inMenuOrDialog) return;

			const { config, isBusy, handlers } = ref.current;
			const entries = Object.entries(config) as [ShortcutAction, ShortcutChord][];

			for (const [action, chord] of entries) {
				const handler = handlers[action];
				if (!handler) continue;

				if (!matchShortcut(event as ShortcutKeyEvent, chord)) continue;

				// Example of busy-sensitive behavior: no new chat while busy
				if (isBusy && action === ShortcutAction.newChat) return;

				event.preventDefault();
				event.stopPropagation();

				handler();
				return;
			}
		};

		window.addEventListener('keydown', onKeyDown, true);
		return () => {
			window.removeEventListener('keydown', onKeyDown, true);
		};
	}, []);
}

/**
 * Build a flat list of display items for all shortcuts.
 *
 * - Includes configurable app-level shortcuts (from `config`).
 * - Also includes static Plate editor shortcuts (marks, blocks, emoji, etc.).
 *
 * This is what UI components should consume for a "Keyboard shortcuts" dialog.
 */
export function buildShortcutDisplay(config: ShortcutConfig): ShortcutDisplayItem[] {
	const items: ShortcutDisplayItem[] = [];

	(Object.entries(ACTION_META) as [ShortcutAction, ShortcutMeta][]).forEach(([action, meta]) => {
		if (meta.source === 'config') {
			const chord = config[action];
			if (!chord) return;
			const keys = formatShortcut(chord);
			if (!keys) return;

			items.push({
				action,
				label: meta.label,
				group: meta.group,
				order: meta.order,
				keys,
			});
		} else {
			// Static Plate/inline shortcuts – always shown
			items.push({
				action,
				label: meta.label,
				group: meta.group,
				order: meta.order,
				keys: meta.keys,
			});
		}
	});

	// Stable order
	items.sort((a, b) => a.order - b.order);
	return items;
}
