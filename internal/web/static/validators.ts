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
const specialCharsString = specialCharacters.join(", ");

const pluralize = (singular: string, amount: number) => (amount === 1 ? singular : singular + "s");

const form = document.querySelector<HTMLFormElement>("#form");
if (!form) throw new Error("Could not find form element");

const submitButton = form.querySelector<HTMLButtonElement>("button[type='submit']");
if (!submitButton) throw new Error("Could not find submit button element");

export const mustNotBeEmpty = (v: string) => (v.length === 0 ? "The input must not be empty" : "");
export const mustBeLongerThan = (minLength: number) => (v: string) =>
  v.length < minLength ? `The input must be at least ${minLength} ${pluralize("character", minLength)} long` : "";
export const mustIncludeNumbers = (amount: number) => (v: string) =>
  v.split("").filter((c) => !isNaN(+c)).length < amount
    ? `The input must include at least ${amount} ${pluralize("number", amount)}`
    : "";
export const mustIncludeSymbols = (amount: number) => (v: string) =>
  v.split("").filter((c) => specialCharacters.includes(c)).length < amount
    ? `The input must include at least ${amount} ${pluralize("symbol", amount)}: ${specialCharsString}}`
    : "";
export const mustIncludeUppercase = (amount: number) => (v: string) =>
  v.split("").filter((c) => c === c.toUpperCase() && c !== c.toLowerCase()).length < amount
    ? `The input must include at least ${amount} uppercase ${pluralize("character", amount)}`
    : "";
export const mustIncludeLowercase = (amount: number) => (v: string) =>
  v.split("").filter((c) => c === c.toLowerCase() && c !== c.toUpperCase()).length < amount
    ? `The input must include at least ${amount} lowercase ${pluralize("character", amount)}`
    : "";

export const mustMatchNewPassword = (v: string) => {
  const passwordInput = form.querySelector<HTMLInputElement>(`#new input`);
  if (!passwordInput) throw new Error("Could not find password input element");

  return passwordInput.value !== v ? "The input must match the new password" : "";
};
export const mustNotMatchCurrentPassword = (v: string) => {
  const passwordInput = form.querySelector<HTMLInputElement>(`#current input`);
  if (!passwordInput) throw new Error("Could not find password input element");

  return passwordInput.value === v ? "The input must not match the current password" : "";
};
export const mustNotIncludeUsername = (v: string) => {
  const passwordInput = form.querySelector<HTMLInputElement>(`#username input`);
  if (!passwordInput) throw new Error("Could not find username input element");

  return v.includes(passwordInput.value) ? "The input must not include the username" : "";
};

export const toggleValidator = (validate: (v: string) => string, enabled: boolean) => (v: string) =>
  enabled ? validate(v) : "";
