import eslint from '@eslint/js'
import globals from 'globals'
import tseslint from 'typescript-eslint'
import pluginVue from 'eslint-plugin-vue'

export default tseslint.config(
  // Global ignores
  {
    ignores: [
      'dist/',
      'node_modules/',
      'src/api/generated.d.ts',
    ],
  },

  // Base JS recommended rules
  eslint.configs.recommended,

  // Browser globals
  {
    languageOptions: {
      globals: globals.browser,
    },
  },

  // TypeScript recommended rules
  ...tseslint.configs.recommended,

  // Vue 3 essential rules (error prevention, no formatting opinions)
  ...pluginVue.configs['flat/essential'],

  // Enable TypeScript parser for .vue files
  {
    files: ['**/*.vue'],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
      },
    },
  },

  // Project-specific rule overrides
  {
    rules: {
      // Align with tsconfig (noUnusedLocals: false, noUnusedParameters: false)
      '@typescript-eslint/no-unused-vars': ['warn', {
        argsIgnorePattern: '^_',
        varsIgnorePattern: '^_',
      }],

      // Allow empty functions (e.g. default callbacks, noop)
      '@typescript-eslint/no-empty-function': 'off',

      // Vue specific adjustments
      'vue/multi-word-component-names': 'off',
      'vue/no-v-html': 'off',
    },
  },
)
