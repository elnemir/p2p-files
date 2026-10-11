import {defineConfig, globalIgnores} from 'eslint/config';
import js from '@eslint/js';
import ts from 'typescript-eslint';
import hooks from 'eslint-plugin-react-hooks';
import a11y from 'eslint-plugin-jsx-a11y';

export default defineConfig([
  {files:['apps/web/**/*.{ts,tsx}'], extends:[js.configs.recommended, ts.configs.recommended, hooks.configs.flat.recommended, a11y.flatConfigs.recommended]},
  globalIgnores(['.cache/**', '**/.next/**', '**/next-env.d.ts', 'node_modules/**']),
]);
