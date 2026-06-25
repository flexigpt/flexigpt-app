import { defineConfig, type DummyRuleMap, type OxlintConfig } from 'oxlint';

// oxlint-disable-next-line no-restricted-imports import/no-relative-parent-imports
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

const eslintRules: DummyRuleMap = {
	// Correctness. Default On.
	'constructor-super': 'off',
	'getter-return': 'off',

	// Restriction. Default On.
	'no-void': 'off',
	complexity: 'off',
	'no-undefined': 'off',
	'no-console': 'off',
	'no-empty-function': 'off',
	'no-use-before-define': 'off',
	'no-plusplus': 'off',
	'no-div-regex': 'off',
	'default-case': 'off',
	'class-methods-use-this': 'off',
	'no-bitwise': 'off',

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

	// Nursery. Default Off.

	// Style. Default Off.
	curly: 'error',
	'default-param-last': 'error',
	'guard-for-in': 'error',
	'no-duplicate-imports': ['error', { allowSeparateTypeImports: true }],
	'no-multi-assign': 'error',
	'no-new-func': 'error',
	'no-return-assign': 'error',
	'no-script-url': 'error',
	'no-template-curly-in-string': 'error',
	'no-useless-computed-key': 'error',
	'prefer-const': 'error',
	'prefer-promise-reject-errors': 'error',
	'prefer-regex-literals': 'error',
	'prefer-rest-params': 'error',
	'prefer-spread': 'error',
	'prefer-numeric-literals': 'error',
	yoda: 'error',

	'prefer-object-has-own': 'off',
	'no-implicit-coercion': 'off',

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

	// Restriction. Default On.
	'typescript/promise-function-async': 'off',
	'typescript/explicit-function-return-type': 'off',
	'typescript/explicit-module-boundary-types': 'off',
	'typescript/non-nullable-type-assertion-style': 'off',
	'typescript/no-explicit-any': 'off',
	'typescript/explicit-member-accessibility': 'off',
	'typescript/no-import-type-side-effects': 'off',

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
	// Restriction. Default on.
	'import/no-default-export': 'off',
	'import/unambiguous': 'off',

	// Pedantic. Default on.
	'import/max-dependencies': 'off',
};

const reactRules: DummyRuleMap = {
	// Restriction. Default on.
	'react/no-multi-comp': 'off',
	'react/jsx-filename-extension': 'off',
	'react/jsx-no-literals': 'off',
	'react/forbid-component-props': 'off',
	'react/button-has-type': 'off',
	'react/only-export-components': 'off',

	// Suspicious. Default on.
	'react/react-in-jsx-scope': 'off',
};

const promiseRules: DummyRuleMap = {
	// Restriction. Default on.
	'promise/catch-or-return': 'off',

	// Suspicious. Default on.
	'promise/always-return': 'off',
};

const oxcRules: DummyRuleMap = {
	// Restriction. Default on.
	'oxc/no-rest-spread-properties': 'off',
	'oxc/no-optional-chaining': 'off',
	'oxc/no-async-await': 'off',
};

const unicornRules: DummyRuleMap = {
	// Restriction. Default on.
	'unicorn/prefer-node-protocol': 'off',
	'unicorn/no-array-for-each': 'off',
	'unicorn/no-array-reduce': 'off',

	// Suspicious. Default On.
	'unicorn/prefer-add-event-listener': 'off',

	// Pedantic. Default On.
	'unicorn/no-negated-condition': 'off',
	'unicorn/no-useless-undefined': 'off',

	// Style. Default off.
	'unicorn/filename-case': [
		'error',
		{
			case: 'snakeCase',
			ignore: ['react-router.config.ts'],
		},
	],
};

const nodeRules: DummyRuleMap = {
	// Restriction. Default on.
	'node/no-process-env': 'off',
};

// oxlint-disable-next-line no-restricted-exports
export default defineConfig({
	categories: {
		correctness: 'error',
		suspicious: 'error',
		restriction: 'error',
		pedantic: 'error',
		perf: 'off',
		style: 'off',
		nursery: 'off',
	},

	extends: [baseConfig as OxlintConfig],

	ignorePatterns: [...new Set([...baseIgnorePatterns, 'dist/**', 'app/apis/wailsjs/**', '.react-router/**'])],

	jsPlugins: ['eslint-plugin-better-tailwindcss'],

	rules: {
		...baseRules,
		...nodeRules,
		...promiseRules,
		...eslintRules,
		...importRules,
		...tsRules,
		...unicornRules,
		...reactRules,
		...oxcRules,
		...betterTailwindCSSRules,
	},

	settings: {
		'eslint-plugin-better-tailwindcss': {
			detectComponentClasses: true,
			entryPoint: './app/globals.css',
		},
	},
});
