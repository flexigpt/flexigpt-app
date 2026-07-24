import type { CSSProperties } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { FiMoon, FiSun } from 'react-icons/fi';

import type { MermaidConfig } from 'mermaid';

import { Base64EncodeUTF8 } from '@/lib/encode_decode';
import { getUUIDv7 } from '@/lib/uuid_utils';

import { useDebounce } from '@/hooks/use_debounce';
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
			svgMarkup: string;
	  }
	| {
			key: string;
			status: 'error';
			message: string;
	  };

interface CachedMermaidRender {
	key: string;
	renderId: string;
	svgMarkup: string;
}

type ZoomState =
	| {
			isOpen: false;
			svgNode: null;
	  }
	| {
			isOpen: true;
			svgNode: SVGSVGElement;
	  };

const MERMAID_ERROR_MESSAGE = 'Failed to render diagram. Please check the syntax.';

// Successful Mermaid renders survive component unmounts. This is important
// for virtualized or rebuilt markdown trees where scrolling can remount the
// same diagram.
const renderedMermaidCache = new Map<string, CachedMermaidRender>();
const pendingMermaidRenders = new Map<string, Promise<CachedMermaidRender>>();

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
			width: 'auto',
			height: 'auto',
			'max-width': '80%',
			'max-height': '60vh',
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

function getCachedMermaidRender(key: string): CachedMermaidRender | null {
	return renderedMermaidCache.get(key) ?? null;
}

function createMermaidRenderState(
	cached: CachedMermaidRender,
	instanceRenderId: string
): Extract<MermaidRenderState, { status: 'rendered' }> {
	// Mermaid incorporates the supplied render ID into the SVG root ID,
	// marker IDs, CSS selectors, and URL references. Give each mounted copy
	// its own ID so identical diagrams can safely coexist in the document.
	const svgMarkup =
		cached.renderId === instanceRenderId
			? cached.svgMarkup
			: cached.svgMarkup.replaceAll(cached.renderId, instanceRenderId);

	return {
		key: cached.key,
		status: 'rendered',
		svgMarkup,
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
		const rendered: CachedMermaidRender = {
			key,
			renderId,
			svgMarkup: prepareMermaidSvgMarkup(renderResult.svg),
		};

		renderedMermaidCache.set(key, rendered);
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
	const inlineDiagramRef = useRef<HTMLDivElement | null>(null);

	const [zoomState, setZoomState] = useState<ZoomState>({ isOpen: false, svgNode: null });
	const [themeMode, setThemeMode] = useState<'auto' | 'light' | 'dark'>(defaultThemeMode);

	const [instanceRenderId] = useState(() => `mermaid-${getUUIDv7()}`);
	const latestToken = useRef(0);

	// Prevent rendering while code is still settling after streaming/markdown rebuild.
	const stableCode = useDebounce(code, 150);

	const effectiveMermaidTheme = useMemo<'dark' | 'default'>(() => {
		if (themeMode === 'auto') {
			return isDark ? 'dark' : 'default';
		}
		return themeMode === 'dark' ? 'dark' : 'default';
	}, [themeMode, isDark]);

	const renderKey = useMemo(() => `${effectiveMermaidTheme}\u0000${stableCode}`, [effectiveMermaidTheme, stableCode]);

	const [renderState, setRenderState] = useState<MermaidRenderState | null>(null);

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
		() => (cachedRender ? createMermaidRenderState(cachedRender, instanceRenderId) : null),
		[cachedRender, instanceRenderId]
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

				setRenderState(createMermaidRenderState(cr, instanceRenderId));

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
	}, [instanceRenderId, mermaidConfig, onRenderStatusChange, renderKey, stableCode]);

	const currentRenderState = cachedRenderState ?? (renderState?.key === renderKey ? renderState : null);
	const svgMarkup = currentRenderState?.status === 'rendered' ? currentRenderState.svgMarkup : null;
	const hasRenderError = currentRenderState?.status === 'error';

	const attachInlineDiagram = useCallback(
		(element: HTMLDivElement | null) => {
			inlineDiagramRef.current = element;

			if (!element) {
				return;
			}

			element.replaceChildren();

			if (!svgMarkup) {
				return;
			}

			const parser = new DOMParser();
			const doc = parser.parseFromString(svgMarkup, 'image/svg+xml');
			if (doc.querySelector('parsererror')) {
				return;
			}

			const svg = doc.querySelector('svg');
			if (!svg) {
				return;
			}

			element.append(document.importNode(svg, true));
		},
		[svgMarkup]
	);

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
		if (!inlineDiagramRef.current) {
			throw new Error('Container not found');
		}

		const svg = inlineDiagramRef.current.querySelector('svg');
		if (!svg) {
			throw new Error('SVG element not found in container');
		}

		const svgData = new XMLSerializer().serializeToString(svg);
		const svgBase64 = Base64EncodeUTF8(svgData);
		const dataUrl = `data:image/svg+xml;base64,${svgBase64}`;

		return new Promise<Blob>((resolve, reject) => {
			const img = new window.Image();
			img.crossOrigin = 'anonymous';

			img.onload = () => {
				const scaleFactor = 2;
				const canvas = document.createElement('canvas');
				canvas.width = img.width * scaleFactor;
				canvas.height = img.height * scaleFactor;

				const ctx = canvas.getContext('2d');
				if (!ctx) {
					reject(new Error('Canvas context is null'));
					return;
				}

				ctx.scale(scaleFactor, scaleFactor);
				ctx.fillStyle = getDiagramBackgroundColor();
				ctx.fillRect(0, 0, img.width, img.height);
				ctx.drawImage(img, 0, 0);

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

			img.src = dataUrl;
		});
	};

	const handleOpenZoom = () => {
		const svg = inlineDiagramRef.current?.querySelector('svg');
		if (!svg) {
			return;
		}

		setZoomState({
			isOpen: true,
			svgNode: svg.cloneNode(true) as SVGSVGElement,
		});
	};

	const handleCloseZoom = () => {
		setZoomState({
			isOpen: false,
			svgNode: null,
		});
	};

	if (!stableCode.trim() || hasRenderError || !svgMarkup) {
		return null;
	}

	return (
		<>
			<div ref={wrapperRef} className="app-bg-mermaid my-4 overflow-hidden rounded-lg" style={surfaceStyle}>
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

						<DownloadButton
							valueFetcher={fetchDiagramAsBlob}
							size={16}
							fileprefix="diagram"
							isBinary={true}
							language="mermaid"
							className="btn btn-sm app-text-code flex items-center border-none bg-transparent shadow-none hover:opacity-60"
						/>
					</div>
				</div>

				{/* A native button would wrap Mermaid SVG links, so retain a keyboard-operable composite trigger. */}
				<button
					className="flex min-h-65 w-full cursor-zoom-in items-center justify-center overflow-auto p-1 text-center"
					type="button"
					tabIndex={0}
					aria-label="Enlarge Mermaid diagram"
					onClick={handleOpenZoom}
					onKeyDown={event => {
						if (event.key !== 'Enter' && event.key !== ' ') {
							return;
						}
						event.preventDefault();
						handleOpenZoom();
					}}
				>
					<div ref={attachInlineDiagram} className="max-h-[60vh] w-full overflow-auto" />
				</button>
			</div>

			<MermaidZoomModal
				isOpen={zoomState.isOpen}
				onClose={handleCloseZoom}
				svgNode={zoomState.svgNode}
				surfaceStyle={surfaceStyle}
			/>
		</>
	);
}
