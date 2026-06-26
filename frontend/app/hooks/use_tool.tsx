import { useEffect, useState } from 'react';

import type { ToolListItem } from '@/spec/tool';

import { getAllTools } from '@/apis/list_helper';

export function useTools() {
	const [data, setData] = useState<ToolListItem[]>([]);
	const [loading, setLoading] = useState(true);

	useEffect(() => {
		let cancelled = false;

		void (async () => {
			try {
				const res = await getAllTools();
				if (cancelled) {
					return;
				}
				setData(res);
			} catch (err) {
				if (!cancelled) {
					console.error('Failed to load tools', err);
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
