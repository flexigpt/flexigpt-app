import type { ReactNode } from 'react';

import { FiAlertCircle, FiHelpCircle } from 'react-icons/fi';

type StatusTone = 'neutral' | 'success' | 'warning' | 'error' | 'info';

const STATUS_TONE_CLASSES: Record<StatusTone, string> = {
	neutral: 'badge-neutral',
	success: 'badge-success',
	warning: 'badge-warning',
	error: 'badge-error',
	info: 'badge-info',
};

interface ManagementPageHeaderProps {
	title: string;
	description?: string;
	leadingActions?: ReactNode;
	actions?: ReactNode;
}

/**
 * @public
 */
export function ManagementPageHeader({ title, description, leadingActions, actions }: ManagementPageHeaderProps) {
	return (
		<header className="bg-base-200/95 border-base-content/10 sticky top-0 z-20 mt-4 w-11/12 rounded-2xl border px-4 py-3 backdrop-blur-sm xl:w-2/3">
			<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<div className="flex min-w-0 items-center gap-2">
					{leadingActions}
					<div className="min-w-0">
						<h1 className="truncate text-xl font-semibold">{title}</h1>
						{description ? <p className="text-base-content/70 mt-1 text-xs">{description}</p> : null}
					</div>
				</div>
				{actions ? <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">{actions}</div> : null}
			</div>
		</header>
	);
}

interface ManagementBundleCardProps {
	title: ReactNode;
	subtitle?: ReactNode;
	headerActions?: ReactNode;
	children?: ReactNode;
	className?: string;
}

/**
 * @public
 */
export function ManagementBundleCard({
	title,
	subtitle,
	headerActions,
	children,
	className = '',
}: ManagementBundleCardProps) {
	return (
		<section className={`bg-base-100 border-base-content/10 mb-6 rounded-2xl border p-4 shadow-sm ${className}`}>
			<div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
				<div className="min-w-0">
					<div className="truncate text-sm font-semibold">{title}</div>
					{subtitle ? <div className="text-base-content/60 mt-1 text-xs">{subtitle}</div> : null}
				</div>
				{headerActions ? <div className="flex flex-wrap items-center justify-end gap-2">{headerActions}</div> : null}
			</div>
			{children}
		</section>
	);
}

interface ManagementItemCardProps {
	title: ReactNode;
	subtitle?: ReactNode;
	status?: ReactNode;
	description?: ReactNode;
	metadata?: ReactNode;
	actions?: ReactNode;
	children?: ReactNode;
	className?: string;
}

export function ManagementItemCard({
	title,
	subtitle,
	status,
	description,
	metadata,
	actions,
	children,
	className = '',
}: ManagementItemCardProps) {
	return (
		<article
			className={`border-base-content/10 hover:border-base-content/20 min-w-0 rounded-2xl border p-4 transition-colors ${className}`}
		>
			<div className="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
				<div className="min-w-0">
					<div className="truncate font-medium">{title}</div>
					{subtitle ? <div className="text-base-content/60 mt-1 text-xs wrap-break-word">{subtitle}</div> : null}
					{description ? (
						<div className="text-base-content/70 mt-2 max-h-16 overflow-hidden text-sm">{description}</div>
					) : null}
				</div>
				{status ? <div className="flex max-w-full shrink-0 flex-wrap items-center gap-2">{status}</div> : null}
			</div>

			{metadata ? <div className="mt-3 flex flex-wrap gap-2">{metadata}</div> : null}
			{children}
			{actions ? <ActionRow>{actions}</ActionRow> : null}
		</article>
	);
}

interface StatusBadgeProps {
	children: ReactNode;
	tone?: StatusTone;
	title?: string;
	className?: string;
}

export function StatusBadge({ children, tone = 'neutral', title, className = '' }: StatusBadgeProps) {
	return (
		<span
			className={`badge h-auto max-w-full px-2 py-1 text-center leading-tight wrap-break-word whitespace-normal ${STATUS_TONE_CLASSES[tone]} ${className}`}
			title={title}
		>
			{children}
		</span>
	);
}

interface MetadataPillProps {
	label?: ReactNode;
	children: ReactNode;
	title?: string;
	className?: string;
}

export function MetadataPill({ label, children, title, className = '' }: MetadataPillProps) {
	return (
		<span
			className={`border-base-content/20 inline-flex min-w-0 items-center gap-1 rounded-xl border px-2 py-1 text-xs ${className}`}
			title={title}
		>
			{label ? <span className="text-base-content/60 shrink-0">{label}</span> : null}
			<span className="min-w-0 truncate">{children}</span>
		</span>
	);
}

interface ActionRowProps {
	children: ReactNode;
	leading?: ReactNode;
	className?: string;
}

export function ActionRow({ children, leading, className = '' }: ActionRowProps) {
	return (
		<div
			className={`border-base-content/10 mt-4 flex flex-col gap-3 border-t pt-3 sm:flex-row sm:items-center sm:justify-between ${className}`}
		>
			<div className="flex min-w-0 flex-wrap items-center gap-2">{leading}</div>
			<div className="flex flex-wrap items-center justify-end gap-2">{children}</div>
		</div>
	);
}

interface ModalFieldProps {
	label: ReactNode;
	children: ReactNode;
	htmlFor?: string;
	required?: boolean;
	hint?: string;
	error?: string;
	align?: 'start' | 'center';
	className?: string;
}

export function ModalField({
	label,
	children,
	htmlFor,
	required = false,
	hint,
	error,
	align = 'center',
	className = '',
}: ModalFieldProps) {
	return (
		<div className={`grid grid-cols-12 gap-2 ${align === 'start' ? 'items-start' : 'items-center'} ${className}`}>
			<label htmlFor={htmlFor} className="label col-span-12 justify-start gap-2 sm:col-span-3">
				<span className="text-sm">
					{label}
					{required ? '*' : ''}
				</span>
				{hint ? (
					<span className="tooltip tooltip-right" data-tip={hint}>
						<FiHelpCircle size={12} />
					</span>
				) : null}
			</label>
			<div className="col-span-12 min-w-0 sm:col-span-9">
				{children}
				{error ? (
					<div className="text-error mt-1 flex items-start gap-1 text-xs" role="alert">
						<FiAlertCircle className="mt-0.5 shrink-0" size={12} />
						<span className="wrap-break-word">{error}</span>
					</div>
				) : null}
			</div>
		</div>
	);
}

interface ModalSectionProps {
	title: ReactNode;
	description?: ReactNode;
	actions?: ReactNode;
	children: ReactNode;
	className?: string;
}

export function ModalSection({ title, description, actions, children, className = '' }: ModalSectionProps) {
	return (
		<section className={`border-base-content/10 rounded-2xl border p-4 ${className}`}>
			<div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
				<div className="min-w-0">
					<h4 className="text-sm font-semibold">{title}</h4>
					{description ? <div className="text-base-content/70 mt-1 text-xs">{description}</div> : null}
				</div>
				{actions ? <div className="flex shrink-0 flex-wrap gap-2">{actions}</div> : null}
			</div>
			<div className="space-y-4">{children}</div>
		</section>
	);
}
