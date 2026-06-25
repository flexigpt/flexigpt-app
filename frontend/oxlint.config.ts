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
	'no-unused-vars': [
		'error',
		{
			argsIgnorePattern: '^_',
			caughtErrorsIgnorePattern: '^_',
			varsIgnorePattern: '^_',
		},
	],

	// Restriction. Default On.
	complexity: 'off',
	'class-methods-use-this': 'off',
	'default-case': 'off',
	'no-bitwise': 'off',
	'no-console': 'off',
	'no-div-regex': 'off',
	'no-empty-function': 'off',
	'no-plusplus': 'off',

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

	'no-undefined': 'off',
	'no-use-before-define': 'off',
	'no-void': 'off',

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

	// Style. Default Off.
	curly: 'error',
	'default-case-last': 'error',
	'default-param-last': 'error',
	'func-name-matching': 'error',
	'func-names': 'error',
	'grouped-accessor-pairs': 'error',
	'guard-for-in': 'error',
	'no-duplicate-imports': ['error', { allowSeparateTypeImports: true }],
	'no-extra-label': 'error',
	'no-label-var': 'error',
	'no-labels': 'error',
	'no-lone-blocks': 'error',
	'no-multi-assign': 'error',
	'no-new-func': 'error',
	'no-return-assign': 'error',
	'no-script-url': 'error',
	'no-template-curly-in-string': 'error',
	'no-useless-computed-key': 'error',
	'prefer-const': 'error',
	'prefer-numeric-literals': 'error',
	'prefer-object-has-own': 'error',
	'prefer-object-spread': 'error',
	'prefer-promise-reject-errors': 'error',
	'prefer-regex-literals': 'error',
	'prefer-rest-params': 'error',
	'prefer-spread': 'error',
	'vars-on-top': 'error',
	yoda: 'error',

	'arrow-body-style': 'off',
	'func-style': 'off',
	'capitalized-comments': 'off',
	'id-length': 'off',
	'id-match': 'off',
	'init-declarations': 'off',
	'logical-assignment-operators': 'off',
	'max-params': 'off',
	'max-statements': 'off',
	'new-cap': 'off',
	'no-continue': 'off',
	'no-implicit-coercion': 'off',
	'no-magic-numbers': 'off',
	'no-multi-str': 'off',
	'no-nested-ternary': 'off',
	'no-ternary': 'off',
	'object-shorthand': 'off',
	'operator-assignment': 'off',
	'prefer-arrow-callback': 'off',
	'prefer-destructuring': 'off',
	'prefer-exponentiation-operator': 'off',
	'prefer-named-capture-group': 'off',
	'prefer-template': 'off',
	'sort-imports': 'off',
	'sort-keys': 'off',

	// Nursery. Default Off.
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
	'typescript/ban-ts-comment': [
		'error',
		{
			minimumDescriptionLength: 10,
		},
	],
	'typescript/switch-exhaustiveness-check': 'off',
	'typescript/no-misused-promises': 'off',
	'typescript/no-unsafe-argument': 'off',
	'typescript/no-unsafe-assignment': 'off',
	'typescript/no-unsafe-call': 'off',
	'typescript/prefer-readonly-parameter-types': 'off',
	'typescript/prefer-nullish-coalescing': 'off',
	'typescript/prefer-promise-reject-errors': 'off',
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
	'typescript/require-await': 'off',
	'typescript/strict-void-return': 'off',
	'typescript/strict-boolean-expressions': 'off',

	// Style. Default Off.
	'typescript/adjacent-overload-signatures': 'error',
	'typescript/ban-tslint-comment': 'error',
	'typescript/class-literal-property-style': 'error',
	'typescript/consistent-generic-constructors': 'error',
	'typescript/consistent-indexed-object-style': 'error',
	'typescript/consistent-type-assertions': 'error',
	'typescript/consistent-type-definitions': 'error',
	'typescript/consistent-type-exports': 'error',
	'typescript/consistent-type-imports': 'error',
	'typescript/no-empty-interface': 'error',
	'typescript/no-inferrable-types': 'error',
	'typescript/no-unnecessary-qualifier': 'error',
	'typescript/parameter-properties': 'error',
	'typescript/prefer-find': 'error',
	'typescript/prefer-function-type': 'error',
	'typescript/prefer-reduce-type-parameter': 'error',
	'typescript/prefer-return-this-type': 'error',
	'typescript/prefer-string-starts-ends-with': 'error',
	'typescript/unified-signatures': 'error',

	'typescript/array-type': 'off',
	'typescript/dot-notation': 'off',
	'typescript/method-signature-style': 'off',
	'typescript/prefer-for-of': 'off',
	'typescript/prefer-readonly': 'off',
	'typescript/prefer-regexp-exec': 'off',

	// Nursery. Default Off.
};

const importRules: DummyRuleMap = {
	// Restriction. Default on.
	'import/no-default-export': 'off',
	'import/unambiguous': 'off',

	// Pedantic. Default on.
	'import/max-dependencies': 'off',

	// Style. Default Off.
	'import/no-duplicates': 'error',
	'import/first': 'error',
	'import/newline-after-import': 'error',
	'import/no-anonymous-default-export': 'error',
	'import/no-mutable-exports': 'error',
	'import/no-namespace': 'error',
	'import/no-nodejs-modules': ['error', { allow: ['path', 'node:path'] }],

	'import/consistent-type-specifier-style': 'off',
	'import/exports-last': 'off',
	'import/group-exports': 'off',
	'import/no-named-default': 'off',
	'import/no-named-export': 'off',
	'import/prefer-default-export': 'off',
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
