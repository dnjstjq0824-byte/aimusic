// ESLint flat config (P1): Nuxt-recommended rules via @nuxt/eslint.
// No formatting rules here (no formatter adopted); correctness only.
import withNuxt from './.nuxt/eslint.config.mjs';

export default withNuxt(
  {
    // import/first false-positives on the repo-wide dual
    // <script>+<script setup> SFC pattern; keep it for plain .ts only.
    files: ['app/**/*.vue'],
    rules: { 'import/first': 'off' },
  },
  {
    // Explicit `any` is widespread in the API layer; warn (don't fail)
    // so new code thinks twice without blocking on legacy uses.
    rules: { '@typescript-eslint/no-explicit-any': 'warn' },
  },
);
