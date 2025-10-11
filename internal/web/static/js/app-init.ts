/**
 * App initialization wrapper - reads configuration from data attributes
 * for CSP compliance (no inline scripts)
 */

import { init } from "./app.js";

// Get the current script element to read data attributes
// Note: document.currentScript can be null in ES6 modules, so we provide fallback
const currentScript = document.currentScript as HTMLScriptElement | null;

// Read configuration from data attributes with defaults
const config = {
  minLength: Number(currentScript?.dataset["minLength"] ?? "8"),
  minNumbers: Number(currentScript?.dataset["minNumbers"] ?? "0"),
  minSymbols: Number(currentScript?.dataset["minSymbols"] ?? "0"),
  minUppercase: Number(currentScript?.dataset["minUppercase"] ?? "0"),
  minLowercase: Number(currentScript?.dataset["minLowercase"] ?? "0"),
  passwordCanIncludeUsername: currentScript?.dataset["passwordCanIncludeUsername"] === "true"
};

// Initialize the app with configuration (always call, even with defaults)
init(config);
