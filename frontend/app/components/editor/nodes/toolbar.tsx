import {
	type AnchorHTMLAttributes,
	type ButtonHTMLAttributes,
	type ComponentType,
	forwardRef,
	type HTMLAttributes,
	type MouseEvent,
	useState,
} from 'react';

import { FiChevronDown } from 'react-icons/fi';

import { cn } from '@udecode/cn';
import { cva, type VariantProps } from 'class-variance-authority';

/* Root toolbar */
export const Toolbar = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(function Toolbar(
	{ className, ...props },
	ref
) {
	return <div ref={ref} className={cn('relative flex items-center gap-1 select-none', className)} {...props} />;
});

/* Grouping container (use with join for button groups) */
/**
 * @public
 */
export const ToolbarToggleGroup = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
	function ToolbarToggleGroup({ className, ...props }, ref) {
		return <div ref={ref} className={cn('join flex items-center', className)} {...props} />;
	}
);

/**
 * @public
 */
export const ToolbarLink = forwardRef<HTMLAnchorElement, AnchorHTMLAttributes<HTMLAnchorElement>>(function ToolbarLink(
	{ className, ...props },
	ref
) {
	return <a ref={ref} className={cn('link link-hover font-medium', className)} {...props} />;
});

/* Vertical separator */
/**
 * @public
 */
export function ToolbarSeparator({ className, ...props }: HTMLAttributes<HTMLSpanElement>) {
	return <span className={cn('bg-base-300 mx-2 my-1 w-px self-stretch', className)} {...props} />;
}

/* Button variants (DaisyUI) */
const toolbarButtonVariants = cva(
	// base
	'btn normal-case gap-2 join-item [&>svg]:shrink-0',
	{
		variants: {
			size: {
				sm: 'btn-sm',
				default: '',
				lg: 'btn-lg',
			},
			variant: {
				default: 'btn-ghost',
				outline: 'btn-outline',
			},
		},
		defaultVariants: {
			size: 'sm',
			variant: 'default',
		},
	}
);

/* For the narrow dropdown arrow piece */
const dropdownArrowVariants = cva('btn btn-square join-item', {
	variants: {
		size: {
			sm: 'btn-sm',
			default: '',
			lg: 'btn-lg',
		},
		variant: {
			default: 'btn-ghost',
			outline: 'btn-outline',
		},
	},
	defaultVariants: {
		size: 'sm',
		variant: 'default',
	},
});

/* DaisyUI tooltip HOC (string-only content) */
type Tooltipable = {
	tooltip?: string;
	tooltipClassName?: string;
	tooltipPosition?: 'top' | 'right' | 'bottom' | 'left';
};

function withTooltip<P extends Tooltipable>(Component: ComponentType<P>) {
	return function WithTooltip(props: P) {
		const { tooltip, tooltipClassName, tooltipPosition = 'bottom', ...rest } = props;

		const content = <Component {...(rest as P)} />;

		if (!tooltip) return content;

		return (
			<div className={cn('tooltip', `tooltip-${tooltipPosition}`, tooltipClassName)} data-tip={tooltip}>
				{content}
			</div>
		);
	};
}

/* Toggle item (DaisyUI) */
type ToolbarToggleItemProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'onChange'> &
	VariantProps<typeof toolbarButtonVariants> & {
		pressed?: boolean;
		defaultPressed?: boolean;
		onPressedChange?: (pressed: boolean) => void;
	};

/**
 * @public
 */
export const ToolbarToggleItem = forwardRef<HTMLButtonElement, ToolbarToggleItemProps>(function ToolbarToggleItem(
	{ className, size = 'sm', variant, pressed, defaultPressed, onPressedChange, onClick, disabled, ...props },
	ref
) {
	const isControlled = typeof pressed === 'boolean';
	const [internalPressed, setInternalPressed] = useState(defaultPressed ?? false);

	const isPressed = isControlled ? pressed : internalPressed;

	const handleClick = (e: MouseEvent<HTMLButtonElement>) => {
		if (disabled) return;
		onClick?.(e);
		if (!e.defaultPrevented) {
			const next = !isPressed;
			if (!isControlled) setInternalPressed(next);
			onPressedChange?.(next);
		}
	};

	return (
		<button
			ref={ref}
			type="button"
			aria-pressed={isPressed}
			disabled={disabled}
			className={cn(toolbarButtonVariants({ size, variant }), isPressed && 'btn-active', className)}
			onClick={handleClick}
			{...props}
		/>
	);
});

/* ToolbarButton: toggling if pressed provided, else normal button */
type ToolbarButtonProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'onChange'> &
	VariantProps<typeof toolbarButtonVariants> &
	Tooltipable & {
		isDropdown?: boolean;
		pressed?: boolean;
		defaultPressed?: boolean;
		onPressedChange?: (pressed: boolean) => void;
	};

const RawToolbarButton = forwardRef<HTMLButtonElement, ToolbarButtonProps>(function RawToolbarButton(
	{ children, className, isDropdown, size = 'sm', variant, pressed, defaultPressed, onPressedChange, ...props },
	ref
) {
	const inner = (
		<span className={cn('inline-flex items-center', isDropdown && 'gap-1')}>
			<span className={cn(isDropdown && 'flex-1 whitespace-nowrap')}>{children}</span>
			{isDropdown && <FiChevronDown className="text-base-content/70" aria-hidden />}
		</span>
	);

	if (typeof pressed === 'boolean' || typeof defaultPressed === 'boolean') {
		// Toggle button
		return (
			<ToolbarToggleItem
				ref={ref}
				className={className}
				size={size}
				variant={variant}
				pressed={pressed}
				defaultPressed={defaultPressed}
				onPressedChange={onPressedChange}
				{...props}
			>
				{inner}
			</ToolbarToggleItem>
		);
	}

	// Plain button
	return (
		<button ref={ref} type="button" className={cn(toolbarButtonVariants({ size, variant }), className)} {...props}>
			{inner}
		</button>
	);
});

export const ToolbarButton = withTooltip(RawToolbarButton);

/* Split button: primary + secondary in a join group */
/**
 * @public
 */
export type ToolbarSplitButtonProps = Omit<ToolbarButtonProps, 'isDropdown'>;

/**
 * @public
 */
export const ToolbarSplitButton = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
	function ToolbarSplitButton({ className, ...props }, ref) {
		return <div ref={ref} className={cn('join', className)} {...props} />;
	}
);

type ToolbarSplitButtonPrimaryProps = Omit<ToolbarToggleItemProps, 'defaultPressed' | 'onPressedChange'> & Tooltipable;

/**
 * @public
 */
const RawToolbarSplitButtonPrimary = forwardRef<HTMLButtonElement, ToolbarSplitButtonPrimaryProps>(
	function RawToolbarSplitButtonPrimary({ className, size = 'sm', variant, children, ...props }, ref) {
		return (
			<ToolbarToggleItem ref={ref} size={size} variant={variant} className={cn('rounded-r-none', className)} {...props}>
				{children}
			</ToolbarToggleItem>
		);
	}
);

/**
 * @public
 */
export const ToolbarSplitButtonPrimary = withTooltip(RawToolbarSplitButtonPrimary);

type ToolbarSplitButtonSecondaryProps = ButtonHTMLAttributes<HTMLButtonElement> &
	VariantProps<typeof dropdownArrowVariants> &
	Tooltipable;

const RawToolbarSplitButtonSecondary = forwardRef<HTMLButtonElement, ToolbarSplitButtonSecondaryProps>(
	function RawToolbarSplitButtonSecondary({ className, size = 'sm', variant, onClick, ...props }, ref) {
		return (
			<button
				ref={ref}
				type="button"
				className={cn(dropdownArrowVariants({ size, variant }), 'rounded-l-none', className)}
				onClick={e => {
					e.stopPropagation();
					onClick?.(e);
				}}
				{...props}
				aria-label="More"
			>
				<FiChevronDown className="text-base-content/70" aria-hidden />
			</button>
		);
	}
);

/**
 * @public
 */
export const ToolbarSplitButtonSecondary = withTooltip(RawToolbarSplitButtonSecondary);

/* Simple grouping wrapper, use ToolbarSeparator between groups */
export function ToolbarGroup({ className, children, ...props }: HTMLAttributes<HTMLDivElement>) {
	return (
		<div className={cn('flex items-center', className)} {...props}>
			{children}
		</div>
	);
}

/* DaisyUI menu group (use inside a <ul className="menu">) */
/**
 * @public
 */
export function ToolbarMenuGroup({
	className,
	label,
	children,
	...props
}: HTMLAttributes<HTMLLIElement> & { label?: string }) {
	return (
		<li className={cn('', className)} {...props}>
			{label && <h2 className="menu-title">{label}</h2>}
			<ul>{children}</ul>
		</li>
	);
}
