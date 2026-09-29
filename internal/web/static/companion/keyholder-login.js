import "./login-companion.js";

// The existing form emits this only after validation and bot verification.
// No field values are passed to the companion and the request is never delayed.
document.getElementById("form")?.addEventListener("password-change-start", () => {
  document.querySelector('login-companion[character="keyholder"]')?.performAction();
});
