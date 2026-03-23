import { type SyntheticEvent, useRef, useState } from 'react';

import { FiCheck, FiChevronDown, FiChevronUp } from 'react-icons/fi';

import { useCloseDetails } from '@/hooks/use_close_details';

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
	} = props;

	const [isOpen, setIsOpen] = useState(false);
	const detailsRef = useRef<HTMLDetailsElement>(null);

	// Close the details dropdown if the user clicks outside.
	useCloseDetails({
		detailsRef,
		events: ['mousedown'],
		onClose: () => {
			setIsOpen(false);
		},
	});

	const handleSelection = (key: K) => {
		const item = dropdownItems[key];
		if (!item?.isEnabled) {
			return;
		}
		onChange(key);
		if (detailsRef.current) {
			// Force-close the details dropdown.
			detailsRef.current.open = false;
		}
		setIsOpen(false);
	};

	// Helper function to derive display name:
	// 1. Use the consumer's getDisplayName callback if provided
	// 2. Otherwise, fall back to the key itself.
	const getItemDisplayName = (key: K) => {
		if (typeof getDisplayName === 'function') {
			return getDisplayName(key);
		}
		return key;
	};

	// We can optionally filter out disabled items, unless they are already selected.
	const sourceKeys = (orderedKeys ?? (Object.keys(dropdownItems) as K[])).filter(
		key => dropdownItems[key] !== undefined
	);

	const filteredKeys = sourceKeys.filter(typedKey => {
		const item = dropdownItems[typedKey];
		if (!item) return false;
		if (!filterDisabled) return true;
		// If the item is enabled or it is the selected key, we keep it.
		return item.isEnabled || typedKey === selectedKey;
	});

	return (
		<details
			ref={detailsRef}
			className="dropdown relative w-full"
			onToggle={(event: SyntheticEvent<HTMLElement>) => {
				const details = event.currentTarget as HTMLDetailsElement;
				if (disabled && details.open) {
					details.open = false;
					setIsOpen(false);
					return;
				}
				setIsOpen(details.open);
			}}
		>
			<summary
				className={`btn border-neutral/20 bg-base-200 flex w-full items-center justify-between rounded-2xl px-4 py-2 text-left shadow-none ${
					disabled ? 'cursor-default opacity-70' : 'cursor-pointer'
				}`}
				title={title}
				onClick={event => {
					if (disabled) {
						event.preventDefault();
					}
				}}
			>
				<span className="truncate font-normal">{selectedKey ? getItemDisplayName(selectedKey) : placeholderLabel}</span>
				{isOpen ? <FiChevronUp size={16} /> : <FiChevronDown size={16} />}
			</summary>

			<ul
				tabIndex={0}
				className="dropdown-content menu border-neutral/20 bg-base-300 flex w-full flex-col flex-nowrap overflow-x-hidden overflow-y-auto rounded-2xl shadow-sm"
				style={{
					maxHeight: typeof maxMenuHeight === 'number' ? `${maxMenuHeight}px` : maxMenuHeight,
				}}
			>
				{filteredKeys.map(key => {
					const item = dropdownItems[key];
					const isItemDisabled = disabled || !item?.isEnabled;

					return (
						<li
							key={key}
							className={`w-full rounded-2xl ${isItemDisabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'}`}
							onClick={() => {
								if (!isItemDisabled) {
									handleSelection(key);
								}
							}}
						>
							<a
								className="m-1 flex items-center justify-between p-2"
								aria-disabled={isItemDisabled}
								onClick={event => {
									event.preventDefault();
								}}
							>
								<span className="truncate">{getItemDisplayName(key)}</span>
								{key === selectedKey && <FiCheck />}
							</a>
						</li>
					);
				})}
			</ul>
		</details>
	);
};
