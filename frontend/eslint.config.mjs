import js from '@eslint/js';
import eslintPluginPrettierRecommended from 'eslint-plugin-prettier/recommended';
import react from 'eslint-plugin-react';
import reactHooks from 'eslint-plugin-react-hooks';
import reactYouMightNotNeedAnEffect from 'eslint-plugin-react-you-might-not-need-an-effect';
import tailwindCanonicalClasses from 'eslint-plugin-tailwind-canonical-classes';
import { defineConfig, globalIgnores } from 'eslint/config';
import globals from 'globals';
import path from 'path';
import { configs } from 'typescript-eslint';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// eslint-disable-next-line no-restricted-exports
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
					project: './tsconfig.json',
				},
			},
			react: {
				version: 'detect',
			},
		},
		rules: {
			'no-restricted-imports': [
				'error',
				{
					paths: [
						{
							name: 'react',
							importNames: ['default', 'React'],
							message: 'Avoid React import directly. Prefer explicitly importing.',
						},
						{
							name: 'react',
							importNames: ['FC', 'FunctionComponent'],
							message: 'Avoid FC imports. Prefer explicitly importing or explicitly typing your component props.',
						},
					],
					patterns: [
						{
							group: ['./*', '../*'],
							message: 'Relative imports are not allowed.',
						},
					],
				},
			],
			'no-restricted-exports': [
				'error',
				{
					restrictDefaultExports: {
						direct: true,
						named: true,
						defaultFrom: true,
						namedFrom: true,
						namespaceFrom: true,
					},
				},
			],

			...reactHooks.configs.flat.recommended.rules,
			'react-hooks/exhaustive-deps': 'error',
			'react-hooks/unsupported-syntax': 'error',
			'react-hooks/incompatible-library': 'error',

			...reactYouMightNotNeedAnEffect.configs.strict.rules,

			'@typescript-eslint/consistent-type-imports': 'error',
			'@typescript-eslint/no-explicit-any': 'off',
			'@typescript-eslint/no-unsafe-assignment': 'off',
			'@typescript-eslint/no-unsafe-call': 'off',
			'@typescript-eslint/no-unsafe-argument': 'off',
			'@typescript-eslint/restrict-template-expressions': 'off',
			'@typescript-eslint/prefer-promise-reject-errors': 'off',
			'@typescript-eslint/require-await': 'off',
			'@typescript-eslint/no-floating-promises': 'off',
			'@typescript-eslint/no-misused-promises': 'off',
			'@typescript-eslint/no-unnecessary-condition': 'off',
			'@typescript-eslint/no-unused-vars': [
				'error',
				{
					argsIgnorePattern: '^_',
					varsIgnorePattern: '^_',
					caughtErrorsIgnorePattern: '^_',
				},
			],

			'tailwind-canonical-classes/tailwind-canonical-classes': [
				'error',
				{
					cssPath: './app/globals.css',
				},
			],
		},
	},

	eslintPluginPrettierRecommended,
	{
		rules: {
			'prettier/prettier': [
				'error',
				{
					endOfLine: 'auto',
				},
				{
					usePrettierrc: true,
					fileInfoOptions: {
						ignorePath: path.resolve(__dirname, '../.prettierignore'),
					},
				},
			],
		},
	}
);
