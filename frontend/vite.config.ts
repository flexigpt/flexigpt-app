/* eslint-disable no-restricted-exports */
import { reactRouter } from '@react-router/dev/vite';
import tailwindcss from '@tailwindcss/vite';
import path from 'node:path';
import license from 'rollup-plugin-license';
import { visualizer } from 'rollup-plugin-visualizer';
import { defineConfig } from 'vite';

// eslint-disable-next-line no-restricted-imports
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

const normalize = (id: string) => id.replace(/\\/g, '/');

const depsToOptimize = [...new Set([...baseDeps, ...extraDepsToOptimize])].filter(
	dep => !excludedDepsToOptimize.has(dep)
);

export default defineConfig(({ mode }) => {
	const isProd = mode === 'production';
	const genLicenses = process.env.GEN_LICENSES === 'true';
	const genLicensesForceWrite = process.env.GEN_LICENSES_FORCE_WRITE === 'true';
	// Allow CI/scripts to override output location deterministically.
	// If not set, keep the existing default (repoRoot/build/licenses/...).
	const jsLicensesOutFile = process.env.LICENSE_JS_OUT
		? path.resolve(process.env.LICENSE_JS_OUT)
		: path.resolve(process.cwd(), '../build/licenses/js-dependency-licenses.txt');

	// const analyze = process.env.ANALYZE === 'true' || !isProd;
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
					output: {
						file: jsLicensesOutFile,
					},
				},
			})
		);
	}
	return {
		plugins: [reactRouter(), tailwindcss()],

		base: isProd ? '/frontend/dist/' : '/',

		resolve: {
			// This replaces the need for the vite-tsconfig-paths plugin
			tsconfigPaths: true,
		},

		// Add these configurations for better ESM support
		optimizeDeps: {
			// set optimizeDeps.noDiscovery to true and optimizeDeps.include as undefined or empty to disable.
			// noDiscovery: true,
			// include: undefined,
			include: depsToOptimize,
			target: 'esnext',
		},

		build: {
			outDir: 'dist',
			target: 'esnext',
			/**
			 * License generation runs a Vite build.
			 * By default we do NOT write dist/ (faster + avoids disturbing local builds),
			 * but allow forcing output if some environment/tooling prevents the license plugin from writing.
			 */
			write: !(genLicenses && !genLicensesForceWrite),
			emptyOutDir: !(genLicenses && !genLicensesForceWrite),
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

		worker: {
			format: 'es',
		},

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
