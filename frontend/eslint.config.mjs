import js from '@eslint/js';
import eslintConfigPrettier from 'eslint-config-prettier';
import oxlint from 'eslint-plugin-oxlint';
// oxlint pending: react compiler rules stability.
import react from 'eslint-plugin-react';
// oxlint pending: react compiler rules stability.
import reactHooks from 'eslint-plugin-react-hooks';
// oxlint pending: react compiler rules stability.
import reactYouMightNotNeedAnEffect from 'eslint-plugin-react-you-might-not-need-an-effect';
import { defineConfig, globalIgnores } from 'eslint/config';
import globals from 'globals';
import path from 'path';
// Oxlint handles almost all typescript-eslint things. mostly here for parser and config base disables etc.
import { configs } from 'typescript-eslint';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const tsconfigPath = path.join(__dirname, 'tsconfig.json');

// oxlint-disable-next-line no-restricted-exports
export default defineConfig(
	globalIgnores(['dist/**', 'app/apis/wailsjs/**', '.react-router/**']),

	js.configs.recommended,

	configs.base,
	configs.eslintRecommended,
	{
		files: ['**/*.{js,jsx,mjs,ts,tsx}'],
		plugins: {
			react,
			'react-hooks': reactHooks,
			'react-you-might-not-need-an-effect': reactYouMightNotNeedAnEffect,
		},
		languageOptions: {
			parserOptions: {
				ecmaFeatures: {
					jsx: true,
				},
				projectService: true,
				tsconfigRootDir: __dirname,
			},
			globals: {
				...globals.browser,
			},
		},
		settings: {
			'import/resolver': {
				typescript: {
					alwaysTryTypes: true,
					project: tsconfigPath,
				},
			},
			react: {
				version: 'detect',
			},
		},
		rules: {
			...reactHooks.configs.flat.recommended.rules,

			'react-hooks/unsupported-syntax': 'error',
			'react-hooks/incompatible-library': 'error',

			...reactYouMightNotNeedAnEffect.configs.strict.rules,
			'react-you-might-not-need-an-effect/no-event-handler': 'off',

			// Oxlint implements.
			'no-restricted-imports': 'off',
			'no-restricted-exports': 'off',
			'no-control-regex': 'off',
			'no-unsafe-finally': 'off',
			'react-hooks/exhaustive-deps': 'off',
			'no-var': 'off', // ts transpiles let/const to var, so no need for vars any more
			'no-with': 'off', // ts(1101) & ts(2410)
			'prefer-const': 'off', // ts provides better types with const
			'prefer-rest-params': 'off', // ts provides better types with rest args over arguments
			'prefer-spread': 'off', // ts transpiles spread to apply, so no need for manual apply
		},
	},

	eslintConfigPrettier,
	...oxlint.buildFromOxlintConfigFile(path.join(__dirname, 'oxlint.config.ts'))
);
