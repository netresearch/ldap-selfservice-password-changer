import {
  mustBeLongerThan,
  mustIncludeLowercase,
  mustIncludeNumbers,
  mustIncludeSymbols,
  mustIncludeUppercase,
  mustMatchNewPassword,
  mustNotBeEmpty,
  mustNotIncludeUsername,
  mustNotMatchCurrentPassword,
  toggleValidator
} from "./validators.js";

type Opts = {
  minLength: number;
  minNumbers: number;
  minSymbols: number;
  minUppercase: number;
  minLowercase: number;
  passwordCanIncludeUsername: boolean;
};

export const init = (opts: Opts) => {
  const successContainer = document.querySelector<HTMLFormElement>("div[data-purpose='successContainer']");
  if (!successContainer) throw new Error("Could not find success container element");

  const form = document.querySelector<HTMLFormElement>("#form");
  if (!form) throw new Error("Could not find form element");

  const submitButton = form.querySelector<HTMLButtonElement>("& > div[data-purpose='submit'] > button[type='submit']");
  if (!submitButton) throw new Error("Could not find submit button element");

  const submitErrorContainer = form.querySelector<HTMLDivElement>(
    "& > div[data-purpose='submit'] > div[data-purpose='errors']"
  );
  if (!submitErrorContainer) throw new Error("Could not find submit error container element");

  type Field = [string, ((v: string) => string)[]];

  const fieldsWithValidators = [
    ["username", [mustNotBeEmpty]],
    ["current", [mustNotBeEmpty]],
    [
      "new",
      [
        mustNotBeEmpty,
        mustBeLongerThan(opts.minLength),
        mustNotMatchCurrentPassword,
        toggleValidator(mustNotIncludeUsername, !opts.passwordCanIncludeUsername),
        mustIncludeNumbers(opts.minNumbers),
        mustIncludeSymbols(opts.minSymbols),
        mustIncludeUppercase(opts.minUppercase),
        mustIncludeLowercase(opts.minLowercase)
      ]
    ],
    ["new2", [mustNotBeEmpty, mustMatchNewPassword]]
  ] satisfies Field[];

  const fields = fieldsWithValidators.map(([name, validators]) => {
    const f = form.querySelector<HTMLDivElement>(`#${name}`);
    if (!f) throw new Error(`Field "${name}" does not exist`);

    const inputContainer = f.querySelector<HTMLDivElement>('div[data-purpose="inputContainer"]');
    if (!inputContainer) throw new Error(`Input container for "${name}" does not exist`);

    const input = inputContainer.querySelector<HTMLInputElement>("input");
    if (!input) throw new Error(`Input for "${name}" does not exist`);

    const revealButton = inputContainer.querySelector<HTMLButtonElement>('button[data-purpose="reveal"]');
    if (!revealButton && input.type === "password") throw new Error(`Reveal button for "${name}" does not exist`);

    const errorContainer = f.querySelector<HTMLDivElement>('div[data-purpose="errors"]');
    if (!errorContainer) throw new Error(`Error for "${name}" does not exist`);

    const getValue = () => input.value;
    const setErrors = (errors: string[]) => {
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
        .reduce((acc, v) => {
          if (v.length > 0) acc.push(v);

          return acc;
        }, [] as string[]);

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

  const toggleFields = (enabled: boolean) => {
    [submitButton, ...fields.map(({ input }) => input)].forEach((el) => (el.disabled = !enabled));
    submitButton.dataset["loading"] = (!enabled).toString();
  };

  form.onsubmit = async (e) => {
    e.preventDefault();
    e.stopPropagation();

    const [username, oldPassword, newPassword] = fields.map((f) => f.getValue());

    const hasErrors = fields.map(({ validate }) => validate()).some((e) => e === true);
    submitButton.disabled = hasErrors;
    if (hasErrors) return;

    console.log("Changing password...");
    toggleFields(false);

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

      form.style.display = "none";
      successContainer.style.display = "block";
    } catch (e) {
      console.error(e);

      submitErrorContainer.innerText = (e as Error).message;

      // Re-enable inputs but keep the submit button disabled,
      // since we know that this isn't going to work. After the validators
      // successfully re-run, it will enable the submit button again.
      toggleFields(true);
      submitButton.disabled = true;
    }
  };

  form.onchange = (e) => {
    e.stopPropagation();

    const hasErrors = fields.map(({ validate }) => validate()).some((e) => e === true);
    submitButton.disabled = hasErrors;
  };
};
