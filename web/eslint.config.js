import js from '@eslint/js'
import globals from 'globals'
import tseslint from 'typescript-eslint'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import boundaries from 'eslint-plugin-boundaries'

export default tseslint.config(
  { ignores: ['dist', 'node_modules', 'coverage'] },
  {
    files: ['*.config.ts', '*.config.js'],
    languageOptions: {
      parser: tseslint.parser,
    },
  },
  {
    extends: [js.configs.recommended, ...tseslint.configs.strict],
    files: ['src/**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2023,
      globals: globals.browser,
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
      boundaries,
    },
    settings: {
      'boundaries/elements': [
        { type: 'app', pattern: ['src/app/**'] },
        { type: 'features', pattern: ['src/features/**'], capture: ['feature'] },
        { type: 'shared', pattern: ['src/shared/**'] },
        { type: 'styles', pattern: ['src/styles/**'] },
      ],
      'boundaries/ignore': ['**/*.test.*', '**/*.spec.*'],
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],

      // Architecture: features cannot import from other features
      'boundaries/element-types': [
        'error',
        {
          default: 'disallow',
          rules: [
            { from: 'app', allow: ['app', 'features', 'shared', 'styles'] },
            { from: 'features', allow: ['shared'] },
            { from: 'shared', allow: ['shared'] },
            { from: 'styles', allow: ['styles'] },
          ],
        },
      ],

      // No direct fetch/axios in component files
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['axios', 'node-fetch'],
              message: 'Use the API client from @/shared/lib/api-client instead.',
            },
          ],
        },
      ],

      // Naming: flag non-PascalCase component files via convention (documented, not auto-fixable)
      '@typescript-eslint/naming-convention': [
        'warn',
        {
          selector: 'function',
          format: ['camelCase', 'PascalCase'],
        },
      ],

      // Prefer named exports
      'no-restricted-syntax': [
        'warn',
        {
          selector: 'ExportDefaultDeclaration',
          message: 'Prefer named exports for components and hooks.',
        },
      ],
    },
  },
)
