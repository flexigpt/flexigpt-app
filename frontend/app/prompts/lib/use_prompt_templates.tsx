import { useEffect, useState } from 'react';

import type { PromptTemplate, PromptTemplateListItem } from '@/spec/prompt';
import { PromptTemplateKind } from '@/spec/prompt';

import { promptStoreAPI } from '@/apis/baseapi';
import { getAllPromptTemplates } from '@/apis/list_helper';

export function usePromptTemplates() {
	const [data, setData] = useState<PromptTemplateListItem[]>([]);
	const [loading, setLoading] = useState(true);

	useEffect(() => {
		let cancelled = false;

		void (async () => {
			try {
				const res = await getAllPromptTemplates(undefined, undefined, false);
				if (cancelled) {
					return;
				}
				setData(
					res.filter(
						item =>
							item.kind === PromptTemplateKind.Generic ||
							(item.kind === PromptTemplateKind.InstructionsOnly && !item.isResolved)
					)
				);
			} catch (err) {
				if (!cancelled) {
					console.error('Failed to load prompt templates', err);
				}
			} finally {
				if (!cancelled) {
					setLoading(false);
				}
			}
		})();

		return () => {
			cancelled = true;
		};
	}, []);

	return { data, loading };
}

/**
 * @public
 */
export function usePromptTemplate(bundleID: string, slug: string, version: string) {
	const [tmpl, setTmpl] = useState<PromptTemplate | undefined>();

	useEffect(() => {
		if (!bundleID || !slug || !version) {
			return;
		}

		let cancelled = false;

		void (async () => {
			try {
				const res = await promptStoreAPI.getPromptTemplate(bundleID, slug, version);
				if (cancelled) {
					return;
				}
				setTmpl(res);
			} catch (err) {
				if (!cancelled) {
					console.error('Failed to load prompt template', err);
				}
			}
		})();

		return () => {
			cancelled = true;
		};
	}, [bundleID, slug, version]);

	return tmpl;
}
