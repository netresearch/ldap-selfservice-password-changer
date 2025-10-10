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

/**
 * Validates that a field is not empty.
 * @param fieldName - Name of the field for error messages
 * @returns Validator function that checks if value is not empty
 * @example
 * const validator = mustNotBeEmpty("Username");
 * validator(""); // Returns: "Username must not be empty"
 * validator("john"); // Returns: ""
 */
export const mustNotBeEmpty = (fieldName: string) => (v: string) =>
  v.length === 0 ? `${fieldName} must not be empty` : "";

/**
 * Validates that a field meets minimum length requirement.
 * @param minLength - Minimum number of characters required
 * @param fieldName - Name of the field for error messages
 * @returns Validator function that checks minimum length
 * @example
 * const validator = mustBeLongerThan(8, "Password");
 * validator("12345"); // Returns: "Password must be at least 8 characters long"
 * validator("12345678"); // Returns: ""
 */
export const mustBeLongerThan = (minLength: number, fieldName: string) => (v: string) =>
  v.length < minLength
    ? `${fieldName} must be at least ${minLength.toString()} ${pluralize("character", minLength)} long`
    : "";

/**
 * Validates that a field contains minimum required numeric digits.
 * @param amount - Minimum number of digits required
 * @param fieldName - Name of the field for error messages
 * @returns Validator function that checks for numeric characters
 * @example
 * const validator = mustIncludeNumbers(2, "Password");
 * validator("abc1"); // Returns: "Password must include at least 2 numbers"
 * validator("abc12"); // Returns: ""
 */
export const mustIncludeNumbers = (amount: number, fieldName: string) => (v: string) =>
  v.split("").filter((c) => !isNaN(+c)).length < amount
    ? `${fieldName} must include at least ${amount.toString()} ${pluralize("number", amount)}`
    : "";

/**
 * Validates that a field contains minimum required special characters.
 * Special characters include: ! " # $ % & ' ( ) * + , - . / : ; < = > ? @ [ \ ] ^ _ ` { | } ~
 * @param amount - Minimum number of special characters required
 * @param fieldName - Name of the field for error messages
 * @returns Validator function that checks for special characters
 * @example
 * const validator = mustIncludeSymbols(1, "Password");
 * validator("abc123"); // Returns: "Password must include at least 1 special character (such as !, @, #, $, %)"
 * validator("abc123!"); // Returns: ""
 */
export const mustIncludeSymbols = (amount: number, fieldName: string) => (v: string) =>
  v.split("").filter((c) => specialCharacters.includes(c)).length < amount
    ? `${fieldName} must include at least ${amount.toString()} special ${pluralize("character", amount)} (such as !, @, #, $, %)`
    : "";

/**
 * Validates that a field contains minimum required uppercase letters.
 * @param amount - Minimum number of uppercase letters required
 * @param fieldName - Name of the field for error messages
 * @returns Validator function that checks for uppercase letters
 * @example
 * const validator = mustIncludeUppercase(1, "Password");
 * validator("abc123"); // Returns: "Password must include at least 1 uppercase character"
 * validator("Abc123"); // Returns: ""
 */
export const mustIncludeUppercase = (amount: number, fieldName: string) => (v: string) =>
  v.split("").filter((c) => c === c.toUpperCase() && c !== c.toLowerCase()).length < amount
    ? `${fieldName} must include at least ${amount.toString()} uppercase ${pluralize("character", amount)}`
    : "";

/**
 * Validates that a field contains minimum required lowercase letters.
 * @param amount - Minimum number of lowercase letters required
 * @param fieldName - Name of the field for error messages
 * @returns Validator function that checks for lowercase letters
 * @example
 * const validator = mustIncludeLowercase(1, "Password");
 * validator("ABC123"); // Returns: "Password must include at least 1 lowercase character"
 * validator("Abc123"); // Returns: ""
 */
export const mustIncludeLowercase = (amount: number, fieldName: string) => (v: string) =>
  v.split("").filter((c) => c === c.toLowerCase() && c !== c.toUpperCase()).length < amount
    ? `${fieldName} must include at least ${amount.toString()} lowercase ${pluralize("character", amount)}`
    : "";

/**
 * Validates that a field matches the new password field.
 * Used for password confirmation fields to ensure passwords match.
 * @param fieldName - Name of the field for error messages
 * @returns Validator function that checks if value matches new password
 * @example
 * const validator = mustMatchNewPassword("Password Confirmation");
 * // Assuming new_password field contains "SecurePass123"
 * validator("WrongPass"); // Returns: "Password Confirmation must match the new password"
 * validator("SecurePass123"); // Returns: ""
 */
export const mustMatchNewPassword = (fieldName: string) => (v: string) => {
  const passwordInput = form.querySelector<HTMLInputElement>(`#new_password input`);
  if (!passwordInput) throw new Error("Could not find password input element");

  return passwordInput.value !== v ? `${fieldName} must match the new password` : "";
};

/**
 * Validates that a field does not match the current password field.
 * Ensures users change to a different password, not reuse current one.
 * @param fieldName - Name of the field for error messages
 * @returns Validator function that checks value doesn't match current password
 * @example
 * const validator = mustNotMatchCurrentPassword("New Password");
 * // Assuming current_password field contains "OldPass123"
 * validator("OldPass123"); // Returns: "New Password must not match the current password"
 * validator("NewPass456"); // Returns: ""
 */
export const mustNotMatchCurrentPassword = (fieldName: string) => (v: string) => {
  const passwordInput = form.querySelector<HTMLInputElement>(`#current_password input`);
  if (!passwordInput) throw new Error("Could not find password input element");

  return passwordInput.value === v ? `${fieldName} must not match the current password` : "";
};

/**
 * Validates that a field does not contain the username.
 * Security requirement to prevent weak passwords containing username.
 * @param fieldName - Name of the field for error messages
 * @returns Validator function that checks value doesn't include username
 * @example
 * const validator = mustNotIncludeUsername("Password");
 * // Assuming username field contains "john"
 * validator("john123"); // Returns: "Password must not include the username"
 * validator("SecurePass123"); // Returns: ""
 */
export const mustNotIncludeUsername = (fieldName: string) => (v: string) => {
  const passwordInput = form.querySelector<HTMLInputElement>(`#username input`);
  if (!passwordInput) throw new Error("Could not find username input element");

  return v.toLowerCase().includes(passwordInput.value.toLowerCase())
    ? `${fieldName} must not include the username`
    : "";
};

/**
 * Conditionally enables or disables a validator based on a boolean flag.
 * Useful for optional validation rules that can be configured.
 * @param validate - The validator function to conditionally apply
 * @param enabled - Whether the validator should be active
 * @returns Validator function that applies validation only when enabled
 * @example
 * const usernameValidator = mustNotIncludeUsername("Password");
 * const conditionalValidator = toggleValidator(usernameValidator, passwordCanIncludeUsername);
 * // If passwordCanIncludeUsername is false, validation runs
 * // If passwordCanIncludeUsername is true, validation is skipped
 */
export const toggleValidator = (validate: (v: string) => string, enabled: boolean) => (v: string) =>
  enabled ? validate(v) : "";

/**
 * Validates that a field contains a valid email address format.
 * Uses a comprehensive regex pattern following RFC 5322 guidelines.
 * Validates:
 * - Local part: alphanumeric, dots, hyphens, underscores, plus signs
 * - Domain: alphanumeric, dots, hyphens
 * - TLD: 2-63 characters (e.g., .com, .museum)
 * @param v - The email value to validate
 * @returns Error message if invalid, empty string if valid
 * @example
 * isValidEmail("invalid"); // Returns: "The input must be a valid email address"
 * isValidEmail("user@example.com"); // Returns: ""
 * isValidEmail("user+tag@sub.example.co.uk"); // Returns: ""
 */
export const isValidEmail = (v: string) => {
  // RFC 5322 compliant email regex (simplified but comprehensive)
  const emailRegex = /^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,63}$/;
  return !emailRegex.test(v) ? "The input must be a valid email address" : "";
};
