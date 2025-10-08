const specialCharacters = (() => {
  // Generate an array of special characters according to the ASCII table:
  // https://en.wikipedia.org/wiki/ASCII
  const specialCharacters = [];

  for (let i = "!".charCodeAt(0); i <= "/".charCodeAt(0); i++) {
    specialCharacters.push(String.fromCharCode(i));
  }

  for (let i = ":".charCodeAt(0); i <= "@".charCodeAt(0); i++) {
    specialCharacters.push(String.fromCharCode(i));
  }

  for (let i = "[".charCodeAt(0); i <= "`".charCodeAt(0); i++) {
    specialCharacters.push(String.fromCharCode(i));
  }

  for (let i = "{".charCodeAt(0); i <= "~".charCodeAt(0); i++) {
    specialCharacters.push(String.fromCharCode(i));
  }

  return specialCharacters;
})();

const pluralize = (singular: string, amount: number) => (amount === 1 ? singular : singular + "s");

const form = document.querySelector<HTMLFormElement>("#form");
if (!form) throw new Error("Could not find form element");

const submitButton = form.querySelector<HTMLButtonElement>("button[type='submit']");
if (!submitButton) throw new Error("Could not find submit button element");

export const mustNotBeEmpty = (fieldName: string) => (v: string) =>
  v.length === 0 ? `${fieldName} must not be empty` : "";
export const mustBeLongerThan = (minLength: number, fieldName: string) => (v: string) =>
  v.length < minLength ? `${fieldName} must be at least ${minLength} ${pluralize("character", minLength)} long` : "";
export const mustIncludeNumbers = (amount: number, fieldName: string) => (v: string) =>
  v.split("").filter((c) => !isNaN(+c)).length < amount
    ? `${fieldName} must include at least ${amount} ${pluralize("number", amount)}`
    : "";
export const mustIncludeSymbols = (amount: number, fieldName: string) => (v: string) =>
  v.split("").filter((c) => specialCharacters.includes(c)).length < amount
    ? `${fieldName} must include at least ${amount} special ${pluralize("character", amount)} (such as !, @, #, $, %)`
    : "";
export const mustIncludeUppercase = (amount: number, fieldName: string) => (v: string) =>
  v.split("").filter((c) => c === c.toUpperCase() && c !== c.toLowerCase()).length < amount
    ? `${fieldName} must include at least ${amount} uppercase ${pluralize("character", amount)}`
    : "";
export const mustIncludeLowercase = (amount: number, fieldName: string) => (v: string) =>
  v.split("").filter((c) => c === c.toLowerCase() && c !== c.toUpperCase()).length < amount
    ? `${fieldName} must include at least ${amount} lowercase ${pluralize("character", amount)}`
    : "";

export const mustMatchNewPassword = (fieldName: string) => (v: string) => {
  const passwordInput = form.querySelector<HTMLInputElement>(`#new input`);
  if (!passwordInput) throw new Error("Could not find password input element");

  return passwordInput.value !== v ? `${fieldName} must match the new password` : "";
};
export const mustNotMatchCurrentPassword = (fieldName: string) => (v: string) => {
  const passwordInput = form.querySelector<HTMLInputElement>(`#current input`);
  if (!passwordInput) throw new Error("Could not find password input element");

  return passwordInput.value === v ? `${fieldName} must not match the current password` : "";
};
export const mustNotIncludeUsername = (fieldName: string) => (v: string) => {
  const passwordInput = form.querySelector<HTMLInputElement>(`#username input`);
  if (!passwordInput) throw new Error("Could not find username input element");

  return v.includes(passwordInput.value) ? `${fieldName} must not include the username` : "";
};

export const toggleValidator = (validate: (v: string) => string, enabled: boolean) => (v: string) =>
  enabled ? validate(v) : "";

export const isValidEmail = (v: string) => {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return !emailRegex.test(v) ? "The input must be a valid email address" : "";
};
