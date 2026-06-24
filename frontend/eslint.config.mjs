import js from '@eslint/js';
import eslintConfigPrettier from 'eslint-config-prettier';
import oxlint from 'eslint-plugin-oxlint';
import react from 'eslint-plugin-react';
import reactHooks from 'eslint-plugin-react-hooks';
import reactYouMightNotNeedAnEffect from 'eslint-plugin-react-you-might-not-need-an-effect';
import tailwindCanonicalClasses from 'eslint-plugin-tailwind-canonical-classes';
import eslintPluginTailwindcss from 'eslint-plugin-tailwindcss';
import { defineConfig, globalIgnores } from 'eslint/config';
import globals from 'globals';
import path from 'path';
import { configs } from 'typescript-eslint';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const tsconfigPath = path.join(__dirname, 'tsconfig.json');
const tailwindCssPath = path.join(__dirname, 'app/globals.css');

// oxlint-disable-next-line no-restricted-exports
export default defineConfig(
	globalIgnores(['dist/**', 'app/apis/wailsjs/**', '.react-router/**']),

	js.configs.recommended,
	configs.strictTypeChecked,

	{
		files: ['**/*.{js,jsx,mjs,ts,tsx}'],
		plugins: {
			react,
			'react-hooks': reactHooks,
			'react-you-might-not-need-an-effect': reactYouMightNotNeedAnEffect,
			'tailwind-canonical-classes': tailwindCanonicalClasses,
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
			'@typescript-eslint/consistent-type-imports': 'error',
			'@typescript-eslint/no-explicit-any': 'off',
			'@typescript-eslint/no-unsafe-assignment': 'off',
			'@typescript-eslint/no-unnecessary-type-assertion': 'off',
			'@typescript-eslint/no-unsafe-call': 'off',
			'@typescript-eslint/no-unsafe-argument': 'off',
			'@typescript-eslint/restrict-template-expressions': 'off',
			'@typescript-eslint/prefer-promise-reject-errors': 'off',
			'@typescript-eslint/require-await': 'off',
			'@typescript-eslint/no-floating-promises': 'off',
			'@typescript-eslint/no-misused-promises': 'off',
			'@typescript-eslint/no-unnecessary-condition': 'off',

			'tailwind-canonical-classes/tailwind-canonical-classes': [
				'error',
				{
					cssPath: tailwindCssPath,
				},
			],
			// Oxlint implements.
			'no-restricted-imports': 'off',
			'no-restricted-exports': 'off',
			'no-control-regex': 'off',
			'no-unsafe-finally': 'off',
			'react-hooks/exhaustive-deps': 'off',
			'@typescript-eslint/no-dynamic-delete': 'off',
			'@typescript-eslint/no-unused-vars': 'off',
		},
	},

	eslintPluginTailwindcss.configs.recommended,
	{
		settings: {
			tailwindcss:
				/** @type {import('eslint-plugin-tailwindcss').PluginSettings} */
				({
					cssConfigPath: tailwindCssPath,
				}),
		},
		rules: {
			'tailwindcss/classnames-order': 'off',
			'tailwindcss/no-custom-classname': 'off',
		},
	},
	eslintConfigPrettier,
	...oxlint.buildFromOxlintConfigFile(path.join(__dirname, '.oxlintrc.json'))
);
