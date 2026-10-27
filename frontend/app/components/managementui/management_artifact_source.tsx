import { useCallback, useEffect, useRef, useState } from 'react';
import { FiAlertCircle, FiCheck, FiChevronDown, FiCopy, FiDownload } from 'react-icons/fi';

import { utf8TextToBase64 } from '@/lib/base64_utils';
import { getErrorMessage } from '@/lib/error_utils';

import { backendAPI } from '@/apis/baseapi';

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
 * Reads source only on Show, Copy, or Export. All three actions share one
 * request and one result. Keep the component keyed to the artifact identity
 * and revision when its parent can change the selected artifact.
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
	const [loading, setLoading] = useState(false);
	const [working, setWorking] = useState<'copy' | 'export' | null>(null);
	const [error, setError] = useState('');
	const [copied, setCopied] = useState(false);
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

		const request = load()
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
			})
			.finally(() => {
				if (requestRef.current === request) {
					requestRef.current = null;
				}
				if (mountedRef.current) {
					setLoading(false);
				}
			});

		requestRef.current = request;
		return request;
	}, [label, load]);

	useEffect(() => {
		if (defaultOpen && !valueRef.current) {
			void ensureSource();
		}
	}, [defaultOpen, ensureSource]);

	const copy = async () => {
		if (working) {
			return;
		}
		setWorking('copy');
		setCopied(false);
		const source = await ensureSource();
		if (source) {
			try {
				if (!navigator.clipboard?.writeText) {
					throw new Error('Clipboard access is unavailable.');
				}
				await navigator.clipboard.writeText(source.text);
				if (mountedRef.current) {
					setCopied(true);
				}
			} catch (cause) {
				if (mountedRef.current) {
					setError(getErrorMessage(cause, `${label} could not be copied.`));
				}
			}
		}
		if (mountedRef.current) {
			setWorking(null);
		}
	};

	const exportSource = async () => {
		if (working) {
			return;
		}
		setWorking('export');
		const source = await ensureSource();
		if (source) {
			try {
				const extension = source.extension.replace(/^\./, '');
				await backendAPI.saveFile(source.fileName, utf8TextToBase64(source.text), [
					{
						DisplayName: `${label} (*.${extension})`,
						Extensions: [extension],
					},
				]);
			} catch (cause) {
				if (mountedRef.current) {
					setError(getErrorMessage(cause, `${label} could not be exported.`));
				}
			}
		}
		if (mountedRef.current) {
			setWorking(null);
		}
	};

	return (
		<div className="border-base-content/10 bg-base-100 overflow-hidden rounded-xl border">
			<div className="flex min-h-8 items-center justify-end gap-0.5 px-1 py-0.5">
				<button
					type="button"
					className="btn btn-ghost btn-xs size-7 min-h-7 rounded-lg p-0"
					disabled={Boolean(working)}
					aria-label={working === 'copy' ? `Copying ${label}` : copied ? `${label} copied` : `Copy ${label}`}
					title={working === 'copy' ? `Copying ${label}` : copied ? `${label} copied` : `Copy ${label}`}
					onClick={() => {
						void copy();
					}}
				>
					{working === 'copy' ? (
						<span className="loading loading-spinner loading-xs" aria-hidden="true" />
					) : copied ? (
						<FiCheck size={14} aria-hidden="true" />
					) : (
						<FiCopy size={14} aria-hidden="true" />
					)}
					<span className="sr-only">{working === 'copy' ? 'Copying' : copied ? 'Copied' : 'Copy'}</span>
				</button>
				<button
					type="button"
					className="btn btn-ghost btn-xs size-7 min-h-7 rounded-lg p-0"
					disabled={Boolean(working)}
					aria-label={working === 'export' ? `Exporting ${label}` : `Export ${label}`}
					title={working === 'export' ? `Exporting ${label}` : `Export ${label}`}
					onClick={() => {
						void exportSource();
					}}
				>
					{working === 'export' ? (
						<span className="loading loading-spinner loading-xs" aria-hidden="true" />
					) : (
						<FiDownload size={14} aria-hidden="true" />
					)}
					<span className="sr-only">{working === 'export' ? 'Exporting' : 'Export'}</span>
				</button>
			</div>

			<details
				open={defaultOpen}
				className="group border-base-content/10 border-t"
				onToggle={event => {
					if (event.currentTarget.open) {
						void ensureSource();
					}
				}}
			>
				<summary className="flex cursor-pointer list-none items-center justify-between gap-2 px-3 py-2 text-sm font-medium [&::-webkit-details-marker]:hidden">
					<span>Show {label}</span>
					<FiChevronDown size={15} className="shrink-0 transition-transform group-open:rotate-180" />
				</summary>
				<div className="border-base-content/10 space-y-2 border-t p-3">
					{loading && !value ? <output className="text-base-content/60 text-sm">Loading {label}...</output> : null}
					{value ? (
						<pre className="app-scrollbar-thin bg-base-300 max-h-96 overflow-auto rounded-xl p-3 font-mono text-xs/5 wrap-break-word whitespace-pre-wrap">
							{value.text}
						</pre>
					) : null}
					{note ? <p className="text-base-content/60 text-xs">{note}</p> : null}
				</div>
			</details>

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
			{copied ? <output className="sr-only">{label} copied.</output> : null}
		</div>
	);
}
