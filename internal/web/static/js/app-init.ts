/**
 * App initialization wrapper — reads configuration from data attributes
 * for CSP compliance (no inline scripts).
 *
 * NOTE: `document.currentScript` is ALWAYS null inside ES modules (per spec),
 * so we look the script up explicitly by src.
 */

import { init } from "./app.js";

const currentScript = document.querySelector<HTMLScriptElement>('script[src*="app-init.js"]');

const config = {
  minLength: Number(currentScript?.dataset["minLength"] ?? "8"),
  minNumbers: Number(currentScript?.dataset["minNumbers"] ?? "0"),
  minSymbols: Number(currentScript?.dataset["minSymbols"] ?? "0"),
  minUppercase: Number(currentScript?.dataset["minUppercase"] ?? "0"),
  minLowercase: Number(currentScript?.dataset["minLowercase"] ?? "0"),
  passwordCanIncludeUsername: currentScript?.dataset["passwordCanIncludeUsername"] === "true"
};

init(config);
