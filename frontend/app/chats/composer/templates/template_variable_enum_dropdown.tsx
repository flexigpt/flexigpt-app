import {
	type CSSProperties,
	type KeyboardEvent as ReactKeyBoardEvent,
	type SyntheticEvent,
	useCallback,
	useEffect,
	useId,
	useLayoutEffect,
	useMemo,
	useRef,
	useState,
} from 'react';

import { createPortal } from 'react-dom';

import { FiCheck, FiChevronDown, FiChevronUp, FiX } from 'react-icons/fi';

type EnumDropdownInlineProps = {
	options: string[];
	value?: string;
	onChange: (val: string | undefined) => void;

	placeholder?: string;
	clearLabel?: string;
	clearable?: boolean;
	disabled?: boolean;
	size?: 'xs' | 'sm' | 'md';
	triggerClassName?: string;
	withinSlate?: boolean;
	autoOpen?: boolean;
	onCancel?: () => void;

	placement?: 'top' | 'bottom' | 'auto';
	minWidthPx?: number;
	menuMaxHeightPx?: number;
};

function getInitialActiveIndex(options: string[], value?: string): number {
	const selectedIdx = value ? options.findIndex(opt => opt === value) : -1;
	return selectedIdx >= 0 ? selectedIdx : 0;
}

function EnumDropdownInlineMenu({
	options,
	value,
	onChange,
	clearLabel,
	clearable,
	size,
	withinSlate,
	placement,
	minWidthPx,
	menuMaxHeightPx,
	triggerRef,
	onRequestClose,
}: {
	options: string[];
	value?: string;
	onChange: (val: string | undefined) => void;
	clearLabel: string;
	clearable: boolean;
	size: 'xs' | 'sm' | 'md';
	withinSlate: boolean;
	onCancel?: () => void;
	placement: 'top' | 'bottom' | 'auto';
	minWidthPx: number;
	menuMaxHeightPx: number;
	triggerRef: React.RefObject<HTMLButtonElement | null>;
	onRequestClose: (cancel?: boolean) => void;
}) {
	const menuRef = useRef<HTMLDivElement | null>(null);
	const optionRefs = useRef<(HTMLButtonElement | null)[]>([]);
	const [activeIndex, setActiveIndex] = useState<number>(() => getInitialActiveIndex(options, value));
	const listboxId = useId();

	const [style, setStyle] = useState<CSSProperties>({
		position: 'absolute',
		top: -9999,
		left: -9999,
		zIndex: 9999,
		visibility: 'hidden',
	});

	const stopForSlate = (e: SyntheticEvent) => {
		if (withinSlate) {
			e.preventDefault();
			e.stopPropagation();
		}
	};

	const updatePosition = useCallback(() => {
		const anchor = triggerRef.current;
		const menu = menuRef.current;
		if (!anchor || !menu) return;

		const prevVis = menu.style.visibility;
		const prevTop = menu.style.top;
		const prevLeft = menu.style.left;
		const prevPos = menu.style.position;

		menu.style.visibility = 'hidden';
		menu.style.position = 'absolute';
		menu.style.top = '0px';
		menu.style.left = '0px';

		const GAP = 6;
		const rect = anchor.getBoundingClientRect();
		const menuRect = menu.getBoundingClientRect();

		const spaceAbove = rect.top;
		const spaceBelow = window.innerHeight - rect.bottom;
		const preferTop = placement === 'top' || (placement === 'auto' && spaceAbove > spaceBelow);

		let top = preferTop ? rect.top + window.scrollY - menuRect.height - GAP : rect.bottom + window.scrollY + GAP;

		if (preferTop && top < window.scrollY + 8 && spaceBelow >= menuRect.height + GAP) {
			top = rect.bottom + window.scrollY + GAP;
		} else if (
			!preferTop &&
			top + menuRect.height > window.scrollY + window.innerHeight - 8 &&
			spaceAbove >= menuRect.height + GAP
		) {
			top = rect.top + window.scrollY - menuRect.height - GAP;
		}

		let left = rect.left + window.scrollX;
		const maxLeft = window.scrollX + window.innerWidth - menuRect.width - 8;
		if (left > maxLeft) left = Math.max(window.scrollX + 8, maxLeft);

		setStyle({
			position: 'absolute',
			top,
			left,
			minWidth: Math.max(minWidthPx, rect.width),
			zIndex: 9999,
			visibility: 'visible',
		});

		menu.style.visibility = prevVis;
		menu.style.position = prevPos;
		menu.style.top = prevTop;
		menu.style.left = prevLeft;
	}, [minWidthPx, placement, triggerRef]);

	useLayoutEffect(() => {
		updatePosition();
	}, [updatePosition]);

	useEffect(() => {
		requestAnimationFrame(() => {
			const btn = optionRefs.current[activeIndex];
			if (btn) {
				btn.focus();
			} else {
				try {
					menuRef.current?.focus();
				} catch {
					/* swallow */
				}
			}
		});
	}, [activeIndex]);

	useEffect(() => {
		const btn = optionRefs.current[activeIndex];
		try {
			btn?.scrollIntoView({ block: 'nearest' });
		} catch {
			// noop
		}
	}, [activeIndex]);

	useEffect(() => {
		const onReposition = () => {
			updatePosition();
		};

		const onKey = (e: KeyboardEvent) => {
			if (e.key === 'Escape') {
				e.preventDefault();
				onRequestClose(true);
			}
		};

		const onOutside = (e: MouseEvent | PointerEvent) => {
			const path = e.composedPath();
			const anchor = triggerRef.current;
			const menu = menuRef.current;
			if (!anchor || !menu) return;
			if (!path.includes(anchor) && !path.includes(menu)) {
				onRequestClose(true);
			}
		};

		window.addEventListener('scroll', onReposition, true);
		window.addEventListener('resize', onReposition);
		document.addEventListener('keydown', onKey, true);
		document.addEventListener('pointerdown', onOutside, true);

		return () => {
			window.removeEventListener('scroll', onReposition, true);
			window.removeEventListener('resize', onReposition);
			document.removeEventListener('keydown', onKey, true);
			document.removeEventListener('pointerdown', onOutside, true);
		};
	}, [onRequestClose, triggerRef, updatePosition]);

	const onMenuKeyDown = (e: ReactKeyBoardEvent) => {
		if (withinSlate) {
			e.stopPropagation();
		}

		const maxIdx = options.length - 1;

		if (e.key === 'ArrowDown') {
			e.preventDefault();
			const next = Math.min(maxIdx, activeIndex < 0 ? 0 : activeIndex + 1);
			setActiveIndex(next);
			requestAnimationFrame(() => optionRefs.current[next]?.focus());
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			const next = Math.max(0, activeIndex < 0 ? 0 : activeIndex - 1);
			setActiveIndex(next);
			requestAnimationFrame(() => optionRefs.current[next]?.focus());
		} else if (e.key === 'Home') {
			e.preventDefault();
			setActiveIndex(0);
			requestAnimationFrame(() => optionRefs.current[0]?.focus());
		} else if (e.key === 'End') {
			e.preventDefault();
			setActiveIndex(maxIdx);
			requestAnimationFrame(() => optionRefs.current[maxIdx]?.focus());
		} else if (e.key === 'Enter') {
			e.preventDefault();
			if (activeIndex >= 0 && activeIndex <= maxIdx) {
				const opt = options[activeIndex];
				onChange(opt);
				onRequestClose(false);
			}
		} else if (e.key === 'Escape') {
			e.preventDefault();
			onRequestClose(true);
		}
	};

	if (typeof document === 'undefined' || !document.body) {
		return null;
	}

	return createPortal(
		<div
			ref={menuRef}
			style={style}
			className="bg-base-100 rounded-xl border p-1 shadow"
			tabIndex={-1}
			onMouseDown={stopForSlate}
			onPointerDown={stopForSlate}
			onClick={stopForSlate}
			onKeyDown={onMenuKeyDown}
		>
			<ul
				role="listbox"
				id={listboxId}
				className="w-full p-1"
				style={{ maxHeight: menuMaxHeightPx, overflow: 'auto', minWidth: style.minWidth }}
			>
				{options.map((opt, idx) => {
					const isActive = (value ?? '') === opt;
					const isFocused = idx === activeIndex;

					return (
						<li key={opt} className="w-full">
							<button
								type="button"
								role="option"
								aria-selected={isActive}
								ref={el => {
									optionRefs.current[idx] = el;
								}}
								className={`w-full justify-start rounded-lg px-2 py-1 text-left font-normal ${
									size === 'xs' ? 'text-xs' : size === 'sm' ? 'text-sm' : 'text-base'
								} ${isFocused ? 'bg-base-200' : 'hover:bg-base-200'}`}
								onClick={e => {
									stopForSlate(e);
									onChange(opt);
									onRequestClose(false);
								}}
							>
								<div className="flex justify-between">
									<span className="truncate">{opt}</span>
									{isActive && <FiCheck size={12} className="ml-2 inline-block" />}
								</div>
							</button>
						</li>
					);
				})}
				{clearable && (
					<li className="w-full">
						<button
							type="button"
							className={`text-error hover:bg-base-200 flex w-full items-center justify-center gap-1 rounded-lg px-2 py-1 text-left ${
								size === 'xs' ? 'text-xs' : size === 'sm' ? 'text-sm' : 'text-base'
							}`}
							onClick={e => {
								stopForSlate(e);
								onChange(undefined);
								onRequestClose(false);
							}}
						>
							<FiX /> {clearLabel}
						</button>
					</li>
				)}
			</ul>
		</div>,
		document.body
	);
}

export function EnumDropdownInline({
	options,
	value,
	onChange,
	placeholder = '-- select --',
	clearLabel = 'Clear',
	clearable = true,
	disabled = false,
	size = 'xs',
	triggerClassName,
	withinSlate = false,
	autoOpen = false,
	onCancel,
	placement = 'auto',
	minWidthPx = 176,
	menuMaxHeightPx = 240,
}: EnumDropdownInlineProps) {
	const [open, setOpen] = useState<boolean>(() => autoOpen);
	const triggerRef = useRef<HTMLButtonElement | null>(null);

	const sizeCls = size === 'md' ? 'btn-md' : size === 'sm' ? 'btn-sm' : 'btn-xs';
	const defaultTrigger = `btn btn-ghost ${sizeCls} font-normal w-40 min-w-24 justify-between truncate bg-transparent`;

	const display = value === undefined || value === '' ? placeholder : value;

	const stopForSlate = (e: SyntheticEvent) => {
		if (withinSlate) {
			e.preventDefault();
			e.stopPropagation();
		}
	};

	const close = useCallback(
		(cancel = true) => {
			setOpen(false);

			requestAnimationFrame(() => {
				try {
					triggerRef.current?.focus();
				} catch {
					/* swallow */
				}
			});

			if (cancel) onCancel?.();
		},
		[onCancel]
	);

	useEffect(() => {
		if (!autoOpen) return;

		requestAnimationFrame(() => {
			triggerRef.current?.focus();
		});
	}, [autoOpen]);

	const menuKey = useMemo(() => `${options.join('\u0000')}|${value ?? ''}`, [options, value]);

	return (
		<>
			<button
				ref={triggerRef}
				type="button"
				className={triggerClassName ?? defaultTrigger}
				aria-haspopup="listbox"
				aria-expanded={open}
				aria-label="Open selection menu"
				title="Open selection"
				disabled={disabled}
				onMouseDown={stopForSlate}
				onPointerDown={stopForSlate}
				onClick={e => {
					stopForSlate(e);
					if (disabled) return;
					setOpen(prev => !prev);
				}}
				onKeyDown={e => {
					stopForSlate(e);
					if (disabled) return;

					if (e.key === 'Enter' || e.key === ' ') {
						e.preventDefault();
						setOpen(true);
					} else if (e.key === 'Escape') {
						e.preventDefault();
						close(true);
					} else if (e.key === 'ArrowDown') {
						e.preventDefault();
						setOpen(true);
					} else if (e.key === 'ArrowUp') {
						e.preventDefault();
						setOpen(true);
					}
				}}
			>
				<span className={`truncate font-normal ${size === 'xs' ? 'text-xs' : size === 'sm' ? 'text-sm' : 'text-base'}`}>
					{display}
				</span>
				{open ? <FiChevronDown size={12} /> : <FiChevronUp size={12} />}
			</button>

			{open && (
				<EnumDropdownInlineMenu
					key={menuKey}
					options={options}
					value={value}
					onChange={onChange}
					clearLabel={clearLabel}
					clearable={clearable}
					size={size}
					withinSlate={withinSlate}
					onCancel={onCancel}
					placement={placement}
					minWidthPx={minWidthPx}
					menuMaxHeightPx={menuMaxHeightPx}
					triggerRef={triggerRef}
					onRequestClose={close}
				/>
			)}
		</>
	);
}
