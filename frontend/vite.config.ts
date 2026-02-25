/* eslint-disable no-restricted-exports */
import { reactRouter } from '@react-router/dev/vite';
import tailwindcss from '@tailwindcss/vite';
import path from 'node:path';
import license from 'rollup-plugin-license';
import { visualizer } from 'rollup-plugin-visualizer';
import { defineConfig } from 'vite';
import checker from 'vite-plugin-checker';
import tsconfigPaths from 'vite-tsconfig-paths';

// eslint-disable-next-line no-restricted-imports
import pkg from './package.json';

const baseDeps = Object.keys(pkg.dependencies ?? {});

const extraDepsToOptimize = [
	'react-icons/fi',
	'platejs/react',
	'@platejs/basic-styles/react',
	'@platejs/basic-nodes/react',
	'@platejs/indent/react',
	'@platejs/emoji/react',
	'@platejs/tabbable/react',
	'@platejs/list/react',
	'@platejs/combobox/react',
	'@ariakit/react/tab',
];

const excludedDepsToOptimize = new Set(['@emoji-mart/data', '@fontsource-variable/inter']);

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
		plugins: [
			tailwindcss(),
			reactRouter(),
			tsconfigPaths(),

			checker({
				typescript: true,
				eslint: {
					lintCommand: 'eslint . -c ./eslint.config.mjs',
					useFlatConfig: true,
				},
			}),
		],

		base: isProd ? '/frontend/dist/' : '/',

		// Add these configurations for better ESM support
		optimizeDeps: {
			// set optimizeDeps.noDiscovery to true and optimizeDeps.include as undefined or empty to disable.
			// noDiscovery: true,
			// include: undefined,
			include: depsToOptimize,
			esbuildOptions: {
				target: 'esnext',
				supported: {
					bigint: true,
				},
			},
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

			rollupOptions: {
				plugins: rollupPlugins,
				output: {
					format: 'es',

					manualChunks(id) {
						// Only vendor modules
						if (!id.includes('node_modules')) return;

						// Cross-platform path normalization
						const n = id.replace(/\\/g, '/');

						// KaTeX (JS). Fonts are emitted separately as assets automatically.
						if (n.includes('/node_modules/katex/')) return 'libkatex';

						if (n.includes('/node_modules/compromise/')) return 'libcompromise';

						// Mermaid and common companions
						if (n.includes('/node_modules/mermaid/')) return 'libmermaid';

						if (/\/node_modules\/(unified|remark(?:-[^/]+)?|rehype(?:-[^/]+)?)\//.test(n)) {
							return 'libunified';
						}

						// PlateJS suite
						if (/\/node_modules\/(@udecode|@platejs|platejs)\//.test(n)) return 'libplate';
						if (/\/node_modules\/@emoji-mart\//.test(n)) return 'libemojimart';

						// Let Rollup decide otherwise
						return;
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
