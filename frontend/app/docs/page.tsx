import { useEffect } from 'react';

import { FiBookOpen } from 'react-icons/fi';

import { useLocation } from 'react-router';

import { useTitleBarContent } from '@/hooks/use_title_bar';

import { EnhancedMarkdown } from '@/components/markdown_enhanced';
import { PageFrame } from '@/components/page_frame';

import { docsCategories } from '@/docs/manifest';

function scrollToSection(sectionID: string) {
	document.getElementById(sectionID)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

// eslint-disable-next-line no-restricted-exports
export default function DocsPage() {
	const location = useLocation();

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

	useEffect(() => {
		const sectionID = decodeURIComponent(location.hash.replace(/^#/, ''));
		if (!sectionID) {
			return;
		}

		const frame = requestAnimationFrame(() => {
			scrollToSection(sectionID);
		});

		return () => {
			cancelAnimationFrame(frame);
		};
	}, [location.hash]);

	return (
		<PageFrame>
			<div className="mx-auto flex w-full flex-col gap-6 px-2 py-6 lg:flex-row">
				<aside className="lg:sticky lg:top-4 lg:h-fit lg:w-60 lg:self-start">
					<div className="bg-base-100 rounded-2xl p-4 shadow-lg">
						<div className="flex items-center gap-2 text-lg font-semibold">
							<FiBookOpen size={18} />
							<span>In-app Docs</span>
						</div>
						<p className="mt-2 text-sm opacity-70">Bundled user guide, and architecture reference for FlexiGPT.</p>

						<div className="mt-4 flex flex-col gap-4">
							{docsCategories.map(category => (
								<div key={category.id} className="border-base-300/60 rounded-2xl border p-2">
									<div className="px-2 pt-1">
										<div className="text-xs font-semibold tracking-wide uppercase opacity-60">{category.title}</div>
										<p className="mt-1 text-xs opacity-70">{category.summary}</p>
									</div>
									<div className="mt-2 flex flex-col gap-2">
										{category.sections.map(section => (
											<button
												key={section.id}
												type="button"
												className="bg-base-200 hover:bg-base-300 rounded-xl px-3 py-3 text-left transition-colors"
												onClick={() => {
													scrollToSection(section.id);
												}}
											>
												<div className="text-sm font-semibold">{section.title}</div>
											</button>
										))}
									</div>
								</div>
							))}
						</div>
					</div>
				</aside>

				<div className="min-w-0 flex-1 space-y-6">
					<section className="bg-base-100 rounded-2xl p-8 shadow-lg">
						<div className="flex items-center gap-3">
							<div className="bg-base-200 rounded-2xl p-3">
								<FiBookOpen size={24} />
							</div>
							<div>
								<h1 className="text-2xl font-semibold">FlexiGPT Docs</h1>
								<p className="mt-1 text-sm opacity-80">User guide and architecture reference.</p>
							</div>
						</div>
					</section>

					{docsCategories.map(category => (
						<>
							{category.sections.map(section => (
								<section key={section.id} id={section.id} className="bg-base-100 rounded-2xl p-8 shadow-lg">
									<div className="mb-4">
										<h3 className="text-xl font-semibold">{section.title}</h3>
										<p className="mt-1 text-sm opacity-70">{section.summary}</p>
									</div>
									<div className="min-w-0 text-sm leading-6">
										<EnhancedMarkdown text={section.body} hideMermaidCode={true} />
									</div>
								</section>
							))}
						</>
					))}
				</div>
			</div>
		</PageFrame>
	);
}
