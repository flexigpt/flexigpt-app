import { useEffect, useState } from 'react';

import type { ToolListItem } from '@/spec/tool';

import { getAllTools } from '@/apis/list_helper';

export function useTools() {
	const [data, setData] = useState<ToolListItem[]>([]);
	const [loading, setLoading] = useState(true);

	useEffect(() => {
		let cancelled = false;

		getAllTools()
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
