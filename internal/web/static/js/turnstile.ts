declare const turnstile:
  | {
      reset: () => void;
    }
  | null
  | undefined;

export const getTurnstileToken = (form: HTMLFormElement): string =>
  form.querySelector<HTMLInputElement>('input[name="cf-turnstile-response"]')?.value ?? "";

export const isTurnstileTokenMissing = (form: HTMLFormElement, token: string): boolean =>
  form.querySelector(".cf-turnstile") !== null && !token;

// Called from the submit handlers' catch block, ahead of re-enabling the
// fields. Anything thrown here would escape that catch and leave the form
// disabled with no way back except a reload, so neither a missing global (the
// Cloudflare script can be blocked by an ad blocker or a corporate proxy) nor
// a throwing reset is allowed to propagate. A widget that cannot be reset
// still holds its previous token, and the server rejects a replayed one.
export const resetTurnstile = (form: HTMLFormElement): void => {
  if (form.querySelector(".cf-turnstile") === null) {
    return;
  }

  // typeof first: the global is undeclared when the script never loaded, and
  // reading an undeclared name any other way is itself a ReferenceError. The
  // null check follows because typeof null is "object", so the property access
  // below would throw ahead of the try.
  if (typeof turnstile === "undefined" || turnstile === null || typeof turnstile.reset !== "function") {
    return;
  }

  try {
    turnstile.reset();
  } catch {
    // Keep the caller's recovery path running.
  }
};

export const turnstileRequestFields = (token: string): { turnstileToken?: string } =>
  token ? { turnstileToken: token } : {};
