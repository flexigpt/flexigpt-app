import { useMemo } from 'react';

import { FiMoreHorizontal } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, Tooltip, useMenuStore, useStoreState, useTooltipStore } from '@ariakit/react';

import { buildShortcutDisplay, type ShortcutConfig } from '@/lib/keyboard_shortcuts';

import { actionTriggerChipButtonClasses, ActionTriggerChipContent } from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';

type TipKey = 'lastWins' | 'remove' | 'collapse' | 'attachmentsScope' | 'toolsPersist' | 'toolErrors';
const tipsText: Record<TipKey, string> = {
	// Original system‑prompt/template tips
	lastWins: 'Last system prompt wins: when multiple templates add system prompts, the last one added becomes active.',
	remove: 'Removing a template also removes the system prompt that came from it (if it was created by that template).',
	collapse:
		'Collapse to text removes system prompts, variables, and tools. The current user block with placeholders for variables is inserted as plain text.',

	// New attachment/tool behavior tips
	attachmentsScope:
		'Attachments you add to a message are available to the model when that message is sent (and persist in history like any other message).',
	toolsPersist:
		'Once you attach tools to a conversation, those tool choices are preserved and sent with every turn until you change them.',
	toolErrors:
		'When a tool run returns an application error, its chip turns red. You can retry it or send the error output to the model explicitly.',
};

interface CommandTipsMenuProps {
	shortcutConfig: ShortcutConfig;
}

const menuClasses =
	'rounded-box bg-base-100 text-base-content z-50 max-w-lg min-w-60 overflow-visible border border-base-300 p-1 shadow-xl';

const menuItemClasses =
	'flex items-center gap-2 rounded-xl px-2 py-1 text-xs outline-none transition-colors ' +
	'hover:bg-base-200 data-[active-item]:bg-base-300';

export function CommandTipsMenu({ shortcutConfig }: CommandTipsMenuProps) {
	const shortcutsMenu = useMenuStore({ placement: 'top-end', focusLoop: true });
	const tipsMenu = useMenuStore({ placement: 'top-end', focusLoop: true });

	const shortcutsOpen = useStoreState(shortcutsMenu, 'open');
	const tipsOpen = useStoreState(tipsMenu, 'open');

	// Shared tooltip for "Additional input tips" items.
	// placement: 'left-end' -> tooltip is to the LEFT, with its BOTTOM aligned to the item.
	const tipsTooltip = useTooltipStore({ placement: 'left-end' });
	const tooltipAnchorEl = useStoreState(tipsTooltip, 'anchorElement');
	const currentTipDescription = tooltipAnchorEl?.dataset.tipDescription ?? '';

	// All tips shown under "Additional input tips"
	const tips = useMemo(
		() => [
			{
				key: 'lastWins' as TipKey,
				title: 'Last system prompt wins',
				description: tipsText.lastWins,
			},
			{
				key: 'remove' as TipKey,
				title: 'Removing a template',
				description: tipsText.remove,
			},
			{
				key: 'collapse' as TipKey,
				title: 'Collapse to text decouples',
				description: tipsText.collapse,
			},
			{
				key: 'attachmentsScope' as TipKey,
				title: 'Attachments apply to all turns',
				description: tipsText.attachmentsScope,
			},
			{
				key: 'toolsPersist' as TipKey,
				title: 'Tool choices persist per conversation',
				description: tipsText.toolsPersist,
			},
			{
				key: 'toolErrors' as TipKey,
				title: 'Errored tool results',
				description: tipsText.toolErrors,
			},
		],
		[]
	);

	// All shortcuts (global + insert) from central config
	const shortcutItems = useMemo(() => buildShortcutDisplay(shortcutConfig), [shortcutConfig]);

	const chatShortcuts = shortcutItems.filter(item => item.group === 'Chat');
	const insertShortcuts = shortcutItems.filter(item => item.group === 'Insert');

	return (
		<div className="flex items-center gap-1">
			<HoverTip content="Keyboard shortcuts" placement="top">
				<MenuButton
					store={shortcutsMenu}
					className={`${actionTriggerChipButtonClasses} ${shortcutsOpen ? 'bg-base-300/80' : ''}`}
					aria-label="Keyboard shortcuts"
				>
					<ActionTriggerChipContent label="Shortcuts" open={shortcutsOpen} labelClassName="text-xs font-normal" />
				</MenuButton>
			</HoverTip>

			<Menu store={shortcutsMenu} gutter={8} overflowPadding={8} portal className={menuClasses} autoFocusOnShow>
				{/* Chat group */}
				{chatShortcuts.length > 0 && (
					<>
						<div className="text-neutral-custom/70 px-3 pt-2 pb-1 text-xs tracking-wide uppercase">Chat shortcuts</div>
						{chatShortcuts.map(item => (
							<MenuItem key={item.action} hideOnClick={false} className={menuItemClasses}>
								<span className="flex-1 text-left">{item.label}</span>
								<span className="text-neutral-custom ml-auto w-22 text-left text-xs whitespace-nowrap">
									{item.keys}
								</span>
							</MenuItem>
						))}
					</>
				)}

				{/* Insert group */}
				{insertShortcuts.length > 0 && (
					<>
						{chatShortcuts.length > 0 && <div className="border-base-200 mx-2 mt-1 border-t" />}
						<div className="text-neutral-custom/70 px-3 pt-2 pb-1 text-xs tracking-wide uppercase">
							Insert shortcuts
						</div>
						{insertShortcuts.map(item => (
							<MenuItem key={item.action} hideOnClick={false} className={menuItemClasses}>
								<span className="flex-1 text-left">{item.label}</span>
								<span className="text-neutral-custom ml-auto w-22 text-left text-xs whitespace-nowrap">
									{item.keys}
								</span>
							</MenuItem>
						))}
					</>
				)}
			</Menu>

			<HoverTip content="Additional input tips" placement="top">
				<MenuButton
					store={tipsMenu}
					className={`${actionTriggerChipButtonClasses} ${tipsOpen ? 'bg-base-300/80' : ''}`}
					aria-label="Additional input tips"
				>
					<ActionTriggerChipContent label="Input tips" open={tipsOpen} labelClassName="text-xs font-normal" />
				</MenuButton>
			</HoverTip>

			<Menu
				store={tipsMenu}
				gutter={8}
				overflowPadding={8}
				portal
				className={menuClasses}
				autoFocusOnShow
				// Explicitly handle Escape at the menu level (capture phase),
				// so it always closes the menu & tooltip and returns focus to the button.
				onKeyDownCapture={event => {
					if (event.key === 'Escape') {
						tipsTooltip.hide();
						tipsMenu.hide();
					}
				}}
			>
				{tips.map(tip => (
					<MenuItem
						key={tip.key}
						hideOnClick={false}
						className={menuItemClasses}
						data-tip-description={tip.description}
						onFocus={event => {
							// Focus via keyboard (arrow keys / Tab) -> show tooltip
							tipsTooltip.setAnchorElement(event.currentTarget);
							tipsTooltip.show();
						}}
						onBlur={() => {
							// Leaving item (focus out) -> hide tooltip
							tipsTooltip.hide();
							tipsTooltip.setAnchorElement(null);
						}}
						onMouseEnter={event => {
							// Hover with mouse -> show tooltip
							tipsTooltip.setAnchorElement(event.currentTarget);
							tipsTooltip.show();
						}}
						onMouseLeave={() => {
							// Leave with mouse -> hide tooltip
							tipsTooltip.hide();
						}}
					>
						<div className="flex items-center gap-2">
							<span className="text-left font-medium">{tip.title}</span>
							<FiMoreHorizontal size={14} className="ml-auto opacity-60" aria-hidden="true" />
						</div>
					</MenuItem>
				))}
			</Menu>

			{/* Tooltip for "Additional input tips" items.
          - Opens on hover AND keyboard focus (arrow keys).
          - Portaled so it escapes the menu box.
          - Positioned to the left with BOTTOM aligned via placement: 'left-end'. */}
			<Tooltip
				store={tipsTooltip}
				portal
				className="rounded-box bg-base-100 text-base-content border-base-300 max-w-xs border p-2 text-xs shadow-xl"
			>
				{currentTipDescription}
			</Tooltip>
		</div>
	);
}
