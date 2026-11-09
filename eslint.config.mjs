import js from "@eslint/js";
import nextPlugin from "@next/eslint-plugin-next";
import tseslint from "typescript-eslint";

export default tseslint.config(
  {
    ignores: ["**/node_modules/**", "**/.next/**", "**/dist/**", "**/*.d.ts", "**/next-env.d.ts"]
  },
  js.configs.recommended,
  ...tseslint.configs.strictTypeChecked,
  {
    languageOptions: {
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname
      }
    },
    plugins: { "@next/next": nextPlugin },
    rules: {
      ...nextPlugin.configs.recommended.rules,
      "max-lines-per-function": ["error", { max: 40, skipBlankLines: true, skipComments: true }],
      "max-statements": ["error", 12],
      "max-depth": ["error", 2],
      "no-else-return": "error",
      "no-continue": "error",
      "no-labels": "error",
      "no-loop-func": "error",
      "complexity": ["error", 6],
      "arrow-body-style": ["error", "as-needed"],
      "prefer-const": "error",
      "func-style": ["error", "declaration"],
      "no-restricted-syntax": [
        "error",
        { selector: "WhileStatement", message: "Циклы while запрещены стилем проекта." },
        { selector: "ForStatement > ContinueStatement", message: "continue запрещён стилем проекта." },
        { selector: "ForStatement > BreakStatement", message: "break запрещён стилем проекта." },
        { selector: "IfStatement > ReturnStatement", message: "Ранний return запрещён: один выход из функции." }
      ],
      "@typescript-eslint/explicit-function-return-type": "error",
      "@typescript-eslint/no-unnecessary-condition": "off"
    }
  }
);
