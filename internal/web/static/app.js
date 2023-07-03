// @ts-check

const opts = {
  minLength: +"{{ .opts.MinLength }}",
  minNumbers: +"{{ .opts.MinNumbers }}",
  minSymbols: +"{{ .opts.MinSymbols }}",
  minUppercase: +"{{ .opts.MinUppercase }}",
  minLowercase: +"{{ .opts.MinLowercase }}"
};

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

/**
 * @param {string} singular
 * @param {number} amount
 */
const pluralize = (singular, amount) => (amount === 1 ? singular : singular + "s");

/** @type {HTMLFormElement | null} */
const form = document.querySelector("#form");
if (!form) throw new Error("Could not find form element");

/** @type {HTMLButtonElement | null} */
const submitButton = form.querySelector("button[type='submit']");
if (!submitButton) throw new Error("Could not find submit button element");

/** @param {string} v */
const mustNotBeEmpty = (v) => (v.length === 0 ? "The input must not be empty" : "");
/** @param {number} minLength */
const mustBeLongerThan =
  (minLength) =>
  /** @param {string} v */
  (v) =>
    v.length < minLength ? `The input must be at least ${minLength} ${pluralize("character", minLength)} long` : "";
/** @param {number} amount */
const mustIncludeNumbers =
  (amount) =>
  /** @param {string} v */
  (v) =>
    v.split("").filter((c) => !isNaN(+c)).length < amount
      ? `The input must include at least ${amount} ${pluralize("number", amount)}`
      : "";
/** @param {number} amount */
const mustIncludeSymbols =
  (amount) =>
  /** @param {string} v */
  (v) =>
    v.split("").filter((c) => specialCharacters.includes(c)).length < amount
      ? `The input must include at least ${amount} ${pluralize("symbol", amount)}: ${specialCharsString}}`
      : "";
/** @param {number} amount */
const mustIncludeUppercase =
  (amount) =>
  /** @param {string} v */
  (v) =>
    v.split("").filter((c) => c === c.toUpperCase() && c !== c.toLowerCase()).length < amount
      ? `The input must include at least ${amount} uppercase ${pluralize("character", amount)}`
      : "";
/** @param {number} amount */
const mustIncludeLowercase =
  (amount) =>
  /** @param {string} v */
  (v) =>
    v.split("").filter((c) => c === c.toLowerCase() && c !== c.toUpperCase()).length < amount
      ? `The input must include at least ${amount} lowercase ${pluralize("character", amount)}`
      : "";

/** @param {string} v */
const mustMatchNewPassword = (v) => {
  /** @type {HTMLInputElement | null} */
  const passwordInput = form.querySelector(`#new input`);
  if (!passwordInput) throw new Error("Could not find password input element");

  return passwordInput.value !== v ? "The input must match the new password" : "";
};
/** @param {string} v */
const mustNotMatchCurrentPassword = (v) => {
  /** @type {HTMLInputElement | null} */
  const passwordInput = form.querySelector(`#current input`);
  if (!passwordInput) throw new Error("Could not find password input element");

  return passwordInput.value === v ? "The input must not match the current password" : "";
};

/**
 * @param {(v: string) => string} validate
 * @param {boolean} enabled
 */
const toggleValidator =
  (validate, enabled) =>
  /** @param {string} v */
  (v) =>
    enabled ? validate(v) : "";

/**
 * @typedef {[string, ((v: string) => string)[]]} Field
 */

/** @type {Field[]} */
const fieldsWithValidators = [
  ["username", [mustNotBeEmpty]],
  ["current", [mustNotBeEmpty]],
  [
    "new",
    [
      mustNotBeEmpty,
      mustBeLongerThan(opts.minLength),
      mustNotMatchCurrentPassword,
      toggleValidator(mustIncludeNumbers(opts.minNumbers), opts.minNumbers > 0),
      toggleValidator(mustIncludeSymbols(opts.minSymbols), opts.minSymbols > 0),
      toggleValidator(mustIncludeUppercase(opts.minUppercase), opts.minUppercase > 0),
      toggleValidator(mustIncludeLowercase(opts.minLowercase), opts.minLowercase > 0)
    ]
  ],
  ["new2", [mustNotBeEmpty, mustMatchNewPassword]]
];

const fields = fieldsWithValidators.map(([name, validators]) => {
  /** @type {HTMLDivElement | null} */
  const f = form.querySelector(`#${name}`);
  if (!f) throw new Error(`Field "${name}" does not exist`);

  /** @type {HTMLDivElement | null} */
  const inputContainer = f.querySelector('div[data-purpose="inputContainer"]');
  if (!inputContainer) throw new Error(`Input container for "${name}" does not exist`);

  /** @type {HTMLInputElement | null} */
  const input = inputContainer.querySelector("input");
  if (!input) throw new Error(`Input for "${name}" does not exist`);

  /** @type {HTMLButtonElement | null} */
  const revealButton = inputContainer.querySelector('button[data-purpose="reveal"]');
  if (!revealButton && input.type === "password") throw new Error(`Reveal button for "${name}" does not exist`);

  /** @type {HTMLDivElement | null} */
  const errorContainer = f.querySelector('div[data-purpose="errorContainer"]');
  if (!errorContainer) throw new Error(`Error for "${name}" does not exist`);

  const getValue = () => input.value;
  /**
   * @param {string[]} errors
   */
  const setErrors = (errors) => {
    errorContainer.innerHTML = "";

    if (errors.length > 0) {
      inputContainer.classList.add("border-red-500");
    } else {
      inputContainer.classList.remove("border-red-500");
    }

    for (const error of errors) {
      const el = document.createElement("p");
      el.innerText = error;

      errorContainer.appendChild(el);
    }
  };

  const validate = () => {
    const value = getValue();

    const errors = validators
      .map((validate) => validate(value))
      .reduce((/** @type {string[]} */ acc, v) => {
        if (v.length > 0) acc.push(v);

        return acc;
      }, []);

    console.log(`Validated "${name}": ${errors.length} error(s)`);

    setErrors(errors);

    return errors.length > 0;
  };

  if (revealButton) {
    revealButton.onclick = (e) => {
      e.preventDefault();
      e.stopPropagation();

      const newType = input.type === "password" ? "text" : "password";
      const revealed = newType === "text";

      console.log(`${revealed ? "Showing" : "Hiding"} content of "${name}"`);

      input.type = newType;
      f.dataset["revealed"] = revealed.toString();
    };
  }

  return { input, errorContainer, getValue, validate };
});

form.onsubmit = async (e) => {
  e.preventDefault();
  e.stopPropagation();

  const [username, oldPassword, newPassword] = fields.map((f) => f.getValue());

  const hasErrors = fields.map(({ validate }) => validate()).some((e) => e === true);
  submitButton.disabled = hasErrors;
  if (hasErrors) return;

  console.log("Changing password...");
  [submitButton, ...fields.map(({ input }) => input)].forEach((el) => (el.disabled = true));
  submitButton.dataset["loading"] = "true";

  try {
    const res = await fetch("/api/rpc", {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify({
        method: "change-password",
        params: [username, oldPassword, newPassword]
      })
    });

    const body = await res.text();

    if (!res.ok) {
      let err = body;

      try {
        const parsed = JSON.parse(body);

        err = parsed.data[0];
      } catch (e) {}

      throw new Error(`An error occurred: ${err}`);
    }

    console.log("Changed successfully");
  } catch (e) {
    console.log(e);
  }
};

form.onchange = (e) => {
  e.stopPropagation();

  const hasErrors = fields.map(({ validate }) => validate()).some((e) => e === true);
  submitButton.disabled = hasErrors;
};
