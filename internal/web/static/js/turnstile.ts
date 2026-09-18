declare const turnstile: {
  reset: () => void;
};

export const getTurnstileToken = (form: HTMLFormElement): string =>
  form.querySelector<HTMLInputElement>('input[name="cf-turnstile-response"]')?.value ?? "";

export const isTurnstileTokenMissing = (form: HTMLFormElement, token: string): boolean =>
  form.querySelector(".cf-turnstile") !== null && !token;

export const resetTurnstile = (form: HTMLFormElement): void => {
  if (form.querySelector(".cf-turnstile")) {
    turnstile.reset();
  }
};

export const turnstileRequestFields = (token: string): { turnstileToken?: string } =>
  token ? { turnstileToken: token } : {};
