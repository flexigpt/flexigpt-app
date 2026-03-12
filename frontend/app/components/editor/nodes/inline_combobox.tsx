import {
	type ComponentProps,
	createContext,
	forwardRef,
	type HTMLAttributes,
	type ReactNode,
	type RefObject,
	startTransition,
	useCallback,
	useContext,
	useEffect,
	useMemo,
	useRef,
	useState,
} from 'react';

import {
	Combobox,
	ComboboxGroup,
	ComboboxGroupLabel,
	ComboboxItem,
	type ComboboxItemProps,
	ComboboxPopover,
	ComboboxProvider,
	ComboboxRow,
	Portal,
	useComboboxContext,
	useComboboxStore,
	useStoreState,
} from '@ariakit/react';
import { filterWords } from '@platejs/combobox';
import { useComboboxInput, type UseComboboxInputResult, useHTMLInputCursorState } from '@platejs/combobox/react';
import { cn } from '@udecode/cn';
import { cva } from 'class-variance-authority';
import type { Point, TElement } from 'platejs';
import { useComposedRef, useEditorRef } from 'platejs/react';

type FilterFn = (
	item: { value: string; group?: string; keywords?: string[]; label?: string },
	search: string
) => boolean;

interface InlineComboboxContextValue {
	filter: FilterFn | false;
	inputProps: UseComboboxInputResult['props'];
	inputRef: RefObject<HTMLInputElement | null>;
	removeInput: UseComboboxInputResult['removeInput'];
	showTrigger: boolean;
	trigger: string;
	setHasEmpty: (hasEmpty: boolean) => void;
}

const InlineComboboxContext = createContext<InlineComboboxContextValue>(null as unknown as InlineComboboxContextValue);

const defaultFilter: FilterFn = ({ group, keywords = [], label, value }, search) => {
	const uniqueTerms = new Set([value, ...keywords, group, label].filter(Boolean));

	return Array.from(uniqueTerms).some(keyword => filterWords(keyword ?? '', search));
};

interface InlineComboboxProps {
	children: ReactNode;
	element: TElement;
	trigger: string;
	filter?: FilterFn | false;
	hideWhenNoValue?: boolean;
	showTrigger?: boolean;
	value?: string;
	setValue?: (value: string) => void;
}

export const InlineCombobox = ({
	children,
	element,
	filter = defaultFilter,
	hideWhenNoValue = false,
	setValue: setValueProp,
	showTrigger = true,
	trigger,
	value: valueProp,
}: InlineComboboxProps) => {
	const editor = useEditorRef();
	const inputRef = useRef<HTMLInputElement>(null);
	const cursorState = useHTMLInputCursorState(inputRef);

	const [valueState, setValueState] = useState('');
	const hasValueProp = valueProp !== undefined;
	const value = hasValueProp ? valueProp : valueState;

	const setValue = useCallback(
		(newValue: string) => {
			setValueProp?.(newValue);

			if (!hasValueProp) {
				setValueState(newValue);
			}
		},
		[setValueProp, hasValueProp]
	);

	/**
	 * Track the point just before the input element so we know where to
	 * insertText if the combobox closes due to a selection change.
	 */
	const insertPoint = useRef<Point | null>(null);

	useEffect(() => {
		const path = editor.api.findPath(element);

		if (!path) return;

		const point = editor.api.before(path);

		if (!point) return;

		const pointRef = editor.api.pointRef(point);
		insertPoint.current = pointRef.current;

		return () => {
			pointRef.unref();
		};
	}, [editor, element]);

	const { props: inputProps, removeInput } = useComboboxInput({
		cancelInputOnBlur: true,
		cursorState,
		ref: inputRef,
		onCancelInput: cause => {
			if (cause !== 'backspace') {
				editor.tf.insertText(trigger + value, {
					at: insertPoint.current ?? undefined,
				});
			}
			if (cause === 'arrowLeft' || cause === 'arrowRight') {
				editor.tf.move({
					distance: 1,
					reverse: cause === 'arrowLeft',
				});
			}
		},
	});

	const [hasEmpty, setHasEmpty] = useState(false);

	const contextValue: InlineComboboxContextValue = useMemo(
		() => ({
			filter,
			inputProps,
			inputRef,
			removeInput,
			setHasEmpty,
			showTrigger,
			trigger,
		}),
		[trigger, showTrigger, filter, inputRef, inputProps, removeInput, setHasEmpty]
	);

	const store = useComboboxStore({
		// open: ,
		setValue: newValue => {
			startTransition(() => {
				setValue(newValue);
			});
		},
	});

	const items = useStoreState(store, 'items');

	/**
	 * If there is no active ID and the list of items changes, select the first
	 * item.
	 */
	useEffect(() => {
		if (!store.getState().activeId) {
			store.setActiveId(store.first());
		}
	}, [items, store]);

	return (
		<span contentEditable={false}>
			<ComboboxProvider open={(items.length > 0 || hasEmpty) && (!hideWhenNoValue || value.length > 0)} store={store}>
				<InlineComboboxContext.Provider value={contextValue}>{children}</InlineComboboxContext.Provider>
			</ComboboxProvider>
		</span>
	);
};

export const InlineComboboxInput = forwardRef<HTMLInputElement, HTMLAttributes<HTMLInputElement>>(
	function InlineComboboxInput({ className, ...props }, propRef) {
		const { inputProps, inputRef: contextRef, showTrigger, trigger } = useContext(InlineComboboxContext);

		const store = useComboboxContext();
		if (!store) {
			throw new Error('Combobox must be wrapped in ComboboxProvider');
		}
		const value = useStoreState(store, 'value');

		const ref = useComposedRef(propRef, contextRef);

		/**
		 * To create an auto-resizing input, we render a visually hidden span
		 * containing the input value and position the input element on top of it.
		 * This works well for all cases except when input exceeds the width of the
		 * container.
		 */

		return (
			<>
				{showTrigger && trigger}

				<span className="relative min-h-[1em]">
					<span className="invisible overflow-hidden whitespace-nowrap" aria-hidden="true">
						{value || '\u200B'}
					</span>

					<Combobox
						ref={ref}
						className={cn('absolute top-0 left-0 size-full bg-transparent outline-none', className)}
						value={value}
						autoSelect
						{...inputProps}
						{...props}
					/>
				</span>
			</>
		);
	}
);

export const InlineComboboxContent: typeof ComboboxPopover = ({ className, ...props }) => {
	// Portal prevents CSS from leaking into popover
	return (
		<Portal>
			<ComboboxPopover
				className={cn(
					'rounded-box bg-base-100 text-base-content z-50 max-h-72 w-75 overflow-y-auto',
					'border-base-300 border p-1 shadow-xl',
					className
				)}
				{...props}
			/>
		</Portal>
	);
};

const comboboxItemVariants = cva(
	[
		'relative mx-1 flex h-[28px] items-center rounded-md px-2 text-sm',
		'text-base-content outline-none select-none',
		// icon size normalizer
		'[&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0',
	].join(' '),
	{
		defaultVariants: {
			interactive: true,
		},
		variants: {
			interactive: {
				false: '',
				true: ['cursor-pointer transition-colors', 'hover:bg-base-200', 'data-[active-item=true]:bg-base-200'].join(
					' '
				),
			},
		},
	}
);

export const InlineComboboxItem = ({
	className,
	focusEditor = true,
	group,
	keywords,
	label,
	onClick,
	...props
}: {
	focusEditor?: boolean;
	group?: string;
	keywords?: string[];
	label?: string;
} & ComboboxItemProps &
	Required<Pick<ComboboxItemProps, 'value'>>) => {
	const { value } = props;

	const { filter, removeInput } = useContext(InlineComboboxContext);

	const store = useComboboxContext();

	if (!store) {
		throw new Error('Combobox must be wrapped in ComboboxProvider');
	}

	// Optimization: Do not subscribe to value if filter is false
	const search = filter && useStoreState(store, 'value');

	const visible = useMemo(
		() => !filter || filter({ group, keywords, label, value }, search as string),
		[filter, group, keywords, label, value, search]
	);

	if (!visible) return null;

	return (
		<ComboboxItem
			className={cn(comboboxItemVariants(), className)}
			onClick={event => {
				removeInput(focusEditor);
				onClick?.(event);
			}}
			{...props}
		/>
	);
};

export const InlineComboboxEmpty = ({ children, className }: HTMLAttributes<HTMLDivElement>) => {
	const { setHasEmpty } = useContext(InlineComboboxContext);
	const store = useComboboxContext();
	if (!store) {
		throw new Error('Combobox must be wrapped in ComboboxProvider');
	}
	const items = useStoreState(store, 'items');

	useEffect(() => {
		setHasEmpty(true);

		return () => {
			setHasEmpty(false);
		};
	}, [setHasEmpty]);

	if (items.length > 0) return null;

	return (
		<div className={cn(comboboxItemVariants({ interactive: false }), 'text-base-content/60', className)}>
			{children}
		</div>
	);
};

/**
 * @public
 */
export const InlineComboboxRow = ComboboxRow;

export function InlineComboboxGroup({ className, ...props }: ComponentProps<typeof ComboboxGroup>) {
	return (
		<ComboboxGroup
			{...props}
			className={cn('hidden [&:has([role=option])]:block', 'border-base-200 border-b py-1 last:border-b-0', className)}
		/>
	);
}

/**
 * @public
 */
export function InlineComboboxGroupLabel({ className, ...props }: ComponentProps<typeof ComboboxGroupLabel>) {
	return (
		<ComboboxGroupLabel
			{...props}
			className={cn('text-base-content/60 mt-1.5 mb-2 px-3 text-xs font-semibold uppercase', className)}
		/>
	);
}
