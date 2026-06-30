/* oxlint-disable no-restricted-exports */
import type { Config } from '@react-router/dev/config';

export default {
	basename: '/',
	splitRouteModules: true,
	appDirectory: 'app',
	ssr: false,
	buildDirectory: 'dist',
	prerender: true,
} satisfies Config;
