/* eslint-disable no-restricted-exports */
import type { Config } from '@react-router/dev/config';

export default {
	future: {
		v8_middleware: true,
		v8_splitRouteModules: true,
		v8_viteEnvironmentApi: true,
		v8_passThroughRequests: true,
		v8_trailingSlashAwareDataRequests: true,
	},
	appDirectory: 'app',
	ssr: false,
	buildDirectory: 'dist',
} satisfies Config;
