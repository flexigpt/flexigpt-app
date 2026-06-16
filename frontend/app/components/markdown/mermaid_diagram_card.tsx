import { type CSSProperties, useEffect, useMemo, useRef, useState } from 'react';

import { FiMoon, FiSun } from 'react-icons/fi';

import { type MermaidConfig } from 'mermaid';

import { Base64EncodeUTF8 } from '@/lib/encode_decode';
import { getUUIDv7 } from '@/lib/uuid_utils';

import { useDebounce } from '@/hooks/use_debounce';
import { renderMermaidQueued, useIsDarkMermaid } from '@/hooks/use_mermaid';

import { DownloadButton } from '@/components/download_button';
import { MermaidZoomModal } from '@/components/markdown/mermaid_zoom_modal';

interface MermaidDiagramProps {
	code: string;
	/**
	 * auto = follow app theme.
	 * light/dark = force Mermaid theme for this diagram only
	 */
	defaultThemeMode?: 'auto' | 'light' | 'dark';
	showThemeToggle?: boolean;
}

export function MermaidDiagram({ code, defaultThemeMode = 'auto', showThemeToggle = true }: MermaidDiagramProps) {
	const isDark = useIsDarkMermaid();

	const wrapperRef = useRef<HTMLDivElement | null>(null);

	const inlineDiagramRef = useRef<HTMLDivElement | null>(null);
	const [error, setError] = useState<string | null>(null);
	const [svgNode, setSvgNode] = useState<SVGSVGElement | null>(null);
	const [zoomOpen, setZoomOpen] = useState(false);

	const [themeMode, setThemeMode] = useState<'auto' | 'light' | 'dark'>(defaultThemeMode);
	const uniqueId = useRef(`mermaid-${getUUIDv7()}`);
	const renderSeq = useRef(0);
	const latestToken = useRef(0);

	// Prevent rendering while code is still settling after streaming/markdown rebuild.
	const stableCode = useDebounce(code, 150);

	const effectiveMermaidTheme = useMemo<'dark' | 'default'>(() => {
		if (themeMode === 'auto') return isDark ? 'dark' : 'default';
		return themeMode === 'dark' ? 'dark' : 'default';
	}, [themeMode, isDark]);

	// Per-diagram surface override: only when user explicitly selects light/dark.
	// In auto mode, it stays consistent with the app’s DaisyUI theme.
	const surfaceStyle = useMemo<CSSProperties | undefined>(() => {
		if (themeMode === 'auto') return undefined;
		const forcedDark = themeMode === 'dark';
		return {
			// override the variables your utility classes use
			['--bg-mermaid' as any]: forcedDark ? 'var(--mermaid-surface-dark)' : 'var(--mermaid-surface-light)',
			['--bg-code-header' as any]: forcedDark ? 'var(--mermaid-header-bg-dark)' : 'var(--mermaid-header-bg-light)',
			['--text-code' as any]: forcedDark ? 'var(--mermaid-header-text-dark)' : 'var(--mermaid-header-text-light)',
			// helps native form controls / scrollbars look correct inside the block
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
			// (Mermaid themes sometimes embed a background; transparency avoids mismatches.)
			themeVariables: {
				background: 'transparent',
			} as any,
		};
	}, [effectiveMermaidTheme]);

	useEffect(() => {
		let isMounted = true;
		const token = ++latestToken.current;

		if (inlineDiagramRef.current) {
			const renderId = `${uniqueId.current}-${renderSeq.current++}`;
			renderMermaidQueued(renderId, stableCode, mermaidConfig)
				.then(renderResult => {
					if (!isMounted || token !== latestToken.current || !inlineDiagramRef.current) return;
					{
						inlineDiagramRef.current.innerHTML = renderResult.svg;
						// Center the SVG with inline styles
						const svg = inlineDiagramRef.current.querySelector('svg');
						if (svg) {
							svg.style.display = 'block';
							svg.style.margin = 'auto';
							svg.style.width = 'auto';
							svg.style.height = 'auto';
							svg.style.maxWidth = '80%';
							svg.style.maxHeight = '60vh';
							// Ensure wrapper background is what you see (not SVG background)
							svg.style.backgroundColor = 'transparent';
							// Some Mermaid outputs use a background rect; harmless if not present
							const bg = svg.querySelector('rect.background');
							if (bg) bg.setAttribute('fill', 'transparent');
							setSvgNode(svg.cloneNode(true) as SVGSVGElement);
						}
						setError(null);
					}
				})
				.catch((e: unknown) => {
					if (!isMounted || token !== latestToken.current) return;
					setError('Failed to render diagram. Please check the syntax.');
					console.error('syntax error:', e);
				});
		}
		return () => {
			isMounted = false;
		};
	}, [stableCode, mermaidConfig]);

	const getDiagramBackgroundColor = (): string => {
		const el = wrapperRef.current;
		if (!el) return '#ffffff';
		const bg = window.getComputedStyle(el).backgroundColor;
		// If transparent, default to white so PNG is not transparent-black-ish
		if (!bg || bg === 'transparent' || bg === 'rgba(0, 0, 0, 0)') return '#ffffff';
		return bg;
	};

	const fetchDiagramAsBlob = async (): Promise<Blob> => {
		if (!inlineDiagramRef.current) throw new Error('Container not found');
		const svg = inlineDiagramRef.current.querySelector('svg');
		if (!svg) throw new Error('SVG element not found in container');

		const svgData = new XMLSerializer().serializeToString(svg);
		// Encode SVG as base64 data URL
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
				// ctx.fillRect(0, 0, canvas.width, canvas.height);
				// After scaling, draw in CSS pixels
				ctx.fillRect(0, 0, img.width, img.height);
				ctx.drawImage(img, 0, 0);
				canvas.toBlob(
					blob => {
						if (blob) resolve(blob);
						else reject(new Error('Canvas is empty'));
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

	if (!stableCode.trim()) {
		setError(null);
		return;
	}

	return (
		<>
			<div ref={wrapperRef} className="bg-mermaid my-4 overflow-hidden rounded-lg" style={surfaceStyle}>
				{/* header bar */}
				<div className="bg-code-header flex items-center justify-between px-4">
					<span className="text-code">Mermaid diagram</span>

					<div className="flex items-center gap-2">
						{showThemeToggle && (
							<div className="join">
								<button
									type="button"
									className={`btn btn-xs text-code join-item border-none bg-transparent shadow-none hover:opacity-60 ${themeMode === 'auto' ? 'btn-active' : ''}`}
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
									className={`btn btn-xs text-code join-item border-none bg-transparent shadow-none hover:opacity-60 ${themeMode === 'light' ? 'btn-active' : ''}`}
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
									className={`btn btn-xs text-code join-item border-none bg-transparent shadow-none hover:opacity-60 ${themeMode === 'dark' ? 'btn-active' : ''}`}
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
							className="btn btn-sm text-code flex items-center border-none bg-transparent shadow-none hover:opacity-60"
						/>
					</div>
				</div>

				<div
					className="flex min-h-65 w-full cursor-zoom-in items-center justify-center overflow-auto p-1 text-center"
					onClick={() => {
						if (!error) {
							setZoomOpen(true);
						}
					}}
				>
					{error ? (
						<svg width="300" height="100" className="rounded border-2 border-red-500">
							<rect width="300" height="100" fill="#fff" stroke="#e53e3e" strokeWidth="2" rx="8" />
							<text x="150" y="55" textAnchor="middle" fill="#e53e3e" fontSize="18" fontFamily="monospace">
								Mermaid syntax error
							</text>
						</svg>
					) : (
						<div ref={inlineDiagramRef} className="max-h-[60vh] w-full overflow-auto" />
					)}
				</div>
			</div>

			<MermaidZoomModal
				isOpen={zoomOpen}
				onClose={() => {
					setZoomOpen(false);
				}}
				svgNode={svgNode}
				surfaceStyle={surfaceStyle}
			/>
		</>
	);
}
