import { useEffect } from 'react';

import { FiBookOpen } from 'react-icons/fi';

import { useLocation } from 'react-router';

import { useTitleBarContent } from '@/hooks/use_title_bar';

import { EnhancedMarkdown } from '@/components/markdown_enhanced';
import { PageFrame } from '@/components/page_frame';

import { docsSections } from '@/docs/manifest';

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
			<div className="mx-auto flex w-full max-w-7xl flex-col gap-6 px-2 py-6 lg:flex-row">
				<aside className="lg:sticky lg:top-4 lg:h-fit lg:w-60 lg:self-start">
					<div className="bg-base-100 rounded-2xl p-2 shadow-lg">
						<div className="flex items-center gap-2 text-lg font-semibold">
							<FiBookOpen size={18} />
							<span>In-app Docs</span>
						</div>
						<div className="mt-4 flex flex-col gap-2">
							{docsSections.map(section => (
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
				</aside>

				<div className="min-w-0 flex-1 space-y-6">
					<section className="bg-base-100 rounded-2xl p-6 shadow-lg">
						<div className="mb-4 flex items-center gap-3">
							<div className="bg-base-200 rounded-2xl p-3">
								<FiBookOpen size={24} />
							</div>
							<div>
								<h1 className="text-2xl font-semibold">FlexiGPT Docs</h1>
								<p className="mt-1 text-sm opacity-80">
									Learn the fastest way to get set up, understand how FlexiGPT works, and get comfortable using LLMs.
								</p>
							</div>
						</div>
					</section>

					{docsSections.map(section => (
						<section key={section.id} id={section.id} className="bg-base-100 rounded-2xl p-6 shadow-lg">
							<div className="mb-4">
								<h2 className="text-xl font-semibold">{section.title}</h2>
								<p className="mt-1 text-sm opacity-70">{section.summary}</p>
							</div>
							<div className="min-w-0 text-sm leading-6">
								<EnhancedMarkdown text={section.body} />
							</div>
						</section>
					))}
				</div>
			</div>
		</PageFrame>
	);
}
