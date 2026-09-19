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

  try {
    // A render into a container that still holds a widget is rejected with a
    // console warning and an undefined return rather than a throw, so the
    // return value is the only reliable signal.
    widgetId = api.render(widgetSelector, { sitekey, theme });
  } catch {
    widgetId = undefined;
  }

  renderedTheme = widgetId === undefined ? undefined : theme;

  return widgetId !== undefined;
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
  if (widgetId === undefined || api === null || currentTheme() === renderedTheme) {
    return;
  }

  try {
    api.remove(widgetId);
  } catch {
    return; // Removal failed: keep the widget that is on the page.
  }

  widgetId = undefined;
  renderedTheme = undefined;

  // Without a widget the submit handlers block on a challenge that is not on
  // screen, so a failed render is retried once before the page is left in that
  // state.
  if (!renderWidget()) {
    renderWidget();
  }
};

export const getTurnstileToken = (form: HTMLFormElement): string =>
  form.querySelector<HTMLInputElement>('input[name="cf-turnstile-response"]')?.value ?? "";

export const isTurnstileTokenMissing = (form: HTMLFormElement, token: string): boolean =>
  form.querySelector(".cf-turnstile") !== null && !token;

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

  document.addEventListener("themechange", applyThemeToWidget);

  if (turnstileApi() !== null) {
    renderWidget();
    return;
  }

  window.onloadTurnstileCallback = renderWidget;
};

bootstrapTurnstile();
