/* eslint-disable no-restricted-exports */
import { index, route, type RouteConfig } from '@react-router/dev/routes';

export default [
	index('./home/page.tsx'),
	route('chats', './chats/page.tsx'),
	route('prompts', './prompts/page.tsx'),
	route('tools', './tools/page.tsx'),
	route('settings', './settings/page.tsx'),
	route('modelpresets', './modelpresets/page.tsx'),
] satisfies RouteConfig;
