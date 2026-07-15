import js from "@eslint/js";
import globals from "globals";
import tseslint from "typescript-eslint";
import nextPlugin from "@next/eslint-plugin-next";

export default tseslint.config(
  { ignores: [".next/"] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      globals: { ...globals.browser },
      parserOptions: {
        projectService: true,
      },
    },
    plugins: {
      "@next/next": nextPlugin,
    },
    rules: {
      ...nextPlugin.configs.recommended.rules,
      ...nextPlugin.configs["core-web-vitals"].rules,
      "@typescript-eslint/no-unused-vars": ["warn", { argsIgnorePattern: "^_" }],
      "@typescript-eslint/no-explicit-any": "warn",
    },
  },
);
