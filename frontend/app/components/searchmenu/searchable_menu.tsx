import type { KeyboardEvent, RefObject } from 'react';
import { useEffect, useMemo, useRef } from 'react';

import { FiSearch, FiX } from 'react-icons/fi';

interface SearchableMenuInputProps {
	open: boolean;
	query: string;
	onQueryChange: (next: string) => void;
	placeholder?: string;
	inputRef?: RefObject<HTMLInputElement | null>;
	resultCount?: number;
	totalCount?: number;
	disabled?: boolean;
	onFocusFirstItem?: () => void;
	onEnterFirstResult?: () => void;
	onEscape?: () => void;
	className?: string;
}

export const searchableMenuEmptyStateClasses = 'text-base-content/60 rounded-xl px-2 py-1 text-xs';

export function SearchableMenuInput({
	open,
	query,
	onQueryChange,
	placeholder = 'Search…',
	inputRef,
	resultCount,
	totalCount,
	disabled = false,
	onFocusFirstItem,
	onEnterFirstResult,
	onEscape,
	className = '',
}: SearchableMenuInputProps) {
	const localInputRef = useRef<HTMLInputElement | null>(null);
	const resolvedInputRef = inputRef ?? localInputRef;
	const hasQuery = query.trim().length > 0;

	useEffect(() => {
		if (!open || disabled || typeof window === 'undefined') {
			return;
		}

		const raf = window.requestAnimationFrame(() => {
			const input = resolvedInputRef.current;
			if (!input) {
				return;
			}

			input.focus({ preventScroll: true });
			input.select();
		});

		return () => {
			window.cancelAnimationFrame(raf);
		};
	}, [disabled, open, resolvedInputRef]);

	const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
		event.stopPropagation();

		if (event.key === 'ArrowDown') {
			event.preventDefault();
			onFocusFirstItem?.();
			return;
		}

		if (event.key === 'Enter') {
			event.preventDefault();
			onEnterFirstResult?.();
			return;
		}

		if (event.key === 'Escape') {
			event.preventDefault();
			if (hasQuery) {
				onQueryChange('');
				return;
			}
			onEscape?.();
		}
	};

	const countLabel = useMemo(() => {
		if (resultCount === undefined || totalCount === undefined) {
			return null;
		}
		return `${resultCount}/${totalCount}`;
	}, [resultCount, totalCount]);

	return (
		<div className={`bg-base-100 sticky top-0 z-10 mb-2 pb-2 ${className}`}>
			<label className="relative block">
				<FiSearch
					size={12}
					className="text-base-content/50 pointer-events-none absolute top-1/2 left-2 -translate-y-1/2"
				/>
				<input
					ref={resolvedInputRef}
					type="search"
					className="input input-xs bg-base-200 border-base-300 h-7 w-full rounded-xl pr-8 pl-7 text-xs outline-none"
					value={query}
					disabled={disabled}
					placeholder={placeholder}
					autoComplete="off"
					autoCorrect="off"
					spellCheck={false}
					aria-label={placeholder}
					data-searchable-menu-input="true"
					onChange={event => {
						onQueryChange(event.currentTarget.value);
					}}
					onKeyDown={handleKeyDown}
					onClick={event => {
						event.stopPropagation();
					}}
				/>
				{hasQuery ? (
					<button
						type="button"
						className="btn btn-ghost btn-xs absolute top-1/2 right-1 size-5 min-h-0 -translate-y-1/2 rounded-full p-0"
						onClick={event => {
							event.preventDefault();
							event.stopPropagation();
							onQueryChange('');
							resolvedInputRef.current?.focus({ preventScroll: true });
						}}
						aria-label="Clear search"
					>
						<FiX size={11} />
					</button>
				) : countLabel ? (
					<span className="text-base-content/45 pointer-events-none absolute top-1/2 right-2 -translate-y-1/2 text-[10px]">
						{countLabel}
					</span>
				) : null}
			</label>
		</div>
	);
}
