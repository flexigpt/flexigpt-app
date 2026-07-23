import type { ButtonHTMLAttributes, ComponentType, HTMLAttributes, MouseEvent } from 'react';
import { forwardRef, useState } from 'react';

import { FiChevronDown } from 'react-icons/fi';

import { cn } from '@udecode/cn';
import type { VariantProps } from 'class-variance-authority';
import { cva } from 'class-variance-authority';

/* Root toolbar */
export const Toolbar = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(function Toolbar(
	{ className, ...props },
	ref
) {
	return <div ref={ref} className={cn('relative flex items-center gap-1 select-none', className)} {...props} />;
});

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

/* DaisyUI tooltip HOC (string-only content) */
interface Tooltipable {
	tooltip?: string;
	tooltipClassName?: string;
	tooltipPosition?: 'top' | 'right' | 'bottom' | 'left';
}

function withTooltip<P extends Tooltipable>(Component: ComponentType<P>) {
	return function WithTooltip(props: P) {
		const { tooltip, tooltipClassName, tooltipPosition = 'bottom', ...rest } = props;

		const content = <Component {...(rest as P)} />;

		if (!tooltip) {
			return content;
		}
		const pos = 'tooltip-' + tooltipPosition;
		return (
			<div className={cn('tooltip', pos, tooltipClassName)} data-tip={tooltip}>
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

const ToolbarToggleItem = forwardRef<HTMLButtonElement, ToolbarToggleItemProps>(function ToolbarToggleItem(
	{ className, size = 'sm', variant, pressed, defaultPressed, onPressedChange, onClick, disabled, ...props },
	ref
) {
	const isControlled = typeof pressed === 'boolean';
	const [internalPressed, setInternalPressed] = useState(defaultPressed ?? false);

	const isPressed = isControlled ? pressed : internalPressed;

	const handleClick = (e: MouseEvent<HTMLButtonElement>) => {
		if (disabled) {
			return;
		}
		onClick?.(e);
		if (!e.defaultPrevented) {
			const next = !isPressed;
			if (!isControlled) {
				setInternalPressed(next);
			}
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

// oxlint-disable-next-line react/only-export-components
export const ToolbarButton = withTooltip(RawToolbarButton);

/* Simple grouping wrapper */
export function ToolbarGroup({ className, children, ...props }: HTMLAttributes<HTMLDivElement>) {
	return (
		<div className={cn('flex items-center', className)} {...props}>
			{children}
		</div>
	);
}
