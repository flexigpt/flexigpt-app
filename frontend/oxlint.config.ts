import { defineConfig, type DummyRuleMap, type OxlintConfig } from 'oxlint';

// oxlint-disable-next-line no-restricted-imports
import baseConfig from '../.oxlintrc.json' with { type: 'json' };

const baseIgnorePatterns = baseConfig.ignorePatterns ?? [];
const baseRules = baseConfig.rules ?? {};

const betterTailwindCSSRules: DummyRuleMap = {
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
};

const unicornRules: DummyRuleMap = {
	// Suspicious. Default On.
	'unicorn/prefer-add-event-listener': 'off',

	// Pedantic. Default On.
	'unicorn/no-negated-condition': 'off',
	'unicorn/no-useless-undefined': 'off',
};

const eslintRules: DummyRuleMap = {
	// Correctness. Default On.
	'constructor-super': 'off',
	'getter-return': 'off',

	// Suspicious. Default on.
	'no-underscore-dangle': 'off',

	// Pedantic. Default On.
	'max-lines-per-function': 'off',
	'max-lines': 'off',
	'no-inline-comments': 'off',
	'no-negated-condition': 'off',
	'require-unicode-regexp': 'off',
	'require-await': 'off',
	'no-promise-executor-return': 'off',
	'max-depth': 'off',

	// Restriction. Default Off.
	'no-empty': 'error',
	'no-regex-spaces': 'error',
	'no-var': 'error',

	// Nursery. Default Off.

	// Style. Default Off.
	'prefer-const': 'error',
	'prefer-rest-params': 'error',
	'prefer-spread': 'error',

	// Configs.
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
	'no-unused-vars': [
		'error',
		{
			argsIgnorePattern: '^_',
			caughtErrorsIgnorePattern: '^_',
			varsIgnorePattern: '^_',
		},
	],
};

const tsRules: DummyRuleMap = {
	// Correctness. Default On.
	'typescript/no-floating-promises': 'off',
	'typescript/restrict-template-expressions': 'off',

	// Suspicious. Default on.
	'typescript/no-unsafe-type-assertion': 'off',
	'typescript/consistent-return': 'off',

	// Pedantic. Default On.
	'typescript/strict-void-return': 'off',
	'typescript/strict-boolean-expressions': 'off',
	'typescript/prefer-readonly-parameter-types': 'off',
	'typescript/prefer-nullish-coalescing': 'off',
	'typescript/prefer-promise-reject-errors': 'off',
	'typescript/switch-exhaustiveness-check': 'off',
	'typescript/no-misused-promises': 'off',
	'typescript/no-unsafe-argument': 'off',
	'typescript/no-unsafe-assignment': 'off',
	'typescript/no-unsafe-call': 'off',
	'typescript/require-await': 'off',

	// Restriction. Default Off.

	// Nursery. Default Off.

	// Style. Default Off.
	'typescript/consistent-type-imports': 'error',

	// Configs.
	'typescript/ban-ts-comment': [
		'error',
		{
			minimumDescriptionLength: 10,
		},
	],

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
};

const importRules: DummyRuleMap = {
	// Pedantic. Default on.
	'import/max-dependencies': 'off',
};

const reactRules: DummyRuleMap = {
	// Suspicious. Default on.
	'react/react-in-jsx-scope': 'off',
};

const promiseRules: DummyRuleMap = {
	// Suspicious. Default on.
	'promise/always-return': 'off',
};

// oxlint-disable-next-line no-restricted-exports
export default defineConfig({
	categories: {
		correctness: 'error',
		pedantic: 'error',
		perf: 'off',
		restriction: 'off',
		style: 'off',
		suspicious: 'error',
		nursery: 'off',
	},

	extends: [baseConfig as OxlintConfig],

	ignorePatterns: [...new Set([...baseIgnorePatterns, 'dist/**', 'app/apis/wailsjs/**', '.react-router/**'])],

	jsPlugins: ['eslint-plugin-better-tailwindcss'],

	rules: {
		...baseRules,
		...promiseRules,
		...eslintRules,
		...importRules,
		...tsRules,
		...unicornRules,
		...reactRules,
		...betterTailwindCSSRules,
	},

	settings: {
		'eslint-plugin-better-tailwindcss': {
			detectComponentClasses: true,
			entryPoint: './app/globals.css',
		},
	},
});
