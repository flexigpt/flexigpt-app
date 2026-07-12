import type { CSSProperties, SyntheticEvent } from 'react';
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiCheck, FiChevronDown, FiChevronUp } from 'react-icons/fi';

/**
 * @public
 */
export interface DropdownItem {
	isEnabled: boolean;
}

/**
 * @public
 */
export interface DropdownProps<K extends string> {
	// The mapped dropdownItems (like modelPresets or aiSettings).
	dropdownItems: Record<K, DropdownItem>;
	//  The currently selected key (e.g. 'gpt-3.5' or 'openai').
	selectedKey: K;
	// Called when user picks a new key.
	onChange: (key: K) => void;
	// Optional. If you want to filter out items that
	// are not enabled unless they are the current selection,
	// set this to true. (Defaults to true).
	filterDisabled?: boolean;
	// Optional text that appears in a tooltip or as a title
	// on the summary element.
	title?: string;
	// Optional callback to get the display name for an item in the dropdown.
	// If the item itself has a `displayName` property, that takes precedence.
	// Otherwise, this function (if present) will be used to determine the label.
	getDisplayName?: (key: K) => string;
	maxMenuHeight?: number | string; // Optional, default 300
	orderedKeys?: readonly K[];
	placeholderLabel?: string;
	disabled?: boolean;
	/**
	 * Render the menu in normal document flow instead of as an absolute DaisyUI dropdown.
	 * Useful inside scrollable modals so opening the menu expands the modal naturally.
	 */
	inlineMenu?: boolean;
	maxSummaryHeight?: number | string;
}

// A single reusable dropdown that can be used by passing the appropriate config.
export const Dropdown = <K extends string>(props: DropdownProps<K>) => {
	const {
		dropdownItems,
		selectedKey,
		onChange,
		filterDisabled = true,
		title = 'Select an option',
		getDisplayName,
		maxMenuHeight = 300,
		orderedKeys,
		placeholderLabel = 'Select an option',
		disabled = false,
		inlineMenu = false,
		maxSummaryHeight,
	} = props;

	const [isOpen, setIsOpen] = useState(false);
	const [floatingStyle, setFloatingStyle] = useState<CSSProperties | null>(null);
	const [portalTarget, setPortalTarget] = useState<Element | null>(null);
	const detailsRef = useRef<HTMLDetailsElement>(null);
	const summaryRef = useRef<HTMLElement>(null);
	const menuRef = useRef<HTMLUListElement>(null);

	const closeMenu = useCallback(() => {
		if (detailsRef.current) {
			detailsRef.current.open = false;
		}
		setIsOpen(false);
		setFloatingStyle(null);
		setPortalTarget(null);
	}, []);

	const handleSelection = (key: K) => {
		const item = dropdownItems[key];
		if (!item?.isEnabled) {
			return;
		}
		onChange(key);
		closeMenu();
	};

	const updateFloatingPosition = useCallback(() => {
		if (inlineMenu || !isOpen || !summaryRef.current || typeof window === 'undefined') {
			return;
		}

		const rect = summaryRef.current.getBoundingClientRect();
		const viewportPadding = 8;
		const menuGap = 6;
		const requestedHeight = typeof maxMenuHeight === 'number' ? maxMenuHeight : 300;
		const spaceBelow = window.innerHeight - rect.bottom - viewportPadding - menuGap;
		const spaceAbove = rect.top - viewportPadding - menuGap;
		const openAbove = spaceBelow < Math.min(180, requestedHeight) && spaceAbove > spaceBelow;
		const availableHeight = Math.max(120, openAbove ? spaceAbove : spaceBelow);
		const width = Math.min(Math.max(rect.width, 220), window.innerWidth - viewportPadding * 2);
		const left = Math.min(
			Math.max(viewportPadding, rect.left),
			Math.max(viewportPadding, window.innerWidth - width - viewportPadding)
		);

		setFloatingStyle({
			position: 'fixed',
			left,
			width,
			top: openAbove ? undefined : rect.bottom + menuGap,
			bottom: openAbove ? window.innerHeight - rect.top + menuGap : undefined,
			maxHeight: Math.min(requestedHeight, availableHeight),
			zIndex: 1000,
		});
	}, [inlineMenu, isOpen, maxMenuHeight]);

	useLayoutEffect(() => {
		updateFloatingPosition();
	}, [updateFloatingPosition]);

	useEffect(() => {
		if (!isOpen) {
			return;
		}

		const handleOutsideInteraction = (event: MouseEvent | FocusEvent) => {
			const target = event.target;
			if (!(target instanceof Node)) {
				return;
			}
			if (detailsRef.current?.contains(target) || menuRef.current?.contains(target)) {
				return;
			}
			closeMenu();
		};

		const handleKeyDown = (event: KeyboardEvent) => {
			if (event.key === 'Escape') {
				event.preventDefault();
				closeMenu();
				summaryRef.current?.focus();
			}
		};

		document.addEventListener('mousedown', handleOutsideInteraction);
		document.addEventListener('focusin', handleOutsideInteraction);
		document.addEventListener('keydown', handleKeyDown);
		window.addEventListener('resize', updateFloatingPosition);
		document.addEventListener('scroll', updateFloatingPosition, true);

		return () => {
			document.removeEventListener('mousedown', handleOutsideInteraction);
			document.removeEventListener('focusin', handleOutsideInteraction);
			document.removeEventListener('keydown', handleKeyDown);
			window.removeEventListener('resize', updateFloatingPosition);
			document.removeEventListener('scroll', updateFloatingPosition, true);
		};
	}, [closeMenu, isOpen, updateFloatingPosition]);

	const getItemDisplayName = (key: K) => {
		if (typeof getDisplayName === 'function') {
			return getDisplayName(key);
		}
		return key;
	};

	const sourceKeys = (orderedKeys ?? (Object.keys(dropdownItems) as K[])).filter(
		key => dropdownItems[key] !== undefined
	);

	const filteredKeys = sourceKeys.filter(typedKey => {
		const item = dropdownItems[typedKey];
		if (!item) {
			return false;
		}
		if (!filterDisabled) {
			return true;
		}
		return item.isEnabled || typedKey === selectedKey;
	});

	const menu = (
		<ul
			ref={menuRef}
			role="listbox"
			aria-label={title}
			className={`menu border-neutral/20 bg-base-300 flex w-full flex-col flex-nowrap overflow-x-hidden overflow-y-auto rounded-2xl border shadow-lg ${
				inlineMenu ? 'mt-2' : ''
			}`}
			style={
				inlineMenu
					? {
							maxHeight: typeof maxMenuHeight === 'number' ? `${maxMenuHeight}px` : maxMenuHeight,
						}
					: (floatingStyle ?? undefined)
			}
		>
			{filteredKeys.map(key => {
				const item = dropdownItems[key];
				const isItemDisabled = disabled || !item?.isEnabled;

				return (
					<li key={key} className="w-full">
						<button
							type="button"
							role="option"
							aria-selected={key === selectedKey}
							disabled={isItemDisabled}
							className="m-1 flex w-[calc(100%-0.5rem)] min-w-0 items-center justify-between gap-2 rounded-xl p-2 text-left disabled:cursor-not-allowed disabled:opacity-50"
							onClick={() => {
								handleSelection(key);
							}}
						>
							<span className="min-w-0 truncate">{getItemDisplayName(key)}</span>
							{key === selectedKey ? <FiCheck className="shrink-0" /> : null}
						</button>
					</li>
				);
			})}

			{filteredKeys.length === 0 ? (
				<li className="text-base-content/60 px-3 py-2 text-sm">{placeholderLabel}</li>
			) : null}
		</ul>
	);

	return (
		<details
			ref={detailsRef}
			className="relative w-full"
			onToggle={(event: SyntheticEvent<HTMLElement>) => {
				const details = event.currentTarget as HTMLDetailsElement;
				if (disabled && details.open) {
					details.open = false;
					setIsOpen(false);
					setPortalTarget(null);
					return;
				}
				setIsOpen(details.open);
				if (details.open && !inlineMenu && typeof document !== 'undefined') {
					setPortalTarget(details.closest('dialog') ?? document.body);
				} else {
					setFloatingStyle(null);
					setPortalTarget(null);
				}
			}}
		>
			<summary
				ref={summaryRef}
				className={`btn border-neutral/20 bg-base-200 flex w-full items-center justify-between rounded-2xl px-4 py-2 text-left shadow-none ${
					disabled ? 'cursor-default opacity-70' : 'cursor-pointer'
				}`}
				title={title}
				aria-expanded={isOpen}
				aria-disabled={disabled}
				tabIndex={disabled ? -1 : 0}
				aria-haspopup="listbox"
				onClick={event => {
					if (disabled) {
						event.preventDefault();
					}
				}}
				style={{
					maxHeight: typeof maxSummaryHeight === 'number' ? `${maxSummaryHeight}px` : maxSummaryHeight,
				}}
			>
				<span className="truncate font-normal">{selectedKey ? getItemDisplayName(selectedKey) : placeholderLabel}</span>
				{isOpen ? <FiChevronUp size={16} /> : <FiChevronDown size={16} />}
			</summary>

			{inlineMenu && isOpen ? menu : null}
			{!inlineMenu && isOpen && floatingStyle && portalTarget ? createPortal(menu, portalTarget) : null}
		</details>
	);
};
