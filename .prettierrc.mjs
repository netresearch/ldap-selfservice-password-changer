/** @type {import("prettier").Config} */
const config = {
  printWidth: 120,
  trailingComma: "none",
  tabWidth: 2,
  semi: true,
  singleQuote: false,
  tailwindStylesheet: "./internal/web/tailwind.css",
  plugins: ["prettier-plugin-tailwindcss", "prettier-plugin-go-template"]
};

export default config;
