import { useEffect, useState } from 'react';

import { type PromptTemplate, PromptTemplateKind, type PromptTemplateListItem } from '@/spec/prompt';

import { promptStoreAPI } from '@/apis/baseapi';
import { getAllPromptTemplates } from '@/apis/list_helper';

export function usePromptTemplates() {
	const [data, setData] = useState<PromptTemplateListItem[]>([]);
	const [loading, setLoading] = useState(true);

	useEffect(() => {
		let cancelled = false;

		getAllPromptTemplates(undefined, undefined, false, [PromptTemplateKind.Generic])
			.then(res => {
				if (cancelled) {
					return;
				}
				setData(res);
			})
			.finally(() => {
				if (cancelled) {
					return;
				}
				setLoading(false);
			});

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
		if (!bundleID || !slug || !version) return;

		let cancelled = false;

		promptStoreAPI.getPromptTemplate(bundleID, slug, version).then(res => {
			if (cancelled) {
				return;
			}
			setTmpl(res);
		});

		return () => {
			cancelled = true;
		};
	}, [bundleID, slug, version]);

	return tmpl;
}
