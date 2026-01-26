import eslint from "@eslint/js";
import tseslint from "typescript-eslint";
import astro from "eslint-plugin-astro";

export default [
  eslint.configs.recommended,
  ...tseslint.configs.recommended,
  ...astro.configs.recommended,
  {
    ignores: [
      "dist/",
      "node_modules/",
      ".astro/",
      "src/components/posthog.astro",
    ],
  },
  {
    files: ["**/*.{js,ts,astro}"],
    rules: {
      // Relax some rules for landing page
      "@typescript-eslint/no-unused-vars": [
        "warn",
        { argsIgnorePattern: "^_" },
      ],
      "no-console": "warn",
    },
  },
];
