import reactHooks from 'eslint-plugin-react-hooks';
import reactYouMightNotNeedAnEffect from 'eslint-plugin-react-you-might-not-need-an-effect';
import { defineConfig, globalIgnores } from 'eslint/config';
import globals from 'globals';
import tseslint from 'typescript-eslint';

// eslint-disable-next-line no-restricted-exports
export default defineConfig(
	globalIgnores(['dist/**', 'app/apis/wailsjs/**', '.react-router/**']),
	{
		linterOptions: {
			reportUnusedDisableDirectives: 'off',
		},
	},
	{
		files: ['**/*.{js,jsx,mjs,ts,tsx}'],
		plugins: {
			'react-hooks': reactHooks,
			'react-you-might-not-need-an-effect': reactYouMightNotNeedAnEffect,
			// Optional here for now, but keep if you may turn on TS ESLint rules later.
			'@typescript-eslint': tseslint.plugin,
		},
		languageOptions: {
			parser: tseslint.parser,
			parserOptions: {
				ecmaVersion: 'latest',
				sourceType: 'module',
				ecmaFeatures: { jsx: true },

				// Turn these on only if you later add type-aware TS rules.
				// projectService: true,
				// tsconfigRootDir: import.meta.dirname,
			},
			globals: {
				...globals.browser,
			},
		},
		rules: {
			// Full React Hooks recommended rule set:
			...reactHooks.configs.flat.recommended.rules,

			// Enable individual rules manually:
			// 'react-hooks/rules-of-hooks': 'error',

			// Uncomment if you want this too without enabling full recommended:
			// 'react-hooks/exhaustive-deps': 'error',

			// 'react-hooks/set-state-in-effect': 'error',

			// ...reactYouMightNotNeedAnEffect.configs.strict.rules,
		},
	}
);
