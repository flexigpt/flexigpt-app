import { useCallback, useEffect, useRef } from 'react';

import { FiBookOpen, FiChevronLeft, FiChevronRight } from 'react-icons/fi';

import { useLocation, useNavigate } from 'react-router';

import { useTitleBarContent } from '@/hooks/use_title_bar';

import { EnhancedMarkdown } from '@/components/markdown_enhanced';
import { PageFrame } from '@/components/page_frame';

import { docsCategories } from '@/docs/manifest';

const DOC_QUERY_PARAM = 'doc';
const DESKTOP_MEDIA_QUERY = '(min-width: 1024px)';
const SCROLL_TOP_OFFSET_PX = 16;

const docSections = docsCategories.flatMap(category =>
	category.sections.map(section => ({
		...section,
		categoryID: category.id,
		categoryTitle: category.title,
	}))
);

const docSectionIDs = new Set(docSections.map(section => section.id));

function getSectionIDFromHash(hash: string) {
	return decodeURIComponent(hash.replace(/^#/, ''));
}

function getHashFromSectionID(sectionID: string) {
	return `#${encodeURIComponent(sectionID)}`;
}

function getDocIDFromSearch(search: string) {
	const params = new URLSearchParams(search);
	const value = params.get(DOC_QUERY_PARAM);
	return value && docSectionIDs.has(value) ? value : '';
}

function buildDocSearch(baseSearch: string, docID: string) {
	const params = new URLSearchParams(baseSearch);
	params.set(DOC_QUERY_PARAM, docID);

	const next = params.toString();
	return next ? `?${next}` : '';
}

function buildDocHref(baseSearch: string, docID: string, headingID?: string) {
	return `${buildDocSearch(baseSearch, docID)}${headingID ? getHashFromSectionID(headingID) : ''}`;
}

function prefersReducedMotion() {
	return typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

function isDesktopViewport() {
	return typeof window !== 'undefined' && window.matchMedia(DESKTOP_MEDIA_QUERY).matches;
}

function replaceHashWithoutRouter(headingID?: string) {
	if (typeof window === 'undefined') {
		return;
	}

	const nextURL = `${window.location.pathname}${window.location.search}${headingID ? getHashFromSectionID(headingID) : ''}`;
	window.history.replaceState(window.history.state, '', nextURL);
}

function tryParseURL(href: string) {
	try {
		return new URL(href, window.location.href);
	} catch {
		return null;
	}
}

function scrollElementInsideContainer(
	container: HTMLElement,
	element: HTMLElement,
	behavior: ScrollBehavior = 'auto',
	offset = SCROLL_TOP_OFFSET_PX
) {
	const containerRect = container.getBoundingClientRect();
	const elementRect = element.getBoundingClientRect();

	const nextTop = container.scrollTop + (elementRect.top - containerRect.top) - offset;

	container.scrollTo({
		top: Math.max(0, nextTop),
		behavior,
	});
}

// eslint-disable-next-line no-restricted-exports
export default function DocsPage() {
	const location = useLocation();
	const navigate = useNavigate();

	const pageScrollRef = useRef<HTMLDivElement | null>(null);
	const mainScrollRef = useRef<HTMLElement | null>(null);
	const sidebarItemRefs = useRef<Record<string, HTMLAnchorElement | null>>({});

	const explicitDocID = getDocIDFromSearch(location.search);
	const hashTargetID = getSectionIDFromHash(location.hash);
	const legacyHashDocID = !explicitDocID && hashTargetID && docSectionIDs.has(hashTargetID) ? hashTargetID : '';
	const activeDocID = explicitDocID || legacyHashDocID || docSections[0]?.id || '';

	const activeDocIndex = docSections.findIndex(section => section.id === activeDocID);
	const activeDocSection = docSections[activeDocIndex] ?? docSections[0] ?? null;
	const prevDocSection = activeDocIndex > 0 ? docSections[activeDocIndex - 1] : null;
	const nextDocSection =
		activeDocIndex >= 0 && activeDocIndex < docSections.length - 1 ? docSections[activeDocIndex + 1] : null;

	useTitleBarContent(
		{
			center: (
				<div className="mx-auto flex items-center justify-center gap-2 text-sm font-medium opacity-70">
					<FiBookOpen size={16} />
					<span>Docs</span>
				</div>
			),
		},
		[]
	);

	const scrollContentToTop = useCallback(() => {
		pageScrollRef.current?.scrollTo({ top: 0, behavior: 'auto' });
		mainScrollRef.current?.scrollTo({ top: 0, behavior: 'auto' });
	}, []);

	const scrollToTargetInActiveContainer = useCallback((targetID: string, behavior: ScrollBehavior = 'auto') => {
		const element = document.getElementById(targetID);
		if (!element) {
			return false;
		}

		const scrollContainer = isDesktopViewport() ? mainScrollRef.current : pageScrollRef.current;

		if (scrollContainer) {
			scrollElementInsideContainer(scrollContainer, element, behavior);
			return true;
		}

		element.scrollIntoView({
			behavior,
			block: 'start',
		});

		return true;
	}, []);

	const scrollInsideCurrentDoc = useCallback(
		(targetID: string) => {
			replaceHashWithoutRouter(targetID);

			requestAnimationFrame(() => {
				scrollToTargetInActiveContainer(targetID, prefersReducedMotion() ? 'auto' : 'smooth');
			});
		},
		[scrollToTargetInActiveContainer]
	);

	const navigateToDocPage = useCallback(
		(docID: string, options?: { headingID?: string; replace?: boolean; baseSearch?: string }) => {
			if (!docID || !docSectionIDs.has(docID)) {
				return;
			}

			const headingID = options?.headingID;

			if (docID === activeDocID) {
				if (headingID) {
					scrollInsideCurrentDoc(headingID);
				} else {
					replaceHashWithoutRouter();
					requestAnimationFrame(() => {
						scrollContentToTop();
					});
				}
				return;
			}

			navigate(
				{
					pathname: location.pathname,
					search: buildDocSearch(options?.baseSearch ?? location.search, docID),
					hash: headingID ? getHashFromSectionID(headingID) : '',
				},
				{ replace: options?.replace ?? false }
			);
		},
		[activeDocID, location.pathname, location.search, navigate, scrollContentToTop, scrollInsideCurrentDoc]
	);

	const handleMarkdownLinkClick = useCallback(
		(href: string) => {
			const rawHref = href.trim();
			if (!rawHref) {
				return false;
			}

			if (rawHref.startsWith('#')) {
				const targetID = getSectionIDFromHash(rawHref);
				if (!targetID) {
					return false;
				}

				if (document.getElementById(targetID)) {
					scrollInsideCurrentDoc(targetID);
					return true;
				}

				if (docSectionIDs.has(targetID)) {
					navigateToDocPage(targetID);
					return true;
				}

				return false;
			}

			const url = tryParseURL(rawHref);
			if (!url) {
				return false;
			}

			if (url.origin !== window.location.origin) {
				return false;
			}

			if (url.pathname === location.pathname) {
				const explicitTargetDocID = getDocIDFromSearch(url.search);
				const urlHashTargetID = getSectionIDFromHash(url.hash);

				if (explicitTargetDocID) {
					const headingID = urlHashTargetID && urlHashTargetID !== explicitTargetDocID ? urlHashTargetID : undefined;

					navigateToDocPage(explicitTargetDocID, {
						headingID,
						baseSearch: url.search,
					});
					return true;
				}

				if (urlHashTargetID) {
					if (document.getElementById(urlHashTargetID)) {
						scrollInsideCurrentDoc(urlHashTargetID);
						return true;
					}

					if (docSectionIDs.has(urlHashTargetID)) {
						navigateToDocPage(urlHashTargetID);
						return true;
					}

					return false;
				}

				navigate(`${url.pathname}${url.search}${url.hash}`);
				return true;
			}

			navigate(`${url.pathname}${url.search}${url.hash}`);
			return true;
		},
		[location.pathname, navigate, navigateToDocPage, scrollInsideCurrentDoc]
	);

	useEffect(() => {
		if (!legacyHashDocID) {
			return;
		}

		navigate(
			{
				pathname: location.pathname,
				search: buildDocSearch(location.search, legacyHashDocID),
				hash: '',
			},
			{ replace: true }
		);
	}, [legacyHashDocID, location.pathname, location.search, navigate]);

	useEffect(() => {
		if (!activeDocID) {
			return;
		}

		sidebarItemRefs.current[activeDocID]?.scrollIntoView({
			block: 'nearest',
			inline: 'nearest',
			behavior: 'auto',
		});
	}, [activeDocID]);

	useEffect(() => {
		if (!activeDocSection) {
			return;
		}

		const frame = requestAnimationFrame(() => {
			const targetID = getSectionIDFromHash(window.location.hash);

			if (targetID) {
				scrollToTargetInActiveContainer(targetID, 'auto');
				return;
			}

			scrollContentToTop();
		});

		return () => {
			cancelAnimationFrame(frame);
		};
	}, [activeDocSection, location.hash, scrollContentToTop, scrollToTargetInActiveContainer]);

	if (!activeDocSection) {
		return (
			<PageFrame contentScrollable={false}>
				<div className="flex h-full items-center justify-center p-6">
					<div className="bg-base-100 rounded-2xl p-8 shadow-lg">
						<h1 className="text-xl font-semibold">Docs unavailable</h1>
						<p className="mt-2 text-sm opacity-70">No bundled documentation sections were found.</p>
					</div>
				</div>
			</PageFrame>
		);
	}

	return (
		<PageFrame contentScrollable={false}>
			<div ref={pageScrollRef} className="h-full min-h-0 overflow-y-auto lg:overflow-hidden">
				<div className="mx-auto flex min-h-full w-full flex-col gap-6 px-2 py-6 lg:h-full lg:min-h-0 lg:flex-row">
					<aside className="lg:h-full lg:min-h-0 lg:w-72 lg:shrink-0">
						<div className="bg-base-100 rounded-2xl shadow-lg lg:h-full lg:overflow-y-auto lg:overscroll-contain">
							<div className="p-4">
								<div className="flex items-center gap-2 text-lg font-semibold">
									<FiBookOpen size={18} />
									<span>In-app Docs</span>
								</div>

								<p className="mt-2 text-sm opacity-70">Bundled user guide and architecture reference for FlexiGPT.</p>

								<p className="mt-3 text-xs opacity-60">
									{activeDocIndex + 1} of {docSections.length}
								</p>
							</div>

							<nav aria-label="Docs navigation" className="px-3 pb-4">
								<div className="flex flex-col gap-4">
									{docsCategories.map(category => {
										const categoryIsActive = category.sections.some(section => section.id === activeDocID);

										return (
											<section
												key={category.id}
												className={`rounded-2xl border p-2 ${
													categoryIsActive ? 'border-primary/30' : 'border-base-300/60'
												}`}
											>
												<div className="bg-base-100/95 sticky top-0 z-10 -mx-1 mb-2 rounded-xl px-3 py-2 backdrop-blur">
													<div className="text-xs font-semibold tracking-wide uppercase opacity-60">
														{category.title}
													</div>
													<p className="mt-1 text-xs opacity-70">{category.summary}</p>
												</div>

												<div className="flex flex-col gap-1">
													{category.sections.map(section => {
														const isActive = activeDocID === section.id;

														return (
															<a
																key={section.id}
																ref={node => {
																	sidebarItemRefs.current[section.id] = node;
																}}
																href={buildDocHref(location.search, section.id)}
																aria-current={isActive ? 'page' : undefined}
																className={`rounded-xl px-3 py-2.5 text-left transition-colors ${
																	isActive
																		? 'bg-primary/10 text-primary'
																		: 'text-base-content/80 hover:bg-base-200 hover:text-base-content'
																}`}
																onClick={event => {
																	event.preventDefault();
																	navigateToDocPage(section.id);
																}}
															>
																<div className="text-sm font-semibold">{section.title}</div>
															</a>
														);
													})}
												</div>
											</section>
										);
									})}
								</div>
							</nav>
						</div>
					</aside>

					<main
						ref={mainScrollRef}
						className="min-w-0 flex-1 lg:h-full lg:min-h-0 lg:overflow-y-auto lg:overscroll-contain lg:pr-2"
					>
						<div className="space-y-6">
							<section className="bg-base-100 rounded-2xl p-8 shadow-lg">
								<div className="flex items-start gap-3">
									<div className="bg-base-200 rounded-2xl p-3">
										<FiBookOpen size={24} />
									</div>

									<div className="min-w-0">
										<div className="text-primary text-xs font-semibold tracking-wide uppercase">
											{activeDocSection.categoryTitle}
										</div>
										<h1 className="mt-1 text-2xl font-semibold">{activeDocSection.title}</h1>
										<p className="mt-1 text-sm opacity-80">{activeDocSection.summary}</p>
									</div>
								</div>
							</section>

							<article id={activeDocSection.id} className="bg-base-100 rounded-2xl p-8 shadow-lg">
								<div className="min-w-0 text-sm leading-6">
									<EnhancedMarkdown
										text={activeDocSection.body}
										hideMermaidCode={true}
										hideH1Title={true}
										onLinkClick={handleMarkdownLinkClick}
									/>
								</div>
							</article>

							<section className="bg-base-100 rounded-2xl p-4 shadow-lg">
								<div className="flex flex-col gap-3 sm:flex-row sm:items-stretch sm:justify-between">
									<div className="flex-1">
										{prevDocSection ? (
											<button
												type="button"
												className="bg-base-200 hover:bg-base-300 flex h-full w-full items-center gap-3 rounded-2xl px-4 py-4 text-left transition-colors"
												onClick={() => {
													navigateToDocPage(prevDocSection.id);
												}}
											>
												<FiChevronLeft size={18} />
												<div className="min-w-0">
													<div className="text-xs opacity-60">Previous</div>
													<div className="truncate text-sm font-semibold">{prevDocSection.title}</div>
												</div>
											</button>
										) : (
											<div className="border-base-300/60 h-full rounded-2xl border px-4 py-4 text-sm opacity-50">
												Start of docs
											</div>
										)}
									</div>

									<div className="flex-1">
										{nextDocSection ? (
											<button
												type="button"
												className="bg-base-200 hover:bg-base-300 flex h-full w-full items-center justify-between gap-3 rounded-2xl px-4 py-4 text-left transition-colors"
												onClick={() => {
													navigateToDocPage(nextDocSection.id);
												}}
											>
												<div className="min-w-0">
													<div className="text-xs opacity-60">Next</div>
													<div className="truncate text-sm font-semibold">{nextDocSection.title}</div>
												</div>
												<FiChevronRight size={18} />
											</button>
										) : (
											<div className="border-base-300/60 h-full rounded-2xl border px-4 py-4 text-sm opacity-50">
												End of docs
											</div>
										)}
									</div>
								</div>
							</section>
						</div>
					</main>
				</div>
			</div>
		</PageFrame>
	);
}
