import { useCallback, useEffect, useRef, useState } from 'react';
import { FiAlertCircle, FiChevronDown } from 'react-icons/fi';

import { getErrorMessage } from '@/lib/error_utils';

import { CopyButton } from '@/components/copy_button';
import { DownloadButton } from '@/components/download_button';

export interface ManagementArtifactSourceValue<T> {
	text: string;
	fileName: string;
	extension: string;
	data: T;
}

interface ManagementArtifactSourceProps<T> {
	label: string;
	load: () => Promise<ManagementArtifactSourceValue<T>>;
	onLoaded?: (data: T) => void;
	prefetched?: ManagementArtifactSourceValue<T> | null;
	defaultOpen?: boolean;
	note?: string;
}

/**
 * Reads source only when the user opens, copies, or exports it. All actions
 * share one request and one cached result. Keep this component keyed to the
 * artifact identity and revision when the selected artifact can change.
 */
export function ManagementArtifactSource<T>({
	label,
	load,
	onLoaded,
	prefetched,
	defaultOpen = false,
	note,
}: ManagementArtifactSourceProps<T>) {
	const [value, setValue] = useState<ManagementArtifactSourceValue<T> | null>(prefetched ?? null);
	const [expanded, setExpanded] = useState(defaultOpen);
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState('');
	const valueRef = useRef<ManagementArtifactSourceValue<T> | null>(prefetched ?? null);
	const requestRef = useRef<Promise<ManagementArtifactSourceValue<T> | null> | null>(null);
	const mountedRef = useRef(false);
	const onLoadedRef = useRef(onLoaded);

	useEffect(() => {
		mountedRef.current = true;
		return () => {
			mountedRef.current = false;
		};
	}, []);

	useEffect(() => {
		onLoadedRef.current = onLoaded;
	}, [onLoaded]);

	const ensureSource = useCallback(async (): Promise<ManagementArtifactSourceValue<T> | null> => {
		if (valueRef.current) {
			return valueRef.current;
		}
		if (requestRef.current) {
			return requestRef.current;
		}

		setLoading(true);
		setError('');

		const request = Promise.resolve()
			.then(() => load())
			.then(result => {
				if (mountedRef.current) {
					valueRef.current = result;
					setValue(result);
					onLoadedRef.current?.(result.data);
				}
				return result;
			})
			.catch((cause: unknown) => {
				if (mountedRef.current) {
					setError(getErrorMessage(cause, `${label} could not be loaded.`));
				}
				return null;
			});

		requestRef.current = request;

		void request.finally(() => {
			if (requestRef.current === request) {
				requestRef.current = null;
			}
			if (mountedRef.current) {
				setLoading(false);
			}
		});

		return request;
	}, [label, load]);

	useEffect(() => {
		if (defaultOpen && !valueRef.current) {
			void ensureSource();
		}
	}, [defaultOpen, ensureSource]);

	const getCopyValue = useCallback(async () => {
		const source = await ensureSource();
		return source?.text;
	}, [ensureSource]);

	const getExportValue = useCallback(async () => {
		const source = await ensureSource();
		if (!source) {
			return null;
		}

		return {
			content: source.text,
			fileName: source.fileName,
			extension: source.extension,
			filterDisplayName: label,
		};
	}, [ensureSource, label]);

	const clearError = useCallback(() => {
		if (mountedRef.current) {
			setError('');
		}
	}, []);

	const handleCopyError = useCallback(
		(cause: unknown) => {
			if (mountedRef.current) {
				setError(getErrorMessage(cause, `${label} could not be copied.`));
			}
		},
		[label]
	);

	const handleExportError = useCallback(
		(cause: unknown) => {
			if (mountedRef.current) {
				setError(getErrorMessage(cause, `${label} could not be exported.`));
			}
		},
		[label]
	);

	const toggleExpanded = () => {
		if (!expanded) {
			void ensureSource();
		}
		setExpanded(current => !current);
	};

	return (
		<div className="border-base-content/10 bg-base-100 overflow-hidden rounded-xl border">
			<div className="flex min-h-8 items-center justify-between gap-2 px-2 py-0.5">
				<span className="min-w-0 truncate text-sm font-medium">View {label}</span>

				<div className="flex shrink-0 items-center gap-0.5">
					<CopyButton
						className="btn btn-ghost btn-xs shrink-0 rounded-lg p-1"
						valueFetcher={getCopyValue}
						size={14}
						label="Copy"
						copyingLabel="Copying"
						copiedLabel="Copied"
						aria-label={`Copy ${label}`}
						title={`Copy ${label}`}
						onClick={clearError}
						onCopied={clearError}
						onCopyError={handleCopyError}
					/>

					<DownloadButton
						className="btn btn-ghost btn-xs shrink-0 rounded-lg p-1"
						valueFetcher={getExportValue}
						size={14}
						label="Export"
						downloadingLabel="Exporting"
						aria-label={`Export ${label}`}
						title={`Export ${label}`}
						onClick={clearError}
						onDownloaded={clearError}
						onDownloadError={handleExportError}
					/>

					<button
						type="button"
						className="btn btn-ghost btn-xs shrink-0 rounded-lg p-1"
						aria-label={expanded ? `Collapse ${label}` : `Expand ${label}`}
						title={expanded ? `Collapse ${label}` : `Expand ${label}`}
						aria-expanded={expanded}
						onClick={toggleExpanded}
					>
						<FiChevronDown
							size={15}
							aria-hidden="true"
							className={`shrink-0 transition-transform ${expanded ? 'rotate-180' : ''}`}
						/>
						<span className="sr-only">{expanded ? `Collapse ${label}` : `View ${label}`}</span>
					</button>
				</div>
			</div>

			{expanded ? (
				<div className="border-base-content/10 space-y-2 border-t p-3">
					{loading && !value ? <output className="text-base-content/60 text-sm">Loading {label}...</output> : null}

					{value ? (
						value.text.trim() ? (
							<pre className="app-scrollbar-thin bg-base-300 max-h-96 overflow-auto rounded-xl p-3 font-mono text-xs/5 wrap-break-word whitespace-pre-wrap">
								{value.text}
							</pre>
						) : (
							<p className="text-base-content/60 text-sm">This source is empty.</p>
						)
					) : null}

					{note ? <p className="text-base-content/60 text-xs">{note}</p> : null}
				</div>
			) : null}

			{error ? (
				<div className="border-base-content/10 flex items-start gap-2 border-t p-3 text-sm" role="alert">
					<FiAlertCircle size={14} className="text-error mt-0.5 shrink-0" />
					<div className="min-w-0 flex-1">
						<p className="text-error wrap-break-word">{error}</p>
						<button
							type="button"
							className="btn btn-ghost btn-xs mt-1 rounded-lg"
							disabled={loading}
							onClick={() => {
								void ensureSource();
							}}
						>
							Retry
						</button>
					</div>
				</div>
			) : null}
		</div>
	);
}
