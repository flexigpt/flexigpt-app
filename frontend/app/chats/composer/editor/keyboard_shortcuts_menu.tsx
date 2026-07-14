import { useMemo } from 'react';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';
import { buildShortcutDisplay } from '@/lib/keyboard_shortcuts';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/hover_tip';

interface KeyboardShortcutsMenuProps {
	shortcutConfig: ShortcutConfig;
}

export function KeyboardShortcutsMenu({ shortcutConfig }: KeyboardShortcutsMenuProps) {
	const shortcutsMenu = useMenuStore({ placement: 'top-end', focusLoop: true });
	const shortcutsOpen = useStoreState(shortcutsMenu, 'open');

	const shortcutItems = useMemo(() => buildShortcutDisplay(shortcutConfig), [shortcutConfig]);
	const chatShortcuts = shortcutItems.filter(item => item.group === 'Chat');
	const composerShortcuts = shortcutItems.filter(item => item.group === 'Composer');

	return (
		<div>
			<HoverTip content="Keyboard shortcuts" placement="top">
				<MenuButton
					store={shortcutsMenu}
					className={`${actionTriggerChipButtonClasses} ${shortcutsOpen ? 'bg-base-300/80' : ''}`}
					aria-label="Keyboard shortcuts"
				>
					<ActionTriggerChipContent label="Shortcuts" open={shortcutsOpen} labelClassName="text-xs font-normal" />
				</MenuButton>
			</HoverTip>

			<Menu
				store={shortcutsMenu}
				gutter={8}
				overflowPadding={8}
				portal
				className={actionTriggerMenuWideClasses}
				autoFocusOnShow
			>
				{chatShortcuts.length > 0 && (
					<>
						<div className="app-text-neutral/70 px-3 pt-2 pb-1 text-xs tracking-wide uppercase">Chat shortcuts</div>
						{chatShortcuts.map(item => (
							<MenuItem key={item.action} hideOnClick={false} className={actionTriggerMenuItemClasses}>
								<span className="flex-1 text-left">{item.label}</span>
								<span className="app-text-neutral ml-auto w-30 text-left text-xs whitespace-nowrap">{item.keys}</span>
							</MenuItem>
						))}
					</>
				)}

				{composerShortcuts.length > 0 && (
					<>
						{chatShortcuts.length > 0 && <div className="border-base-200 mx-2 mt-1 border-t" />}
						<div className="app-text-neutral/70 px-3 pt-2 pb-1 text-xs tracking-wide uppercase">Composer shortcuts</div>
						{composerShortcuts.map(item => (
							<MenuItem key={item.action} hideOnClick={false} className={actionTriggerMenuItemClasses}>
								<span className="flex-1 text-left">{item.label}</span>
								<span className="app-text-neutral ml-auto w-30 text-left text-xs whitespace-nowrap">{item.keys}</span>
							</MenuItem>
						))}
					</>
				)}
			</Menu>
		</div>
	);
}
