import { FiAlertCircle, FiRefreshCw } from 'react-icons/fi';

interface ManagementResourceErrorProps {
	title: string;
	error: unknown;
	onRetry: () => Promise<unknown>;
	isRetrying?: boolean;
	className?: string;
}

export function ManagementResourceError({
	title,
	error,
	onRetry,
	isRetrying = false,
	className = '',
}: ManagementResourceErrorProps) {
	const message =
		error instanceof Error && error.message.trim().length > 0
			? error.message
			: 'The requested data could not be loaded.';

	return (
		<div className={`alert alert-error rounded-2xl text-sm ${className}`} role="alert">
			<FiAlertCircle className="shrink-0" size={16} />

			<div className="min-w-0 grow">
				<div className="font-semibold">{title}</div>
				<div className="mt-1 wrap-break-word">{message}</div>
			</div>

			<button
				type="button"
				className="btn btn-sm rounded-xl"
				disabled={isRetrying}
				onClick={() => {
					void Promise.resolve(onRetry()).catch(() => undefined);
				}}
			>
				<FiRefreshCw size={14} />
				<span>{isRetrying ? 'Retrying...' : 'Retry'}</span>
			</button>
		</div>
	);
}
