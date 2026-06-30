/* oxlint-disable no-restricted-exports */
import { reactRouter } from '@react-router/dev/vite';
import tailwindcss from '@tailwindcss/vite';
import path from 'node:path';
import license from 'rollup-plugin-license';
import { visualizer } from 'rollup-plugin-visualizer';
import { defineConfig } from 'vite';

// oxlint-disable-next-line no-restricted-imports
import pkg from './package.json';

const baseDeps = Object.keys(pkg.dependencies ?? {});

const extraDepsToOptimize = [
	'react-icons/fi',
	'platejs/react',
	'@platejs/basic-styles/react',
	'@platejs/basic-nodes/react',
	'@platejs/indent/react',
	'@platejs/tabbable/react',
	'@platejs/list/react',
	'@ariakit/react/tab',
];

const excludedDepsToOptimize = new Set(['@fontsource-variable/inter']);

const normalize = (id: string) => id.replaceAll('\\', '/');

const depsToOptimize = [...new Set([...baseDeps, ...extraDepsToOptimize])].filter(
	dep => !excludedDepsToOptimize.has(dep)
);

// oxlint-disable-next-line no-unused-vars
export default defineConfig(({ mode }) => {
	const genLicenses = process.env.GEN_LICENSES === 'true';
	const jsLicensesOutFile = process.env.LICENSE_JS_OUT
		? path.resolve(process.env.LICENSE_JS_OUT)
		: path.resolve(process.cwd(), '../build/licenses/js-dependency-licenses.txt');

	const analyze = false;
	const analyzerPlugin = visualizer({
		open: true,
		gzipSize: true,
		brotliSize: true,
		filename: 'dist/stats.html',
	});

	const rollupPlugins = [];
	if (analyze) {
		rollupPlugins.push(analyzerPlugin);
	}
	if (genLicenses) {
		rollupPlugins.push(
			license({
				thirdParty: {
					includePrivate: false,
					output: { file: jsLicensesOutFile },
				},
			})
		);
	}

	return {
		// App-wide global settings remain at the top level
		envDir: false,
		plugins: [reactRouter(), tailwindcss()],
		resolve: {
			tsconfigPaths: true,
		},

		// Environment API definitions
		environments: {
			client: {
				// Dependency optimization shifted targeting the client environment
				optimizeDeps: {
					include: depsToOptimize,
					target: 'esnext',
				},
				// Compilation targets isolated cleanly to your client asset runtime
				build: {
					outDir: 'dist',
					target: 'esnext',
					write: true,
					emptyOutDir: true,
					rolldownOptions: {
						plugins: rollupPlugins,
						output: {
							format: 'es',
							codeSplitting: {
								groups: [
									{
										name: 'libkatex',
										test: (id: string) => normalize(id).includes('/node_modules/katex/'),
									},
									{
										name: 'libcompromise',
										test: (id: string) => normalize(id).includes('/node_modules/compromise/'),
									},
									{
										name: 'libmermaid',
										test: (id: string) => normalize(id).includes('/node_modules/mermaid/'),
									},
									{
										name: 'libunified',
										test: (id: string) =>
											/\/node_modules\/(unified|remark(?:-[^/]+)?|rehype(?:-[^/]+)?)\//.test(normalize(id)),
									},
									{
										name: 'libplate',
										test: (id: string) => /\/node_modules\/(@udecode|@platejs|platejs)\//.test(normalize(id)),
									},
								],
							},
						},
					},
				},
			},
		},

		// Native Web Worker compilation settings
		worker: {
			format: 'es',
		},

		// Test blocks are maintained independently by Vitest integration layers
		test: {
			globals: true,
			environment: 'jsdom',
			setupFiles: './vitest.setup.ts',
			coverage: {
				reporter: ['text', 'html'],
				exclude: ['wailsjs', 'dist', 'vite.config.ts'],
			},
		},
	};
});
