import type { ReactNode } from 'react';

import { FiAlertCircle, FiHelpCircle } from 'react-icons/fi';

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
