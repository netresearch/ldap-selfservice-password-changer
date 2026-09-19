// Cloudflare Turnstile widget integration.
//
// The widget is rendered explicitly rather than through the `cf-turnstile`
// class, because the widget's theme is fixed at render time and this
// application has its own theme toggle that may deliberately differ from the
// operating system setting. Owning the render call is what makes it possible
// to re-render on a theme change.
//
// Every call into the Cloudflare global goes through `turnstileApi()`: the
// script is a third-party resource an ad blocker or a corporate proxy can
// remove, and a ReferenceError in a submit handler would leave the form
// disabled with no way back except a page reload.

import { THEME_CHANGE_EVENT } from "./toggles.js";

declare const turnstile: TurnstileApi | null | undefined;

declare global {
  interface Window {
    onloadTurnstileCallback?: () => void;
  }
}

interface TurnstileApi {
  render: (container: string, options: TurnstileRenderOptions) => string | undefined;
  remove: (widgetId: string) => void;
  reset: (widgetId: string) => void;
}

interface TurnstileRenderOptions {
  sitekey: string;
  theme: TurnstileTheme;
}

type TurnstileTheme = "light" | "dark";

/** Must match the id rendered by templates/molecules/turnstile.html. */
const widgetSelector = "#cf-turnstile-widget";

/** The id Cloudflare hands back for the rendered widget, if it rendered. */
let widgetId: string | undefined;

/** The theme the widget on the page was rendered with. */
let renderedTheme: TurnstileTheme | undefined;

/**
 * The Cloudflare API, or null when it is not usable. `typeof` has to come
 * first: reading an undeclared name any other way is a ReferenceError, and
 * that is the case this guard exists for. The null check follows because
 * `typeof null` is "object".
 */
const turnstileApi = (): TurnstileApi | null =>
  typeof turnstile === "undefined" || turnstile === null ? null : turnstile;

const widgetElement = (): HTMLElement | null => document.querySelector<HTMLElement>(widgetSelector);

/**
 * The theme actually applied to the page. `toggles.ts` keeps the `dark` class
 * on <html> in step with the toggle, including its "auto" state, so reading
 * the class covers the OS preference too.
 */
const currentTheme = (): TurnstileTheme => (document.documentElement.classList.contains("dark") ? "dark" : "light");

/** Renders the widget in the current theme. Reports whether a widget exists afterwards. */
const renderWidget = (): boolean => {
  if (widgetId !== undefined) {
    // A render into an occupied container is rejected, and the rejection is
    // indistinguishable from a failure — so never issue one.
    return true;
  }

  const element = widgetElement();
  const api = turnstileApi();
  if (element === null || api === null) {
    return false;
  }

  const sitekey = element.dataset["sitekey"] ?? "";
  if (sitekey === "") {
    return false;
  }

  const theme = currentTheme();

  let id: string | undefined;
  try {
    // A rejected render returns undefined rather than throwing, so the return
    // value is the only reliable signal that a widget exists.
    id = api.render(widgetSelector, { sitekey, theme });
  } catch {
    return false;
  }

  if (id === undefined) {
    return false;
  }

  widgetId = id;
  renderedTheme = theme;

  return true;
};

/**
 * Re-renders the widget when the applied theme changed. A rendered widget
 * cannot change its theme, so the old one is removed first — which discards a
 * token the visitor may already have earned. Managed and non-interactive
 * widgets re-issue one without any interaction; an interactive widget asks
 * again. That cost is why an event that did not change the applied theme, such
 * as switching between "auto" and the theme the system already uses, returns
 * here without touching the widget.
 */
const applyThemeToWidget = (): void => {
  const api = turnstileApi();
  if (api === null) {
    return;
  }

  if (widgetId === undefined) {
    // An earlier render failed and the page carries the marker without a
    // widget, which blocks every submit. Rebuild it here rather than skipping
    // the event; a refused submit does the same through
    // ensureTurnstileWidget, for a visitor who never touches the toggle.
    renderWidget();

    return;
  }

  if (currentTheme() === renderedTheme) {
    return;
  }

  try {
    api.remove(widgetId);
  } catch {
    return; // Removal failed: keep the widget that is on the page.
  }

  widgetId = undefined;
  renderedTheme = undefined;

  renderWidget();
};

export const getTurnstileToken = (form: HTMLFormElement): string =>
  form.querySelector<HTMLInputElement>('input[name="cf-turnstile-response"]')?.value ?? "";

export const isTurnstileTokenMissing = (form: HTMLFormElement, token: string): boolean =>
  form.querySelector(".cf-turnstile") !== null && !token;

/**
 * Renders the widget if an earlier attempt failed and the page is left with
 * the marker but no challenge, which blocks every submit. Called from the
 * refusal branch, it turns that state into one refused submit followed by a
 * challenge the visitor can answer — otherwise only a theme change rebuilds
 * it, and a visitor need never trigger one. A no-op while a widget is present.
 */
export const ensureTurnstileWidget = (): void => {
  renderWidget();
};

/**
 * Called from the submit handlers' catch block, ahead of re-enabling the
 * fields. Anything thrown here would escape that catch and leave the form
 * disabled until a reload, so nothing is allowed to propagate. A widget that
 * cannot be reset still holds its previous token, and the server rejects a
 * replayed one.
 */
export const resetTurnstile = (): void => {
  const api = turnstileApi();
  if (widgetId === undefined || api === null) {
    return;
  }

  try {
    api.reset(widgetId);
  } catch {
    // Keep the caller's recovery path running.
  }
};

export const turnstileRequestFields = (token: string): { turnstileToken?: string } =>
  token ? { turnstileToken: token } : {};

/**
 * Renders the widget once the Cloudflare script is available. This module is
 * an ES module and therefore deferred, so the container is parsed by the time
 * this runs, and either load order is covered.
 *
 * The two branches are deliberately asymmetric. `turnstile.ready()` is not
 * used because the script tag carries `async defer`, which that function
 * explicitly rejects. And the onload callback is registered only when the
 * global is absent: Cloudflare looks the callback name up again a second after
 * loading, so registering it on the synchronous path as well would produce a
 * second render into a container that already holds a widget — rejected, and
 * it would clear the widget id.
 */
const bootstrapTurnstile = (): void => {
  if (widgetElement() === null) {
    return; // Turnstile is not configured for this deployment.
  }

  document.addEventListener(THEME_CHANGE_EVENT, applyThemeToWidget);

  if (turnstileApi() !== null) {
    renderWidget();
    return;
  }

  window.onloadTurnstileCallback = renderWidget;
};

bootstrapTurnstile();
