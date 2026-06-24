import { defineConfig, type OxlintConfig } from 'oxlint';

// oxlint-disable-next-line no-restricted-imports
import baseConfig from '../.oxlintrc.json' with { type: 'json' };

const baseIgnorePatterns = baseConfig.ignorePatterns ?? [];
const baseRules = baseConfig.rules ?? {};

// oxlint-disable-next-line no-restricted-exports
export default defineConfig({
	categories: {
		correctness: 'error',
		nursery: 'off',
		pedantic: 'off',
		perf: 'off',
		restriction: 'off',
		style: 'off',
		suspicious: 'off',
	},

	extends: [baseConfig as OxlintConfig],

	ignorePatterns: [...new Set([...baseIgnorePatterns, 'dist/**', 'app/apis/wailsjs/**', '.react-router/**'])],

	jsPlugins: ['eslint-plugin-better-tailwindcss'],

	rules: {
		...baseRules,

		'better-tailwindcss/enforce-canonical-classes': 'error',
		'better-tailwindcss/enforce-consistent-class-order': 'off',
		'better-tailwindcss/enforce-consistent-important-position': 'error',
		'better-tailwindcss/enforce-consistent-line-wrapping': 'off',
		'better-tailwindcss/enforce-consistent-variable-syntax': 'error',
		'better-tailwindcss/enforce-consistent-variant-order': 'error',
		'better-tailwindcss/enforce-logical-properties': 'off',
		'better-tailwindcss/enforce-shorthand-classes': 'error',
		'better-tailwindcss/no-conflicting-classes': 'error',
		'better-tailwindcss/no-deprecated-classes': 'error',
		'better-tailwindcss/no-duplicate-classes': 'error',
		'better-tailwindcss/no-restricted-classes': 'error',
		'better-tailwindcss/no-unknown-classes': [
			'error',
			{
				ignore: ['app-*'],
			},
		],
		'better-tailwindcss/no-unnecessary-whitespace': 'error',

		'constructor-super': 'off',
		'getter-return': 'off',

		'no-class-assign': 'off',
		'no-const-assign': 'off',
		'no-dupe-class-members': 'off',
		'no-dupe-keys': 'off',
		'no-func-assign': 'off',
		'no-import-assign': 'off',
		'no-new-native-nonconstructor': 'off',
		'no-obj-calls': 'off',
		'no-redeclare': 'off',
		'no-rest-spread-properties': 'off',
		'no-restricted-exports': [
			'error',
			{
				restrictDefaultExports: {
					defaultFrom: true,
					direct: true,
					named: true,
					namedFrom: true,
					namespaceFrom: true,
				},
			},
		],
		'no-restricted-imports': [
			'error',
			{
				paths: [
					{
						importNames: ['default', 'React'],
						message: 'Avoid React import directly. Prefer explicitly importing.',
						name: 'react',
					},
					{
						importNames: ['FC', 'FunctionComponent'],
						message: 'Avoid FC imports. Prefer explicitly importing or explicitly typing your component props.',
						name: 'react',
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
		'no-setter-return': 'off',
		'no-this-before-super': 'off',
		'no-undef': 'off',
		'no-unreachable': 'off',
		'no-unsafe-negation': 'off',
		'no-unused-vars': [
			'error',
			{
				argsIgnorePattern: '^_',
				caughtErrorsIgnorePattern: '^_',
				varsIgnorePattern: '^_',
			},
		],
		'no-var': 'error',
		'no-with': 'off',
		'prefer-const': 'error',
		'prefer-rest-params': 'error',
		'prefer-spread': 'error',
		'react/exhaustive-deps': 'error',
		'react/react-compiler': 'off',
		'react/rules-of-hooks': 'error',
		'typescript/ban-ts-comment': [
			'error',
			{
				minimumDescriptionLength: 10,
			},
		],
		'typescript/consistent-type-imports': 'error',
		'typescript/no-explicit-any': 'off',
		'typescript/no-floating-promises': 'off',
		'typescript/no-misused-promises': 'off',
		'typescript/no-unnecessary-condition': 'off',
		'typescript/no-unnecessary-type-assertion': 'off',
		'typescript/no-unsafe-argument': 'off',
		'typescript/no-unsafe-assignment': 'off',
		'typescript/no-unsafe-call': 'off',
		'typescript/no-unsafe-type-assertion': 'off',
		'typescript/prefer-optional-chain': 'off',
		'typescript/prefer-promise-reject-errors': 'off',
		'typescript/require-await': 'off',
		'typescript/restrict-template-expressions': 'off',
		'typescript/restrict-plus-operands': [
			'error',
			{
				allowAny: false,
				allowBoolean: false,
				allowNullish: false,
				allowNumberAndString: false,
				allowRegExp: false,
			},
		],
		'typescript/return-await': ['error', 'error-handling-correctness-only'],
	},

	settings: {
		'eslint-plugin-better-tailwindcss': {
			detectComponentClasses: true,
			entryPoint: './app/globals.css',
		},
	},
});
