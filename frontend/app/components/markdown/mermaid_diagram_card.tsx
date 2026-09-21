import type { CSSProperties } from 'react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { FiMoon, FiSun } from 'react-icons/fi';

import type { MermaidConfig } from 'mermaid';

import { getUUIDv7 } from '@/lib/uuid_utils';

import { renderMermaidQueued, useIsDarkMermaid } from '@/hooks/use_mermaid';

import { DownloadButton } from '@/components/download_button';
import { MermaidZoomModal } from '@/components/markdown/mermaid_zoom_modal';

export type MermaidRenderStatus = 'idle' | 'rendering' | 'rendered' | 'error';

interface MermaidDiagramProps {
	code: string;
	/**
	 * auto = follow app theme.
	 * light/dark = force Mermaid theme for this diagram only.
	 */
	defaultThemeMode?: 'auto' | 'light' | 'dark';
	showThemeToggle?: boolean;
	onRenderStatusChange?: (status: MermaidRenderStatus, message?: string) => void;
}

type MermaidSurfaceStyle = CSSProperties & {
	'--app-bg-mermaid'?: string;
	'--app-bg-code-header'?: string;
	'--app-text-code'?: string;
};

type MermaidRenderState =
	| {
			key: string;
			status: 'rendered';
			imageSrc: string;
	  }
	| {
			key: string;
			status: 'error';
			message: string;
	  };

interface CachedMermaidRender {
	key: string;
	imageSrc: string;
	estimatedBytes: number;
}

const MERMAID_ERROR_MESSAGE = 'Failed to render diagram. Please check the syntax.';
const MERMAID_CACHE_MAX_ENTRIES = 32;
const MERMAID_CACHE_MAX_ESTIMATED_BYTES = 32 * 1024 * 1024;
const MERMAID_PNG_MAX_DIMENSION = 8192;
const MERMAID_PNG_MAX_PIXELS = 16_000_000;

// Successful Mermaid renders survive component unmounts. This is important
// for virtualized or rebuilt markdown trees where scrolling can remount the
// same diagram.
const renderedMermaidCache = new Map<string, CachedMermaidRender>();
const pendingMermaidRenders = new Map<string, Promise<CachedMermaidRender>>();
const activeMermaidRenderKeys = new Map<string, number>();
let renderedMermaidCacheEstimatedBytes = 0;

const appendInlineStyles = (element: Element, styles: Record<string, string>) => {
	const existingStyle = element.getAttribute('style')?.trim();
	const styleText = Object.entries(styles)
		.map(([key, value]) => `${key}: ${value}`)
		.join('; ');

	element.setAttribute('style', existingStyle ? `${existingStyle}; ${styleText}` : styleText);
};

const prepareMermaidSvgMarkup = (svgMarkup: string): string => {
	if (typeof window === 'undefined') {
		return svgMarkup;
	}

	try {
		const parser = new DOMParser();
		const doc = parser.parseFromString(svgMarkup, 'image/svg+xml');

		if (doc.querySelector('parsererror')) {
			return svgMarkup;
		}

		const svg = doc.querySelector('svg');
		if (!svg) {
			return svgMarkup;
		}

		appendInlineStyles(svg, {
			display: 'block',
			margin: 'auto',
			width: '100%',
			height: '100%',
			'max-width': 'none',
			'max-height': 'none',
			'background-color': 'transparent',
		});

		const backgroundRect = svg.querySelector('rect.background');
		if (backgroundRect) {
			backgroundRect.setAttribute('fill', 'transparent');
		}

		return new XMLSerializer().serializeToString(svg);
	} catch {
		return svgMarkup;
	}
};

function deleteCachedMermaidRender(key: string): void {
	const cached = renderedMermaidCache.get(key);
	if (!cached) {
		return;
	}

	renderedMermaidCache.delete(key);
	renderedMermaidCacheEstimatedBytes = Math.max(0, renderedMermaidCacheEstimatedBytes - cached.estimatedBytes);
	URL.revokeObjectURL(cached.imageSrc);
}

function pruneRenderedMermaidCache(): void {
	if (
		renderedMermaidCache.size <= MERMAID_CACHE_MAX_ENTRIES &&
		renderedMermaidCacheEstimatedBytes <= MERMAID_CACHE_MAX_ESTIMATED_BYTES
	) {
		return;
	}

	for (const key of renderedMermaidCache.keys()) {
		if (
			renderedMermaidCache.size <= MERMAID_CACHE_MAX_ENTRIES &&
			renderedMermaidCacheEstimatedBytes <= MERMAID_CACHE_MAX_ESTIMATED_BYTES
		) {
			break;
		}

		if ((activeMermaidRenderKeys.get(key) ?? 0) > 0) {
			continue;
		}

		deleteCachedMermaidRender(key);
	}
}

function storeCachedMermaidRender(rendered: CachedMermaidRender): void {
	if (renderedMermaidCache.has(rendered.key)) {
		deleteCachedMermaidRender(rendered.key);
	}

	renderedMermaidCache.set(rendered.key, rendered);
	renderedMermaidCacheEstimatedBytes += rendered.estimatedBytes;
	pruneRenderedMermaidCache();
}

function retainMermaidRenderKey(key: string): () => void {
	activeMermaidRenderKeys.set(key, (activeMermaidRenderKeys.get(key) ?? 0) + 1);

	return () => {
		const nextCount = (activeMermaidRenderKeys.get(key) ?? 1) - 1;
		if (nextCount > 0) {
			activeMermaidRenderKeys.set(key, nextCount);
		} else {
			activeMermaidRenderKeys.delete(key);
		}

		pruneRenderedMermaidCache();
	};
}

function getCachedMermaidRender(key: string): CachedMermaidRender | null {
	return renderedMermaidCache.get(key) ?? null;
}

function createMermaidRenderState(cached: CachedMermaidRender): Extract<MermaidRenderState, { status: 'rendered' }> {
	return {
		key: cached.key,
		status: 'rendered',
		imageSrc: cached.imageSrc,
	};
}

function renderMermaidCached(key: string, code: string, config: MermaidConfig): Promise<CachedMermaidRender> {
	const cached = getCachedMermaidRender(key);
	if (cached) {
		return Promise.resolve(cached);
	}

	const pending = pendingMermaidRenders.get(key);
	if (pending) {
		return pending;
	}

	const renderId = `mermaid-cache-${getUUIDv7()}`;
	const next = renderMermaidQueued(renderId, code, config).then(renderResult => {
		const preparedSvgMarkup = prepareMermaidSvgMarkup(renderResult.svg);
		const imageSrc = URL.createObjectURL(new Blob([preparedSvgMarkup], { type: 'image/svg+xml;charset=utf-8' }));
		const rendered: CachedMermaidRender = {
			key,
			imageSrc,
			estimatedBytes: (key.length + preparedSvgMarkup.length) * 2,
		};

		storeCachedMermaidRender(rendered);
		return rendered;
	});

	pendingMermaidRenders.set(key, next);

	void next.then(
		() => {
			if (pendingMermaidRenders.get(key) === next) {
				pendingMermaidRenders.delete(key);
			}
		},
		() => {
			if (pendingMermaidRenders.get(key) === next) {
				pendingMermaidRenders.delete(key);
			}
		}
	);

	return next;
}

export function MermaidDiagram({
	code,
	defaultThemeMode = 'auto',
	showThemeToggle = true,
	onRenderStatusChange,
}: MermaidDiagramProps) {
	const isDark = useIsDarkMermaid();

	const wrapperRef = useRef<HTMLDivElement | null>(null);

	const [isZoomOpen, setIsZoomOpen] = useState(false);
	const [themeMode, setThemeMode] = useState<'auto' | 'light' | 'dark'>(defaultThemeMode);

	const latestToken = useRef(0);

	// CodeBlock mounts Mermaid only after streaming has completed.
	const stableCode = code;

	const effectiveMermaidTheme = useMemo<'dark' | 'default'>(() => {
		if (themeMode === 'auto') {
			return isDark ? 'dark' : 'default';
		}
		return themeMode === 'dark' ? 'dark' : 'default';
	}, [themeMode, isDark]);

	const renderKey = useMemo(() => `${effectiveMermaidTheme}\u0000${stableCode}`, [effectiveMermaidTheme, stableCode]);

	const [renderState, setRenderState] = useState<MermaidRenderState | null>(null);

	useEffect(() => {
		return retainMermaidRenderKey(renderKey);
	}, [renderKey]);

	// Per-diagram surface override: only when user explicitly selects light/dark.
	// In auto mode, it stays consistent with the app’s DaisyUI theme.
	const surfaceStyle = useMemo<MermaidSurfaceStyle | undefined>(() => {
		if (themeMode === 'auto') {
			return;
		}

		const forcedDark = themeMode === 'dark';

		return {
			'--app-bg-mermaid': forcedDark ? 'var(--mermaid-surface-dark)' : 'var(--mermaid-surface-light)',
			'--app-bg-code-header': forcedDark ? 'var(--mermaid-header-bg-dark)' : 'var(--mermaid-header-bg-light)',
			'--app-text-code': forcedDark ? 'var(--mermaid-header-text-dark)' : 'var(--mermaid-header-text-light)',
			colorScheme: forcedDark ? 'dark' : 'light',
		};
	}, [themeMode]);

	const mermaidConfig = useMemo<MermaidConfig>(() => {
		return {
			startOnLoad: false,
			theme: effectiveMermaidTheme,
			suppressErrorRendering: true,
			securityLevel: 'loose',
			// Important: keep outer background controlled by the container, not SVG.
			// Mermaid themes sometimes embed a background; transparency avoids mismatches.
			themeVariables: {
				background: 'transparent',
			} as any,
		};
	}, [effectiveMermaidTheme]);

	const cachedRender = getCachedMermaidRender(renderKey);
	const cachedRenderState = useMemo(
		() => (cachedRender ? createMermaidRenderState(cachedRender) : null),
		[cachedRender]
	);

	useEffect(() => {
		const token = ++latestToken.current;

		if (!stableCode.trim()) {
			return;
		}

		const cached = getCachedMermaidRender(renderKey);
		if (cached) {
			onRenderStatusChange?.('rendered');
			return;
		}

		let isCancelled = false;

		renderMermaidCached(renderKey, stableCode, mermaidConfig)
			.then(cr => {
				if (isCancelled || token !== latestToken.current) {
					return;
				}

				setRenderState(createMermaidRenderState(cr));

				onRenderStatusChange?.('rendered');
			})
			.catch((e: unknown) => {
				if (isCancelled || token !== latestToken.current) {
					return;
				}

				setRenderState({
					key: renderKey,
					status: 'error',
					message: MERMAID_ERROR_MESSAGE,
				});

				onRenderStatusChange?.('error', MERMAID_ERROR_MESSAGE);
				console.error('syntax error:', e);
			});

		return () => {
			isCancelled = true;
		};
	}, [mermaidConfig, onRenderStatusChange, renderKey, stableCode]);

	const currentRenderState = cachedRenderState ?? (renderState?.key === renderKey ? renderState : null);
	const imageSrc = currentRenderState?.status === 'rendered' ? currentRenderState.imageSrc : null;
	const hasRenderError = currentRenderState?.status === 'error';

	const getDiagramBackgroundColor = (): string => {
		const el = wrapperRef.current;
		if (!el) {
			return '#ffffff';
		}

		const bg = window.getComputedStyle(el).backgroundColor;

		// If transparent, default to white so PNG is not transparent-black-ish.
		if (!bg || bg === 'transparent' || bg === 'rgba(0, 0, 0, 0)') {
			return '#ffffff';
		}

		return bg;
	};

	const fetchDiagramAsBlob = async (): Promise<Blob> => {
		if (!imageSrc) {
			throw new Error('Mermaid image is not ready');
		}

		return new Promise<Blob>((resolve, reject) => {
			const img = new window.Image();

			img.onload = () => {
				const width = img.naturalWidth || img.width;
				const height = img.naturalHeight || img.height;
				if (width <= 0 || height <= 0) {
					reject(new Error('Mermaid image has invalid dimensions'));
					return;
				}

				const scaleFactor = Math.min(
					2,
					MERMAID_PNG_MAX_DIMENSION / width,
					MERMAID_PNG_MAX_DIMENSION / height,
					Math.sqrt(MERMAID_PNG_MAX_PIXELS / (width * height))
				);
				const canvas = document.createElement('canvas');
				canvas.width = Math.max(1, Math.round(width * scaleFactor));
				canvas.height = Math.max(1, Math.round(height * scaleFactor));

				const ctx = canvas.getContext('2d');
				if (!ctx) {
					reject(new Error('Canvas context is null'));
					return;
				}

				ctx.scale(scaleFactor, scaleFactor);
				ctx.fillStyle = getDiagramBackgroundColor();
				ctx.fillRect(0, 0, width, height);
				ctx.drawImage(img, 0, 0, width, height);

				canvas.toBlob(
					blob => {
						if (blob) {
							resolve(blob);
							return;
						}

						reject(new Error('Canvas is empty'));
					},
					'image/png',
					1.0
				);
			};

			img.onerror = err => {
				reject(err);
			};

			img.src = imageSrc;
		});
	};

	const handleOpenZoom = () => {
		if (!imageSrc) {
			return;
		}

		setIsZoomOpen(true);
	};

	const handleCloseZoom = () => {
		setIsZoomOpen(false);
	};

	if (!stableCode.trim() || hasRenderError) {
		return null;
	}

	return (
		<>
			<div
				ref={wrapperRef}
				className="app-bg-mermaid my-4 overflow-hidden rounded-lg"
				style={{
					...surfaceStyle,
					contain: 'layout paint style',
					contentVisibility: 'auto',
					containIntrinsicSize: 'auto 18rem',
				}}
			>
				<div className="app-bg-code-header flex items-center justify-between px-4">
					<span className="app-text-code">Mermaid Diagram</span>

					<div className="flex items-center gap-2">
						{showThemeToggle && (
							<div className="join">
								<button
									type="button"
									className={`btn btn-xs app-text-code join-item border-none bg-transparent shadow-none hover:opacity-60 ${
										themeMode === 'auto' ? 'btn-active' : ''
									}`}
									onClick={() => {
										setThemeMode('auto');
									}}
									aria-pressed={themeMode === 'auto'}
									title="Follow app theme"
								>
									Auto
								</button>

								<button
									type="button"
									className={`btn btn-xs app-text-code join-item border-none bg-transparent shadow-none hover:opacity-60 ${
										themeMode === 'light' ? 'btn-active' : ''
									}`}
									onClick={() => {
										setThemeMode('light');
									}}
									aria-pressed={themeMode === 'light'}
									title="Force light Mermaid theme"
								>
									<FiSun />
								</button>

								<button
									type="button"
									className={`btn btn-xs app-text-code join-item border-none bg-transparent shadow-none hover:opacity-60 ${
										themeMode === 'dark' ? 'btn-active' : ''
									}`}
									onClick={() => {
										setThemeMode('dark');
									}}
									aria-pressed={themeMode === 'dark'}
									title="Force dark Mermaid theme"
								>
									<FiMoon />
								</button>
							</div>
						)}

						{imageSrc ? (
							<DownloadButton
								valueFetcher={fetchDiagramAsBlob}
								size={16}
								fileprefix="diagram"
								isBinary={true}
								language="mermaid"
								className="btn btn-sm app-text-code flex items-center border-none bg-transparent shadow-none hover:opacity-60"
							/>
						) : null}
					</div>
				</div>

				{imageSrc ? (
					<button
						className="flex min-h-65 w-full cursor-zoom-in items-center justify-center overflow-hidden p-1 text-center"
						type="button"
						aria-label="Enlarge Mermaid diagram"
						onClick={handleOpenZoom}
					>
						<img
							src={imageSrc}
							alt=""
							aria-hidden="true"
							loading="lazy"
							decoding="async"
							draggable={false}
							className="pointer-events-none block size-auto max-h-[60vh] max-w-[80%] object-contain"
						/>
					</button>
				) : (
					<div className="text-base-content/60 flex min-h-65 items-center justify-center text-sm" aria-busy="true">
						Rendering diagram
					</div>
				)}
			</div>

			{imageSrc ? (
				<MermaidZoomModal
					isOpen={isZoomOpen}
					onClose={handleCloseZoom}
					imageSrc={imageSrc}
					surfaceStyle={surfaceStyle}
				/>
			) : null}
		</>
	);
}
