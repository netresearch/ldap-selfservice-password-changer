import plugin from "tailwindcss/plugin";

/** @type {import('tailwindcss').Config} */
const config = {
  content: ["internal/web/templates/*.html"],
  plugins: [
    plugin(({ addVariant }) => {
      addVariant("reveal-button", '&[data-revealed="true"] [data-purpose="reveal"]');
      addVariant("reveal-eye", '&[data-revealed="true"] .on-content-revealed');
      addVariant("reveal-eye-slash", '&[data-revealed="true"] .on-content-hidden');
      addVariant("nonreveal-eye", '&[data-revealed="false"] .on-content-revealed');
      addVariant("nonreveal-eye-slash", '&[data-revealed="false"] .on-content-hidden');
      addVariant("input-focus", "&:has(input:focus)");
      addVariant("hocus", ["&:hover", "&:focus"]);
    })
  ],
  theme: {
    extend: {}
  }
};

export default config;
