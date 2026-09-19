declare const turnstile:
  | {
      reset: () => void;
    }
  | undefined;

export const getTurnstileToken = (form: HTMLFormElement): string =>
  form.querySelector<HTMLInputElement>('input[name="cf-turnstile-response"]')?.value ?? "";

export const isTurnstileTokenMissing = (form: HTMLFormElement, token: string): boolean =>
  form.querySelector(".cf-turnstile") !== null && !token;

// The global is absent when the Cloudflare script was blocked (ad blocker,
// corporate proxy). Calling it anyway throws inside the caller's catch block
// and leaves the form disabled with no way back except a reload, so the
// absence is tolerated here.
export const resetTurnstile = (form: HTMLFormElement): void => {
  if (form.querySelector(".cf-turnstile") === null) {
    return;
  }

  if (typeof turnstile === "undefined" || typeof turnstile.reset !== "function") {
    return;
  }

  turnstile.reset();
};

export const turnstileRequestFields = (token: string): { turnstileToken?: string } =>
  token ? { turnstileToken: token } : {};
