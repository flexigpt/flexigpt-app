/* eslint-disable @typescript-eslint/no-unsafe-member-access */
import { TabbablePlugin } from '@platejs/tabbable/react';
import { KEYS } from 'platejs';

export const TabbableKit = [
	TabbablePlugin.configure(({ editor }) => ({
		node: {
			isElement: true,
		},
		options: {
			query: () => {
				if (editor.api.isAt({ start: true }) || editor.api.isAt({ end: true })) return false;

				return !editor.api.some({
					match: (n: any) => {
						return !!(
							(n.type && [KEYS.codeBlock, KEYS.li, KEYS.listTodoClassic, KEYS.table].includes(n.type)) ||
							n.listStyleType
						);
					},
				});
			},
		},
		override: {
			enabled: {
				indent: false,
			},
		},
	})),
];
